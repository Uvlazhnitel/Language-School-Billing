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
	"langschool/internal/app"
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
		SetLessonsCount(3).
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
	if gotPerLesson.BillingMode != app.BillingModePerLesson || gotPerLesson.Count != 3 || !gotPerLesson.HasRecord {
		t.Fatalf("unexpected per_lesson row: %+v", *gotPerLesson)
	}
	if gotSubscription.BillingMode != app.BillingModeSubscription || gotSubscription.Count != 0 || gotSubscription.HasRecord {
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
		SetLessonsCount(2).
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
		SetLessonsCount(1).
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
	if am.LessonsCount != 1 {
		t.Fatalf("lessons count = %d, want 1", am.LessonsCount)
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
	if am.LessonsCount != 5 {
		t.Fatalf("lessons count = %d, want 5", am.LessonsCount)
	}
}
