package recipient

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"langschool/ent/enttest"
)

func TestResolveInvoiceRecipient(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:recipienthelper?mode=memory&_fk=1")
	defer client.Close()

	t.Run("adult student uses self", func(t *testing.T) {
		st, err := client.Student.Create().
			SetFullName("Adult Student").
			SetPersonalCode("111111-11111").
			SetPhone("111").
			SetEmail("adult@example.com").
			SetIsMinor(false).
			Save(ctx)
		if err != nil {
			t.Fatalf("Student.Create: %v", err)
		}

		got, err := ResolveInvoiceRecipient(ctx, client, st.ID)
		if err != nil {
			t.Fatalf("ResolveInvoiceRecipient: %v", err)
		}
		if got.RecipientName != "Adult Student" {
			t.Fatalf("RecipientName = %q, want %q", got.RecipientName, "Adult Student")
		}
		if got.RecipientPhone != "111" {
			t.Fatalf("RecipientPhone = %q, want %q", got.RecipientPhone, "111")
		}
		if got.ChildName != "Adult Student" {
			t.Fatalf("ChildName = %q, want %q", got.ChildName, "Adult Student")
		}
		if got.StudentPersonalCode != "111111-11111" {
			t.Fatalf("StudentPersonalCode = %q, want %q", got.StudentPersonalCode, "111111-11111")
		}
		if got.IsMinor {
			t.Fatalf("IsMinor = true, want false")
		}
	})

	t.Run("minor uses payer fields", func(t *testing.T) {
		st, err := client.Student.Create().
			SetFullName("Child One").
			SetPersonalCode("222222-22222").
			SetPhone("300").
			SetEmail("payer@example.com").
			SetIsMinor(true).
			SetPayerName("Payer Adult").
			SetPayerRole("mother").
			Save(ctx)
		if err != nil {
			t.Fatalf("Student.Create: %v", err)
		}

		got, err := ResolveInvoiceRecipient(ctx, client, st.ID)
		if err != nil {
			t.Fatalf("ResolveInvoiceRecipient: %v", err)
		}
		if got.RecipientName != "Payer Adult" {
			t.Fatalf("RecipientName = %q, want %q", got.RecipientName, "Payer Adult")
		}
		if got.RecipientPhone != "300" {
			t.Fatalf("RecipientPhone = %q, want %q", got.RecipientPhone, "300")
		}
		if got.ChildName != "Child One" {
			t.Fatalf("ChildName = %q, want %q", got.ChildName, "Child One")
		}
		if got.StudentPersonalCode != "222222-22222" {
			t.Fatalf("StudentPersonalCode = %q, want %q", got.StudentPersonalCode, "222222-22222")
		}
		if !got.IsMinor {
			t.Fatalf("IsMinor = false, want true")
		}
	})

	t.Run("minor without payer falls back to child", func(t *testing.T) {
		st, err := client.Student.Create().
			SetFullName("Child Three").
			SetPersonalCode("333333-33333").
			SetPhone("500").
			SetEmail("child3@example.com").
			SetIsMinor(true).
			Save(ctx)
		if err != nil {
			t.Fatalf("Student.Create: %v", err)
		}

		got, err := ResolveInvoiceRecipient(ctx, client, st.ID)
		if err != nil {
			t.Fatalf("ResolveInvoiceRecipient: %v", err)
		}
		if got.RecipientName != "Child Three" {
			t.Fatalf("RecipientName = %q, want %q", got.RecipientName, "Child Three")
		}
		if got.StudentPersonalCode != "333333-33333" {
			t.Fatalf("StudentPersonalCode = %q, want %q", got.StudentPersonalCode, "333333-33333")
		}
	})
}
