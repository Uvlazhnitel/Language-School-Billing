package attendance

import (
	"context"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

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
