package payment

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"langschool/ent"
	"langschool/ent/enttest"
	"langschool/internal/app"
	"langschool/internal/apperrors"
)

func operationClient(t *testing.T, path string) *ent.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:"+path)
	t.Cleanup(func() { client.Close() })
	return client
}

func TestPaymentOperationReplayAfterRestartAndDeletion(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "payments.sqlite")
	client := operationClient(t, path)
	svc := New(client)
	student := createTestStudent(t, ctx, client, "Replay")
	iv := createTestInvoice(t, ctx, client, student.ID, 2026, 10, 10, app.InvoiceStatusIssued)
	key := uuid.NewString()
	first, replayed, err := svc.CreateWithOperation(ctx, 1, key, student.ID, &iv.ID, 10, "cash", "2026-10-19", "test")
	if err != nil || replayed {
		t.Fatalf("first: replay=%v err=%v", replayed, err)
	}
	// A replay must succeed even though the invoice is now paid in full.
	second, replayed, err := svc.CreateWithOperation(ctx, 1, key, student.ID, &iv.ID, 10, "cash", "2026-10-19", "test")
	if err != nil || !replayed || !reflect.DeepEqual(first, second) {
		t.Fatalf("repeat: result=%+v replay=%v err=%v", second, replayed, err)
	}
	if err := svc.Delete(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := operationClient(t, path)
	third, replayed, err := New(reopened).CreateWithOperation(ctx, 1, key, student.ID, &iv.ID, 10, "cash", "2026-10-19", "test")
	if err != nil || !replayed || !reflect.DeepEqual(first, third) {
		t.Fatalf("after reopen/delete: %+v %v %v", third, replayed, err)
	}
	if count := reopened.Payment.Query().CountX(ctx); count != 0 {
		t.Fatalf("deleted payment recreated: count=%d", count)
	}
}

func TestPaymentOperationAllocationsAndIdentity(t *testing.T) {
	ctx := context.Background()
	client := operationClient(t, filepath.Join(t.TempDir(), "payments.sqlite"))
	svc := New(client)
	student := createTestStudent(t, ctx, client, "Allocated")
	createTestInvoice(t, ctx, client, student.ID, 2026, 9, 10, app.InvoiceStatusIssued)
	createTestInvoice(t, ctx, client, student.ID, 2026, 10, 20, app.InvoiceStatusIssued)
	key := uuid.NewString()
	first, _, err := svc.CreateWithOperation(ctx, 1, key, student.ID, nil, 40, "bank", "2026-10-19", "test")
	if err != nil {
		t.Fatal(err)
	}
	second, replayed, err := svc.CreateWithOperation(ctx, 1, key, student.ID, nil, 40, "bank", "2026-10-19", "test")
	if err != nil || !replayed || !reflect.DeepEqual(first, second) {
		t.Fatalf("repeat allocations: %v %v", replayed, err)
	}
	if count := client.Payment.Query().CountX(ctx); count != 3 {
		t.Fatalf("allocation count=%d want 3", count)
	}
	balance, err := svc.StudentBalance(ctx, student.ID)
	if err != nil || balance.TotalPaid != 40 || balance.Balance != 10 {
		t.Fatalf("balance=%+v err=%v", balance, err)
	}
	for _, change := range []struct {
		name               string
		student            int
		invoice            *int
		amount             float64
		method, date, note string
	}{
		{"amount", student.ID, nil, 41, "bank", "2026-10-19", "test"},
		{"student", student.ID + 1, nil, 40, "bank", "2026-10-19", "test"},
		{"invoice", student.ID, &first.ID, 40, "bank", "2026-10-19", "test"},
		{"method", student.ID, nil, 40, "cash", "2026-10-19", "test"},
		{"date", student.ID, nil, 40, "bank", "2026-10-20", "test"},
		{"note", student.ID, nil, 40, "bank", "2026-10-19", "changed"},
	} {
		t.Run(change.name, func(t *testing.T) {
			_, _, err := svc.CreateWithOperation(ctx, 1, key, change.student, change.invoice, change.amount, change.method, change.date, change.note)
			if !apperrors.IsConflict(err) {
				t.Fatalf("want conflict, got %v", err)
			}
		})
	}
	// Different operations and different actors are independently valid.
	for _, operation := range []struct {
		actor int
		key   string
	}{{1, uuid.NewString()}, {2, key}} {
		_, replayed, err := svc.CreateWithOperation(ctx, operation.actor, operation.key, student.ID, nil, 40, "bank", "2026-10-19", "test")
		if err != nil || replayed {
			t.Fatalf("independent operation: %v %v", replayed, err)
		}
	}
	balance, err = svc.StudentBalance(ctx, student.ID)
	if err != nil || balance.TotalPaid != 120 {
		t.Fatalf("independent total: %+v %v", balance, err)
	}
}

func TestPaymentOperationConcurrentRequests(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "payments.sqlite")
	one := operationClient(t, path)
	two := operationClient(t, path)
	student := createTestStudent(t, ctx, one, "Concurrent")
	key := uuid.NewString()
	var wg sync.WaitGroup
	var fresh atomic.Int32
	results := make(chan *PaymentDTO, 8)
	errors := make(chan error, 8)
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		client := one
		if i%2 == 0 {
			client = two
		}
		wg.Add(1)
		go func(client *ent.Client) {
			defer wg.Done()
			<-start
			item, replayed, err := New(client).CreateWithOperation(ctx, 1, key, student.ID, nil, 10, "cash", "2026-10-19", "")
			if err != nil {
				errors <- err
				return
			}
			if !replayed {
				fresh.Add(1)
			}
			results <- item
		}(client)
	}
	close(start)
	wg.Wait()
	close(errors)
	close(results)
	for err := range errors {
		t.Error(err)
	}
	var first *PaymentDTO
	for item := range results {
		if first == nil {
			first = item
		}
		if !reflect.DeepEqual(first, item) {
			t.Errorf("different replay result: %+v", item)
		}
	}
	if fresh.Load() != 1 || one.Payment.Query().CountX(ctx) != 1 || one.PaymentOperation.Query().CountX(ctx) != 1 {
		t.Fatalf("concurrent writes: fresh=%d", fresh.Load())
	}
}

func TestPaymentOperationRollbackAndValidation(t *testing.T) {
	ctx := context.Background()
	client := operationClient(t, filepath.Join(t.TempDir(), "payments.sqlite"))
	svc := New(client)
	student := createTestStudent(t, ctx, client, "Rollback")
	iv := createTestInvoice(t, ctx, client, student.ID, 2026, 10, 10, app.InvoiceStatusIssued)
	var calls atomic.Int32
	failure := errors.New("simulated second allocation failure")
	client.Payment.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if calls.Add(1) == 2 {
				return nil, failure
			}
			return next.Mutate(ctx, m)
		})
	})
	key := uuid.NewString()
	_, _, err := svc.CreateWithOperation(ctx, 1, key, student.ID, nil, 20, "cash", "2026-10-19", "")
	if !errors.Is(err, failure) {
		t.Fatalf("expected injected failure: %v", err)
	}
	if client.Payment.Query().CountX(ctx) != 0 || client.PaymentOperation.Query().CountX(ctx) != 0 {
		t.Fatal("partial operation persisted")
	}
	if got := client.Invoice.GetX(ctx, iv.ID).Status; string(got) != app.InvoiceStatusIssued {
		t.Fatalf("status changed on rollback: %s", got)
	}
	_, replayed, err := svc.CreateWithOperation(ctx, 1, key, student.ID, nil, 20, "cash", "2026-10-19", "")
	if err != nil || replayed {
		t.Fatalf("retry after rollback: %v %v", replayed, err)
	}
	for _, invalid := range []string{"", "not-a-uuid", uuid.Nil.String()} {
		_, _, err := svc.CreateWithOperation(ctx, 1, invalid, student.ID, nil, 10, "cash", "2026-10-19", "")
		if !errors.Is(err, ErrInvalidOperationID) {
			t.Fatalf("invalid UUID: %v", err)
		}
	}
}

func TestPaymentOperationSchemaAdditionPreservesExistingPayments(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	client := operationClient(t, path)
	student := createTestStudent(t, ctx, client, "Existing data")
	existing, err := New(client).Create(ctx, student.ID, nil, 10, "cash", "2026-10-19", "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	// Model the old schema, which has payments but no operation records.
	_, err = raw.ExecContext(ctx, "DROP TABLE payment_operations")
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	migrated := operationClient(t, path)
	if saved := migrated.Payment.GetX(ctx, existing.ID); saved.AmountCents != 1000 || saved.Note != "legacy" {
		t.Fatalf("existing payment changed: %+v", saved)
	}
	if migrated.PaymentOperation.Query().CountX(ctx) != 0 {
		t.Fatal("migration must not invent operations for old payments")
	}
	_, replayed, err := New(migrated).CreateWithOperation(ctx, 1, uuid.NewString(), student.ID, nil, 10, "cash", "2026-10-19", "new")
	if err != nil || replayed || migrated.Payment.Query().CountX(ctx) != 2 {
		t.Fatalf("new operation after migration: %v %v", replayed, err)
	}
}
