// app.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"langschool/ent/settings"
	"langschool/internal/app/attendance"
	"langschool/internal/infra"
	"langschool/internal/paths"
)

type App struct {
	ctx       context.Context
	dirs      paths.Dirs
	db        *infra.DB
	appDBPath string

	// services
	att *attendance.Service
}

func NewApp() *App { return &App{} }

// startup is called by Wails when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	base := filepath.Join(userHome(), "LangSchool")
	dirs, err := paths.Ensure(base)
	if err != nil {
		log.Fatal(err)
	}
	a.dirs = dirs

	a.appDBPath = filepath.Join(dirs.Data, "app.sqlite")
	log.Println("Data path:", a.appDBPath)

	db, err := infra.Open(ctx, a.appDBPath)
	if err != nil {
		log.Fatal(err)
	}
	a.db = db
	log.Println("DB ready")

	// Ensure single Settings record with singleton_id=1 exists
	exists, err := a.db.Ent.Settings.
		Query().
		Where(settings.SingletonIDEQ(1)).
		Exist(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		if _, err := a.db.Ent.Settings.
			Create().
			SetSingletonID(1).
			SetOrgName("").
			SetAddress("").
			SetInvoicePrefix("LS").
			SetNextSeq(1).
			SetInvoiceDayOfMonth(1).
			SetAutoIssue(false).
			SetCurrency("EUR").
			SetLocale("ru-RU").
			Save(ctx); err != nil {
			log.Fatal(err)
		}
	}

	// init services
	a.att = attendance.New(a.db.Ent)
}

// domReady is called by Wails when the frontend is ready.
func (a *App) domReady(ctx context.Context) {}

// shutdown is called by Wails when the app is quitting.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		_ = a.db.Ent.Close()
	}
}

func (a *App) DevSeed() (int, error) {
	ctx := a.ctx
	db := a.db.Ent

	// students
	sJohn, _ := db.Student.Create().
		SetFullName("John Doe").SetPhone("+1234567890").SetEmail("john@doe.com").Save(ctx)
	sJane, _ := db.Student.Create().
		SetFullName("Jane Johnson").SetPhone("+9876543210").SetEmail("jane@johnson.com").Save(ctx)

	// courses
	cA2, _ := db.Course.Create().
		SetName("English A2").
		SetType("group").
		SetLessonPrice(15).
		SetSubscriptionPrice(120).
		SetScheduleJSON(`{"daysOfWeek":[1,3]}`).
		Save(ctx)

	cB1, _ := db.Course.Create().
		SetName("English B1").
		SetType("individual").
		SetLessonPrice(25).
		SetSubscriptionPrice(0).
		Save(ctx)

	now := time.Now()

	// enrollments
	_, _ = db.Enrollment.Create().
		SetStudentID(sJane.ID).
		SetCourseID(cA2.ID).
		SetBillingMode("subscription").
		SetStartDate(now).
		Save(ctx)

	_, _ = db.Enrollment.Create().
		SetStudentID(sJohn.ID).
		SetCourseID(cA2.ID).
		SetBillingMode("per_lesson").
		SetStartDate(now).
		Save(ctx)

	_, _ = db.Enrollment.Create().
		SetStudentID(sJohn.ID).
		SetCourseID(cB1.ID).
		SetBillingMode("per_lesson").
		SetStartDate(now).
		Save(ctx)

	return 2, nil // number of students added as an example
}

func userHome() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

// ---------- Simple diagnostics ----------

func (a *App) Ping() string { return "ok" }

func (a *App) Greet(name string) string {
	if name == "" {
		return "HI!"
	}
	return fmt.Sprintf("Hi, %s!", name)
}

// ---------- App info / utilities ----------

// AppDirs returns application directories for UI (useful for exports/backups).
func (a *App) AppDirs() map[string]string {
	return map[string]string{
		"base":     a.dirs.Base,
		"data":     a.dirs.Data,
		"backups":  a.dirs.Backups,
		"invoices": a.dirs.Invoices,
		"exports":  a.dirs.Exports,
	}
}

// BackupNow creates a timestamped copy of the SQLite DB in Backups/ and returns the file path.
func (a *App) BackupNow() (string, error) {
	if a.appDBPath == "" {
		return "", fmt.Errorf("db path is empty")
	}
	ts := time.Now().Format("20060102-150405")
	dst := filepath.Join(a.dirs.Backups, fmt.Sprintf("app-%s.sqlite", ts))
	if err := copyFile(a.appDBPath, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// ---------- Attendance bindings ----------

func (a *App) AttendanceListPerLesson(year, month int, courseID *int) ([]attendance.Row, error) {
	return a.att.ListPerLesson(a.ctx, year, month, courseID)
}

func (a *App) AttendanceUpsert(studentID, courseID, year, month, count int) error {
	return a.att.Upsert(a.ctx, studentID, courseID, year, month, count)
}

func (a *App) AttendanceAddOne(year, month int, courseID *int) (int, error) {
	return a.att.AddOneForFilter(a.ctx, year, month, courseID)
}

func (a *App) AttendanceEstimate(year, month int, courseID *int) (map[string]int, error) {
	return a.att.EstimateBySchedule(a.ctx, year, month, courseID)
}

func (a *App) AttendanceSetLocked(year, month int, courseID *int, lock bool) (int, error) {
	return a.att.SetLocked(a.ctx, year, month, courseID, lock)
}
