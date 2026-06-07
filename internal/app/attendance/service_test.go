package attendance

import (
	"context"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/internal/app"
	invsvc "langschool/internal/app/invoice"
	"langschool/internal/money"
)

func TestListPerLessonIncludesSubscriptionRows(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-list?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	perLessonStudent, err := client.Student.Create().SetFullName("Per Lesson Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create per_lesson: %v", err)
	}
	subscriptionStudent, err := client.Student.Create().SetFullName("Subscription Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create subscription: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Group A").
		SetType(course.TypeGroup).
		SetLessonPrice(25).
		SetSubscriptionPrice(80).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	perLessonEnr, err := client.Enrollment.Create().
		SetStudentID(perLessonStudent.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create per_lesson: %v", err)
	}

	subscriptionEnr, err := client.Enrollment.Create().
		SetStudentID(subscriptionStudent.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModeSubscription).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create subscription: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(perLessonStudent.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(5).
		SetHours(3).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	rows, err := svc.ListPerLesson(ctx, 2026, 5, &crs.ID)
	if err != nil {
		t.Fatalf("ListPerLesson: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}

	var gotPerLesson, gotSubscription *Row
	for i := range rows {
		switch rows[i].EnrollmentID {
		case perLessonEnr.ID:
			gotPerLesson = &rows[i]
		case subscriptionEnr.ID:
			gotSubscription = &rows[i]
		}
	}

	if gotPerLesson == nil || gotSubscription == nil {
		t.Fatalf("expected both rows, got %+v", rows)
	}
	if gotPerLesson.BillingMode != app.BillingModePerLesson || gotPerLesson.Hours != 3 || !gotPerLesson.HasRecord {
		t.Fatalf("unexpected per_lesson row: %+v", *gotPerLesson)
	}
	if gotSubscription.BillingMode != app.BillingModeSubscription || gotSubscription.Hours != 0 || gotSubscription.HasRecord {
		t.Fatalf("unexpected subscription row: %+v", *gotSubscription)
	}
}

func TestListPerLessonLocksNonDraftInvoiceMonths(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-locks?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().SetFullName("Locked Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Group B").
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
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	if _, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(5).
		SetStatus(app.InvoiceStatusIssued).
		SetTotalAmount(40).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	rows, err := svc.ListPerLesson(ctx, 2026, 5, &crs.ID)
	if err != nil {
		t.Fatalf("ListPerLesson: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	if rows[0].EnrollmentID != enr.ID {
		t.Fatalf("enrollment id = %d, want %d", rows[0].EnrollmentID, enr.ID)
	}
	if !rows[0].AttendanceLocked || rows[0].InvoiceStatus != app.InvoiceStatusIssued {
		t.Fatalf("expected locked row with issued status, got %+v", rows[0])
	}
}

func TestUpsertRejectsNonDraftInvoiceMonths(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-upsert-lock?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().SetFullName("Guarded Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Group C").
		SetType(course.TypeGroup).
		SetLessonPrice(15).
		SetSubscriptionPrice(0).
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
		SetMonth(6).
		SetHours(1).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	if _, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(6).
		SetStatus(app.InvoiceStatusPaid).
		SetTotalAmount(15).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	err = svc.Upsert(ctx, st.ID, crs.ID, 2026, 6, 4)
	if err == nil {
		t.Fatalf("expected Upsert to reject non-draft invoice month")
	}
	if !strings.Contains(err.Error(), app.InvoiceStatusPaid) {
		t.Fatalf("unexpected error: %v", err)
	}

	am, err := client.AttendanceMonth.Query().
		Where(
			attendancemonth.StudentIDEQ(st.ID),
			attendancemonth.CourseIDEQ(crs.ID),
			attendancemonth.YearEQ(2026),
			attendancemonth.MonthEQ(6),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("AttendanceMonth.Query: %v", err)
	}
	if am.Hours != 1 {
		t.Fatalf("hours = %v, want 1", am.Hours)
	}
}

func TestUpsertAllowsDraftOrMissingInvoice(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-upsert-draft?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().SetFullName("Draft Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Group D").
		SetType(course.TypeGroup).
		SetLessonPrice(15).
		SetSubscriptionPrice(0).
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

	if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 7, 2); err != nil {
		t.Fatalf("Upsert without invoice: %v", err)
	}

	if _, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(7).
		SetStatus(app.InvoiceStatusDraft).
		SetTotalAmount(30).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 7, 5); err != nil {
		t.Fatalf("Upsert with draft invoice: %v", err)
	}

	am, err := client.AttendanceMonth.Query().
		Where(
			attendancemonth.StudentIDEQ(st.ID),
			attendancemonth.CourseIDEQ(crs.ID),
			attendancemonth.YearEQ(2026),
			attendancemonth.MonthEQ(7),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("AttendanceMonth.Query: %v", err)
	}
	if am.Hours != 5 {
		t.Fatalf("hours = %v, want 5", am.Hours)
	}
}

func TestListPerLessonAllowsDeleteAfterReopenToDraft(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-delete-reopened?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)
	invoiceSvc := invsvc.New(client)

	st, err := client.Student.Create().SetFullName("Reopened Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Group E").
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

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(8).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	iv, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(8).
		SetStatus(app.InvoiceStatusIssued).
		SetNumber("LS-202608-001").
		SetTotalAmount(60).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Group E").
		SetQty(2).
		SetUnitPrice(30).
		SetAmount(60).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	if err := invoiceSvc.ReopenDraft(ctx, iv.ID, t.TempDir()); err != nil {
		t.Fatalf("ReopenDraft: %v", err)
	}

	rows, err := svc.ListPerLesson(ctx, 2026, 8, &crs.ID)
	if err != nil {
		t.Fatalf("ListPerLesson: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	if !rows[0].CanDelete {
		t.Fatalf("expected enrollment to be deletable after reopen to draft, got %+v", rows[0])
	}

	if err := svc.DeleteEnrollment(ctx, enr.ID); err != nil {
		t.Fatalf("DeleteEnrollment: %v", err)
	}

	invoiceCount, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(8)).
		Count(ctx)
	if err != nil {
		t.Fatalf("Invoice.Count: %v", err)
	}
	if invoiceCount != 0 {
		t.Fatalf("invoice count = %d, want 0", invoiceCount)
	}
}

func TestDeleteEnrollmentRejectsNonDraftInvoiceUsage(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-delete-issued?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().SetFullName("Protected Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Group F").
		SetType(course.TypeGroup).
		SetLessonPrice(18).
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
		SetStatus(app.InvoiceStatusIssued).
		SetNumber("LS-202609-001").
		SetTotalAmount(18).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Dalības maksa par Group F").
		SetQty(1).
		SetUnitPrice(18).
		SetAmount(18).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	rows, err := svc.ListPerLesson(ctx, 2026, 9, &crs.ID)
	if err != nil {
		t.Fatalf("ListPerLesson: %v", err)
	}
	if len(rows) != 1 || rows[0].CanDelete {
		t.Fatalf("expected non-draft invoice usage to block delete, got %+v", rows)
	}

	err = svc.DeleteEnrollment(ctx, enr.ID)
	if err == nil {
		t.Fatalf("expected DeleteEnrollment to reject issued invoice usage")
	}
	if !strings.Contains(err.Error(), "выставленных, оплаченных или отменённых счетах") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteEnrollmentRebuildsRemainingDraftInvoice(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-delete-rebuild-draft?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)
	invoiceSvc := invsvc.New(client)

	st, err := client.Student.Create().SetFullName("Draft Rebuild Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	courseA, err := client.Course.Create().
		SetName("Group G1").
		SetType(course.TypeGroup).
		SetLessonPrice(20).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("CourseA.Create: %v", err)
	}
	courseB, err := client.Course.Create().
		SetName("Group G2").
		SetType(course.TypeGroup).
		SetLessonPrice(15).
		SetSubscriptionPrice(0).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("CourseB.Create: %v", err)
	}

	enrA, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(courseA.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("EnrollmentA.Create: %v", err)
	}
	_, err = client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(courseB.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("EnrollmentB.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(courseA.ID).
		SetYear(2026).
		SetMonth(10).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonthA.Create: %v", err)
	}
	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(courseB.ID).
		SetYear(2026).
		SetMonth(10).
		SetHours(1).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonthB.Create: %v", err)
	}

	res, err := invoiceSvc.GenerateDrafts(ctx, 2026, 10)
	if err != nil {
		t.Fatalf("GenerateDrafts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}

	rows, err := svc.ListPerLesson(ctx, 2026, 10, nil)
	if err != nil {
		t.Fatalf("ListPerLesson: %v", err)
	}
	for _, row := range rows {
		if row.EnrollmentID == enrA.ID && !row.CanDelete {
			t.Fatalf("expected draft-only enrollment to be deletable, got %+v", row)
		}
	}

	if err := svc.DeleteEnrollment(ctx, enrA.ID); err != nil {
		t.Fatalf("DeleteEnrollment: %v", err)
	}

	iv, err := client.Invoice.Query().
		Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(10)).
		Only(ctx)
	if err != nil {
		t.Fatalf("Invoice.Query: %v", err)
	}
	if iv.Status != app.InvoiceStatusDraft {
		t.Fatalf("invoice status = %q, want draft", iv.Status)
	}
	if iv.TotalAmountCents != money.EurosToCents(20) {
		t.Fatalf("invoice total = %v, want 20", money.CentsToEuros(iv.TotalAmountCents))
	}

	lines, err := client.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(iv.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("InvoiceLine.Query: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("invoice line count = %d, want 2", len(lines))
	}
	for _, line := range lines {
		if line.EnrollmentID == enrA.ID {
			t.Fatalf("deleted enrollment line still present: %+v", line)
		}
	}
}

func TestUpsertAcceptsQuarterHours(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-quarter-hours?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().SetFullName("Quarter Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Quarter Group").
		SetType(course.TypeGroup).
		SetLessonPrice(24).
		SetSubscriptionPrice(0).
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

	if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 11, 1.5); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	am, err := client.AttendanceMonth.Query().
		Where(
			attendancemonth.StudentIDEQ(st.ID),
			attendancemonth.CourseIDEQ(crs.ID),
			attendancemonth.YearEQ(2026),
			attendancemonth.MonthEQ(11),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("AttendanceMonth.Query: %v", err)
	}
	if am.Hours != 1.5 {
		t.Fatalf("hours = %v, want 1.5", am.Hours)
	}
}

func TestUpsertRejectsNonQuarterHours(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:attendance-non-quarter-hours?mode=memory&_fk=1")
	defer client.Close()

	svc := New(client)

	st, err := client.Student.Create().SetFullName("Invalid Quarter Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Invalid Quarter Group").
		SetType(course.TypeGroup).
		SetLessonPrice(24).
		SetSubscriptionPrice(0).
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

	if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 11, 1.2); err == nil {
		t.Fatalf("expected quarter-hour validation error")
	}
}
