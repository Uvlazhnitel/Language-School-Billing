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

	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/settings"
	"langschool/ent/student"
	"langschool/internal/app/attendance"
	invsvc "langschool/internal/app/invoice"
	"langschool/internal/infra"
	"langschool/internal/paths"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx       context.Context
	dirs      paths.Dirs
	db        *infra.DB
	appDBPath string

	// services
	att *attendance.Service
	inv *invsvc.Service
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

	// initialize services
	a.att = attendance.New(a.db.Ent)
	a.inv = invsvc.New(a.db.Ent)
}

// domReady is called by Wails when the frontend is ready.
func (a *App) domReady(ctx context.Context) {}

// shutdown is called by Wails when the app is quitting.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		_ = a.db.Ent.Close()
	}
}

// DevReset deletes all demo data (except Settings)
func (a *App) DevReset() (int, error) {
	ctx := a.ctx
	db := a.db.Ent

	n1, err := db.AttendanceMonth.Delete().Exec(ctx)
	if err != nil {
		return 0, err
	}
	n2, err := db.Enrollment.Delete().Exec(ctx)
	if err != nil {
		return 0, err
	}
	n3, err := db.Course.Delete().Exec(ctx)
	if err != nil {
		return 0, err
	}
	n4, err := db.Student.Delete().Exec(ctx)
	if err != nil {
		return 0, err
	}

	return n1 + n2 + n3 + n4, nil
}

// DevSeed inserts demo data, avoiding duplicates
func (a *App) DevSeed() (int, error) {
	ctx := a.ctx
	db := a.db.Ent

	// students
	sAnna, err := db.Student.Query().Where(student.FullNameEQ("Anna K.")).Only(ctx)
	if err != nil {
		sAnna, _ = db.Student.Create().SetFullName("Anna K.").SetPhone("+37120000001").SetEmail("anna@example.com").Save(ctx)
	}
	sDima, err := db.Student.Query().Where(student.FullNameEQ("Dmitry L.")).Only(ctx)
	if err != nil {
		sDima, _ = db.Student.Create().SetFullName("Dmitry L.").SetPhone("+37120000002").SetEmail("dima@example.com").Save(ctx)
	}

	// courses
	cA2, err := db.Course.Query().Where(course.NameEQ("English A2 (group)")).Only(ctx)
	if err != nil {
		cA2, _ = db.Course.Create().
			SetName("English A2 (group)").
			SetType("group").SetLessonPrice(15).SetSubscriptionPrice(120).
			SetScheduleJSON(`{"daysOfWeek":[1,3]}`).
			Save(ctx)
	}
	cB1, err := db.Course.Query().Where(course.NameEQ("English B1 (individual)")).Only(ctx)
	if err != nil {
		cB1, _ = db.Course.Create().
			SetName("English B1 (individual)").
			SetType("individual").SetLessonPrice(25).SetSubscriptionPrice(0).
			Save(ctx)
	}

	now := time.Now()

	// enrollments (create only if not exists)
	if _, err := db.Enrollment.Query().
		Where(enrollment.StudentIDEQ(sAnna.ID), enrollment.CourseIDEQ(cA2.ID)).
		Only(ctx); err != nil {
		_, _ = db.Enrollment.Create().
			SetStudentID(sAnna.ID).SetCourseID(cA2.ID).
			SetBillingMode("subscription").SetStartDate(now).Save(ctx)
	}

	if _, err := db.Enrollment.Query().
		Where(enrollment.StudentIDEQ(sDima.ID), enrollment.CourseIDEQ(cA2.ID)).
		Only(ctx); err != nil {
		_, _ = db.Enrollment.Create().
			SetStudentID(sDima.ID).SetCourseID(cA2.ID).
			SetBillingMode("per_lesson").SetStartDate(now).Save(ctx)
	}

	if _, err := db.Enrollment.Query().
		Where(enrollment.StudentIDEQ(sDima.ID), enrollment.CourseIDEQ(cB1.ID)).
		Only(ctx); err != nil {
		_, _ = db.Enrollment.Create().
			SetStudentID(sDima.ID).SetCourseID(cB1.ID).
			SetBillingMode("per_lesson").SetStartDate(now).Save(ctx)
	}

	return 2, nil // return the number of base students
}

// Delete enrollment (for the "Delete" button in the attendance sheet)
func (a *App) EnrollmentDelete(enrollmentID int) error {
	return a.att.DeleteEnrollment(a.ctx, enrollmentID)
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
		"base": a.dirs.Base, "data": a.dirs.Data, "backups": a.dirs.Backups,
		"invoices": a.dirs.Invoices, "exports": a.dirs.Exports,
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

// ---------- Invoice issuing & PDF bindings ----------

type InvoiceListItem = invsvc.ListItem
type InvoiceDTO = invsvc.InvoiceDTO

func (a *App) InvoiceGenerateDrafts(year, month int) (int, error) {
	return a.inv.GenerateDrafts(a.ctx, year, month)
}

func (a *App) InvoiceListDrafts(year, month int) ([]InvoiceListItem, error) {
	return a.inv.ListDrafts(a.ctx, year, month)
}

func (a *App) InvoiceGet(id int) (*InvoiceDTO, error) {
	return a.inv.Get(a.ctx, id)
}

func (a *App) InvoiceDeleteDraft(id int) error {
	return a.inv.DeleteDraft(a.ctx, id)
}

// Result of "issue one invoice"
type IssueResult struct {
	Number  string `json:"number"`
	PdfPath string `json:"pdfPath"`
}

type IssueAllResult struct {
	Count    int      `json:"count"`
	PdfPaths []string `json:"pdfPaths"`
}

// Список по статусу
func (a *App) InvoiceList(year, month int, status string) ([]invsvc.ListItem, error) {
	return a.inv.List(a.ctx, year, month, status)
}

// Выставить один: возвращаем объект
func (a *App) InvoiceIssue(id int) (IssueResult, error) {
	num, path, err := a.inv.Issue(a.ctx, id, a.dirs.Invoices, filepath.Join(a.dirs.Base, "Fonts"))
	if err != nil {
		return IssueResult{}, err
	}
	return IssueResult{Number: num, PdfPath: path}, nil
}

// Выставить все за период
func (a *App) InvoiceIssueAll(year, month int) (IssueAllResult, error) {
	cnt, paths, err := a.inv.IssueAll(a.ctx, year, month, a.dirs.Invoices, filepath.Join(a.dirs.Base, "Fonts"))
	if err != nil {
		return IssueAllResult{}, err
	}
	return IssueAllResult{Count: cnt, PdfPaths: paths}, nil
}

// Открыть файл (PDF) в ОС
func (a *App) OpenFile(path string) error {
	// на всякий случай приводим к абсолютному пути
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	// в Wails v2 открываем URL через браузер
	runtime.BrowserOpenURL(a.ctx, "file://"+path)
	return nil
}
