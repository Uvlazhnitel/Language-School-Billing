package invoice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"langschool/internal/app"
)

func TestReopenDraft(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-reopen?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Invoice Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}
	crs, err := client.Course.Create().
		SetName("Test Course").
		SetType(course.TypeGroup).
		SetLessonPrice(20).
		SetSubscriptionPrice(60).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}
	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModeSubscription).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	t.Run("reopens issued invoice without payments and removes pdf", func(t *testing.T) {
		tmpDir := t.TempDir()
		number := "AL-202604-123"
		iv, err := client.Invoice.Create().
			SetStudentID(st.ID).
			SetPeriodYear(2026).
			SetPeriodMonth(4).
			SetTotalAmount(70).
			SetStatus(StatusIssued).
			SetNumber(number).
			Save(ctx)
		if err != nil {
			t.Fatalf("Invoice.Create: %v", err)
		}
		if _, err := client.InvoiceLine.Create().
			SetInvoiceID(iv.ID).
			SetEnrollmentID(enr.ID).
			SetDescription("Subscription").
			SetQty(1).
			SetUnitPrice(70).
			SetAmount(70).
			Save(ctx); err != nil {
			t.Fatalf("InvoiceLine.Create: %v", err)
		}

		pdfPath := PDFPathByNumber(tmpDir, iv.PeriodYear, iv.PeriodMonth, number)
		if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(pdfPath, []byte("demo"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		if err := svc.ReopenDraft(ctx, iv.ID, tmpDir); err != nil {
			t.Fatalf("ReopenDraft: %v", err)
		}

		got, err := client.Invoice.Get(ctx, iv.ID)
		if err != nil {
			t.Fatalf("Invoice.Get: %v", err)
		}
		if got.Status != StatusDraft {
			t.Fatalf("Status = %q, want %q", got.Status, StatusDraft)
		}
		if got.Number != nil {
			t.Fatalf("Number = %v, want nil", got.Number)
		}
		if got.TotalAmount != 70 {
			t.Fatalf("TotalAmount = %v, want 70", got.TotalAmount)
		}
		count, err := client.InvoiceLine.Query().Count(ctx)
		if err != nil {
			t.Fatalf("InvoiceLine.Count: %v", err)
		}
		if count != 1 {
			t.Fatalf("invoice line count = %d, want 1", count)
		}
		if _, err := os.Stat(pdfPath); !os.IsNotExist(err) {
			t.Fatalf("pdf still exists or unexpected err: %v", err)
		}
	})

	t.Run("fails when issued invoice has payments", func(t *testing.T) {
		number := "AL-202604-124"
		iv, err := client.Invoice.Create().
			SetStudentID(st.ID).
			SetPeriodYear(2026).
			SetPeriodMonth(8).
			SetTotalAmount(50).
			SetStatus(StatusIssued).
			SetNumber(number).
			Save(ctx)
		if err != nil {
			t.Fatalf("Invoice.Create: %v", err)
		}
		if _, err := client.Payment.Create().
			SetStudentID(st.ID).
			SetInvoiceID(iv.ID).
			SetAmount(10).
			SetMethod("cash").
			SetPaidAt(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)).
			SetCreatedAt(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)).
			Save(ctx); err != nil {
			t.Fatalf("Payment.Create: %v", err)
		}

		if err := svc.ReopenDraft(ctx, iv.ID, t.TempDir()); err == nil {
			t.Fatalf("expected ReopenDraft to fail for issued invoice with payments")
		}
	})

	t.Run("fails for paid invoice", func(t *testing.T) {
		iv, err := client.Invoice.Create().
			SetStudentID(st.ID).
			SetPeriodYear(2026).
			SetPeriodMonth(5).
			SetTotalAmount(30).
			SetStatus(StatusPaid).
			SetNumber("AL-202605-125").
			Save(ctx)
		if err != nil {
			t.Fatalf("Invoice.Create: %v", err)
		}
		if err := svc.ReopenDraft(ctx, iv.ID, t.TempDir()); err == nil {
			t.Fatalf("expected ReopenDraft to fail for paid invoice")
		}
	})

	t.Run("fails for canceled invoice", func(t *testing.T) {
		iv, err := client.Invoice.Create().
			SetStudentID(st.ID).
			SetPeriodYear(2026).
			SetPeriodMonth(9).
			SetTotalAmount(35).
			SetStatus(StatusCanceled).
			SetNumber("AL-202605-126").
			Save(ctx)
		if err != nil {
			t.Fatalf("Invoice.Create: %v", err)
		}
		if err := svc.ReopenDraft(ctx, iv.ID, t.TempDir()); err == nil {
			t.Fatalf("expected ReopenDraft to fail for canceled invoice")
		}
	})

	t.Run("fails for draft invoice", func(t *testing.T) {
		iv, err := client.Invoice.Create().
			SetStudentID(st.ID).
			SetPeriodYear(2026).
			SetPeriodMonth(6).
			SetTotalAmount(40).
			SetStatus(StatusDraft).
			Save(ctx)
		if err != nil {
			t.Fatalf("Invoice.Create: %v", err)
		}
		if err := svc.ReopenDraft(ctx, iv.ID, t.TempDir()); err == nil {
			t.Fatalf("expected ReopenDraft to fail for draft invoice")
		}
	})

	t.Run("does not roll back next sequence", func(t *testing.T) {
		if _, err := client.Settings.Create().
			SetSingletonID(app.SettingsSingletonID).
			SetOrgName("ArtLab").
			SetAddress("Latgales iela 260, Rīga, Latvija").
			SetInvoicePrefix("AL").
			SetNextSeq(42).
			SetInvoiceDayOfMonth(1).
			SetCurrency("EUR").
			SetLocale("en-IE").
			Save(ctx); err != nil {
			t.Fatalf("Settings.Create: %v", err)
		}

		iv, err := client.Invoice.Create().
			SetStudentID(st.ID).
			SetPeriodYear(2026).
			SetPeriodMonth(7).
			SetTotalAmount(90).
			SetStatus(StatusIssued).
			SetNumber("AL-202607-041").
			Save(ctx)
		if err != nil {
			t.Fatalf("Invoice.Create: %v", err)
		}

		if err := svc.ReopenDraft(ctx, iv.ID, t.TempDir()); err != nil {
			t.Fatalf("ReopenDraft: %v", err)
		}

		settings, err := client.Settings.Query().Only(ctx)
		if err != nil {
			t.Fatalf("Settings.Query: %v", err)
		}
		if settings.NextSeq != 42 {
			t.Fatalf("NextSeq = %d, want 42", settings.NextSeq)
		}
	})
}
