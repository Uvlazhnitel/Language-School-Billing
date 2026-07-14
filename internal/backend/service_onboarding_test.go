package backend

import (
	"context"
	"testing"

	"langschool/ent/student"
)

func testOnboardingStudentInput(name string) StudentCreateInput {
	return StudentCreateInput{FullName: name}
}

func TestStudentOnboardCreatesStudentAndEnrollmentAtomically(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	course, err := svc.rt.DB.Ent.Course.Create().
		SetName("Onboarding Group").
		SetType("group").
		SetLessonPriceCents(1500).
		SetSubscriptionPriceCents(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	result, err := svc.StudentOnboard(ctx, testOnboardingStudentInput("New Student"), &EnrollmentCreateInput{
		CourseID:            course.ID,
		BillingMode:         BillingModePerLesson,
		ChargeMaterials:     true,
		LessonPriceOverride: 15,
	})
	if err != nil {
		t.Fatalf("StudentOnboard: %v", err)
	}
	if result.Enrollment == nil {
		t.Fatal("expected enrollment in result")
	}
	if result.Enrollment.StudentID != result.Student.ID || result.Enrollment.CourseID != course.ID {
		t.Fatalf("unexpected onboarding result: %+v", result)
	}

	studentCount, err := svc.rt.DB.Ent.Student.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Student.Count: %v", err)
	}
	enrollmentCount, err := svc.rt.DB.Ent.Enrollment.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Count: %v", err)
	}
	if studentCount != 1 || enrollmentCount != 1 {
		t.Fatalf("counts = students %d, enrollments %d; want 1 and 1", studentCount, enrollmentCount)
	}
}

func TestStudentOnboardManyCreatesAllEnrollmentsInOrder(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	firstCourse, err := svc.rt.DB.Ent.Course.Create().
		SetName("First Onboarding Course").
		SetType("group").
		SetLessonPriceCents(1500).
		SetSubscriptionPriceCents(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create first: %v", err)
	}
	secondCourse, err := svc.rt.DB.Ent.Course.Create().
		SetName("Second Onboarding Course").
		SetType("individual").
		SetLessonPriceCents(2500).
		SetSubscriptionPriceCents(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create second: %v", err)
	}

	result, err := svc.StudentOnboardMany(ctx, testOnboardingStudentInput("Multi-course Student"), []EnrollmentCreateInput{
		{CourseID: firstCourse.ID, BillingMode: BillingModePerLesson, ChargeMaterials: true, LessonPriceOverride: 15},
		{CourseID: secondCourse.ID, BillingMode: BillingModePerLesson, ChargeMaterials: false, LessonPriceOverride: 25},
	})
	if err != nil {
		t.Fatalf("StudentOnboardMany: %v", err)
	}
	if len(result.Enrollments) != 2 {
		t.Fatalf("enrollments = %+v, want 2", result.Enrollments)
	}
	if result.Enrollments[0].CourseID != firstCourse.ID || result.Enrollments[1].CourseID != secondCourse.ID {
		t.Fatalf("enrollment order = %+v", result.Enrollments)
	}
	if result.Enrollment == nil || result.Enrollment.CourseID != firstCourse.ID {
		t.Fatalf("legacy enrollment = %+v, want first course", result.Enrollment)
	}
}

func TestStudentOnboardManyRollsBackWhenLaterEnrollmentFails(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	course, err := svc.rt.DB.Ent.Course.Create().
		SetName("Valid Course Before Failure").
		SetType("group").
		SetLessonPriceCents(1500).
		SetSubscriptionPriceCents(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	_, err = svc.StudentOnboardMany(ctx, testOnboardingStudentInput("Fully Rolled Back Student"), []EnrollmentCreateInput{
		{CourseID: course.ID, BillingMode: BillingModePerLesson, LessonPriceOverride: 15},
		{CourseID: 999999, BillingMode: BillingModePerLesson, LessonPriceOverride: 25},
	})
	if err == nil {
		t.Fatal("expected second enrollment to fail")
	}
	studentExists, err := svc.rt.DB.Ent.Student.Query().
		Where(student.FullNameEQ("Fully Rolled Back Student")).
		Exist(ctx)
	if err != nil {
		t.Fatalf("Student.Exist: %v", err)
	}
	if studentExists {
		t.Fatal("student persisted after multi-enrollment failure")
	}
	enrollmentCount, err := svc.rt.DB.Ent.Enrollment.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Count: %v", err)
	}
	if enrollmentCount != 0 {
		t.Fatalf("enrollment count = %d, want 0", enrollmentCount)
	}
}

func TestStudentOnboardManyRejectsDuplicateCourseIDs(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	input := EnrollmentCreateInput{CourseID: 10, BillingMode: BillingModePerLesson, LessonPriceOverride: 15}

	if _, err := svc.StudentOnboardMany(ctx, testOnboardingStudentInput("Duplicate Course Student"), []EnrollmentCreateInput{input, input}); err == nil {
		t.Fatal("expected duplicate courseID error")
	}
	studentCount, err := svc.rt.DB.Ent.Student.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Student.Count: %v", err)
	}
	if studentCount != 0 {
		t.Fatalf("student count = %d, want 0", studentCount)
	}
}

func TestEnrollmentCreateManySkipsExistingAndCreatesMissing(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	st, err := svc.rt.DB.Ent.Student.Create().SetFullName("Existing Multi-course Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}
	firstCourse, err := svc.rt.DB.Ent.Course.Create().SetName("Existing Course").SetType("group").SetLessonPriceCents(1500).SetSubscriptionPriceCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create first: %v", err)
	}
	secondCourse, err := svc.rt.DB.Ent.Course.Create().SetName("Missing Course").SetType("individual").SetLessonPriceCents(2500).SetSubscriptionPriceCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create second: %v", err)
	}
	if _, err := svc.EnrollmentCreate(ctx, st.ID, firstCourse.ID, BillingModePerLesson, true, 15, 0, ""); err != nil {
		t.Fatalf("EnrollmentCreate existing: %v", err)
	}

	result, err := svc.EnrollmentCreateMany(ctx, st.ID, []EnrollmentCreateInput{
		{CourseID: firstCourse.ID, BillingMode: BillingModePerLesson, ChargeMaterials: true, LessonPriceOverride: 15},
		{CourseID: secondCourse.ID, BillingMode: BillingModePerLesson, ChargeMaterials: false, LessonPriceOverride: 25},
	})
	if err != nil {
		t.Fatalf("EnrollmentCreateMany: %v", err)
	}
	if len(result.Enrollments) != 1 || result.Enrollments[0].CourseID != secondCourse.ID {
		t.Fatalf("created enrollments = %+v", result.Enrollments)
	}
	if len(result.SkippedCourseIDs) != 1 || result.SkippedCourseIDs[0] != firstCourse.ID {
		t.Fatalf("skipped course IDs = %+v", result.SkippedCourseIDs)
	}
}

func TestEnrollmentCreateManyRollsBackAllMissingCoursesOnFailure(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	st, err := svc.rt.DB.Ent.Student.Create().SetFullName("Bulk Rollback Student").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}
	course, err := svc.rt.DB.Ent.Course.Create().SetName("Bulk Valid Course").SetType("group").SetLessonPriceCents(1500).SetSubscriptionPriceCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	_, err = svc.EnrollmentCreateMany(ctx, st.ID, []EnrollmentCreateInput{
		{CourseID: course.ID, BillingMode: BillingModePerLesson, LessonPriceOverride: 15},
		{CourseID: 999999, BillingMode: BillingModePerLesson, LessonPriceOverride: 25},
	})
	if err == nil {
		t.Fatal("expected bulk enrollment failure")
	}
	enrollmentCount, err := svc.rt.DB.Ent.Enrollment.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Count: %v", err)
	}
	if enrollmentCount != 0 {
		t.Fatalf("enrollment count = %d, want 0", enrollmentCount)
	}
}

func TestEnrollmentCreateManyRejectsInactiveStudentWithoutChanges(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	st, err := svc.rt.DB.Ent.Student.Create().SetFullName("Inactive Bulk Student").SetIsActive(false).Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}
	course, err := svc.rt.DB.Ent.Course.Create().SetName("Inactive Bulk Course").SetType("group").SetLessonPriceCents(1500).SetSubscriptionPriceCents(0).Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	if _, err := svc.EnrollmentCreateMany(ctx, st.ID, []EnrollmentCreateInput{{
		CourseID: course.ID, BillingMode: BillingModePerLesson, LessonPriceOverride: 15,
	}}); err == nil {
		t.Fatal("expected inactive student error")
	}
	enrollmentCount, err := svc.rt.DB.Ent.Enrollment.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Count: %v", err)
	}
	if enrollmentCount != 0 {
		t.Fatalf("enrollment count = %d, want 0", enrollmentCount)
	}
	freshStudent, err := svc.rt.DB.Ent.Student.Get(ctx, st.ID)
	if err != nil {
		t.Fatalf("Student.Get: %v", err)
	}
	if freshStudent.IsActive {
		t.Fatal("inactive student was activated")
	}
}

func TestStudentOnboardWithoutEnrollmentCreatesOnlyStudent(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()

	result, err := svc.StudentOnboard(ctx, testOnboardingStudentInput("Student Only"), nil)
	if err != nil {
		t.Fatalf("StudentOnboard: %v", err)
	}
	if result.Enrollment != nil {
		t.Fatalf("unexpected enrollment: %+v", result.Enrollment)
	}

	enrollmentCount, err := svc.rt.DB.Ent.Enrollment.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Count: %v", err)
	}
	if enrollmentCount != 0 {
		t.Fatalf("enrollment count = %d, want 0", enrollmentCount)
	}
}

func TestStudentOnboardRollsBackStudentWhenEnrollmentFails(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()

	_, err := svc.StudentOnboard(ctx, testOnboardingStudentInput("Rolled Back Student"), &EnrollmentCreateInput{
		CourseID:            999999,
		BillingMode:         BillingModePerLesson,
		ChargeMaterials:     true,
		LessonPriceOverride: 15,
	})
	if err == nil {
		t.Fatal("expected onboarding to fail")
	}

	exists, err := svc.rt.DB.Ent.Student.Query().
		Where(student.FullNameEQ("Rolled Back Student")).
		Exist(ctx)
	if err != nil {
		t.Fatalf("Student.Exist: %v", err)
	}
	if exists {
		t.Fatal("student persisted after enrollment failure")
	}
}

func TestStudentOnboardRollsBackStudentWhenEnrollmentTermsAreInvalid(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	course, err := svc.rt.DB.Ent.Course.Create().
		SetName("Invalid Terms Group").
		SetType("group").
		SetLessonPriceCents(1500).
		SetSubscriptionPriceCents(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	_, err = svc.StudentOnboard(ctx, testOnboardingStudentInput("Invalid Terms Student"), &EnrollmentCreateInput{
		CourseID:            course.ID,
		BillingMode:         "invalid",
		ChargeMaterials:     true,
		LessonPriceOverride: 15,
	})
	if err == nil {
		t.Fatal("expected invalid enrollment terms to fail")
	}

	exists, err := svc.rt.DB.Ent.Student.Query().
		Where(student.FullNameEQ("Invalid Terms Student")).
		Exist(ctx)
	if err != nil {
		t.Fatalf("Student.Exist: %v", err)
	}
	if exists {
		t.Fatal("student persisted after invalid enrollment terms")
	}
}

func TestStudentOnboardRejectsDuplicatePersonalCodeWithoutPartialRecords(t *testing.T) {
	svc := newTestEnrollmentService(t)
	ctx := context.Background()
	if _, err := svc.rt.DB.Ent.Student.Create().
		SetFullName("Existing Student").
		SetPersonalCode("010101-12345").
		SetIsActive(true).
		Save(ctx); err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	input := testOnboardingStudentInput("Duplicate Student")
	input.PersonalCode = "010101-12345"
	if _, err := svc.StudentOnboard(ctx, input, nil); err == nil {
		t.Fatal("expected duplicate personal code error")
	}

	studentCount, err := svc.rt.DB.Ent.Student.Query().Count(ctx)
	if err != nil {
		t.Fatalf("Student.Count: %v", err)
	}
	if studentCount != 1 {
		t.Fatalf("student count = %d, want 1", studentCount)
	}
}
