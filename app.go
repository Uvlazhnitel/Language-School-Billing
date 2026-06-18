// app.go
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"langschool/ent/invoice"
	"langschool/ent/settings"
	"langschool/internal/app"
	"langschool/internal/app/attendance"
	"langschool/internal/backend"

	invsvc "langschool/internal/app/invoice"
	"langschool/internal/infra"
	"langschool/internal/paths"
	appruntime "langschool/internal/runtime"

	paysvc "langschool/internal/app/payment"
)

// App is a lightweight test/helper facade over the backend services.
// It remains in the root package so the legacy CRUD and invoice tests can
// exercise service behavior without going through the HTTP layer.
type App struct {
	ctx       context.Context // Application context for database operations
	dirs      paths.Dirs      // Application directory paths (data, backups, invoices, etc.)
	db        *infra.DB       // Database connection wrapper
	appDBPath string          // Path to the SQLite database file
	runtime   *appruntime.Runtime
	svc       *backend.Service

	// services provide business logic for different domains
	att *attendance.Service // Attendance tracking service
	inv *invsvc.Service     // Invoice generation and management service
	pay *paysvc.Service     // Payment processing service
}

const appDirName = appruntime.AppDirName
const legacyAppDirName = appruntime.LegacyAppDirName

func resolveAppBaseDir(home string) string {
	return appruntime.ResolveAppBaseDir(home)
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
	return appruntime.UserHome()
}

// resolveFontsDir tries multiple locations and logs the decision.
func (a *App) resolveFontsDir() (string, error) {
	cfg := appruntime.LoadConfig(userHome())
	if a.runtime != nil {
		cfg = a.runtime.Config
	}
	fontsDir, err := appruntime.ResolveFontsDir(cfg, a.dirs)
	if err == nil {
		log.Printf("resolveFontsDir: using %s", fontsDir)
	}
	return fontsDir, err
}

// BackupNow creates a timestamped copy of the SQLite DB in Backups/ and returns the file path.
func (a *App) BackupNow() (string, error) {
	if a.svc != nil {
		return a.svc.BackupNow()
	}
	return appruntime.BackupNow(a.appDBPath, a.dirs.Backups)
}

// backupBeforeMigration creates a timestamped copy of the existing SQLite DB
// before startup runs schema migrations. If the DB does not exist yet, the
// first-run startup continues without creating a backup.
func (a *App) backupBeforeMigration() (string, error) {
	return appruntime.BackupBeforeMigration(a.appDBPath, a.dirs.Backups)
}

func (a *App) cleanupOldPreMigrationBackups(limit int) error {
	return appruntime.CleanupOldPreMigrationBackups(a.dirs.Backups, limit)
}

func (a *App) attachRuntime(runtimeInstance *appruntime.Runtime) {
	a.runtime = runtimeInstance
	a.svc = backend.New(runtimeInstance)
	a.dirs = runtimeInstance.Dirs
	a.db = runtimeInstance.DB
	a.appDBPath = runtimeInstance.AppDBPath
	a.att = runtimeInstance.Attendance
	a.inv = runtimeInstance.Invoice
	a.pay = runtimeInstance.Payment
}

func (a *App) appContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func (a *App) ensureBackendService() *backend.Service {
	if a.svc != nil {
		return a.svc
	}
	if a.db == nil || a.db.Ent == nil {
		return nil
	}
	if a.att == nil {
		a.att = attendance.New(a.db.Ent)
	}
	runtimeInstance := &appruntime.Runtime{
		DB:         a.db,
		Attendance: a.att,
	}
	a.svc = backend.New(runtimeInstance)
	return a.svc
}

// ---------- Attendance bindings ----------

// AttendanceListPerLesson retrieves attendance records for per-lesson billing
// enrollments for the specified year, month, and optionally filtered by course.
// Returns a list of rows containing student, course, and tracked hours information.
func (a *App) AttendanceListPerLesson(year, month int, courseID *int) ([]attendance.Row, error) {
	if a.svc != nil {
		return a.svc.AttendanceListPerLesson(a.ctx, year, month, courseID)
	}
	return a.att.ListPerLesson(a.ctx, year, month, courseID)
}

// AttendanceUpsert creates or updates an attendance record for a student-course
// pair for a specific month.
func (a *App) AttendanceUpsert(studentID, courseID, year, month int, hours float64) error {
	if a.svc != nil {
		return a.svc.AttendanceUpsert(a.ctx, studentID, courseID, year, month, hours)
	}
	return a.att.Upsert(a.ctx, studentID, courseID, year, month, hours)
}

// AttendanceAddOne increments tracked hours by 0.25 for all attendance
// records matching the filter (year, month, optional courseID).
// Returns the number of records that were successfully updated.
func (a *App) AttendanceAddOne(year, month int, courseID *int) (int, error) {
	if a.svc != nil {
		return a.svc.AttendanceAddOne(a.ctx, year, month, courseID)
	}
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
	if a.svc != nil {
		return a.svc.InvoiceGenerateDrafts(a.ctx, year, month)
	}
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
	if a.svc != nil {
		return a.svc.InvoiceRebuildStudentDraft(a.ctx, studentID, year, month)
	}
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
	if a.svc != nil {
		return a.svc.InvoiceGet(a.ctx, id)
	}
	return a.inv.Get(a.ctx, id)
}

// InvoiceDeleteDraft deletes a draft invoice and all its line items.
// Only draft invoices can be deleted; issued/paid/canceled invoices are protected.
func (a *App) InvoiceDeleteDraft(id int) error {
	if a.svc != nil {
		return a.svc.InvoiceDeleteDraft(a.ctx, id)
	}
	dto, err := a.inv.Get(a.ctx, id)
	if err != nil {
		return err
	}
	return a.inv.DeleteDraftWithVersion(a.ctx, id, dto.Version)
}

// InvoiceReopenDraft moves an issued invoice with no payments back to draft state.
// The invoice lines remain unchanged so the draft can be reviewed and issued again.
func (a *App) InvoiceReopenDraft(id int) error {
	if a.svc != nil {
		return a.svc.InvoiceReopenDraft(a.ctx, id)
	}
	dto, err := a.inv.Get(a.ctx, id)
	if err != nil {
		return err
	}
	return a.inv.ReopenDraftWithVersion(a.ctx, id, dto.Version, a.dirs.Invoices)
}

// IssueResult contains the result of issuing a single invoice.
type IssueResult = backend.IssueResult

// IssueAllResult contains the result of issuing all draft invoices for a period.
type IssueAllResult = backend.IssueAllResult
type AuditLogListItem = backend.AuditLogListItem
type AuditLogListResult = backend.AuditLogListResult

// InvoiceList returns invoices for the specified year and month, optionally
// filtered by status. Status can be "draft", "issued", "paid", "canceled", or "all".
func (a *App) InvoiceList(year, month int, status string) ([]invsvc.ListItem, error) {
	if a.svc != nil {
		return a.svc.InvoiceList(a.ctx, year, month, status)
	}
	return a.inv.List(a.ctx, year, month, status)
}

// InvoiceIssue issues a single invoice, applies credit, and attempts to
// generate its canonical PDF immediately. If PDF generation fails after
// numbering, the invoice remains in a pending-PDF status.
func (a *App) InvoiceIssue(id int) (IssueResult, error) {
	if a.svc != nil {
		return a.svc.InvoiceIssue(a.ctx, id)
	}
	dto, err := a.inv.Get(a.ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	num, _, err := a.inv.IssueAndApplyCreditWithVersion(a.ctx, id, dto.Version)
	if err != nil {
		return IssueResult{}, err
	}
	if _, err := a.InvoiceEnsurePDF(id); err != nil {
		log.Printf("InvoiceIssue ensure PDF fallback for invoice %d failed: %v", id, err)
	}
	return IssueResult{Number: num}, nil
}

// InvoiceIssueAll issues all draft invoices for the specified year and month.
// Each invoice is assigned a number and then attempts PDF generation.
// Returns the count of PDFs generated and paths to the generated files.
func (a *App) InvoiceIssueAll(year, month int) (IssueAllResult, error) {
	if a.svc != nil {
		return a.svc.InvoiceIssueAll(a.ctx, year, month)
	}
	fonts, err := a.resolveFontsDir()
	if err != nil {
		return IssueAllResult{}, err
	}
	cnt, paths, err := a.inv.IssueAll(a.ctx, year, month, a.dirs.Invoices, fonts)
	if err != nil {
		return IssueAllResult{}, err
	}
	// Apply credit for all students whose invoices were issued in this period.
	items, err := a.inv.List(a.ctx, year, month, "all")
	if err != nil {
		return IssueAllResult{}, err
	}
	seen := make(map[int]struct{})
	for _, item := range items {
		if item.Status == app.InvoiceStatusDraft {
			continue
		}
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

// InvoiceEnsurePDF ensures that a canonical PDF exists for a numbered invoice.
// If a ready canonical PDF already exists, returns its path. Otherwise,
// regenerates it and upgrades pending statuses to ready statuses.
func (a *App) InvoiceEnsurePDF(id int) (string, error) {
	if a.svc != nil {
		return a.svc.InvoiceEnsurePDF(a.ctx, id)
	}
	iv, err := a.db.Ent.Invoice.Query().
		Where(invoice.IDEQ(id)).
		WithStudent().
		Only(a.ctx)
	if err != nil {
		return "", err
	}
	if iv.Number == nil || *iv.Number == "" {
		return "", fmt.Errorf("счёт ещё не выставлен")
	}
	subjectName := ""
	if iv.Edges.Student != nil {
		subjectName = iv.Edges.Student.FullName
	}
	info := invsvc.NewPDFLocator(a.dirs.Invoices).Evaluate(iv, subjectName)
	if info.Status == invsvc.PDFStatusReady {
		log.Printf("EnsurePDF: invoice=%d number=%s existing=%s", id, *iv.Number, info.Path)
		return info.Path, nil
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

// InvoiceHasPDF reports whether an invoice has a ready canonical PDF.
func (a *App) InvoiceHasPDF(id int) (bool, error) {
	if a.svc != nil {
		return a.svc.InvoiceHasPDF(a.ctx, id)
	}
	iv, err := a.db.Ent.Invoice.Query().
		Where(invoice.IDEQ(id)).
		WithStudent().
		Only(a.ctx)
	if err != nil {
		return false, err
	}
	subjectName := ""
	if iv.Edges.Student != nil {
		subjectName = iv.Edges.Student.FullName
	}
	info := invsvc.NewPDFLocator(a.dirs.Invoices).Evaluate(iv, subjectName)
	return info.Status == invsvc.PDFStatusReady, nil
}

// SettingsSetLocale updates the application locale setting.
// The locale affects date, number, and currency formatting in invoices.
func (a *App) SettingsSetLocale(loc string) error {
	if a.svc != nil {
		return a.svc.SettingsSetLocale(a.ctx, loc)
	}
	_, err := a.db.Ent.Settings.
		Update().Where(settings.SingletonIDEQ(app.SettingsSingletonID)).
		SetLocale(loc).
		Save(a.ctx)
	return err
}

// SettingsGetLocale returns the saved application locale.
func (a *App) SettingsGetLocale() (string, error) {
	if a.svc != nil {
		return a.svc.SettingsGetLocale(a.ctx)
	}
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
	if a.svc != nil {
		return a.svc.PaymentCreate(a.ctx, studentID, invoiceID, amount, method, paidAt, note)
	}
	return a.pay.Create(a.ctx, studentID, invoiceID, amount, method, paidAt, note)
}

// PaymentDelete deletes a payment record. If the payment was linked to an invoice,
// the invoice status will be recomputed (e.g., from "paid" back to "issued" if needed).
func (a *App) PaymentDelete(paymentID int) error {
	if a.svc != nil {
		return a.svc.PaymentDelete(a.ctx, paymentID)
	}
	return a.pay.Delete(a.ctx, paymentID)
}

// PaymentListForStudent returns all payments for a specific student,
// ordered by payment date (most recent first).
func (a *App) PaymentListForStudent(studentID int) ([]PaymentDTO, error) {
	if a.svc != nil {
		return a.svc.PaymentListForStudent(a.ctx, studentID)
	}
	return a.pay.ListForStudent(a.ctx, studentID)
}

// StudentBalance calculates the financial balance for a student, including
// total invoiced amount, total paid amount, current balance, and debt (if any).
func (a *App) StudentBalance(studentID int) (*BalanceDTO, error) {
	if a.svc != nil {
		return a.svc.StudentBalance(a.ctx, studentID)
	}
	return a.pay.StudentBalance(a.ctx, studentID)
}

// DebtorsList returns a list of all active students who have outstanding debt,
// sorted by debt amount (highest first).
func (a *App) DebtorsList() ([]DebtorDTO, error) {
	if a.svc != nil {
		return a.svc.DebtorsList(a.ctx)
	}
	return a.pay.ListDebtors(a.ctx)
}

// MonthOverview returns a read-only monthly dashboard snapshot.
func (a *App) MonthOverview(year, month int) (*MonthOverviewDTO, error) {
	if a.svc != nil {
		return a.svc.MonthOverview(a.ctx, year, month)
	}
	return a.pay.MonthOverview(a.ctx, year, month)
}

// RecentPayments returns the latest payments for dashboard activity feeds.
func (a *App) RecentPayments(limit int) ([]RecentPaymentDTO, error) {
	if a.svc != nil {
		return a.svc.RecentPayments(a.ctx, limit)
	}
	return a.pay.ListRecent(a.ctx, limit)
}

// StudentDebtDetails returns open invoice debt details for one student.
func (a *App) StudentDebtDetails(studentID int) ([]DebtInvoiceDTO, error) {
	if a.svc != nil {
		return a.svc.StudentDebtDetails(a.ctx, studentID)
	}
	return a.pay.StudentDebtDetails(a.ctx, studentID)
}

// InvoicePaymentSummary returns a summary of payment status for a specific invoice,
// including total amount, paid amount, remaining balance, and current status.
func (a *App) InvoicePaymentSummary(invoiceID int) (*InvoiceSummaryDTO, error) {
	if a.svc != nil {
		return a.svc.InvoicePaymentSummary(a.ctx, invoiceID)
	}
	return a.pay.InvoiceSummary(a.ctx, invoiceID)
}

// PaymentQuickCash creates an unlinked cash payment for immediate payment scenarios
// (e.g., "cash for lesson now"). The payment is not linked to any invoice.
func (a *App) PaymentQuickCash(studentID int, amount float64, note string) (*PaymentDTO, error) {
	if a.svc != nil {
		return a.svc.PaymentQuickCash(a.ctx, studentID, amount, note)
	}
	return a.pay.QuickCash(a.ctx, studentID, amount, note)
}

func (a *App) AuditLogList(q, actorLabel, entityType, action, dateFrom, dateTo string, page, pageSize int) (*AuditLogListResult, error) {
	if a.svc == nil {
		return &AuditLogListResult{Items: []AuditLogListItem{}, Total: 0, Page: 1, PageSize: 50}, nil
	}
	return a.svc.AuditLogList(a.ctx, backend.AuditLogListFilter{
		Query:      q,
		ActorLabel: actorLabel,
		EntityType: entityType,
		Action:     action,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Page:       page,
		PageSize:   pageSize,
	})
}

func (a *App) allowedFileRoots() []string {
	roots := []string{a.dirs.Base, a.dirs.Data, a.dirs.Backups, a.dirs.Invoices, a.dirs.Exports}
	seen := make(map[string]struct{}, len(roots))
	filtered := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		filtered = append(filtered, root)
	}
	return filtered
}
