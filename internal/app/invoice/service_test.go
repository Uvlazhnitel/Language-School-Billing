package invoice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/internal/app"
	"langschool/internal/money"
)

func TestGenerateDraftsSubscriptionWithoutLessonsDoesNotAddMaterials(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Materials Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Gleznošana").
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
		SetSubscriptionDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}
	if _, err := client.CourseMonthStat.Create().
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(4).
		SetSubscriptionLessonsHeld(3).
		Save(ctx); err != nil {
		t.Fatalf("CourseMonthStat.Create: %v", err)
	}

	assertDraft := func(expectedStatus string, expectedTotal float64) {
		t.Helper()

		iv, err := client.Invoice.Query().
			Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(4)).
			Only(ctx)
		if err != nil {
			t.Fatalf("Invoice.Query: %v", err)
		}
		if string(iv.Status) != expectedStatus {
			t.Fatalf("invoice status = %q, want %q", iv.Status, expectedStatus)
		}
		if money.CentsToEuros(iv.TotalAmountCents) != expectedTotal {
			t.Fatalf("invoice total = %v, want %v", money.CentsToEuros(iv.TotalAmountCents), expectedTotal)
		}

		lines, err := client.InvoiceLine.Query().
			Where(invoiceline.InvoiceIDEQ(iv.ID)).
			Order(ent.Asc(invoiceline.FieldID)).
			All(ctx)
		if err != nil {
			t.Fatalf("InvoiceLine.Query: %v", err)
		}
		if len(lines) != 1 {
			t.Fatalf("invoice line count = %d, want 1", len(lines))
		}
		if lines[0].Description != "Dalības maksa par Gleznošana" {
			t.Fatalf("service description = %q, want %q", lines[0].Description, "Dalības maksa par Gleznošana")
		}
	}

	res, err := svc.GenerateDrafts(ctx, 2026, 4)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 1 || res.Updated != 0 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected first GenerateDrafts result: %+v", res)
	}
	if enr.ID <= 0 {
		t.Fatalf("unexpected enrollment id: %d", enr.ID)
	}
	assertDraft(string(StatusDraft), 60)

	res, err = svc.GenerateDrafts(ctx, 2026, 4)
	if err != nil {
		t.Fatalf("GenerateDrafts second run: %v", err)
	}
	if res.Created != 0 || res.Updated != 1 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected second GenerateDrafts result: %+v", res)
	}
	assertDraft(string(StatusDraft), 60)
}

func TestGenerateDraftsPerLessonDescriptionIsLatvian(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate-per-lesson?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Per Lesson Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Zīmēšana").
		SetType(course.TypeGroup).
		SetLessonPrice(25).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(5).
		SetHours(4).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	res, err := svc.GenerateDrafts(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}

	iv, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(5)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Invoice.Query: %v", err)
	}
	if iv.TotalAmountCents != money.EurosToCents(105) {
		t.Fatalf("invoice total = %v, want 105", money.CentsToEuros(iv.TotalAmountCents))
	}

	lines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		Order(ent.Asc(invoiceline.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("invoice line count = %d, want 2", len(lines))
	}
	if lines[0].EnrollmentID != enr.ID {
		t.Fatalf("service enrollment_id = %d, want %d", lines[0].EnrollmentID, enr.ID)
	}
	if lines[0].Description != "Dalības maksa par Zīmēšana" {
		t.Fatalf("service description = %q, want %q", lines[0].Description, "Dalības maksa par Zīmēšana")
	}
	if lines[0].Qty != 4 {
		t.Fatalf("service qty = %v, want 4", lines[0].Qty)
	}
	if lines[1].Description != materialsLineDescription {
		t.Fatalf("materials description = %q, want %q", lines[1].Description, materialsLineDescription)
	}
}

func TestGenerateDraftsPerLessonSupportsFractionalHours(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate-fractional-hours?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Fractional Hours Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Skicēšana").
		SetType(course.TypeGroup).
		SetLessonPrice(20).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(5).
		SetHours(1.5).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	res, err := svc.GenerateDrafts(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}

	iv, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(5)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Invoice.Query: %v", err)
	}
	if iv.TotalAmountCents != money.EurosToCents(35) {
		t.Fatalf("invoice total = %v, want 35", money.CentsToEuros(iv.TotalAmountCents))
	}

	lines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		Order(ent.Asc(invoiceline.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("invoice line count = %d, want 2", len(lines))
	}
	if lines[0].EnrollmentID != enr.ID {
		t.Fatalf("service enrollment_id = %d, want %d", lines[0].EnrollmentID, enr.ID)
	}
	if lines[0].Qty != 1.5 {
		t.Fatalf("service qty = %v, want 1.5", lines[0].Qty)
	}
	if lines[0].AmountCents != money.EurosToCents(30) {
		t.Fatalf("service amount = %v, want 30", money.CentsToEuros(lines[0].AmountCents))
	}
	if lines[1].Description != materialsLineDescription {
		t.Fatalf("materials description = %q, want %q", lines[1].Description, materialsLineDescription)
	}
}

func TestGenerateDraftsPerLessonZeroLessonsDoesNotAddMaterials(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate-zero-lessons?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Zero Lessons Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Keramika").
		SetType(course.TypeGroup).
		SetLessonPrice(25).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	_, err = client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	res, err := svc.GenerateDrafts(ctx, 2026, 6)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 0 || res.SkippedNoLines != 1 {
		t.Fatalf("unexpected GenerateDrafts result: %+v", res)
	}

	count, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(6)).
		Count(ctx)
	if err != nil {
		t.Fatalf("Invoice.Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("invoice count = %d, want 0", count)
	}
}

func TestGenerateDraftsDeletesExistingZeroDraft(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-delete-zero-draft?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Draft Cleanup Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Keramika").
		SetType(course.TypeGroup).
		SetLessonPrice(25).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	iv, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(7).
		SetStatus(StatusDraft).
		SetTotalAmount(25).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Keramika").
		SetQty(1).
		SetUnitPrice(25).
		SetAmount(25).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	res, err := svc.GenerateDrafts(ctx, 2026, 7)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 0 || res.Updated != 0 || res.SkippedNoLines != 1 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected GenerateDrafts result: %+v", res)
	}

	count, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(7)).
		Count(ctx)
	if err != nil {
		t.Fatalf("Invoice.Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("invoice count = %d, want 0", count)
	}

	lineCount, err := client.InvoiceLine.Query().Count(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Count: %v", err)
	}
	if lineCount != 0 {
		t.Fatalf("invoice line count = %d, want 0", lineCount)
	}
}

func TestRebuildStudentDraft(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-rebuild-student-draft?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Auto Rebuild Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	otherStudent, err := client.Student.Create().
		SetFullName("Other Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create(other): %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Grafika").
		SetType(course.TypeGroup).
		SetLessonPrice(25).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	if _, err := client.Enrollment.Create().
		SetStudentID(otherStudent.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx); err != nil {
		t.Fatalf("Enrollment.Create(other): %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(8).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	res, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 8)
	if err != nil {
		t.Fatalf("RebuildStudentDraft create: %v", err)
	}
	if res.Created != 1 || res.Updated != 0 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected create result: %+v", res)
	}

	iv, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(8)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Invoice.Query: %v", err)
	}
	if iv.TotalAmountCents != money.EurosToCents(55) {
		t.Fatalf("invoice total = %v, want 55", money.CentsToEuros(iv.TotalAmountCents))
	}

	otherCount, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(otherStudent.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(8)).
		Count(ctx)
	if err != nil {
		t.Fatalf("Invoice.Count(other): %v", err)
	}
	if otherCount != 0 {
		t.Fatalf("other student invoice count = %d, want 0", otherCount)
	}

	if _, err := client.AttendanceMonth.Update().
		Where(
			attendancemonth.StudentIDEQ(st.ID),
			attendancemonth.CourseIDEQ(crs.ID),
			attendancemonth.YearEQ(2026),
			attendancemonth.MonthEQ(8),
		).
		SetHours(3).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Update: %v", err)
	}

	res, err = svc.RebuildStudentDraft(ctx, st.ID, 2026, 8)
	if err != nil {
		t.Fatalf("RebuildStudentDraft update: %v", err)
	}
	if res.Created != 0 || res.Updated != 1 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected update result: %+v", res)
	}

	iv, err = client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if iv.TotalAmountCents != money.EurosToCents(80) {
		t.Fatalf("updated invoice total = %v, want 80", money.CentsToEuros(iv.TotalAmountCents))
	}

	lines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		Order(ent.Asc(invoiceline.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("invoice line count = %d, want 2", len(lines))
	}
	if lines[0].EnrollmentID != enr.ID {
		t.Fatalf("service enrollment_id = %d, want %d", lines[0].EnrollmentID, enr.ID)
	}

	if _, err := client.AttendanceMonth.Update().
		Where(
			attendancemonth.StudentIDEQ(st.ID),
			attendancemonth.CourseIDEQ(crs.ID),
			attendancemonth.YearEQ(2026),
			attendancemonth.MonthEQ(8),
		).
		SetHours(0).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Update to zero: %v", err)
	}

	res, err = svc.RebuildStudentDraft(ctx, st.ID, 2026, 8)
	if err != nil {
		t.Fatalf("RebuildStudentDraft delete: %v", err)
	}
	if res.Created != 0 || res.Updated != 0 || res.SkippedNoLines != 1 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected delete result: %+v", res)
	}

	count, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(8)).
		Count(ctx)
	if err != nil {
		t.Fatalf("Invoice.Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("invoice count after delete = %d, want 0", count)
	}
}

func TestRebuildStudentDraftSkipsIssuedInvoice(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-rebuild-student-issued?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Issued Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Tēlniecība").
		SetType(course.TypeGroup).
		SetLessonPrice(30).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	iv, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(9).
		SetStatus(StatusIssued).
		SetNumber("LS-202609-001").
		SetTotalAmount(30).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Tēlniecība").
		SetQty(1).
		SetUnitPrice(30).
		SetAmount(30).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(9).
		SetHours(4).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	res, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 9)
	if err != nil {
		t.Fatalf("RebuildStudentDraft: %v", err)
	}
	if res.Created != 0 || res.Updated != 0 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 1 {
		t.Fatalf("unexpected issued result: %+v", res)
	}

	got, err := client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if got.TotalAmountCents != money.EurosToCents(30) || got.Status != StatusIssued {
		t.Fatalf("issued invoice changed unexpectedly: total=%v status=%q", money.CentsToEuros(got.TotalAmountCents), got.Status)
	}
}

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

		pdfPath := PDFPathByNumberAndName(tmpDir, iv.PeriodYear, iv.PeriodMonth, number, "Invoice Student")
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
		if got.TotalAmountCents != money.EurosToCents(70) {
			t.Fatalf("TotalAmount = %v, want 70", money.CentsToEuros(got.TotalAmountCents))
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
