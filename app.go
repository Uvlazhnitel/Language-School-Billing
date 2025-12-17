// app.go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	rt "runtime"
	"time"

	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/settings"
	"langschool/ent/student"
	"langschool/internal/app/attendance"

	invsvc "langschool/internal/app/invoice"
	"langschool/internal/infra"
	"langschool/internal/paths"

	paysvc "langschool/internal/app/payment"
)

type App struct {
	ctx       context.Context
	dirs      paths.Dirs
	db        *infra.DB
	appDBPath string

	// services
	att *attendance.Service
	inv *invsvc.Service
	pay *paysvc.Service
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

	if err := a.db.Ent.Schema.Create(a.ctx); err != nil {
		log.Fatalf("schema migrate failed: %v", err)
	}

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
	a.pay = paysvc.New(a.db.Ent)
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
		sAnna, _ = db.Student.Create().
			SetFullName("Anna K.").
			SetPhone("+37120000001").
			SetEmail("anna@example.com").
			SetIsActive(true). // IMPORTANT
			Save(ctx)
	} else {
		_, _ = sAnna.Update().SetIsActive(true).Save(ctx) // IMPORTANT
	}

	sDima, err := db.Student.Query().Where(student.FullNameEQ("Dmitry L.")).Only(ctx)
	if err != nil {
		sDima, _ = db.Student.Create().
			SetFullName("Dmitry L.").
			SetPhone("+37120000002").
			SetEmail("dima@example.com").
			SetIsActive(true). // IMPORTANT
			Save(ctx)
	} else {
		_, _ = sDima.Update().SetIsActive(true).Save(ctx) // IMPORTANT
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

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dirHasFonts checks that both TTF files are present in dir.
func dirHasFonts(dir string) bool {
	return fileExists(filepath.Join(dir, "DejaVuSans.ttf")) &&
		fileExists(filepath.Join(dir, "DejaVuSans-Bold.ttf"))
}

// resolveFontsDir tries multiple locations and logs the decision.
func (a *App) resolveFontsDir() (string, error) {
	var candidates []string

	// 1) Explicit env override
	if env := os.Getenv("LS_FONTS_DIR"); env != "" {
		candidates = append(candidates, env)
	}

	// 2) Our app data base: ~/LangSchool/Fonts
	candidates = append(candidates, filepath.Join(a.dirs.Base, "Fonts"))

	// 3) Next to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "Fonts"),
			filepath.Join(exeDir, "fonts"),
		)
	}

	// 4) Current working directory (dev)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "Fonts"),
			filepath.Join(wd, "fonts"),
		)
	}

	for _, c := range candidates {
		if dirHasFonts(c) {
			log.Printf("resolveFontsDir: using %s", c)
			return c, nil
		}
		log.Printf("resolveFontsDir: not found in %s", c)
	}

	return "", fmt.Errorf("DejaVuSans.ttf & DejaVuSans-Bold.ttf not found in any known location; set LS_FONTS_DIR or place fonts into ~/LangSchool/Fonts or ./Fonts")
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

func (a *App) InvoiceGenerateDrafts(year, month int) (invsvc.GenerateResult, error) {
	log.Printf("InvoiceGenerateDrafts called for %04d-%02d", year, month)
	res, err := a.inv.GenerateDrafts(a.ctx, year, month)
	if err != nil {
		log.Printf("InvoiceGenerateDrafts error: %v", err)
		return res, err
	}
	log.Printf("InvoiceGenerateDrafts result: created=%d updated=%d skippedHasInvoice=%d skippedNoLines=%d",
		res.Created, res.Updated, res.SkippedHasInvoice, res.SkippedNoLines)
	return res, nil
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

// List by status
func (a *App) InvoiceList(year, month int, status string) ([]invsvc.ListItem, error) {
	return a.inv.List(a.ctx, year, month, status)
}

// Issue one: return object
func (a *App) InvoiceIssue(id int) (IssueResult, error) {
	fonts, err := a.resolveFontsDir()
	if err != nil {
		return IssueResult{}, err
	}
	num, path, err := a.inv.Issue(a.ctx, id, a.dirs.Invoices, fonts)
	if err != nil {
		return IssueResult{}, err
	}
	return IssueResult{Number: num, PdfPath: path}, nil
}

// Issue all for the period
func (a *App) InvoiceIssueAll(year, month int) (IssueAllResult, error) {
	fonts, err := a.resolveFontsDir()
	if err != nil {
		return IssueAllResult{}, err
	}
	cnt, paths, err := a.inv.IssueAll(a.ctx, year, month, a.dirs.Invoices, fonts)
	if err != nil {
		return IssueAllResult{}, err
	}
	return IssueAllResult{Count: cnt, PdfPaths: paths}, nil
}

// Open file (PDF) in OS
func (a *App) OpenFile(path string) error {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch rt.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func (a *App) InvoiceEnsurePDF(id int) (string, error) {
	dto, err := a.inv.Get(a.ctx, id)
	if err != nil {
		return "", err
	}
	if dto.Number == nil || *dto.Number == "" {
		return "", fmt.Errorf("invoice not issued yet")
	}
	path := invsvc.PDFPathByNumber(a.dirs.Invoices, dto.Year, dto.Month, *dto.Number)
	log.Printf("EnsurePDF: invoice=%d number=%s want=%s", id, *dto.Number, path)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	fonts, err := a.resolveFontsDir()
	if err != nil {
		return "", err
	}
	log.Printf("EnsurePDF: regenerating with fonts=%s", fonts)
	_, p, err := a.inv.Issue(a.ctx, id, a.dirs.Invoices, fonts)
	if err != nil {
		return "", err
	}
	return p, nil
}

func (a *App) DevClearInvoices(year, month int) (int, error) {
	ctx := a.ctx
	db := a.db.Ent

	// Collect invoice IDs for the period
	invs, err := db.Invoice.
		Query().
		Where(
			invoice.PeriodYearEQ(year),
			invoice.PeriodMonthEQ(month),
		).All(ctx)
	if err != nil {
		return 0, err
	}
	for _, iv := range invs {
		if _, err := db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(iv.ID)).Exec(ctx); err != nil {
			return 0, err
		}
		if err := db.Invoice.DeleteOneID(iv.ID).Exec(ctx); err != nil {
			return 0, err
		}
	}
	return len(invs), nil
}

func (a *App) SettingsSetLocale(loc string) error {
	_, err := a.db.Ent.Settings.
		Update().Where(settings.SingletonIDEQ(1)).
		SetLocale(loc).
		Save(a.ctx)
	return err
}

// ---------- Payment bindings ----------

type PaymentDTO = paysvc.PaymentDTO
type BalanceDTO = paysvc.BalanceDTO
type DebtorDTO = paysvc.DebtorDTO
type InvoiceSummaryDTO = paysvc.InvoiceSummaryDTO

// PaymentCreate creates a payment. paidAt accepts "YYYY-MM-DD" or RFC3339.
func (a *App) PaymentCreate(studentID int, invoiceID *int, amount float64, method string, paidAt string, note string) (*PaymentDTO, error) {
	return a.pay.Create(a.ctx, studentID, invoiceID, amount, method, paidAt, note)
}

func (a *App) PaymentDelete(paymentID int) error {
	return a.pay.Delete(a.ctx, paymentID)
}

func (a *App) PaymentListForStudent(studentID int) ([]PaymentDTO, error) {
	return a.pay.ListForStudent(a.ctx, studentID)
}

func (a *App) StudentBalance(studentID int) (*BalanceDTO, error) {
	return a.pay.StudentBalance(a.ctx, studentID)
}

func (a *App) DebtorsList() ([]DebtorDTO, error) {
	return a.pay.ListDebtors(a.ctx)
}

func (a *App) InvoicePaymentSummary(invoiceID int) (*InvoiceSummaryDTO, error) {
	return a.pay.InvoiceSummary(a.ctx, invoiceID)
}

// QuickCash creates an unlinked cash payment (e.g. "cash for lesson now").
func (a *App) PaymentQuickCash(studentID int, amount float64, note string) (*PaymentDTO, error) {
	return a.pay.QuickCash(a.ctx, studentID, amount, note)
}
