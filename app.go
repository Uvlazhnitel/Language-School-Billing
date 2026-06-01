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
	"sort"
	"strings"
	"time"

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

const appDisplayName = "StudentDesk"
const appDirName = "StudentDesk"
const legacyAppDirName = "LangSchool"
const defaultSchoolDisplayName = "ArtLab"
const defaultSchoolAddress = "Latgales iela 260, Rīga, Latvija"
const preMigrationBackupLimit = 30

// startup is called by Wails when the application starts.
// It initializes the database connection, ensures required directories exist,
// creates the singleton Settings record if it doesn't exist, and initializes
// all service instances. This is where all one-time setup logic runs.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	base := resolveAppBaseDir(userHome())
	dirs, err := paths.Ensure(base)
	if err != nil {
		log.Fatal(err)
	}
	a.dirs = dirs

	a.appDBPath = filepath.Join(dirs.Data, "app.sqlite")
	log.Println("Data path:", a.appDBPath)

	if backupPath, err := a.backupBeforeMigration(); err != nil {
		log.Fatal(err)
	} else if backupPath != "" {
		log.Println("Pre-migration backup created:", backupPath)
		if err := a.cleanupOldPreMigrationBackups(preMigrationBackupLimit); err != nil {
			log.Fatal(err)
		}
	}

	db, err := infra.Open(ctx, a.appDBPath)
	if err != nil {
		log.Fatal(err)
	}
	a.db = db
	log.Println("DB ready")

	if err := migrateLegacyCourseTeachers(ctx, a.db.Ent); err != nil {
		log.Fatal(err)
	}

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
			SetOrgName(defaultSchoolDisplayName).
			SetAddress(defaultSchoolAddress).
			SetInvoicePrefix("LS").
			SetNextSeq(1).
			SetInvoiceDayOfMonth(1).
			SetCurrency("EUR").
			SetLocale("en-US").
			Save(ctx); err != nil {
			log.Fatal(err)
		}
	} else {
		st, err := a.db.Ent.Settings.
			Query().
			Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
			Only(ctx)
		if err != nil {
			log.Fatal(err)
		}

		upd := a.db.Ent.Settings.
			Update().
			Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
			SetCurrency("EUR")

		orgName := strings.TrimSpace(st.OrgName)
		if orgName == "" || strings.EqualFold(orgName, "North Star Language Studio") {
			upd.SetOrgName(defaultSchoolDisplayName)
		}

		address := strings.TrimSpace(st.Address)
		if address == "" || strings.EqualFold(address, "Brivibas iela 88, Riga, Latvia") {
			upd.SetAddress(defaultSchoolAddress)
		}

		if _, err := upd.Save(ctx); err != nil {
			log.Fatal(err)
		}
	}

	// initialize services
	a.att = attendance.New(a.db.Ent)
	a.inv = invsvc.New(a.db.Ent)
	a.pay = paysvc.New(a.db.Ent)
}

func resolveAppBaseDir(home string) string {
	base := filepath.Join(home, appDirName)
	legacyBase := filepath.Join(home, legacyAppDirName)

	if info, err := os.Stat(base); err == nil && info.IsDir() {
		return base
	}

	if info, err := os.Stat(legacyBase); err == nil && info.IsDir() {
		if err := os.Rename(legacyBase, base); err == nil {
			log.Printf("Migrated app data directory from %s to %s", legacyBase, base)
			return base
		}
		log.Printf("Using legacy app data directory %s because migration to %s failed", legacyBase, base)
		return legacyBase
	}

	return base
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

	// 2) Our app data base: ~/StudentDesk/Fonts
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

	return "", fmt.Errorf("DejaVuSans.ttf & DejaVuSans-Bold.ttf not found in any known location; set LS_FONTS_DIR or place fonts into ~/StudentDesk/Fonts or ./Fonts")
}

// ---------- App info / utilities ----------

// AppDirs returns application directories for UI (useful for exports/backups).
func (a *App) AppDirs() map[string]string {
	if a.dirs.Base == "" {
		base := resolveAppBaseDir(userHome())
		dirs, err := paths.Ensure(base)
		if err == nil {
			a.dirs = dirs
		}
	}
	return map[string]string{
		"base": a.dirs.Base, "data": a.dirs.Data, "backups": a.dirs.Backups,
		"invoices": a.dirs.Invoices, "exports": a.dirs.Exports,
	}
}

// AppReady reports whether startup completed enough for frontend data calls.
func (a *App) AppReady() bool {
	return a.ctx != nil && a.db != nil && a.db.Ent != nil && a.att != nil && a.inv != nil && a.pay != nil
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

// backupBeforeMigration creates a timestamped copy of the existing SQLite DB
// before startup runs schema migrations. If the DB does not exist yet, the
// first-run startup continues without creating a backup.
func (a *App) backupBeforeMigration() (string, error) {
	if a.appDBPath == "" {
		return "", fmt.Errorf("db path is empty")
	}

	info, err := os.Stat(a.appDBPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("db path points to a directory: %s", a.appDBPath)
	}

	ts := time.Now().Format("20060102-150405")
	dst := filepath.Join(a.dirs.Backups, fmt.Sprintf("pre-migration-%s.sqlite", ts))
	if err := copyFile(a.appDBPath, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func (a *App) cleanupOldPreMigrationBackups(limit int) error {
	if limit <= 0 {
		return fmt.Errorf("backup retention limit must be positive")
	}

	entries, err := os.ReadDir(a.dirs.Backups)
	if err != nil {
		return err
	}

	type backupFile struct {
		path    string
		modTime time.Time
	}

	backups := make([]backupFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "pre-migration-") || !strings.HasSuffix(name, ".sqlite") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		backups = append(backups, backupFile{
			path:    filepath.Join(a.dirs.Backups, name),
			modTime: info.ModTime(),
		})
	}

	if len(backups) <= limit {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	for _, backup := range backups[limit:] {
		if err := os.Remove(backup.path); err != nil {
			return err
		}
	}

	return nil
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
// Returns a list of rows containing student, course, and tracked hours information.
func (a *App) AttendanceListPerLesson(year, month int, courseID *int) ([]attendance.Row, error) {
	return a.att.ListPerLesson(a.ctx, year, month, courseID)
}

// AttendanceUpsert creates or updates an attendance record for a student-course
// pair for a specific month.
func (a *App) AttendanceUpsert(studentID, courseID, year, month int, hours float64) error {
	return a.att.Upsert(a.ctx, studentID, courseID, year, month, hours)
}

// AttendanceAddOne increments tracked hours by 0.25 for all attendance
// records matching the filter (year, month, optional courseID).
// Returns the number of records that were successfully updated.
func (a *App) AttendanceAddOne(year, month int, courseID *int) (int, error) {
	return a.att.AddOneForFilter(a.ctx, year, month, courseID)
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

// InvoiceRebuildStudentDraft rebuilds the draft invoice for a single student
// in the specified year and month. Issued, paid, and canceled invoices are skipped.
func (a *App) InvoiceRebuildStudentDraft(studentID, year, month int) (invsvc.GenerateResult, error) {
	log.Printf("InvoiceRebuildStudentDraft called for student=%d period=%04d-%02d", studentID, year, month)
	res, err := a.inv.RebuildStudentDraft(a.ctx, studentID, year, month)
	if err != nil {
		log.Printf("InvoiceRebuildStudentDraft error: %v", err)
		return res, err
	}
	log.Printf("InvoiceRebuildStudentDraft result: created=%d updated=%d skippedHasInvoice=%d skippedNoLines=%d",
		res.Created, res.Updated, res.SkippedHasInvoice, res.SkippedNoLines)
	return res, nil
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

// InvoiceReopenDraft moves an issued invoice with no payments back to draft state.
// The invoice lines remain unchanged so the draft can be reviewed and issued again.
func (a *App) InvoiceReopenDraft(id int) error {
	return a.inv.ReopenDraft(a.ctx, id, a.dirs.Invoices)
}

// IssueResult contains the result of issuing a single invoice.
type IssueResult struct {
	Number string `json:"number"` // The assigned invoice number
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

// InvoiceIssue issues a single draft invoice by assigning it a number
// and changing its status to "issued". PDF generation is handled separately
// by InvoiceEnsurePDF when the user explicitly requests a document.
func (a *App) InvoiceIssue(id int) (IssueResult, error) {
	num, err := a.inv.IssueOne(a.ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	dto, err := a.inv.Get(a.ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	if err := a.pay.ApplyCreditToOldestInvoices(a.ctx, dto.StudentID); err != nil {
		return IssueResult{}, err
	}
	return IssueResult{Number: num}, nil
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
	// Apply credit for all students whose invoices were issued in this period.
	items, err := a.inv.List(a.ctx, year, month, app.InvoiceStatusIssued)
	if err != nil {
		return IssueAllResult{}, err
	}
	seen := make(map[int]struct{})
	for _, item := range items {
		if _, ok := seen[item.StudentID]; ok {
			continue
		}
		seen[item.StudentID] = struct{}{}
		if err := a.pay.ApplyCreditToOldestInvoices(a.ctx, item.StudentID); err != nil {
			return IssueAllResult{}, err
		}
	}
	return IssueAllResult{Count: cnt, PdfPaths: paths}, nil
}

// OpenFile opens a directory or reveals a file in the OS file manager.
// Only allows paths within the StudentDesk directory tree to prevent path traversal attacks.
func (a *App) OpenFile(path string) error {
	// Normalize the path
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}

	// Security check: Ensure file is within allowed directories
	allowedBase := filepath.Clean(a.dirs.Base)
	cleanPath := filepath.Clean(path)

	// Check if the path is within the StudentDesk directory
	if !strings.HasPrefix(cleanPath, allowedBase) {
		return fmt.Errorf("доступ запрещён: файл должен находиться внутри каталога %s", allowedBase)
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd

	if info.IsDir() {
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

	switch rt.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,"+path)
	default:
		// Linux file-manager support for selecting a file is inconsistent,
		// so fall back to opening the containing directory.
		cmd = exec.Command("xdg-open", filepath.Dir(path))
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
		return "", fmt.Errorf("счёт ещё не выставлен")
	}
	paths := a.invoicePDFPaths(dto)
	for _, path := range paths {
		log.Printf("EnsurePDF: invoice=%d number=%s want=%s", id, *dto.Number, path)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
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

// InvoiceHasPDF reports whether an issued invoice already has a PDF file on disk.
// It checks both the current named file convention and the legacy number-only path.
func (a *App) InvoiceHasPDF(id int) (bool, error) {
	dto, err := a.inv.Get(a.ctx, id)
	if err != nil {
		return false, err
	}
	if dto.Number == nil || *dto.Number == "" {
		return false, nil
	}

	for _, path := range a.invoicePDFPaths(dto) {
		if _, err := os.Stat(path); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}

	return false, nil
}

func (a *App) invoicePDFPaths(dto *invsvc.InvoiceDTO) []string {
	subjectName := dto.StudentName
	if dto.IsMinor && strings.TrimSpace(dto.ChildName) != "" {
		subjectName = dto.ChildName
	}
	return []string{
		invsvc.PDFPathByNumberAndName(a.dirs.Invoices, dto.Year, dto.Month, *dto.Number, subjectName),
		invsvc.PDFPathByNumber(a.dirs.Invoices, dto.Year, dto.Month, *dto.Number),
	}
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

// SettingsGetLocale returns the saved application locale.
func (a *App) SettingsGetLocale() (string, error) {
	if a.db == nil || a.db.Ent == nil {
		return "en-US", nil
	}
	st, err := a.db.Ent.Settings.
		Query().
		Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
		Only(a.ctx)
	if err != nil {
		return "", err
	}
	return st.Locale, nil
}

// ---------- Payment bindings ----------

// Type aliases for payment-related DTOs
type PaymentDTO = paysvc.PaymentDTO
type BalanceDTO = paysvc.BalanceDTO
type DebtorDTO = paysvc.DebtorDTO
type InvoiceSummaryDTO = paysvc.InvoiceSummaryDTO
type DebtInvoiceDTO = paysvc.DebtInvoiceDTO
type MonthOverviewDTO = paysvc.MonthOverviewDTO
type RecentPaymentDTO = paysvc.RecentPaymentDTO

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

// MonthOverview returns a read-only monthly dashboard snapshot.
func (a *App) MonthOverview(year, month int) (*MonthOverviewDTO, error) {
	return a.pay.MonthOverview(a.ctx, year, month)
}

// RecentPayments returns the latest payments for dashboard activity feeds.
func (a *App) RecentPayments(limit int) ([]RecentPaymentDTO, error) {
	return a.pay.ListRecent(a.ctx, limit)
}

// StudentDebtDetails returns open invoice debt details for one student.
func (a *App) StudentDebtDetails(studentID int) ([]DebtInvoiceDTO, error) {
	return a.pay.StudentDebtDetails(a.ctx, studentID)
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
