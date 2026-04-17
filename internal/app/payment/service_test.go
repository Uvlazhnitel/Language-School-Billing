package payment

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"langschool/ent"
	"langschool/ent/enttest"
	entinvoice "langschool/ent/invoice"
	entpayment "langschool/ent/payment"
	"langschool/internal/app"
)

func TestCreateDirectPaymentUpdatesInvoiceSummaryAndStatus(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Arina Osipova")
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 3, 100, app.InvoiceStatusIssued)

	invoiceID := inv.ID
	if _, err := svc.Create(ctx, st.ID, &invoiceID, 30, app.PaymentMethodCash, "2026-03-12", "partial"); err != nil {
		t.Fatalf("create partial payment: %v", err)
	}

	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary after partial payment: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 30)
	assertFloatEqual(t, summary.Remaining, 70)
	assertEqual(t, summary.Status, app.InvoiceStatusIssued)

	if _, err := svc.Create(ctx, st.ID, &invoiceID, 70, app.PaymentMethodBank, "2026-03-15", "final"); err != nil {
		t.Fatalf("create final payment: %v", err)
	}

	summary, err = svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary after final payment: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 100)
	assertFloatEqual(t, summary.Remaining, 0)
	assertEqual(t, summary.Status, app.InvoiceStatusPaid)
}

func TestCreateGlobalPaymentAllocatesToOldestInvoices(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Mila Petrova")
	oldest := createTestInvoice(t, ctx, client, st.ID, 2026, 1, 50, app.InvoiceStatusIssued)
	newer := createTestInvoice(t, ctx, client, st.ID, 2026, 2, 30, app.InvoiceStatusIssued)

	dto, err := svc.Create(ctx, st.ID, nil, 60, app.PaymentMethodCash, "2026-02-20", "debtor payment")
	if err != nil {
		t.Fatalf("create global payment: %v", err)
	}
	if dto.InvoiceID == nil || *dto.InvoiceID != oldest.ID {
		t.Fatalf("expected first allocated payment to target oldest invoice %d, got %+v", oldest.ID, dto.InvoiceID)
	}

	payments, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID)).
		Order(ent.Asc(entpayment.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query payments: %v", err)
	}
	if len(payments) != 2 {
		t.Fatalf("expected 2 allocated payments, got %d", len(payments))
	}

	assertLinkedPayment(t, payments[0], oldest.ID, 50)
	assertLinkedPayment(t, payments[1], newer.ID, 10)

	oldestSummary, err := svc.InvoiceSummary(ctx, oldest.ID)
	if err != nil {
		t.Fatalf("oldest invoice summary: %v", err)
	}
	assertFloatEqual(t, oldestSummary.Paid, 50)
	assertFloatEqual(t, oldestSummary.Remaining, 0)
	assertEqual(t, oldestSummary.Status, app.InvoiceStatusPaid)

	newerSummary, err := svc.InvoiceSummary(ctx, newer.ID)
	if err != nil {
		t.Fatalf("newer invoice summary: %v", err)
	}
	assertFloatEqual(t, newerSummary.Paid, 10)
	assertFloatEqual(t, newerSummary.Remaining, 20)
	assertEqual(t, newerSummary.Status, app.InvoiceStatusIssued)

	balance, err := svc.StudentBalance(ctx, st.ID)
	if err != nil {
		t.Fatalf("student balance: %v", err)
	}
	assertFloatEqual(t, balance.TotalInvoiced, 80)
	assertFloatEqual(t, balance.TotalPaid, 60)
	assertFloatEqual(t, balance.Debt, 20)
}

func TestCreateGlobalPaymentLeavesExtraAsCredit(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Nikita Smirnov")
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 4, 50, app.InvoiceStatusIssued)

	if _, err := svc.Create(ctx, st.ID, nil, 80, app.PaymentMethodBank, "2026-04-25", "overpayment"); err != nil {
		t.Fatalf("create overpayment: %v", err)
	}

	payments, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID)).
		Order(ent.Asc(entpayment.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query payments: %v", err)
	}
	if len(payments) != 2 {
		t.Fatalf("expected 2 payments (invoice + credit), got %d", len(payments))
	}

	assertLinkedPayment(t, payments[0], inv.ID, 50)
	if payments[1].InvoiceID != nil {
		t.Fatalf("expected extra payment to stay unlinked as credit, got invoice_id=%v", *payments[1].InvoiceID)
	}
	assertFloatEqual(t, payments[1].Amount, 30)

	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 50)
	assertFloatEqual(t, summary.Remaining, 0)
	assertEqual(t, summary.Status, app.InvoiceStatusPaid)

	balance, err := svc.StudentBalance(ctx, st.ID)
	if err != nil {
		t.Fatalf("student balance: %v", err)
	}
	assertFloatEqual(t, balance.TotalInvoiced, 50)
	assertFloatEqual(t, balance.TotalPaid, 80)
	assertFloatEqual(t, balance.Balance, 30)
	assertFloatEqual(t, balance.Debt, 0)
}

func TestDeletePaymentRevertsPaidInvoiceBackToIssued(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Diana Volkova")
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 5, 40, app.InvoiceStatusIssued)

	invoiceID := inv.ID
	created, err := svc.Create(ctx, st.ID, &invoiceID, 40, app.PaymentMethodCash, "2026-05-10", "full payment")
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete payment: %v", err)
	}

	updatedInvoice, err := client.Invoice.Get(ctx, inv.ID)
	if err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	assertEqual(t, string(updatedInvoice.Status), app.InvoiceStatusIssued)

	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary after delete: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 0)
	assertFloatEqual(t, summary.Remaining, 40)
	assertEqual(t, summary.Status, app.InvoiceStatusIssued)
}

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	return enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
}

func createTestStudent(t *testing.T, ctx context.Context, client *ent.Client, fullName string) *ent.Student {
	t.Helper()
	st, err := client.Student.Create().
		SetFullName(fullName).
		Save(ctx)
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	return st
}

func createTestInvoice(t *testing.T, ctx context.Context, client *ent.Client, studentID, year, month int, total float64, status string) *ent.Invoice {
	t.Helper()
	inv, err := client.Invoice.Create().
		SetStudentID(studentID).
		SetPeriodYear(year).
		SetPeriodMonth(month).
		SetTotalAmount(total).
		SetStatus(entinvoice.Status(status)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	return inv
}

func assertLinkedPayment(t *testing.T, p *ent.Payment, invoiceID int, amount float64) {
	t.Helper()
	if p.InvoiceID == nil {
		t.Fatalf("expected payment to be linked to invoice %d, got nil", invoiceID)
	}
	if *p.InvoiceID != invoiceID {
		t.Fatalf("expected linked invoice %d, got %d", invoiceID, *p.InvoiceID)
	}
	assertFloatEqual(t, p.Amount, amount)
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertFloatEqual(t *testing.T, got, want float64) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.0001 {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}
