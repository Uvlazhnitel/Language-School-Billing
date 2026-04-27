package main

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"langschool/ent/enttest"
	"langschool/internal/infra"
)

func TestCourseTeacherCRUDAndSearch(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:crudcourse?mode=memory&_fk=1")
	defer client.Close()

	app := &App{
		ctx: ctx,
		db:  &infra.DB{Ent: client},
	}

	created, err := app.CourseCreate("Conversation Club", "Anna Petrova", "group", 25, 90)
	if err != nil {
		t.Fatalf("CourseCreate: %v", err)
	}

	if created.TeacherName != "Anna Petrova" {
		t.Fatalf("created.TeacherName = %q, want %q", created.TeacherName, "Anna Petrova")
	}

	got, err := app.CourseGet(created.ID)
	if err != nil {
		t.Fatalf("CourseGet: %v", err)
	}
	if got.TeacherName != "Anna Petrova" {
		t.Fatalf("got.TeacherName = %q, want %q", got.TeacherName, "Anna Petrova")
	}

	updated, err := app.CourseUpdate(created.ID, "Conversation Club", "Elina Ozola", "group", 25, 90)
	if err != nil {
		t.Fatalf("CourseUpdate: %v", err)
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
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:crudenrollment?mode=memory&_fk=1")
	defer client.Close()

	app := &App{
		ctx: ctx,
		db:  &infra.DB{Ent: client},
	}

	st, err := app.StudentCreate("Mila Test", "", "", "")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}

	crs, err := app.CourseCreate("Grammar Lab", "Janis Kalnins", "group", 20, 80)
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
	if list[0].TeacherName != "Janis Kalnins" {
		t.Fatalf("TeacherName = %q, want %q", list[0].TeacherName, "Janis Kalnins")
	}
	if list[0].CourseName != "Grammar Lab" {
		t.Fatalf("CourseName = %q, want %q", list[0].CourseName, "Grammar Lab")
	}
}
