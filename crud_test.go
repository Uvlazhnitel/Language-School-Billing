package main

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"langschool/ent"
	"langschool/ent/course"
	"langschool/ent/enttest"
	"langschool/ent/teacher"
	"langschool/internal/infra"
)

func newCRUDTestApp(t *testing.T, name string) (*App, *ent.Client) {
	t.Helper()

	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:"+name+"?mode=memory&_fk=1")

	app := &App{
		ctx: ctx,
		db:  &infra.DB{Ent: client},
	}
	return app, client
}

func TestTeacherCreateAndList(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudteacher")
	defer client.Close()

	created, err := app.TeacherCreate("Anna Petrova")
	if err != nil {
		t.Fatalf("TeacherCreate: %v", err)
	}

	dup, err := app.TeacherCreate("anna petrova")
	if err != nil {
		t.Fatalf("TeacherCreate duplicate: %v", err)
	}
	if dup.ID != created.ID {
		t.Fatalf("duplicate teacher id = %d, want %d", dup.ID, created.ID)
	}

	list, err := app.TeacherList("petro")
	if err != nil {
		t.Fatalf("TeacherList: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].FullName != "Anna Petrova" {
		t.Fatalf("FullName = %q, want %q", list[0].FullName, "Anna Petrova")
	}
}

func TestCourseTeacherCRUDAndSearch(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudcourse")
	defer client.Close()

	anna, err := app.TeacherCreate("Anna Petrova")
	if err != nil {
		t.Fatalf("TeacherCreate Anna: %v", err)
	}
	elina, err := app.TeacherCreate("Elina Ozola")
	if err != nil {
		t.Fatalf("TeacherCreate Elina: %v", err)
	}

	created, err := app.CourseCreate("Conversation Club", &anna.ID, "group", 25, 90)
	if err != nil {
		t.Fatalf("CourseCreate: %v", err)
	}

	if created.TeacherID == nil || *created.TeacherID != anna.ID {
		t.Fatalf("created.TeacherID = %v, want %d", created.TeacherID, anna.ID)
	}
	if created.TeacherName != "Anna Petrova" {
		t.Fatalf("created.TeacherName = %q, want %q", created.TeacherName, "Anna Petrova")
	}

	got, err := app.CourseGet(created.ID)
	if err != nil {
		t.Fatalf("CourseGet: %v", err)
	}
	if got.TeacherID == nil || *got.TeacherID != anna.ID {
		t.Fatalf("got.TeacherID = %v, want %d", got.TeacherID, anna.ID)
	}

	updated, err := app.CourseUpdate(created.ID, "Conversation Club", &elina.ID, "group", 25, 90)
	if err != nil {
		t.Fatalf("CourseUpdate: %v", err)
	}
	if updated.TeacherID == nil || *updated.TeacherID != elina.ID {
		t.Fatalf("updated.TeacherID = %v, want %d", updated.TeacherID, elina.ID)
	}
	if updated.TeacherName != "Elina Ozola" {
		t.Fatalf("updated.TeacherName = %q, want %q", updated.TeacherName, "Elina Ozola")
	}

	found, err := app.CourseList("ozola")
	if err != nil {
		t.Fatalf("CourseList search by teacher: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("len(found) = %d, want 1", len(found))
	}
	if found[0].ID != created.ID {
		t.Fatalf("found[0].ID = %d, want %d", found[0].ID, created.ID)
	}
}

func TestEnrollmentListIncludesTeacherName(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudenrollment")
	defer client.Close()

	st, err := app.StudentCreate("Mila Test", "010101-12345", "", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}

	tch, err := app.TeacherCreate("Janis Kalnins")
	if err != nil {
		t.Fatalf("TeacherCreate: %v", err)
	}

	crs, err := app.CourseCreate("Grammar Lab", &tch.ID, "group", 20, 80)
	if err != nil {
		t.Fatalf("CourseCreate: %v", err)
	}

	_, err = app.EnrollmentCreate(st.ID, crs.ID, "per_lesson", 5, "evening group")
	if err != nil {
		t.Fatalf("EnrollmentCreate: %v", err)
	}

	list, err := app.EnrollmentList(&st.ID, nil)
	if err != nil {
		t.Fatalf("EnrollmentList: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].TeacherID == nil || *list[0].TeacherID != tch.ID {
		t.Fatalf("TeacherID = %v, want %d", list[0].TeacherID, tch.ID)
	}
	if list[0].TeacherName != "Janis Kalnins" {
		t.Fatalf("TeacherName = %q, want %q", list[0].TeacherName, "Janis Kalnins")
	}
	if list[0].CourseName != "Grammar Lab" {
		t.Fatalf("CourseName = %q, want %q", list[0].CourseName, "Grammar Lab")
	}
}

func TestMigrateLegacyCourseTeachers(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:legacyteachers?mode=memory&_fk=1")
	defer client.Close()

	if _, err := client.Course.Create().
		SetName("Legacy Course").
		SetTeacherName("Ieva Liepa").
		SetType(course.TypeGroup).
		SetLessonPrice(15).
		SetSubscriptionPrice(60).
		Save(ctx); err != nil {
		t.Fatalf("create legacy course: %v", err)
	}

	if err := migrateLegacyCourseTeachers(ctx, client); err != nil {
		t.Fatalf("migrateLegacyCourseTeachers: %v", err)
	}

	teachers, err := client.Teacher.Query().All(ctx)
	if err != nil {
		t.Fatalf("Teacher.Query: %v", err)
	}
	if len(teachers) != 1 {
		t.Fatalf("len(teachers) = %d, want 1", len(teachers))
	}
	if teachers[0].FullName != "Ieva Liepa" {
		t.Fatalf("teacher.FullName = %q, want %q", teachers[0].FullName, "Ieva Liepa")
	}

	crs, err := client.Course.Query().
		Where(course.NameEQ("Legacy Course")).
		Only(ctx)
	if err != nil {
		t.Fatalf("Course.Query: %v", err)
	}
	if crs.TeacherID == nil || *crs.TeacherID != teachers[0].ID {
		t.Fatalf("course.TeacherID = %v, want %d", crs.TeacherID, teachers[0].ID)
	}

	// idempotency
	if err := migrateLegacyCourseTeachers(ctx, client); err != nil {
		t.Fatalf("migrateLegacyCourseTeachers second run: %v", err)
	}

	count, err := client.Teacher.Query().Where(teacher.FullNameEQ("Ieva Liepa")).Count(ctx)
	if err != nil {
		t.Fatalf("Teacher.Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("teacher count = %d, want 1", count)
	}
}

func TestStudentCreateAndUpdateIsMinor(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentminor")
	defer client.Close()

	created, err := app.StudentCreate("Nika Test", "020202-23456", "", "", "", true, "Anna Test", "mother")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}
	if !created.IsMinor {
		t.Fatalf("created.IsMinor = false, want true")
	}
	if created.PayerName != "Anna Test" {
		t.Fatalf("created.PayerName = %q, want %q", created.PayerName, "Anna Test")
	}
	if created.PersonalCode != "020202-23456" {
		t.Fatalf("created.PersonalCode = %q, want %q", created.PersonalCode, "020202-23456")
	}

	updated, err := app.StudentUpdate(created.ID, "Nika Test", "030303-34567", "123", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentUpdate: %v", err)
	}
	if updated.IsMinor {
		t.Fatalf("updated.IsMinor = true, want false")
	}
	if updated.PayerName != "" {
		t.Fatalf("updated.PayerName = %q, want empty", updated.PayerName)
	}
	if updated.PersonalCode != "030303-34567" {
		t.Fatalf("updated.PersonalCode = %q, want %q", updated.PersonalCode, "030303-34567")
	}
}

func TestStudentCreateMinorRequiresPayerFields(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentvalidation")
	defer client.Close()

	if _, err := app.StudentCreate("Minor Missing Payer", "", "", "", "", true, "", ""); err == nil {
		t.Fatalf("expected StudentCreate to fail when minor payer fields are missing")
	}

	if _, err := app.StudentCreate("Minor Missing Role", "", "", "", "", true, "Anna Parent", ""); err == nil {
		t.Fatalf("expected StudentCreate to fail when minor payerRole is missing")
	}

	if _, err := app.StudentCreate("Minor Bad Role", "", "", "", "", true, "Anna Parent", "uncle"); err == nil {
		t.Fatalf("expected StudentCreate to fail for invalid payerRole")
	}
}
