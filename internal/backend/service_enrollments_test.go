package backend

import (
	"context"
	"path/filepath"
	"strings"
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

func TestEnrollmentFinishPreservesHistoryAndRemovesOnlyZeroInvoiceLines(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()

	st, err := svc.rt.DB.Ent.Student.Create().SetFullName("Moved Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}
	oldCourse, err := svc.rt.DB.Ent.Course.Create().
		SetName("Latvian").SetTeacherName("Old Teacher").SetType("group").
		SetLessonPriceCents(1250).SetSubscriptionPriceCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create old: %v", err)
	}
	newCourse, err := svc.rt.DB.Ent.Course.Create().
		SetName("Latvian").SetTeacherName("New Teacher").SetType("group").
		SetLessonPriceCents(1500).SetSubscriptionPriceCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create new: %v", err)
	}
	oldEnrollment, err := svc.EnrollmentCreate(ctx, st.ID, oldCourse.ID, BillingModePerLesson, true, 12.5, 0, "")
	if err != nil {
		t.Fatalf("EnrollmentCreate old: %v", err)
	}
	newEnrollment, err := svc.EnrollmentCreate(ctx, st.ID, newCourse.ID, BillingModePerLesson, true, 15, 0, "")
	if err != nil {
		t.Fatalf("EnrollmentCreate new: %v", err)
	}

	if _, err := svc.rt.DB.Ent.AttendanceMonth.Create().
		SetStudentID(st.ID).SetCourseID(oldCourse.ID).SetYear(2026).SetMonth(6).SetHours(1).Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create old: %v", err)
	}
	if _, err := svc.rt.DB.Ent.AttendanceMonth.Create().
		SetStudentID(st.ID).SetCourseID(newCourse.ID).SetYear(2026).SetMonth(7).SetHours(9).Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create new: %v", err)
	}

	juneInvoice, err := svc.rt.DB.Ent.Invoice.Create().
		SetStudentID(st.ID).SetPeriodYear(2026).SetPeriodMonth(6).
		SetStatus(entinvoice.StatusPaid).SetNumber("LS-202606-001").SetTotalAmountCents(1250).Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create June: %v", err)
	}
	juneLine, err := svc.rt.DB.Ent.InvoiceLine.Create().
		SetInvoiceID(juneInvoice.ID).SetEnrollmentID(oldEnrollment.ID).
		SetDescription("Old course June").SetQty(1).SetUnitPriceCents(1250).SetAmountCents(1250).Save(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Create June: %v", err)
	}
	julyInvoice, err := svc.rt.DB.Ent.Invoice.Create().
		SetStudentID(st.ID).SetPeriodYear(2026).SetPeriodMonth(7).
		SetStatus(entinvoice.StatusIssued).SetNumber("LS-202607-001").SetTotalAmountCents(14000).
		SetPdfFilename("LS-202607-001.pdf").SetPdfRevision(1).SetPdfGeneratedAt(time.Now()).Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create July: %v", err)
	}
	zeroLine, err := svc.rt.DB.Ent.InvoiceLine.Create().
		SetInvoiceID(julyInvoice.ID).SetEnrollmentID(oldEnrollment.ID).
		SetDescription("Old course July").SetQty(0).SetUnitPriceCents(1250).SetAmountCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Create zero: %v", err)
	}
	newLine, err := svc.rt.DB.Ent.InvoiceLine.Create().
		SetInvoiceID(julyInvoice.ID).SetEnrollmentID(newEnrollment.ID).
		SetDescription("New course July").SetQty(9).SetUnitPriceCents(1500).SetAmountCents(13500).Save(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Create new: %v", err)
	}
	materialsLine, err := svc.rt.DB.Ent.InvoiceLine.Create().
		SetInvoiceID(julyInvoice.ID).SetEnrollmentID(oldEnrollment.ID).
		SetDescription("Materials").SetQty(1).SetUnitPriceCents(500).SetAmountCents(500).Save(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Create materials: %v", err)
	}
	payment, err := svc.rt.DB.Ent.Payment.Create().
		SetStudentID(st.ID).SetInvoiceID(julyInvoice.ID).SetAmountCents(8000).
		SetMethod("cash").SetPaidAt(time.Now()).SetNote("partial").Save(ctx)
	if err != nil {
		t.Fatalf("Payment.Create: %v", err)
	}

	if err := svc.EnrollmentSetActiveWithVersion(ctx, oldEnrollment.ID, oldEnrollment.Version, false); err != nil {
		t.Fatalf("EnrollmentSetActiveWithVersion: %v", err)
	}

	finished, err := svc.rt.DB.Ent.Enrollment.Get(ctx, oldEnrollment.ID)
	if err != nil {
		t.Fatalf("Enrollment.Get: %v", err)
	}
	if finished.IsActive || finished.Version != oldEnrollment.Version+1 {
		t.Fatalf("finished enrollment = %+v", finished)
	}
	active, err := svc.EnrollmentList(ctx, &st.ID, nil)
	if err != nil {
		t.Fatalf("EnrollmentList: %v", err)
	}
	if len(active) != 1 || active[0].ID != newEnrollment.ID {
		t.Fatalf("active enrollments = %+v, want new enrollment only", active)
	}
	if _, err := svc.rt.DB.Ent.InvoiceLine.Get(ctx, zeroLine.ID); !ent.IsNotFound(err) {
		t.Fatalf("zero line lookup error = %v, want not found", err)
	}
	for _, lineID := range []int{juneLine.ID, newLine.ID, materialsLine.ID} {
		if _, err := svc.rt.DB.Ent.InvoiceLine.Get(ctx, lineID); err != nil {
			t.Fatalf("preserved line %d: %v", lineID, err)
		}
	}
	if _, err := svc.rt.DB.Ent.Payment.Get(ctx, payment.ID); err != nil {
		t.Fatalf("preserved payment: %v", err)
	}
	freshJuly, err := svc.rt.DB.Ent.Invoice.Get(ctx, julyInvoice.ID)
	if err != nil {
		t.Fatalf("Invoice.Get July: %v", err)
	}
	if freshJuly.TotalAmountCents != 14000 || freshJuly.Status != entinvoice.StatusIssued || freshJuly.PdfRevision != nil {
		t.Fatalf("July invoice after finish = %+v", freshJuly)
	}

	juneRows, err := svc.rt.Attendance.ListPerLesson(ctx, 2026, 6, &oldCourse.ID)
	if err != nil {
		t.Fatalf("ListPerLesson June: %v", err)
	}
	if len(juneRows) != 1 || juneRows[0].EnrollmentID != oldEnrollment.ID || juneRows[0].TeacherName != "Old Teacher" {
		t.Fatalf("June rows = %+v", juneRows)
	}
	julyRows, err := svc.rt.Attendance.ListPerLesson(ctx, 2026, 7, &oldCourse.ID)
	if err != nil {
		t.Fatalf("ListPerLesson July: %v", err)
	}
	if len(julyRows) != 0 {
		t.Fatalf("July old-course rows = %+v, want none", julyRows)
	}
	if err := svc.rt.Attendance.Upsert(ctx, st.ID, oldCourse.ID, 2026, 8, 1); err == nil || !strings.Contains(err.Error(), "завершённого зачисления") {
		t.Fatalf("future attendance error = %v", err)
	}
	if err := svc.EnrollmentSetActiveWithVersion(ctx, oldEnrollment.ID, oldEnrollment.Version, false); err == nil {
		t.Fatal("expected stale enrollment version to fail")
	}
}

func TestEnrollmentCreateReactivatesFinishedEnrollment(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	st, err := svc.rt.DB.Ent.Student.Create().SetFullName("Returning Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}
	course, err := svc.rt.DB.Ent.Course.Create().
		SetName("Returning Course").SetType("group").SetLessonPriceCents(1500).SetSubscriptionPriceCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}
	created, err := svc.EnrollmentCreate(ctx, st.ID, course.ID, BillingModePerLesson, true, 15, 0, "old")
	if err != nil {
		t.Fatalf("EnrollmentCreate: %v", err)
	}
	if err := svc.EnrollmentSetActiveWithVersion(ctx, created.ID, created.Version, false); err != nil {
		t.Fatalf("finish enrollment: %v", err)
	}

	reactivated, err := svc.EnrollmentCreate(ctx, st.ID, course.ID, BillingModePerLesson, false, 20, 0, "new")
	if err != nil {
		t.Fatalf("reactivate enrollment: %v", err)
	}
	if reactivated.ID != created.ID || reactivated.Version != created.Version+2 || reactivated.Note != "new" || reactivated.LessonPriceOverride != 20 {
		t.Fatalf("reactivated enrollment = %+v", reactivated)
	}
	fresh, err := svc.rt.DB.Ent.Enrollment.Get(ctx, created.ID)
	if err != nil || !fresh.IsActive {
		t.Fatalf("active enrollment after reactivation = %+v, err=%v", fresh, err)
	}
	if err := svc.EnrollmentSetActiveWithVersion(ctx, fresh.ID, fresh.Version, false); err != nil {
		t.Fatalf("finish enrollment again: %v", err)
	}
	result, err := svc.EnrollmentCreateMany(ctx, st.ID, []EnrollmentCreateInput{{
		CourseID: course.ID, BillingMode: BillingModePerLesson, LessonPriceOverride: 22, Note: "bulk",
	}})
	if err != nil {
		t.Fatalf("EnrollmentCreateMany reactivate: %v", err)
	}
	if len(result.Enrollments) != 1 || result.Enrollments[0].ID != created.ID || len(result.SkippedCourseIDs) != 0 {
		t.Fatalf("bulk reactivation result = %+v", result)
	}
	if _, err := svc.EnrollmentCreate(ctx, st.ID, course.ID, BillingModePerLesson, true, 22, 0, "duplicate"); err == nil {
		t.Fatal("expected active duplicate enrollment to fail")
	}
}
