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
	"strings"
	"time"

	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/settings"
	"langschool/internal/app"
	"langschool/internal/app/attendance"

	invsvc "langschool/internal/app/invoice"
	"langschool/internal/infra"
	"langschool/internal/paths"

	paysvc "langschool/internal/app/payment"
)

// App is the main application struct that holds all application state and services.
// It provides methods that are bound to the frontend via Wails, allowing the
// frontend to interact with the database, file system, and business logic.
type App struct {
	ctx       context.Context // Application context for database operations
	dirs      paths.Dirs      // Application directory paths (data, backups, invoices, etc.)
	db        *infra.DB       // Database connection wrapper
	appDBPath string          // Path to the SQLite database file

	// services provide business logic for different domains
	att *attendance.Service // Attendance tracking service
	inv *invsvc.Service     // Invoice generation and management service
	pay *paysvc.Service     // Payment processing service
}

// NewApp creates a new App instance. The instance is initialized with
// default values and will be fully configured during the startup lifecycle hook.
func NewApp() *App { return &App{} }

// startup is called by Wails when the application starts.
// It initializes the database connection, ensures required directories exist,
// creates the singleton Settings record if it doesn't exist, and initializes
// all service instances. This is where all one-time setup logic runs.
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
		Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
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
			SetLocale("lv-LV").
			Save(ctx); err != nil {
			log.Fatal(err)
		}
	}

	// initialize services
	a.att = attendance.New(a.db.Ent)
	a.inv = invsvc.New(a.db.Ent)
	a.pay = paysvc.New(a.db.Ent)
}

// domReady is called by Wails when the frontend DOM is ready.
// Currently unused, but available for any initialization that needs to
// happen after the frontend has fully loaded.
func (a *App) domReady(ctx context.Context) {}

// shutdown is called by Wails when the application is quitting.
// It performs cleanup operations, such as closing the database connection
// to ensure data integrity and proper resource cleanup.
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		_ = a.db.Ent.Close()
	}
}

// EnrollmentDelete deletes an enrollment and all associated attendance records.
// This is used by the attendance sheet UI to remove enrollments.
// The deletion is handled by the attendance service to ensure all related
// attendance data is properly cleaned up.
func (a *App) EnrollmentDelete(enrollmentID int) error {
	return a.att.DeleteEnrollment(a.ctx, enrollmentID)
}

// userHome returns the user's home directory path.
// Falls back to "." if the home directory cannot be determined.
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

// Ping is a simple health check endpoint that returns "ok".
// Used to verify that the backend is responding to frontend calls.
func (a *App) Ping() string { return "ok" }

// Greet returns a greeting message for the given name.
// Used as a simple test function to verify frontend-backend communication.
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

// AttendanceListPerLesson retrieves attendance records for per-lesson billing
// enrollments for the specified year, month, and optionally filtered by course.
// Returns a list of rows containing student, course, and attendance count information.
func (a *App) AttendanceListPerLesson(year, month int, courseID *int) ([]attendance.Row, error) {
	return a.att.ListPerLesson(a.ctx, year, month, courseID)
}

// AttendanceUpsert creates or updates an attendance record for a student-course
// pair for a specific month. If the month is locked, the update will fail.
func (a *App) AttendanceUpsert(studentID, courseID, year, month, count int) error {
	return a.att.Upsert(a.ctx, studentID, courseID, year, month, count)
}

// AttendanceAddOne increments the lesson count by 1 for all unlocked attendance
// records matching the filter (year, month, optional courseID).
// Returns the number of records that were successfully updated.
func (a *App) AttendanceAddOne(year, month int, courseID *int) (int, error) {
	return a.att.AddOneForFilter(a.ctx, year, month, courseID)
}

// AttendanceSetLocked sets the locked status for all attendance records matching
// the filter (year, month, optional courseID). Locked records cannot be modified.
// Returns the number of records that were updated.
func (a *App) AttendanceSetLocked(year, month int, courseID *int, lock bool) (int, error) {
	return a.att.SetLocked(a.ctx, year, month, courseID, lock)
}

// ---------- Invoice issuing & PDF bindings ----------

// Type aliases for invoice DTOs to simplify the API surface
type InvoiceListItem = invsvc.ListItem
type InvoiceDTO = invsvc.InvoiceDTO

// InvoiceGenerateDrafts generates draft invoices for all active students
// for the specified year and month. Existing drafts are rebuilt, while
// issued/paid/canceled invoices are skipped. Returns statistics about
// the generation process.
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

// InvoiceListDrafts returns a list of draft invoices for the specified year and month.
func (a *App) InvoiceListDrafts(year, month int) ([]InvoiceListItem, error) {
	return a.inv.ListDrafts(a.ctx, year, month)
}

// InvoiceGet retrieves a single invoice by ID with all its line items.
// Works for invoices in any status (draft, issued, paid, canceled).
func (a *App) InvoiceGet(id int) (*InvoiceDTO, error) {
	return a.inv.Get(a.ctx, id)
}

// InvoiceDeleteDraft deletes a draft invoice and all its line items.
// Only draft invoices can be deleted; issued/paid/canceled invoices are protected.
func (a *App) InvoiceDeleteDraft(id int) error {
	return a.inv.DeleteDraft(a.ctx, id)
}

// IssueResult contains the result of issuing a single invoice.
type IssueResult struct {
	Number  string `json:"number"`  // The assigned invoice number
	PdfPath string `json:"pdfPath"` // Path to the generated PDF file
}

// IssueAllResult contains the result of issuing all draft invoices for a period.
type IssueAllResult struct {
	Count    int      `json:"count"`    // Number of invoices issued
	PdfPaths []string `json:"pdfPaths"` // Paths to all generated PDF files
}

// InvoiceList returns invoices for the specified year and month, optionally
// filtered by status. Status can be "draft", "issued", "paid", "canceled", or "all".
func (a *App) InvoiceList(year, month int, status string) ([]invsvc.ListItem, error) {
	return a.inv.List(a.ctx, year, month, status)
}

// InvoiceIssue issues a single draft invoice by assigning it a number,
// changing its status to "issued", and generating a PDF file.
// Returns the invoice number and path to the generated PDF.
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

// InvoiceIssueAll issues all draft invoices for the specified year and month.
// Each invoice is assigned a number, marked as issued, and a PDF is generated.
// Returns the count of issued invoices and paths to all generated PDFs.
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
// Only allows opening files within the LangSchool directory tree to prevent path traversal attacks.
func (a *App) OpenFile(path string) error {
	// Normalize the path
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}

	// Security check: Ensure file is within allowed directories
	allowedBase := filepath.Clean(a.dirs.Base)
	cleanPath := filepath.Clean(path)

	// Check if the path is within the LangSchool directory
	if !strings.HasPrefix(cleanPath, allowedBase) {
		return fmt.Errorf("access denied: file must be within %s directory", allowedBase)
	}

	// Verify file exists
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

// InvoiceEnsurePDF ensures that a PDF exists for an issued invoice.
// If the PDF already exists, returns its path. Otherwise, regenerates it.
// Only works for invoices that have been issued (have a number).
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

// DevClearInvoices is a development utility that deletes all invoices
// (and their line items) for a specific period. Use with caution as
// this permanently removes financial records.
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

// SettingsSetLocale updates the application locale setting.
// The locale affects date, number, and currency formatting in invoices.
func (a *App) SettingsSetLocale(loc string) error {
	_, err := a.db.Ent.Settings.
		Update().Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
		SetLocale(loc).
		Save(a.ctx)
	return err
}

// ---------- Payment bindings ----------

// Type aliases for payment-related DTOs
type PaymentDTO = paysvc.PaymentDTO
type BalanceDTO = paysvc.BalanceDTO
type DebtorDTO = paysvc.DebtorDTO
type InvoiceSummaryDTO = paysvc.InvoiceSummaryDTO

// PaymentCreate creates a new payment record. The paidAt parameter accepts
// either "YYYY-MM-DD" format or RFC3339. If invoiceID is provided, the payment
// is linked to that invoice and the invoice status may be updated automatically.
func (a *App) PaymentCreate(studentID int, invoiceID *int, amount float64, method string, paidAt string, note string) (*PaymentDTO, error) {
	return a.pay.Create(a.ctx, studentID, invoiceID, amount, method, paidAt, note)
}

// PaymentDelete deletes a payment record. If the payment was linked to an invoice,
// the invoice status will be recomputed (e.g., from "paid" back to "issued" if needed).
func (a *App) PaymentDelete(paymentID int) error {
	return a.pay.Delete(a.ctx, paymentID)
}

// PaymentListForStudent returns all payments for a specific student,
// ordered by payment date (most recent first).
func (a *App) PaymentListForStudent(studentID int) ([]PaymentDTO, error) {
	return a.pay.ListForStudent(a.ctx, studentID)
}

// StudentBalance calculates the financial balance for a student, including
// total invoiced amount, total paid amount, current balance, and debt (if any).
func (a *App) StudentBalance(studentID int) (*BalanceDTO, error) {
	return a.pay.StudentBalance(a.ctx, studentID)
}

// DebtorsList returns a list of all active students who have outstanding debt,
// sorted by debt amount (highest first).
func (a *App) DebtorsList() ([]DebtorDTO, error) {
	return a.pay.ListDebtors(a.ctx)
}

// InvoicePaymentSummary returns a summary of payment status for a specific invoice,
// including total amount, paid amount, remaining balance, and current status.
func (a *App) InvoicePaymentSummary(invoiceID int) (*InvoiceSummaryDTO, error) {
	return a.pay.InvoiceSummary(a.ctx, invoiceID)
}

// PaymentQuickCash creates an unlinked cash payment for immediate payment scenarios
// (e.g., "cash for lesson now"). The payment is not linked to any invoice.
func (a *App) PaymentQuickCash(studentID int, amount float64, note string) (*PaymentDTO, error) {
	return a.pay.QuickCash(a.ctx, studentID, amount, note)
}
