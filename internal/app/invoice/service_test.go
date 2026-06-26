package invoice

import (
	"context"
	"errors"
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
	"langschool/ent/payment"
	"langschool/ent/settings"
	"langschool/internal/app"
	"langschool/internal/money"
)

func setInvoiceCurrentTime(t *testing.T, ts time.Time) {
	t.Helper()
	previous := currentTime
	currentTime = func() time.Time { return ts }
	t.Cleanup(func() {
		currentTime = previous
	})
}

func TestGenerateDraftsSubscriptionMaterialsUseCourseMonthStats(t *testing.T) {
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
		SetLessonPriceCents(money.EurosToCents(20)).
		SetSubscriptionPriceCents(money.EurosToCents(60)).
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
		SetSubscriptionLessonPriceCents(money.EurosToCents(20)).
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
		if len(lines) != 2 {
			t.Fatalf("invoice line count = %d, want 2", len(lines))
		}
		if lines[0].Description != "Dalības maksa par Gleznošana" {
			t.Fatalf("service description = %q, want %q", lines[0].Description, "Dalības maksa par Gleznošana")
		}
		if lines[1].Description != materialsLineDescription {
			t.Fatalf("materials description = %q, want %q", lines[1].Description, materialsLineDescription)
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
	assertDraft(string(StatusDraft), 65)

	res, err = svc.GenerateDrafts(ctx, 2026, 4)
	if err != nil {
		t.Fatalf("GenerateDrafts second run: %v", err)
	}
	if res.Created != 0 || res.Updated != 1 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected second GenerateDrafts result: %+v", res)
	}
	assertDraft(string(StatusDraft), 65)
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
		SetLessonPriceCents(money.EurosToCents(25)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
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

func TestGenerateDraftsPerLessonChargeMaterialsFalseDoesNotAddMaterials(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate-materials-disabled?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Online Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Tiešsaistes angļu valoda").
		SetType(course.TypeIndividual).
		SetLessonPriceCents(money.EurosToCents(30)).
		SetSubscriptionPriceCents(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetChargeMaterials(false).
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
		SetHours(2).
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
	if iv.TotalAmountCents != money.EurosToCents(60) {
		t.Fatalf("invoice total = %v, want 60", money.CentsToEuros(iv.TotalAmountCents))
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
	if lines[0].EnrollmentID != enr.ID {
		t.Fatalf("service enrollment_id = %d, want %d", lines[0].EnrollmentID, enr.ID)
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
		SetLessonPriceCents(money.EurosToCents(20)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
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

func TestGenerateDraftsMixedEnrollmentsAddsOneMaterialsLineForEligibleEnrollment(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate-mixed-materials?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Mixed Delivery Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	onlineCourse, err := client.Course.Create().
		SetName("Online Speaking").
		SetType(course.TypeIndividual).
		SetLessonPriceCents(money.EurosToCents(20)).
		SetSubscriptionPriceCents(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create online: %v", err)
	}

	offlineCourse, err := client.Course.Create().
		SetName("Classroom Grammar").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(25)).
		SetSubscriptionPriceCents(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create offline: %v", err)
	}

	onlineEnrollment, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(onlineCourse.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetChargeMaterials(false).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create online: %v", err)
	}

	offlineEnrollment, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(offlineCourse.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetChargeMaterials(true).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create offline: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(onlineCourse.ID).
		SetYear(2026).
		SetMonth(5).
		SetHours(1).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create online: %v", err)
	}
	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(offlineCourse.ID).
		SetYear(2026).
		SetMonth(5).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create offline: %v", err)
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
	if iv.TotalAmountCents != money.EurosToCents(75) {
		t.Fatalf("invoice total = %v, want 75", money.CentsToEuros(iv.TotalAmountCents))
	}

	lines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		Order(ent.Asc(invoiceline.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("invoice line count = %d, want 3", len(lines))
	}

	var materialsLines []*ent.InvoiceLine
	for _, line := range lines {
		if line.Description == materialsLineDescription {
			materialsLines = append(materialsLines, line)
		}
	}
	if len(materialsLines) != 1 {
		t.Fatalf("materials line count = %d, want 1", len(materialsLines))
	}
	if materialsLines[0].EnrollmentID != offlineEnrollment.ID {
		t.Fatalf("materials enrollment_id = %d, want %d", materialsLines[0].EnrollmentID, offlineEnrollment.ID)
	}
	if onlineEnrollment.ID == offlineEnrollment.ID {
		t.Fatalf("expected distinct enrollments for mixed test")
	}
}

func TestGenerateDraftsSubscriptionLessonsAddMaterialsWithoutAttendanceRows(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate-subscription-materials-from-course-month?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Subscription Materials Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Matemātika").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(12.5)).
		SetSubscriptionPriceCents(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModeSubscription).
		SetChargeMaterials(true).
		SetDiscountPct(0).
		SetSubscriptionLessonPriceCents(money.EurosToCents(12.5)).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	if _, err := client.CourseMonthStat.Create().
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(6).
		SetSubscriptionLessonsHeld(4).
		Save(ctx); err != nil {
		t.Fatalf("CourseMonthStat.Create: %v", err)
	}

	res, err := svc.GenerateDrafts(ctx, 2026, 6)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}

	iv, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(6)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Invoice.Query: %v", err)
	}
	if iv.TotalAmountCents != money.EurosToCents(55) {
		t.Fatalf("invoice total = %v, want 55", money.CentsToEuros(iv.TotalAmountCents))
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
		t.Fatalf("subscription enrollment_id = %d, want %d", lines[0].EnrollmentID, enr.ID)
	}
	if lines[1].Description != materialsLineDescription {
		t.Fatalf("materials description = %q, want %q", lines[1].Description, materialsLineDescription)
	}
}

func TestGenerateDraftsMixedEnrollmentsAddMaterialsFromSubscriptionLessons(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-generate-mixed-subscription-materials?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Mixed Subscription Materials Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	perLessonCourse, err := client.Course.Create().
		SetName("Online Speaking").
		SetType(course.TypeIndividual).
		SetLessonPriceCents(money.EurosToCents(20)).
		SetSubscriptionPriceCents(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create per lesson: %v", err)
	}

	subscriptionCourse, err := client.Course.Create().
		SetName("Classroom Grammar").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(15)).
		SetSubscriptionPriceCents(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create subscription: %v", err)
	}

	_, err = client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(perLessonCourse.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetChargeMaterials(false).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create per lesson: %v", err)
	}

	subscriptionEnrollment, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(subscriptionCourse.ID).
		SetBillingMode(enrollment.BillingModeSubscription).
		SetChargeMaterials(true).
		SetDiscountPct(0).
		SetSubscriptionLessonPriceCents(money.EurosToCents(15)).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create subscription: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(perLessonCourse.ID).
		SetYear(2026).
		SetMonth(6).
		SetHours(1).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create per lesson: %v", err)
	}

	if _, err := client.CourseMonthStat.Create().
		SetCourseID(subscriptionCourse.ID).
		SetYear(2026).
		SetMonth(6).
		SetSubscriptionLessonsHeld(2).
		Save(ctx); err != nil {
		t.Fatalf("CourseMonthStat.Create subscription: %v", err)
	}

	res, err := svc.GenerateDrafts(ctx, 2026, 6)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}

	iv, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(6)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Invoice.Query: %v", err)
	}
	if iv.TotalAmountCents != money.EurosToCents(55) {
		t.Fatalf("invoice total = %v, want 55", money.CentsToEuros(iv.TotalAmountCents))
	}

	lines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		Order(ent.Asc(invoiceline.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query: %v", err)
	}

	var materialsLines []*ent.InvoiceLine
	for _, line := range lines {
		if line.Description == materialsLineDescription {
			materialsLines = append(materialsLines, line)
		}
	}
	if len(materialsLines) != 1 {
		t.Fatalf("materials line count = %d, want 1", len(materialsLines))
	}
	if materialsLines[0].EnrollmentID != subscriptionEnrollment.ID {
		t.Fatalf("materials enrollment_id = %d, want %d", materialsLines[0].EnrollmentID, subscriptionEnrollment.ID)
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
		SetLessonPriceCents(money.EurosToCents(25)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
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
		SetLessonPriceCents(money.EurosToCents(25)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
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
		SetTotalAmountCents(money.EurosToCents(25)).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Keramika").
		SetQty(1).
		SetUnitPriceCents(money.EurosToCents(25)).
		SetAmountCents(money.EurosToCents(25)).
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
		SetLessonPriceCents(money.EurosToCents(25)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
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

func TestRebuildStudentDraftCreateRollsBackOnInvoiceLineError(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-rebuild-create-rollback?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Create Rollback Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Keramika").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(25)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	if _, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx); err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(10).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	wantErr := errors.New("invoice line create failed")
	failingCreates := 0
	client.InvoiceLine.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if lineMutation, ok := m.(*ent.InvoiceLineMutation); ok && lineMutation.Op().Is(ent.OpCreate) {
				failingCreates++
				if failingCreates == 2 {
					return nil, wantErr
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	res, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RebuildStudentDraft error = %v, want %v", err, wantErr)
	}
	if res != (GenerateResult{}) {
		t.Fatalf("unexpected result on error: %+v", res)
	}

	count, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(10)).
		Count(ctx)
	if err != nil {
		t.Fatalf("Invoice.Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("invoice count after rollback = %d, want 0", count)
	}

	lineCount, err := client.InvoiceLine.Query().Count(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Count: %v", err)
	}
	if lineCount != 0 {
		t.Fatalf("invoice line count after rollback = %d, want 0", lineCount)
	}
}

func TestRebuildStudentDraftUpdateRollsBackOnInvoiceLineError(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-rebuild-update-rollback?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Update Rollback Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Ilustrācija").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(20)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
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
		SetMonth(11).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	initial, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 11)
	if err != nil {
		t.Fatalf("initial RebuildStudentDraft: %v", err)
	}
	if initial.Created != 1 {
		t.Fatalf("initial Created = %d, want 1", initial.Created)
	}

	iv, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(11)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Invoice.Query: %v", err)
	}
	if iv.TotalAmountCents != money.EurosToCents(45) {
		t.Fatalf("initial invoice total = %v, want 45", money.CentsToEuros(iv.TotalAmountCents))
	}

	beforeLines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		Order(ent.Asc(invoiceline.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query before update: %v", err)
	}
	if len(beforeLines) != 2 {
		t.Fatalf("initial invoice line count = %d, want 2", len(beforeLines))
	}

	if _, err := client.AttendanceMonth.Update().
		Where(
			attendancemonth.StudentIDEQ(st.ID),
			attendancemonth.CourseIDEQ(crs.ID),
			attendancemonth.YearEQ(2026),
			attendancemonth.MonthEQ(11),
		).
		SetHours(3).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Update: %v", err)
	}

	wantErr := errors.New("invoice line update failed")
	failingCreates := 0
	client.InvoiceLine.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if lineMutation, ok := m.(*ent.InvoiceLineMutation); ok && lineMutation.Op().Is(ent.OpCreate) {
				failingCreates++
				if failingCreates == 2 {
					return nil, wantErr
				}
			}
			return next.Mutate(ctx, m)
		})
	})

	res, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 11)
	if !errors.Is(err, wantErr) {
		t.Fatalf("RebuildStudentDraft error = %v, want %v", err, wantErr)
	}
	if res != (GenerateResult{}) {
		t.Fatalf("unexpected result on error: %+v", res)
	}

	afterInvoice, err := client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get after rollback: %v", err)
	}
	if afterInvoice.TotalAmountCents != money.EurosToCents(45) {
		t.Fatalf("invoice total after rollback = %v, want 45", money.CentsToEuros(afterInvoice.TotalAmountCents))
	}

	afterLines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		Order(ent.Asc(invoiceline.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query after rollback: %v", err)
	}
	if len(afterLines) != len(beforeLines) {
		t.Fatalf("invoice line count after rollback = %d, want %d", len(afterLines), len(beforeLines))
	}
	for i := range beforeLines {
		if afterLines[i].Description != beforeLines[i].Description || afterLines[i].AmountCents != beforeLines[i].AmountCents || afterLines[i].Qty != beforeLines[i].Qty || afterLines[i].EnrollmentID != enr.ID {
			t.Fatalf("invoice line %d changed after rollback: got %+v want %+v", i, afterLines[i], beforeLines[i])
		}
	}
}

func TestRebuildStudentDraftSkipsIssuedInvoice(t *testing.T) {
	setInvoiceCurrentTime(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

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
		SetLessonPriceCents(money.EurosToCents(30)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
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
		SetTotalAmountCents(money.EurosToCents(30)).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Tēlniecība").
		SetQty(1).
		SetUnitPriceCents(money.EurosToCents(30)).
		SetAmountCents(money.EurosToCents(30)).
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

func TestRebuildStudentDraftUpdatesCurrentPaidInvoiceAndKeepsNumber(t *testing.T) {
	setInvoiceCurrentTime(t, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))

	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-rebuild-current-paid?mode=memory&_fk=1")
	defer client.Close()

	invoicesDir := t.TempDir()
	svc := NewWithInvoicesDir(client, invoicesDir)

	st, err := client.Student.Create().
		SetFullName("Current Paid Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Keramika Live").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(30)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetChargeMaterials(false).
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
		SetStatus(StatusPaid).
		SetNumber("LS-202609-001").
		SetTotalAmountCents(money.EurosToCents(30)).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Keramika Live").
		SetQty(1).
		SetUnitPriceCents(money.EurosToCents(30)).
		SetAmountCents(money.EurosToCents(30)).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	if _, err := client.Payment.Create().
		SetStudentID(st.ID).
		SetInvoiceID(iv.ID).
		SetAmountCents(money.EurosToCents(30)).
		SetMethod(payment.Method(app.PaymentMethodCash)).
		SetPaidAt(time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)).
		SetCreatedAt(time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)).
		Save(ctx); err != nil {
		t.Fatalf("Payment.Create: %v", err)
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

	pdfPath := PDFPathByNumberAndName(invoicesDir, 2026, 9, "LS-202609-001", st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := client.Invoice.UpdateOneID(iv.ID).
		SetPdfFilename(filepath.Base(pdfPath)).
		SetPdfGeneratedAt(time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)).
		SetPdfRevision(iv.Version).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Update metadata: %v", err)
	}

	res, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 9)
	if err != nil {
		t.Fatalf("RebuildStudentDraft: %v", err)
	}
	if res.Created != 0 || res.Updated != 1 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected rebuild result: %+v", res)
	}

	got, err := client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if got.Number == nil || *got.Number != "LS-202609-001" {
		t.Fatalf("number = %v, want LS-202609-001", got.Number)
	}
	if got.TotalAmountCents != money.EurosToCents(120) {
		t.Fatalf("total = %v, want 120", money.CentsToEuros(got.TotalAmountCents))
	}
	if got.Status != StatusIssuedPendingPDF {
		t.Fatalf("status = %q, want %q", got.Status, StatusIssuedPendingPDF)
	}
	if got.PdfRevision != nil || got.PdfGeneratedAt != nil {
		t.Fatalf("expected PDF metadata cleared, got revision=%v generatedAt=%v", got.PdfRevision, got.PdfGeneratedAt)
	}
	if _, err := os.Stat(pdfPath); !os.IsNotExist(err) {
		t.Fatalf("expected pdf to be invalidated, stat err=%v", err)
	}

	paymentCount, err := client.Payment.Query().Where(payment.InvoiceIDEQ(iv.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("Payment.Count: %v", err)
	}
	if paymentCount != 1 {
		t.Fatalf("payment count = %d, want 1", paymentCount)
	}
}

func TestRebuildStudentDraftUpdatesCurrentIssuedInvoiceAndInvalidatesPDF(t *testing.T) {
	setInvoiceCurrentTime(t, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))

	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-rebuild-current-issued-pdf?mode=memory&_fk=1")
	defer client.Close()

	invoicesDir := t.TempDir()
	svc := NewWithInvoicesDir(client, invoicesDir)

	st, err := client.Student.Create().
		SetFullName("Current Issued Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Keramika Basic").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(30)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetChargeMaterials(false).
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
		SetNumber("LS-202609-002").
		SetTotalAmountCents(money.EurosToCents(120)).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Keramika Basic").
		SetQty(4).
		SetUnitPriceCents(money.EurosToCents(30)).
		SetAmountCents(money.EurosToCents(120)).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(9).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	pdfPath := PDFPathByNumberAndName(invoicesDir, 2026, 9, "LS-202609-002", st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := client.Invoice.UpdateOneID(iv.ID).
		SetPdfFilename(filepath.Base(pdfPath)).
		SetPdfGeneratedAt(time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)).
		SetPdfRevision(iv.Version).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Update metadata: %v", err)
	}

	res, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 9)
	if err != nil {
		t.Fatalf("RebuildStudentDraft: %v", err)
	}
	if res.Created != 0 || res.Updated != 1 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected rebuild result: %+v", res)
	}

	got, err := client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if got.TotalAmountCents != money.EurosToCents(60) {
		t.Fatalf("total = %v, want 60", money.CentsToEuros(got.TotalAmountCents))
	}
	if got.Status != StatusIssuedPendingPDF {
		t.Fatalf("status = %q, want %q", got.Status, StatusIssuedPendingPDF)
	}
	if got.PdfRevision != nil || got.PdfGeneratedAt != nil {
		t.Fatalf("expected PDF metadata cleared, got revision=%v generatedAt=%v", got.PdfRevision, got.PdfGeneratedAt)
	}
	if _, err := os.Stat(pdfPath); !os.IsNotExist(err) {
		t.Fatalf("expected pdf to be invalidated, stat err=%v", err)
	}
}

func TestRebuildStudentDraftKeepsCurrentPaidInvoicePaidWhenAmountDropsBelowPayments(t *testing.T) {
	setInvoiceCurrentTime(t, time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))

	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-rebuild-current-paid-overpaid?mode=memory&_fk=1")
	defer client.Close()

	invoicesDir := t.TempDir()
	svc := NewWithInvoicesDir(client, invoicesDir)

	st, err := client.Student.Create().
		SetFullName("Overpaid Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Keramika Pro").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(30)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetChargeMaterials(false).
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
		SetStatus(StatusPaid).
		SetNumber("LS-202609-003").
		SetTotalAmountCents(money.EurosToCents(120)).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Keramika Pro").
		SetQty(4).
		SetUnitPriceCents(money.EurosToCents(30)).
		SetAmountCents(money.EurosToCents(120)).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	if _, err := client.Payment.Create().
		SetStudentID(st.ID).
		SetInvoiceID(iv.ID).
		SetAmountCents(money.EurosToCents(120)).
		SetMethod(payment.Method(app.PaymentMethodCash)).
		SetPaidAt(time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)).
		SetCreatedAt(time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)).
		Save(ctx); err != nil {
		t.Fatalf("Payment.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(9).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	pdfPath := PDFPathByNumberAndName(invoicesDir, 2026, 9, "LS-202609-003", st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := client.Invoice.UpdateOneID(iv.ID).
		SetPdfFilename(filepath.Base(pdfPath)).
		SetPdfGeneratedAt(time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)).
		SetPdfRevision(iv.Version).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Update metadata: %v", err)
	}

	res, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 9)
	if err != nil {
		t.Fatalf("RebuildStudentDraft: %v", err)
	}
	if res.Created != 0 || res.Updated != 1 || res.SkippedNoLines != 0 || res.SkippedHasInvoice != 0 {
		t.Fatalf("unexpected rebuild result: %+v", res)
	}

	got, err := client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if got.TotalAmountCents != money.EurosToCents(60) {
		t.Fatalf("total = %v, want 60", money.CentsToEuros(got.TotalAmountCents))
	}
	if got.Status != StatusPaidPendingPDF {
		t.Fatalf("status = %q, want %q", got.Status, StatusPaidPendingPDF)
	}
	if got.PdfRevision != nil || got.PdfGeneratedAt != nil {
		t.Fatalf("expected PDF metadata cleared, got revision=%v generatedAt=%v", got.PdfRevision, got.PdfGeneratedAt)
	}
	if _, err := os.Stat(pdfPath); !os.IsNotExist(err) {
		t.Fatalf("expected pdf to be invalidated, stat err=%v", err)
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
		SetLessonPriceCents(money.EurosToCents(20)).
		SetSubscriptionPriceCents(money.EurosToCents(60)).
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
			SetTotalAmountCents(money.EurosToCents(70)).
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
			SetUnitPriceCents(money.EurosToCents(70)).
			SetAmountCents(money.EurosToCents(70)).
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
		if _, err := client.Invoice.UpdateOneID(iv.ID).
			SetPdfFilename(filepath.Base(pdfPath)).
			SetPdfGeneratedAt(time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)).
			SetPdfRevision(iv.Version).
			SetEmailDeliveryStatus(invoice.EmailDeliveryStatusSent).
			SetLastEmailedAt(time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC)).
			SetLastEmailedTo("billing@example.com").
			SetLastEmailedRevision(iv.Version).
			SetLastEmailError("old error").
			SetLastEmailFailedAt(time.Date(2026, 4, 7, 8, 0, 0, 0, time.UTC)).
			Save(ctx); err != nil {
			t.Fatalf("Invoice.Update metadata: %v", err)
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
		if got.PdfFilename != nil || got.PdfRevision != nil || got.PdfGeneratedAt != nil {
			t.Fatalf(
				"expected PDF metadata cleared, got filename=%v revision=%v generatedAt=%v",
				got.PdfFilename,
				got.PdfRevision,
				got.PdfGeneratedAt,
			)
		}
		if got.EmailDeliveryStatus != invoice.EmailDeliveryStatusNotSent {
			t.Fatalf("EmailDeliveryStatus = %q, want %q", got.EmailDeliveryStatus, invoice.EmailDeliveryStatusNotSent)
		}
		if got.LastEmailedAt != nil || got.LastEmailedTo != nil || got.LastEmailedRevision != nil || got.LastEmailError != nil || got.LastEmailFailedAt != nil {
			t.Fatalf(
				"expected email metadata cleared, got lastEmailedAt=%v lastEmailedTo=%v lastEmailedRevision=%v lastEmailError=%v lastEmailFailedAt=%v",
				got.LastEmailedAt,
				got.LastEmailedTo,
				got.LastEmailedRevision,
				got.LastEmailError,
				got.LastEmailFailedAt,
			)
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
			SetTotalAmountCents(money.EurosToCents(50)).
			SetStatus(StatusIssued).
			SetNumber(number).
			Save(ctx)
		if err != nil {
			t.Fatalf("Invoice.Create: %v", err)
		}
		if _, err := client.Payment.Create().
			SetStudentID(st.ID).
			SetInvoiceID(iv.ID).
			SetAmountCents(money.EurosToCents(10)).
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
			SetTotalAmountCents(money.EurosToCents(30)).
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
			SetTotalAmountCents(money.EurosToCents(35)).
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
			SetTotalAmountCents(money.EurosToCents(40)).
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
			SetTotalAmountCents(money.EurosToCents(90)).
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

func TestGetComputesStaleEmailCommunicationStatus(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-email-status?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Status Student").
		SetEmail("family@example.com").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}
	iv, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(6).
		SetTotalAmountCents(money.EurosToCents(55)).
		SetStatus(StatusIssued).
		SetNumber("LS-202606-017").
		SetVersion(4).
		SetEmailDeliveryStatus(invoice.EmailDeliveryStatusSent).
		SetLastEmailedAt(time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)).
		SetLastEmailedTo("family@example.com").
		SetLastEmailedRevision(3).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	dto, err := svc.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if dto.EmailCommunicationStatus != app.InvoiceEmailStatusStale {
		t.Fatalf("EmailCommunicationStatus = %q, want %q", dto.EmailCommunicationStatus, app.InvoiceEmailStatusStale)
	}
}

func TestIssueOneUsesTransactionalSettingsSequence(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-issue-sequence?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Issue Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

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
		SetPeriodMonth(12).
		SetTotalAmountCents(money.EurosToCents(90)).
		SetStatus(StatusDraft).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	number, err := svc.IssueOne(ctx, iv.ID)
	if err != nil {
		t.Fatalf("IssueOne: %v", err)
	}
	if number != "AL-202612-042" {
		t.Fatalf("number = %q, want %q", number, "AL-202612-042")
	}

	issued, err := client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if issued.Status != StatusIssuedPendingPDF {
		t.Fatalf("Status = %q, want %q", issued.Status, StatusIssuedPendingPDF)
	}
	if issued.Number == nil || *issued.Number != number {
		t.Fatalf("Number = %v, want %q", issued.Number, number)
	}

	settingsItem, err := client.Settings.Query().
		Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Settings.Query: %v", err)
	}
	if settingsItem.NextSeq != 43 {
		t.Fatalf("NextSeq = %d, want 43", settingsItem.NextSeq)
	}
}

func TestIssueOneAlreadyIssuedDoesNotIncrementSequenceAgain(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-issue-idempotent?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Issued Again Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	if _, err := client.Settings.Create().
		SetSingletonID(app.SettingsSingletonID).
		SetOrgName("ArtLab").
		SetAddress("Latgales iela 260, Rīga, Latvija").
		SetInvoicePrefix("AL").
		SetNextSeq(50).
		SetInvoiceDayOfMonth(1).
		SetCurrency("EUR").
		SetLocale("en-IE").
		Save(ctx); err != nil {
		t.Fatalf("Settings.Create: %v", err)
	}

	iv, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(12).
		SetTotalAmountCents(money.EurosToCents(90)).
		SetStatus(StatusDraft).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	firstNumber, err := svc.IssueOne(ctx, iv.ID)
	if err != nil {
		t.Fatalf("first IssueOne: %v", err)
	}
	secondNumber, err := svc.IssueOne(ctx, iv.ID)
	if err != nil {
		t.Fatalf("second IssueOne: %v", err)
	}
	if secondNumber != firstNumber {
		t.Fatalf("secondNumber = %q, want %q", secondNumber, firstNumber)
	}

	settingsItem, err := client.Settings.Query().
		Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Settings.Query: %v", err)
	}
	if settingsItem.NextSeq != 51 {
		t.Fatalf("NextSeq = %d, want 51", settingsItem.NextSeq)
	}
}

func TestIssueOneRollsBackWhenSequenceUpdateFails(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:invoice-issue-rollback?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().
		SetFullName("Rollback Issue Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	if _, err := client.Settings.Create().
		SetSingletonID(app.SettingsSingletonID).
		SetOrgName("ArtLab").
		SetAddress("Latgales iela 260, Rīga, Latvija").
		SetInvoicePrefix("AL").
		SetNextSeq(7).
		SetInvoiceDayOfMonth(1).
		SetCurrency("EUR").
		SetLocale("en-IE").
		Save(ctx); err != nil {
		t.Fatalf("Settings.Create: %v", err)
	}

	iv, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(10).
		SetTotalAmountCents(money.EurosToCents(55)).
		SetStatus(StatusDraft).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	wantErr := errors.New("settings update failed")
	client.Settings.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if settingsMutation, ok := m.(*ent.SettingsMutation); ok && settingsMutation.Op().Is(ent.OpUpdate|ent.OpUpdateOne) {
				return nil, wantErr
			}
			return next.Mutate(ctx, m)
		})
	})

	number, err := svc.IssueOne(ctx, iv.ID)
	if !errors.Is(err, wantErr) {
		t.Fatalf("IssueOne error = %v, want %v", err, wantErr)
	}
	if number != "" {
		t.Fatalf("number = %q, want empty", number)
	}

	gotInvoice, err := client.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if gotInvoice.Status != StatusDraft {
		t.Fatalf("Status after rollback = %q, want %q", gotInvoice.Status, StatusDraft)
	}
	if gotInvoice.Number != nil {
		t.Fatalf("Number after rollback = %v, want nil", gotInvoice.Number)
	}

	settingsItem, err := client.Settings.Query().
		Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Settings.Query: %v", err)
	}
	if settingsItem.NextSeq != 7 {
		t.Fatalf("NextSeq after rollback = %d, want 7", settingsItem.NextSeq)
	}
}
