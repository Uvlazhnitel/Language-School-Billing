package backend

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"langschool/ent"
	"langschool/ent/enrollment"
	entinvoice "langschool/ent/invoice"
	"langschool/internal/money"
	appruntime "langschool/internal/runtime"
)

func setEnrollmentCurrentTime(t *testing.T, ts time.Time) {
	t.Helper()
	previous := enrollmentCurrentTime
	enrollmentCurrentTime = func() time.Time { return ts }
	t.Cleanup(func() {
		enrollmentCurrentTime = previous
	})
}

func newTestEnrollmentService(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	rt, err := appruntime.Start(context.Background(), appruntime.Config{
		BaseDir:       filepath.Join(root, "base"),
		DataDir:       filepath.Join(root, "data"),
		BackupsDir:    filepath.Join(root, "backups"),
		InvoicesDir:   filepath.Join(root, "invoices"),
		ExportsDir:    filepath.Join(root, "exports"),
		FontsDir:      filepath.Join(root, "fonts"),
		AdminUsername: "admin",
		AdminPassword: "test-password-123",
		SessionSecret: "test-session-secret",
	})
	if err != nil {
		t.Fatalf("runtime.Start: %v", err)
	}
	t.Cleanup(func() {
		_ = rt.Close()
	})
	return New(rt)
}

func TestResolveEnrollmentInvoiceRebuildMonthRollsDecemberToJanuary(t *testing.T) {
	setEnrollmentCurrentTime(t, time.Date(2026, 12, 20, 10, 0, 0, 0, time.UTC))

	svc := newTestEnrollmentService(t)
	ctx := context.Background()

	st, err := svc.rt.DB.Ent.Student.Create().
		SetFullName("Year Boundary Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	if _, err := svc.rt.DB.Ent.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(12).
		SetStatus(entinvoice.StatusIssued).
		SetNumber("LS-202612-001").
		SetTotalAmountCents(money.EurosToCents(25)).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	year, month, err := svc.resolveEnrollmentInvoiceRebuildMonth(ctx, st.ID)
	if err != nil {
		t.Fatalf("resolveEnrollmentInvoiceRebuildMonth: %v", err)
	}
	if year != 2027 || month != 1 {
		t.Fatalf("resolved month = %04d-%02d, want 2027-01", year, month)
	}
}

func TestEnrollmentInvoiceAffectingFieldsChangedIgnoresNoteOnlyEdits(t *testing.T) {
	before := &ent.Enrollment{
		BillingMode:                  enrollment.BillingModePerLesson,
		ChargeMaterials:              true,
		LessonPriceOverrideCents:     money.EurosToCents(12.5),
		SubscriptionLessonPriceCents: 0,
		Note:                         "before",
	}

	changed := enrollmentInvoiceAffectingFieldsChanged(before, BillingModePerLesson, true, 12.5, 0)
	if changed {
		t.Fatal("expected note-only edit to be ignored")
	}
}

func TestResolveEnrollmentInvoiceRebuildMonthUsesCurrentOpenMonthWithoutCurrentInvoice(t *testing.T) {
	setEnrollmentCurrentTime(t, time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC))

	svc := newTestEnrollmentService(t)
	ctx := context.Background()

	st, err := svc.rt.DB.Ent.Student.Create().
		SetFullName("Open Month Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	year, month, err := svc.resolveEnrollmentInvoiceRebuildMonth(ctx, st.ID)
	if err != nil {
		t.Fatalf("resolveEnrollmentInvoiceRebuildMonth: %v", err)
	}
	if year != 2026 || month != 7 {
		t.Fatalf("resolved month = %04d-%02d, want 2026-07", year, month)
	}
}

func TestResolveEnrollmentInvoiceRebuildMonthUsesNextMonthWhenCurrentMonthIssued(t *testing.T) {
	setEnrollmentCurrentTime(t, time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC))

	svc := newTestEnrollmentService(t)
	ctx := context.Background()

	st, err := svc.rt.DB.Ent.Student.Create().
		SetFullName("Issued Current Month Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	if _, err := svc.rt.DB.Ent.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(7).
		SetStatus(entinvoice.StatusIssued).
		SetNumber("LS-202607-001").
		SetTotalAmountCents(money.EurosToCents(25)).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	year, month, err := svc.resolveEnrollmentInvoiceRebuildMonth(ctx, st.ID)
	if err != nil {
		t.Fatalf("resolveEnrollmentInvoiceRebuildMonth: %v", err)
	}
	if year != 2026 || month != 8 {
		t.Fatalf("resolved month = %04d-%02d, want 2026-08", year, month)
	}
}
