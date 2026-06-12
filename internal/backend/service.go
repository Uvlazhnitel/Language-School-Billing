package backend

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/payment"
	"langschool/ent/settings"
	"langschool/ent/student"
	"langschool/ent/teacher"
	sharedapp "langschool/internal/app"
	"langschool/internal/app/attendance"
	auditsvc "langschool/internal/app/audit"
	invsvc "langschool/internal/app/invoice"
	paysvc "langschool/internal/app/payment"
	"langschool/internal/apperrors"
	"langschool/internal/auth"
	"langschool/internal/money"
	appruntime "langschool/internal/runtime"
)

const (
	CourseTypeGroup         = sharedapp.CourseTypeGroup
	CourseTypeIndividual    = sharedapp.CourseTypeIndividual
	BillingModeSubscription = sharedapp.BillingModeSubscription
	BillingModePerLesson    = sharedapp.BillingModePerLesson
)

type StudentDTO struct {
	ID           int     `json:"id"`
	Version      int     `json:"version"`
	FullName     string  `json:"fullName"`
	PersonalCode string  `json:"personalCode"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Note         string  `json:"note"`
	IsMinor      bool    `json:"isMinor"`
	PayerName    string  `json:"payerName"`
	PayerRole    string  `json:"payerRole"`
	IsActive     bool    `json:"isActive"`
	Balance      float64 `json:"balance"`
	Debt         float64 `json:"debt"`
}

type CourseDTO struct {
	ID                int     `json:"id"`
	Version           int     `json:"version"`
	Name              string  `json:"name"`
	TeacherID         *int    `json:"teacherId,omitempty"`
	TeacherName       string  `json:"teacherName"`
	Type              string  `json:"type"`
	LessonPrice       float64 `json:"lessonPrice"`
	SubscriptionPrice float64 `json:"subscriptionPrice"`
}

type EnrollmentDTO struct {
	ID                      int     `json:"id"`
	Version                 int     `json:"version"`
	StudentID               int     `json:"studentId"`
	StudentName             string  `json:"studentName"`
	CourseID                int     `json:"courseId"`
	CourseName              string  `json:"courseName"`
	CourseType              string  `json:"courseType"`
	TeacherID               *int    `json:"teacherId,omitempty"`
	TeacherName             string  `json:"teacherName"`
	BillingMode             string  `json:"billingMode"`
	DiscountPct             float64 `json:"discountPct"`
	SubscriptionDiscountPct float64 `json:"subscriptionDiscountPct"`
	Note                    string  `json:"note"`
	CreatedAt               string  `json:"createdAt"`
}

type CourseMonthSubscriptionDTO struct {
	CourseID    int     `json:"courseId"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	LessonsHeld float64 `json:"lessonsHeld"`
}

type TeacherDTO struct {
	ID       int    `json:"id"`
	FullName string `json:"fullName"`
	IsActive bool   `json:"isActive"`
}

type InvoiceListItem = invsvc.ListItem
type InvoiceDTO = invsvc.InvoiceDTO
type PaymentDTO = paysvc.PaymentDTO
type BalanceDTO = paysvc.BalanceDTO
type DebtorDTO = paysvc.DebtorDTO
type InvoiceSummaryDTO = paysvc.InvoiceSummaryDTO
type DebtInvoiceDTO = paysvc.DebtInvoiceDTO
type MonthOverviewDTO = paysvc.MonthOverviewDTO
type RecentPaymentDTO = paysvc.RecentPaymentDTO
type AttendanceRow = attendance.Row

type IssueResult struct {
	Number string `json:"number"`
}

type IssueAllResult struct {
	Count    int      `json:"count"`
	PdfPaths []string `json:"pdfPaths"`
}

type Meta struct {
	Ready        bool            `json:"ready"`
	Locale       string          `json:"locale"`
	Capabilities map[string]bool `json:"capabilities"`
}

type SessionDTO struct {
	Authenticated bool            `json:"authenticated"`
	User          *auth.UserInfo  `json:"user,omitempty"`
	Locale        string          `json:"locale"`
	Capabilities  map[string]bool `json:"capabilities"`
	Ready         bool            `json:"ready"`
}

type UserDTO = auth.UserRecord

type Service struct {
	rt *appruntime.Runtime
}

func New(rt *appruntime.Runtime) *Service {
	return &Service{rt: rt}
}

func (s *Service) Ready() bool {
	return s != nil && s.rt != nil && s.rt.DB != nil && s.rt.DB.Ent != nil && s.rt.Attendance != nil && s.rt.Invoice != nil && s.rt.Payment != nil
}

func (s *Service) Meta(ctx context.Context) (*Meta, error) {
	locale, err := s.SettingsGetLocale(ctx)
	if err != nil {
		return nil, err
	}
	return &Meta{
		Ready:        s.Ready(),
		Locale:       locale,
		Capabilities: capabilitiesForRole(auth.RoleAdmin, false),
	}, nil
}

func (s *Service) SessionState(ctx context.Context, currentUser *auth.UserInfo) (*SessionDTO, error) {
	locale := "lv-LV"
	if currentUser != nil {
		locale = normalizeUILocale(currentUser.UILocale)
	}
	return &SessionDTO{
		Authenticated: currentUser != nil,
		User:          currentUser,
		Locale:        locale,
		Capabilities:  capabilitiesForCurrentUser(currentUser, false),
		Ready:         s.Ready(),
	}, nil
}

func (s *Service) Login(ctx context.Context, username, password string, rememberMe bool) (*auth.UserInfo, string, time.Time, bool, error) {
	if s.rt == nil || s.rt.Auth == nil {
		return nil, "", time.Time{}, false, auth.ErrUnauthorized
	}
	return s.rt.Auth.Login(ctx, username, password, rememberMe)
}

func (s *Service) Session(ctx context.Context, signedToken string) (*auth.UserInfo, error) {
	if s.rt == nil || s.rt.Auth == nil {
		return nil, auth.ErrUnauthorized
	}
	return s.rt.Auth.Session(ctx, signedToken)
}

func (s *Service) Logout(ctx context.Context, signedToken string) error {
	if s.rt == nil || s.rt.Auth == nil {
		return nil
	}
	return s.rt.Auth.Logout(ctx, signedToken)
}

func (s *Service) SessionCookie(signedToken string, expiresAt time.Time, persistent bool) *http.Cookie {
	if s.rt == nil || s.rt.Auth == nil {
		return nil
	}
	return s.rt.Auth.SessionCookie(signedToken, expiresAt, persistent)
}

func (s *Service) ClearSessionCookie() *http.Cookie {
	if s.rt == nil || s.rt.Auth == nil {
		return nil
	}
	return s.rt.Auth.ClearSessionCookie()
}

func (s *Service) BackupNow() (string, error) {
	return appruntime.BackupNow(s.rt.AppDBPath, s.rt.Dirs.Backups)
}

func (s *Service) FullBackupNow() (string, error) {
	path, err := appruntime.FullBackupNow(s.rt.AppDBPath, s.rt.Dirs.Invoices, s.rt.Dirs.Backups)
	if err != nil {
		return "", err
	}
	if err := appruntime.CleanupOldFullBackups(s.rt.Dirs.Backups, appruntime.FullBackupLimit); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) UserList(ctx context.Context) ([]UserDTO, error) {
	return s.rt.Auth.ListUsers(ctx)
}

func (s *Service) UserCreate(ctx context.Context, username, password, role string) (*UserDTO, error) {
	return s.rt.Auth.CreateUser(ctx, username, password, role)
}

func (s *Service) UserUpdate(ctx context.Context, id int, username, role string, isActive bool) (*UserDTO, error) {
	return s.rt.Auth.UpdateUser(ctx, id, username, role, isActive)
}

func (s *Service) UserSetPassword(ctx context.Context, id int, password string) error {
	return s.rt.Auth.SetUserPassword(ctx, id, password)
}

func (s *Service) UserSetActive(ctx context.Context, id int, active bool) (*UserDTO, error) {
	return s.rt.Auth.SetUserActive(ctx, id, active)
}

func (s *Service) UserDelete(ctx context.Context, currentUserID, targetUserID int) error {
	return s.rt.Auth.DeleteUser(ctx, currentUserID, targetUserID)
}

func (s *Service) UserGetLocale(ctx context.Context, userID int) (string, error) {
	if s.rt == nil || s.rt.DB == nil || s.rt.DB.Ent == nil {
		return "lv-LV", nil
	}
	u, err := s.rt.DB.Ent.User.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	return normalizeUILocale(u.UILocale), nil
}

func (s *Service) UserSetLocale(ctx context.Context, userID int, loc string) (string, error) {
	if err := validateUILocale(loc); err != nil {
		return "", err
	}
	if s.rt == nil || s.rt.DB == nil || s.rt.DB.Ent == nil {
		return "", errors.New("user store unavailable")
	}
	item, err := s.rt.DB.Ent.User.UpdateOneID(userID).SetUILocale(loc).Save(ctx)
	if err != nil {
		return "", err
	}
	return normalizeUILocale(item.UILocale), nil
}

func (s *Service) AttendanceListPerLesson(ctx context.Context, year, month int, courseID *int) ([]attendance.Row, error) {
	return s.rt.Attendance.ListPerLesson(ctx, year, month, courseID)
}

func (s *Service) AttendanceUpsert(ctx context.Context, studentID, courseID, year, month int, hours float64) error {
	return s.rt.Attendance.Upsert(ctx, studentID, courseID, year, month, hours)
}

func (s *Service) CourseMonthSubscriptionList(ctx context.Context, year, month int, courseID *int) ([]CourseMonthSubscriptionDTO, error) {
	items, err := s.rt.Attendance.ListCourseMonthSubscriptions(ctx, year, month, courseID)
	if err != nil {
		return nil, err
	}
	out := make([]CourseMonthSubscriptionDTO, 0, len(items))
	for _, item := range items {
		out = append(out, CourseMonthSubscriptionDTO{
			CourseID:    item.CourseID,
			Year:        item.Year,
			Month:       item.Month,
			LessonsHeld: item.LessonsHeld,
		})
	}
	return out, nil
}

func (s *Service) CourseMonthSubscriptionUpsert(ctx context.Context, courseID, year, month int, lessonsHeld float64) (*CourseMonthSubscriptionDTO, error) {
	item, err := s.rt.Attendance.UpsertCourseMonthSubscription(ctx, courseID, year, month, lessonsHeld)
	if err != nil {
		return nil, err
	}
	return &CourseMonthSubscriptionDTO{
		CourseID:    item.CourseID,
		Year:        item.Year,
		Month:       item.Month,
		LessonsHeld: item.LessonsHeld,
	}, nil
}

func (s *Service) AttendanceAddOne(ctx context.Context, year, month int, courseID *int) (int, error) {
	return s.rt.Attendance.AddOneForFilter(ctx, year, month, courseID)
}

func (s *Service) EnrollmentDelete(ctx context.Context, enrollmentID int) error {
	return s.rt.Attendance.DeleteEnrollment(ctx, enrollmentID)
}

func (s *Service) EnrollmentDeleteWithVersion(ctx context.Context, enrollmentID, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	return s.rt.Attendance.DeleteEnrollmentWithVersion(ctx, enrollmentID, version)
}

func (s *Service) InvoiceGenerateDrafts(ctx context.Context, year, month int) (invsvc.GenerateResult, error) {
	before, _ := s.auditSnapshotForPeriod(ctx, year, month)
	result, err := s.rt.Invoice.GenerateDrafts(ctx, year, month)
	if err != nil {
		return result, err
	}
	after, _ := s.auditSnapshotForPeriod(ctx, year, month)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice_batch",
		Action:     "invoice.generate_drafts",
		Summary: fmt.Sprintf(
			"Generated drafts for %04d-%02d: created %d, updated %d, skipped with invoices %d, skipped empty %d",
			year, month, result.Created, result.Updated, result.SkippedHasInvoice, result.SkippedNoLines,
		),
		Before: before,
		After:  after,
	})
	return result, nil
}

func (s *Service) InvoiceRebuildStudentDraft(ctx context.Context, studentID, year, month int) (invsvc.GenerateResult, error) {
	before, _ := s.auditSnapshotForStudentMonth(ctx, studentID, year, month)
	result, err := s.rt.Invoice.RebuildStudentDraft(ctx, studentID, year, month)
	if err != nil {
		return result, err
	}
	after, _ := s.auditSnapshotForStudentMonth(ctx, studentID, year, month)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice_batch",
		Action:     "invoice.rebuild_student_draft",
		Summary: fmt.Sprintf(
			"Rebuilt draft invoices for student %d in %04d-%02d: created %d, updated %d, skipped with invoices %d, skipped empty %d",
			studentID, year, month, result.Created, result.Updated, result.SkippedHasInvoice, result.SkippedNoLines,
		),
		Before:    before,
		After:     after,
		StudentID: intPtr(studentID),
	})
	return result, nil
}

func (s *Service) InvoiceGet(ctx context.Context, id int) (*InvoiceDTO, error) {
	return s.rt.Invoice.Get(ctx, id)
}

func (s *Service) InvoiceDeleteDraft(ctx context.Context, id int) error {
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.DeleteDraft(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.delete_draft",
		Summary:    fmt.Sprintf("Deleted draft invoice for %s, %04d-%02d", meta.StudentName, meta.Year, meta.Month),
		Before:     before,
		After: map[string]any{
			"deleted":   true,
			"invoiceId": id,
		},
		StudentID: meta.StudentID,
		InvoiceID: intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceDeleteDraftWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.DeleteDraftWithVersion(ctx, id, version); err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.delete_draft",
		Summary:    fmt.Sprintf("Deleted draft invoice for %s, %04d-%02d", meta.StudentName, meta.Year, meta.Month),
		Before:     before,
		After: map[string]any{
			"deleted":   true,
			"invoiceId": id,
		},
		StudentID: meta.StudentID,
		InvoiceID: intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceReopenDraft(ctx context.Context, id int) error {
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.ReopenDraft(ctx, id, s.rt.Dirs.Invoices); err != nil {
		return err
	}
	after, _, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.reopen_draft",
		Summary:    fmt.Sprintf("Returned invoice %s to draft", invoiceLabel(meta.Number, id)),
		Before:     before,
		After:      after,
		StudentID:  meta.StudentID,
		InvoiceID:  intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceReopenDraftWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	before, meta, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rt.Invoice.ReopenDraftWithVersion(ctx, id, version, s.rt.Dirs.Invoices); err != nil {
		return err
	}
	after, _, err := s.auditInvoiceSnapshot(ctx, id)
	if err != nil {
		return err
	}
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice",
		EntityID:   intPtr(id),
		Action:     "invoice.reopen_draft",
		Summary:    fmt.Sprintf("Returned invoice %s to draft", invoiceLabel(meta.Number, id)),
		Before:     before,
		After:      after,
		StudentID:  meta.StudentID,
		InvoiceID:  intPtr(id),
	})
	return nil
}

func (s *Service) InvoiceList(ctx context.Context, year, month int, status string) ([]invsvc.ListItem, error) {
	return s.rt.Invoice.List(ctx, year, month, status)
}

func (s *Service) InvoiceIssue(ctx context.Context, id int) (IssueResult, error) {
	before, meta, err := s.auditStudentFinanceSnapshotByInvoice(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	num, err := s.rt.Invoice.IssueOne(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	if err := s.rt.Payment.ApplyCreditToOldestInvoices(ctx, dto.StudentID); err != nil {
		return IssueResult{}, err
	}
	after, _, err := s.auditStudentFinanceSnapshot(ctx, dto.StudentID)
	if err == nil {
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "invoice",
			EntityID:   intPtr(id),
			Action:     "invoice.issue",
			Summary:    fmt.Sprintf("Issued invoice %s for %s, total %.2f", num, meta.StudentName, dto.Total),
			Before:     before,
			After:      after,
			StudentID:  intPtr(dto.StudentID),
			InvoiceID:  intPtr(id),
		})
	}
	return IssueResult{Number: num}, nil
}

func (s *Service) InvoiceIssueWithVersion(ctx context.Context, id, version int) (IssueResult, error) {
	if err := validateVersion(version); err != nil {
		return IssueResult{}, err
	}
	before, meta, err := s.auditStudentFinanceSnapshotByInvoice(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	num, studentID, err := s.rt.Invoice.IssueAndApplyCreditWithVersion(ctx, id, version)
	if err != nil {
		return IssueResult{}, err
	}
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return IssueResult{}, err
	}
	after, _, err := s.auditStudentFinanceSnapshot(ctx, studentID)
	if err == nil {
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "invoice",
			EntityID:   intPtr(id),
			Action:     "invoice.issue",
			Summary:    fmt.Sprintf("Issued invoice %s for %s, total %.2f", num, meta.StudentName, dto.Total),
			Before:     before,
			After:      after,
			StudentID:  intPtr(studentID),
			InvoiceID:  intPtr(id),
		})
	}
	return IssueResult{Number: num}, nil
}

func (s *Service) InvoiceIssueAll(ctx context.Context, year, month int) (IssueAllResult, error) {
	before, _ := s.auditSnapshotForPeriod(ctx, year, month)
	fonts, err := appruntime.ResolveFontsDir(s.rt.Config, s.rt.Dirs)
	if err != nil {
		return IssueAllResult{}, err
	}
	cnt, paths, err := s.rt.Invoice.IssueAll(ctx, year, month, s.rt.Dirs.Invoices, fonts)
	if err != nil {
		return IssueAllResult{}, err
	}
	items, err := s.rt.Invoice.List(ctx, year, month, sharedapp.InvoiceStatusIssued)
	if err != nil {
		return IssueAllResult{}, err
	}
	seen := make(map[int]struct{})
	for _, item := range items {
		if _, ok := seen[item.StudentID]; ok {
			continue
		}
		seen[item.StudentID] = struct{}{}
		if err := s.rt.Payment.ApplyCreditToOldestInvoices(ctx, item.StudentID); err != nil {
			return IssueAllResult{}, err
		}
	}
	after, _ := s.auditSnapshotForPeriod(ctx, year, month)
	s.recordAudit(ctx, auditsvc.RecordEvent{
		EntityType: "invoice_batch",
		Action:     "invoice.issue_all",
		Summary:    fmt.Sprintf("Issued %d invoices for %04d-%02d", cnt, year, month),
		Before:     before,
		After:      after,
	})
	return IssueAllResult{Count: cnt, PdfPaths: paths}, nil
}

func (s *Service) InvoiceEnsurePDF(ctx context.Context, id int) (string, error) {
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if dto.Number == nil || *dto.Number == "" {
		return "", fmt.Errorf("счёт ещё не выставлен")
	}
	paths := s.invoicePDFPaths(dto)
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	fonts, err := appruntime.ResolveFontsDir(s.rt.Config, s.rt.Dirs)
	if err != nil {
		return "", err
	}
	_, path, err := s.rt.Invoice.Issue(ctx, id, s.rt.Dirs.Invoices, fonts)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) InvoiceHasPDF(ctx context.Context, id int) (bool, error) {
	dto, err := s.rt.Invoice.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if dto.Number == nil || *dto.Number == "" {
		return false, nil
	}
	for _, path := range s.invoicePDFPaths(dto) {
		if _, err := os.Stat(path); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}
	return false, nil
}

func (s *Service) SettingsSetLocale(ctx context.Context, loc string) error {
	_, err := s.rt.DB.Ent.Settings.
		Update().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		SetLocale(loc).
		Save(ctx)
	return err
}

func (s *Service) SettingsGetLocale(ctx context.Context) (string, error) {
	if s.rt == nil || s.rt.DB == nil || s.rt.DB.Ent == nil {
		return "lv-LV", nil
	}
	st, err := s.rt.DB.Ent.Settings.
		Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	return st.Locale, nil
}

func (s *Service) PaymentCreate(ctx context.Context, studentID int, invoiceID *int, amount float64, method string, paidAt string, note string) (*PaymentDTO, error) {
	before, _, err := s.auditStudentFinanceSnapshot(ctx, studentID)
	if err != nil {
		return nil, err
	}
	item, err := s.rt.Payment.Create(ctx, studentID, invoiceID, amount, method, paidAt, note)
	if err != nil {
		return nil, err
	}
	after, studentMeta, err := s.auditStudentFinanceSnapshot(ctx, studentID)
	if err == nil {
		action := "payment.create"
		if invoiceID == nil {
			action = "payment.allocate_or_credit"
		}
		summary := fmt.Sprintf("Recorded payment of %.2f for %s", item.Amount, studentMeta.StudentName)
		if item.InvoiceID != nil {
			summary = fmt.Sprintf("%s on invoice %d", summary, *item.InvoiceID)
		} else {
			summary = fmt.Sprintf("%s as student credit", summary)
		}
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "payment",
			EntityID:   intPtr(item.ID),
			Action:     action,
			Summary:    summary,
			Before:     before,
			After:      after,
			StudentID:  intPtr(studentID),
			InvoiceID:  item.InvoiceID,
		})
	}
	return item, nil
}

func (s *Service) PaymentDelete(ctx context.Context, paymentID int) error {
	before, meta, err := s.auditPaymentDeleteSnapshot(ctx, paymentID)
	if err != nil {
		return err
	}
	if err := s.rt.Payment.Delete(ctx, paymentID); err != nil {
		return err
	}
	after, _, err := s.auditStudentFinanceSnapshot(ctx, meta.StudentID)
	if err == nil {
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "payment",
			EntityID:   intPtr(paymentID),
			Action:     "payment.delete",
			Summary:    fmt.Sprintf("Deleted payment %d for %s, amount %.2f", paymentID, meta.StudentName, meta.Amount),
			Before:     before,
			After:      after,
			StudentID:  intPtr(meta.StudentID),
			InvoiceID:  meta.InvoiceID,
		})
	}
	return nil
}

func (s *Service) PaymentListForStudent(ctx context.Context, studentID int) ([]PaymentDTO, error) {
	return s.rt.Payment.ListForStudent(ctx, studentID)
}

func (s *Service) StudentBalance(ctx context.Context, studentID int) (*BalanceDTO, error) {
	return s.rt.Payment.StudentBalance(ctx, studentID)
}

func (s *Service) DebtorsList(ctx context.Context) ([]DebtorDTO, error) {
	return s.rt.Payment.ListDebtors(ctx)
}

func (s *Service) MonthOverview(ctx context.Context, year, month int) (*MonthOverviewDTO, error) {
	return s.rt.Payment.MonthOverview(ctx, year, month)
}

func (s *Service) RecentPayments(ctx context.Context, limit int) ([]RecentPaymentDTO, error) {
	return s.rt.Payment.ListRecent(ctx, limit)
}

func (s *Service) StudentDebtDetails(ctx context.Context, studentID int) ([]DebtInvoiceDTO, error) {
	return s.rt.Payment.StudentDebtDetails(ctx, studentID)
}

func (s *Service) InvoicePaymentSummary(ctx context.Context, invoiceID int) (*InvoiceSummaryDTO, error) {
	return s.rt.Payment.InvoiceSummary(ctx, invoiceID)
}

func (s *Service) PaymentQuickCash(ctx context.Context, studentID int, amount float64, note string) (*PaymentDTO, error) {
	return s.rt.Payment.QuickCash(ctx, studentID, amount, note)
}

func (s *Service) StudentList(ctx context.Context, q string, includeInactive bool) ([]StudentDTO, error) {
	q = strings.TrimSpace(q)
	query := s.rt.DB.Ent.Student.Query()
	if !includeInactive {
		query = query.Where(student.IsActiveEQ(true))
	}
	if q != "" {
		query = query.Where(student.Or(
			student.FullNameContainsFold(q),
			student.PhoneContainsFold(q),
			student.EmailContainsFold(q),
		))
	}
	studs, err := query.Order(ent.Asc(student.FieldFullName)).All(ctx)
	if err != nil {
		return nil, err
	}
	summaries, err := s.studentBalanceSummaries(ctx, studs)
	if err != nil {
		return nil, err
	}
	out := make([]StudentDTO, 0, len(studs))
	for _, item := range studs {
		out = append(out, toStudentDTO(item, summaries[item.ID]))
	}
	return out, nil
}

func (s *Service) StudentGet(ctx context.Context, id int) (*StudentDTO, error) {
	item, err := s.rt.DB.Ent.Student.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := toStudentDTO(item, s.studentBalanceSummary(ctx, item.ID))
	return &dto, nil
}

type studentBalanceSummary struct {
	Balance float64
	Debt    float64
}

type studentMoneyAggregate struct {
	StudentID int   `json:"student_id"`
	Total     int64 `json:"total"`
}

func (s *Service) studentBalanceSummaries(ctx context.Context, studs []*ent.Student) (map[int]studentBalanceSummary, error) {
	summaries := make(map[int]studentBalanceSummary, len(studs))
	if len(studs) == 0 {
		return summaries, nil
	}
	studentIDs := make([]int, 0, len(studs))
	for _, st := range studs {
		studentIDs = append(studentIDs, st.ID)
	}
	invoicedByStudent, err := s.aggregateInvoiceTotalsByStudent(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	paidByStudent, err := s.aggregatePaymentTotalsByStudent(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	for _, st := range studs {
		summaries[st.ID] = makeStudentBalanceSummary(invoicedByStudent[st.ID], paidByStudent[st.ID])
	}
	return summaries, nil
}

func (s *Service) studentBalanceSummary(ctx context.Context, studentID int) studentBalanceSummary {
	invoicedByStudent, err := s.aggregateInvoiceTotalsByStudent(ctx, []int{studentID})
	if err != nil {
		return studentBalanceSummary{}
	}
	paidByStudent, err := s.aggregatePaymentTotalsByStudent(ctx, []int{studentID})
	if err != nil {
		return studentBalanceSummary{}
	}
	return makeStudentBalanceSummary(invoicedByStudent[studentID], paidByStudent[studentID])
}

func makeStudentBalanceSummary(invoicedCents, paidCents int64) studentBalanceSummary {
	balanceCents := paidCents - invoicedCents
	summary := studentBalanceSummary{
		Balance: money.CentsToEuros(balanceCents),
	}
	if balanceCents < 0 {
		summary.Debt = money.CentsToEuros(-balanceCents)
	}
	return summary
}

func (s *Service) aggregateInvoiceTotalsByStudent(ctx context.Context, studentIDs []int) (map[int]int64, error) {
	rows := []studentMoneyAggregate{}
	err := s.rt.DB.Ent.Invoice.Query().
		Where(
			invoice.StudentIDIn(studentIDs...),
			invoice.StatusIn(invoice.StatusIssued, invoice.StatusPaid),
		).
		GroupBy(invoice.FieldStudentID).
		Aggregate(ent.As(ent.Sum(invoice.FieldTotalAmountCents), "total")).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, row := range rows {
		out[row.StudentID] = row.Total
	}
	return out, nil
}

func (s *Service) aggregatePaymentTotalsByStudent(ctx context.Context, studentIDs []int) (map[int]int64, error) {
	rows := []studentMoneyAggregate{}
	err := s.rt.DB.Ent.Payment.Query().
		Where(payment.StudentIDIn(studentIDs...)).
		GroupBy(payment.FieldStudentID).
		Aggregate(ent.As(ent.Sum(payment.FieldAmountCents), "total")).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, row := range rows {
		out[row.StudentID] = row.Total
	}
	return out, nil
}

func (s *Service) StudentCreate(ctx context.Context, fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	fullName = sanitizeInput(fullName)
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}
	personalCode = sanitizeInput(personalCode)
	payerName = sanitizeInput(payerName)
	payerRole = normalizePayerRole(payerRole)
	if err := validateMinorPayer(isMinor, payerName, payerRole); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Student.Create().
		SetFullName(fullName).
		SetPersonalCode(personalCode).
		SetPhone(sanitizeInput(phone)).
		SetEmail(sanitizeInput(email)).
		SetNote(sanitizeInput(note)).
		SetIsMinor(isMinor).
		SetPayerName(payerName).
		SetPayerRole(payerRole).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	dto := toStudentDTO(item, studentBalanceSummary{})
	return &dto, nil
}

func (s *Service) StudentUpdate(ctx context.Context, id int, fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	fullName = sanitizeInput(fullName)
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}
	personalCode = sanitizeInput(personalCode)
	payerName = sanitizeInput(payerName)
	payerRole = normalizePayerRole(payerRole)
	if err := validateMinorPayer(isMinor, payerName, payerRole); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Student.UpdateOneID(id).
		AddVersion(1).
		SetFullName(fullName).
		SetPersonalCode(personalCode).
		SetPhone(sanitizeInput(phone)).
		SetEmail(sanitizeInput(email)).
		SetNote(sanitizeInput(note)).
		SetIsMinor(isMinor).
		SetPayerName(payerName).
		SetPayerRole(payerRole).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	dto := toStudentDTO(item, s.studentBalanceSummary(ctx, item.ID))
	return &dto, nil
}

func (s *Service) StudentUpdateWithVersion(ctx context.Context, id, version int, fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	fullName = sanitizeInput(fullName)
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}
	personalCode = sanitizeInput(personalCode)
	payerName = sanitizeInput(payerName)
	payerRole = normalizePayerRole(payerRole)
	if err := validateMinorPayer(isMinor, payerName, payerRole); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Student.UpdateOneID(id).
		Where(student.VersionEQ(version)).
		SetVersion(version + 1).
		SetFullName(fullName).
		SetPersonalCode(personalCode).
		SetPhone(sanitizeInput(phone)).
		SetEmail(sanitizeInput(email)).
		SetNote(sanitizeInput(note)).
		SetIsMinor(isMinor).
		SetPayerName(payerName).
		SetPayerRole(payerRole).
		Save(ctx)
	if err != nil {
		return nil, staleOnNotFound(err)
	}
	dto := toStudentDTO(item, s.studentBalanceSummary(ctx, item.ID))
	return &dto, nil
}

func (s *Service) StudentSetActive(ctx context.Context, id int, active bool) error {
	_, err := s.rt.DB.Ent.Student.UpdateOneID(id).AddVersion(1).SetIsActive(active).Save(ctx)
	return err
}

func (s *Service) StudentSetActiveWithVersion(ctx context.Context, id, version int, active bool) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	_, err := s.rt.DB.Ent.Student.UpdateOneID(id).
		Where(student.VersionEQ(version)).
		SetVersion(version + 1).
		SetIsActive(active).
		Save(ctx)
	return staleOnNotFound(err)
}

func (s *Service) StudentDelete(ctx context.Context, id int) error {
	st, err := s.rt.DB.Ent.Student.Get(ctx, id)
	if err != nil {
		return err
	}
	if st.IsActive {
		return errors.New("cannot delete active student; deactivate first")
	}
	hasPayments, err := s.rt.DB.Ent.Payment.Query().Where(payment.StudentIDEQ(id)).Exist(ctx)
	if err != nil {
		return err
	}
	if hasPayments {
		return errors.New("cannot delete student: has payments (financial records)")
	}
	hasProtectedInvoices, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusNEQ(sharedapp.InvoiceStatusDraft)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasProtectedInvoices {
		return errors.New("cannot delete student: has issued, paid, or canceled invoices")
	}

	tx, err := s.rt.DB.Ent.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	draftInvoiceIDs, err := tx.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusEQ(sharedapp.InvoiceStatusDraft)).
		IDs(ctx)
	if err != nil {
		return err
	}
	if len(draftInvoiceIDs) > 0 {
		if _, err := tx.InvoiceLine.Delete().Where(invoiceline.InvoiceIDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.Invoice.Delete().Where(invoice.IDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
	}
	if _, err := tx.AttendanceMonth.Delete().Where(attendancemonth.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.Enrollment.Delete().Where(enrollment.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if err := tx.Student.DeleteOneID(id).Exec(ctx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) StudentDeleteWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	st, err := s.rt.DB.Ent.Student.Get(ctx, id)
	if err != nil {
		return err
	}
	if st.Version != version {
		return apperrors.StaleRevision()
	}
	if st.IsActive {
		return errors.New("cannot delete active student; deactivate first")
	}
	hasPayments, err := s.rt.DB.Ent.Payment.Query().Where(payment.StudentIDEQ(id)).Exist(ctx)
	if err != nil {
		return err
	}
	if hasPayments {
		return errors.New("cannot delete student: has payments (financial records)")
	}
	hasProtectedInvoices, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusNEQ(sharedapp.InvoiceStatusDraft)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasProtectedInvoices {
		return errors.New("cannot delete student: has issued, paid, or canceled invoices")
	}

	tx, err := s.rt.DB.Ent.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	draftInvoiceIDs, err := tx.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusEQ(sharedapp.InvoiceStatusDraft)).
		IDs(ctx)
	if err != nil {
		return err
	}
	if len(draftInvoiceIDs) > 0 {
		if _, err := tx.InvoiceLine.Delete().Where(invoiceline.InvoiceIDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.Invoice.Delete().Where(invoice.IDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
	}
	if _, err := tx.AttendanceMonth.Delete().Where(attendancemonth.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.Enrollment.Delete().Where(enrollment.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if err := tx.Student.DeleteOneID(id).Where(student.VersionEQ(version)).Exec(ctx); err != nil {
		return staleOnNotFound(err)
	}
	return tx.Commit()
}

func (s *Service) TeacherList(ctx context.Context, q string) ([]TeacherDTO, error) {
	q = strings.TrimSpace(q)
	query := s.rt.DB.Ent.Teacher.Query().Where(teacher.IsActiveEQ(true))
	if q != "" {
		query = query.Where(teacher.FullNameContainsFold(q))
	}
	items, err := query.Order(ent.Asc(teacher.FieldFullName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TeacherDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toTeacherDTO(item))
	}
	return out, nil
}

func (s *Service) TeacherCreate(ctx context.Context, fullName string) (*TeacherDTO, error) {
	fullName = sanitizeInput(fullName)
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}
	existing, err := s.rt.DB.Ent.Teacher.Query().Where(teacher.FullNameEqualFold(fullName)).Only(ctx)
	if err == nil {
		dto := toTeacherDTO(existing)
		return &dto, nil
	}
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Teacher.Create().SetFullName(fullName).SetIsActive(true).Save(ctx)
	if err != nil {
		return nil, err
	}
	dto := toTeacherDTO(item)
	return &dto, nil
}

func (s *Service) CourseList(ctx context.Context, q string) ([]CourseDTO, error) {
	q = strings.TrimSpace(q)
	query := s.rt.DB.Ent.Course.Query().WithTeacher()
	if q != "" {
		query = query.Where(course.Or(
			course.NameContainsFold(q),
			course.TeacherNameContainsFold(q),
			course.HasTeacherWith(teacher.FullNameContainsFold(q)),
		))
	}
	items, err := query.Order(ent.Asc(course.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CourseDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toCourseDTO(item))
	}
	return out, nil
}

func (s *Service) CourseGet(ctx context.Context, id int) (*CourseDTO, error) {
	item, err := s.rt.DB.Ent.Course.Query().Where(course.IDEQ(id)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseCreate(ctx context.Context, name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = sanitizeInput(name)
	courseType = strings.TrimSpace(courseType)
	if err := validateNonEmpty(name, "name"); err != nil {
		return nil, err
	}
	if err := validateCourseType(courseType); err != nil {
		return nil, err
	}
	if err := validatePrices(lessonPrice, subscriptionPrice); err != nil {
		return nil, err
	}
	selectedTeacher, err := s.resolveTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	create := s.rt.DB.Ent.Course.Create().
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPriceCents(money.EurosToCents(lessonPrice)).
		SetSubscriptionPriceCents(money.EurosToCents(subscriptionPrice))
	if selectedTeacher != nil {
		create = create.SetTeacherID(selectedTeacher.ID).SetTeacherName(selectedTeacher.FullName)
	} else {
		create = create.SetTeacherName("")
	}
	item, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	item, err = s.rt.DB.Ent.Course.Query().Where(course.IDEQ(item.ID)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseUpdate(ctx context.Context, id int, name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = sanitizeInput(name)
	courseType = strings.TrimSpace(courseType)
	if err := validateNonEmpty(name, "name"); err != nil {
		return nil, err
	}
	if err := validateCourseType(courseType); err != nil {
		return nil, err
	}
	if err := validatePrices(lessonPrice, subscriptionPrice); err != nil {
		return nil, err
	}
	selectedTeacher, err := s.resolveTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	update := s.rt.DB.Ent.Course.UpdateOneID(id).
		AddVersion(1).
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPriceCents(money.EurosToCents(lessonPrice)).
		SetSubscriptionPriceCents(money.EurosToCents(subscriptionPrice))
	if selectedTeacher != nil {
		update = update.SetTeacherID(selectedTeacher.ID).SetTeacherName(selectedTeacher.FullName)
	} else {
		update = update.ClearTeacher().SetTeacherName("")
	}
	item, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	item, err = s.rt.DB.Ent.Course.Query().Where(course.IDEQ(item.ID)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseUpdateWithVersion(ctx context.Context, id, version int, name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = sanitizeInput(name)
	courseType = strings.TrimSpace(courseType)
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := validateNonEmpty(name, "name"); err != nil {
		return nil, err
	}
	if err := validateCourseType(courseType); err != nil {
		return nil, err
	}
	if err := validatePrices(lessonPrice, subscriptionPrice); err != nil {
		return nil, err
	}
	selectedTeacher, err := s.resolveTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	update := s.rt.DB.Ent.Course.UpdateOneID(id).
		Where(course.VersionEQ(version)).
		SetVersion(version + 1).
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPriceCents(money.EurosToCents(lessonPrice)).
		SetSubscriptionPriceCents(money.EurosToCents(subscriptionPrice))
	if selectedTeacher != nil {
		update = update.SetTeacherID(selectedTeacher.ID).SetTeacherName(selectedTeacher.FullName)
	} else {
		update = update.ClearTeacher().SetTeacherName("")
	}
	item, err := update.Save(ctx)
	if err != nil {
		return nil, staleOnNotFound(err)
	}
	item, err = s.rt.DB.Ent.Course.Query().Where(course.IDEQ(item.ID)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseDelete(ctx context.Context, id int) error {
	enrollmentCount, err := s.rt.DB.Ent.Enrollment.Query().Where(enrollment.CourseIDEQ(id)).Count(ctx)
	if err != nil {
		return err
	}
	if enrollmentCount > 0 {
		return errors.New("cannot delete course: it has enrollments; remove enrollments first or keep course")
	}
	err = s.rt.DB.Ent.Course.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return errors.New("cannot delete course: it is still referenced by existing records")
		}
		return err
	}
	return nil
}

func (s *Service) CourseDeleteWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	enrollmentCount, err := s.rt.DB.Ent.Enrollment.Query().Where(enrollment.CourseIDEQ(id)).Count(ctx)
	if err != nil {
		return err
	}
	if enrollmentCount > 0 {
		return errors.New("cannot delete course: it has enrollments; remove enrollments first or keep course")
	}
	err = s.rt.DB.Ent.Course.DeleteOneID(id).Where(course.VersionEQ(version)).Exec(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return errors.New("cannot delete course: it is still referenced by existing records")
		}
		return staleOnNotFound(err)
	}
	return nil
}

func (s *Service) EnrollmentList(ctx context.Context, studentID *int, courseID *int) ([]EnrollmentDTO, error) {
	q := s.rt.DB.Ent.Enrollment.Query().
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) {
			cq.WithTeacher()
		})
	if studentID != nil {
		q = q.Where(enrollment.StudentIDEQ(*studentID))
	}
	if courseID != nil {
		q = q.Where(enrollment.CourseIDEQ(*courseID))
	}
	items, err := q.Order(ent.Asc(enrollment.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]EnrollmentDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toEnrollmentDTO(item))
	}
	return out, nil
}

func (s *Service) EnrollmentCreate(ctx context.Context, studentID, courseID int, billingMode string, discountPct, subscriptionDiscountPct float64, note string) (*EnrollmentDTO, error) {
	if studentID <= 0 || courseID <= 0 {
		return nil, errors.New("studentID and courseID must be > 0")
	}
	billingMode = strings.TrimSpace(billingMode)
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(discountPct); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(subscriptionDiscountPct); err != nil {
		return nil, err
	}
	st, err := s.rt.DB.Ent.Student.Get(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if !st.IsActive {
		return nil, errors.New("cannot enroll a deactivated student")
	}
	if _, err := s.rt.DB.Ent.Course.Get(ctx, courseID); err != nil {
		return nil, err
	}
	exists, err := s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.StudentIDEQ(studentID), enrollment.CourseIDEQ(courseID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("an enrollment for this student and course already exists")
	}
	item, err := s.rt.DB.Ent.Enrollment.Create().
		SetStudentID(studentID).
		SetCourseID(courseID).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetDiscountPct(discountPct).
		SetSubscriptionDiscountPct(subscriptionDiscountPct).
		SetNote(sanitizeInput(note)).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	item, err = s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(item.ID)).
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) { cq.WithTeacher() }).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toEnrollmentDTO(item)
	return &dto, nil
}

func (s *Service) EnrollmentUpdate(ctx context.Context, enrollmentID int, billingMode string, discountPct, subscriptionDiscountPct float64, note string) (*EnrollmentDTO, error) {
	billingMode = strings.TrimSpace(billingMode)
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(discountPct); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(subscriptionDiscountPct); err != nil {
		return nil, err
	}
	if _, err := s.rt.DB.Ent.Enrollment.UpdateOneID(enrollmentID).
		AddVersion(1).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetDiscountPct(discountPct).
		SetSubscriptionDiscountPct(subscriptionDiscountPct).
		SetNote(sanitizeInput(note)).
		Save(ctx); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(enrollmentID)).
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) { cq.WithTeacher() }).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toEnrollmentDTO(item)
	return &dto, nil
}

func (s *Service) EnrollmentUpdateWithVersion(ctx context.Context, enrollmentID, version int, billingMode string, discountPct, subscriptionDiscountPct float64, note string) (*EnrollmentDTO, error) {
	billingMode = strings.TrimSpace(billingMode)
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(discountPct); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(subscriptionDiscountPct); err != nil {
		return nil, err
	}
	if _, err := s.rt.DB.Ent.Enrollment.UpdateOneID(enrollmentID).
		Where(enrollment.VersionEQ(version)).
		SetVersion(version + 1).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetDiscountPct(discountPct).
		SetSubscriptionDiscountPct(subscriptionDiscountPct).
		SetNote(sanitizeInput(note)).
		Save(ctx); err != nil {
		return nil, staleOnNotFound(err)
	}
	item, err := s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(enrollmentID)).
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) { cq.WithTeacher() }).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toEnrollmentDTO(item)
	return &dto, nil
}

func (s *Service) resolveTeacher(ctx context.Context, teacherID *int) (*ent.Teacher, error) {
	if teacherID == nil {
		return nil, nil
	}
	if *teacherID <= 0 {
		return nil, errors.New("teacherID must be > 0 when provided")
	}
	return s.rt.DB.Ent.Teacher.Get(ctx, *teacherID)
}

func (s *Service) invoicePDFPaths(dto *invsvc.InvoiceDTO) []string {
	subjectName := dto.StudentName
	if dto.IsMinor && strings.TrimSpace(dto.ChildName) != "" {
		subjectName = dto.ChildName
	}
	return []string{
		invsvc.PDFPathByNumberAndName(s.rt.Dirs.Invoices, dto.Year, dto.Month, *dto.Number, subjectName),
		invsvc.PDFPathByNumber(s.rt.Dirs.Invoices, dto.Year, dto.Month, *dto.Number),
	}
}

func toStudentDTO(s *ent.Student, summary studentBalanceSummary) StudentDTO {
	return StudentDTO{
		ID:           s.ID,
		Version:      s.Version,
		FullName:     s.FullName,
		PersonalCode: s.PersonalCode,
		Phone:        s.Phone,
		Email:        s.Email,
		Note:         s.Note,
		IsMinor:      s.IsMinor,
		PayerName:    s.PayerName,
		PayerRole:    s.PayerRole,
		IsActive:     s.IsActive,
		Balance:      summary.Balance,
		Debt:         summary.Debt,
	}
}

func toCourseDTO(c *ent.Course) CourseDTO {
	dto := CourseDTO{
		ID:                c.ID,
		Version:           c.Version,
		Name:              c.Name,
		TeacherName:       c.TeacherName,
		Type:              string(c.Type),
		LessonPrice:       money.CentsToEuros(c.LessonPriceCents),
		SubscriptionPrice: money.CentsToEuros(c.SubscriptionPriceCents),
	}
	if c.TeacherID != nil {
		id := *c.TeacherID
		dto.TeacherID = &id
	}
	if c.Edges.Teacher != nil {
		dto.TeacherName = c.Edges.Teacher.FullName
		id := c.Edges.Teacher.ID
		dto.TeacherID = &id
	}
	return dto
}

func toEnrollmentDTO(e *ent.Enrollment) EnrollmentDTO {
	dto := EnrollmentDTO{
		ID:                      e.ID,
		Version:                 e.Version,
		StudentID:               e.StudentID,
		CourseID:                e.CourseID,
		BillingMode:             string(e.BillingMode),
		DiscountPct:             e.DiscountPct,
		SubscriptionDiscountPct: e.SubscriptionDiscountPct,
		Note:                    e.Note,
		CreatedAt:               e.CreatedAt.Format(time.RFC3339),
	}
	if e.Edges.Student != nil {
		dto.StudentName = e.Edges.Student.FullName
	}
	if e.Edges.Course != nil {
		dto.CourseName = e.Edges.Course.Name
		dto.CourseType = string(e.Edges.Course.Type)
		dto.TeacherName = e.Edges.Course.TeacherName
		if e.Edges.Course.TeacherID != nil {
			id := *e.Edges.Course.TeacherID
			dto.TeacherID = &id
		}
		if e.Edges.Course.Edges.Teacher != nil {
			dto.TeacherName = e.Edges.Course.Edges.Teacher.FullName
			id := e.Edges.Course.Edges.Teacher.ID
			dto.TeacherID = &id
		}
	}
	return dto
}

func toTeacherDTO(t *ent.Teacher) TeacherDTO {
	return TeacherDTO{
		ID:       t.ID,
		FullName: t.FullName,
		IsActive: t.IsActive,
	}
}

func validateVersion(version int) error {
	if version <= 0 {
		return errors.New("version must be > 0")
	}
	return nil
}

func staleOnNotFound(err error) error {
	if ent.IsNotFound(err) {
		return apperrors.StaleRevision()
	}
	return err
}

func validateNonEmpty(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

func validatePrices(lessonPrice, subscriptionPrice float64) error {
	if lessonPrice < 0 || subscriptionPrice < 0 {
		return errors.New("prices must be >= 0")
	}
	return nil
}

func validateCourseType(courseType string) error {
	if courseType != CourseTypeGroup && courseType != CourseTypeIndividual {
		return fmt.Errorf("courseType must be '%s' or '%s'", CourseTypeGroup, CourseTypeIndividual)
	}
	return nil
}

func validateBillingMode(billingMode string) error {
	if billingMode != BillingModeSubscription && billingMode != BillingModePerLesson {
		return fmt.Errorf("billingMode must be '%s' or '%s'", BillingModeSubscription, BillingModePerLesson)
	}
	return nil
}

func validateDiscountPct(discountPct float64) error {
	if discountPct < 0 || discountPct > 100 {
		return errors.New("discountPct must be between 0 and 100")
	}
	return nil
}

func validatePayerRole(role string) error {
	role = strings.TrimSpace(role)
	if role == "" {
		return nil
	}
	switch role {
	case "mother", "father", "grandmother", "grandfather", "guardian", "other":
		return nil
	default:
		return errors.New("payerRole must be one of: mother, father, grandmother, grandfather, guardian, other")
	}
}

func validateMinorPayer(isMinor bool, payerName, payerRole string) error {
	if err := validatePayerRole(payerRole); err != nil {
		return err
	}
	if !isMinor {
		return nil
	}
	if strings.TrimSpace(payerName) == "" {
		return errors.New("payerName is required when isMinor is true")
	}
	if strings.TrimSpace(payerRole) == "" {
		return errors.New("payerRole is required when isMinor is true")
	}
	return nil
}

func validateUILocale(locale string) error {
	switch strings.TrimSpace(locale) {
	case "lv-LV", "ru-RU", "en-US":
		return nil
	default:
		return errors.New("locale must be one of: lv-LV, ru-RU, en-US")
	}
}

func normalizeUILocale(locale string) string {
	switch strings.TrimSpace(locale) {
	case "en-US":
		return "en-US"
	case "ru-RU":
		return "ru-RU"
	case "lv-LV":
		return "lv-LV"
	default:
		return "lv-LV"
	}
}

func normalizePayerRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func capabilitiesForCurrentUser(currentUser *auth.UserInfo, isDesktop bool) map[string]bool {
	if currentUser == nil {
		return capabilitiesForRole("", isDesktop)
	}
	return capabilitiesForRole(currentUser.Role, isDesktop)
}

func capabilitiesForRole(role string, isDesktop bool) map[string]bool {
	isAdmin := role == auth.RoleAdmin || isDesktop
	return map[string]bool{
		"backups":        isAdmin,
		"pdfDownload":    true,
		"pdfGenerate":    true,
		"desktopPaths":   isDesktop,
		"manageUsers":    isAdmin,
		"manageSettings": isAdmin,
		"viewAuditLog":   isAdmin,
		"deletePayments": isAdmin,
		"deleteStudents": isAdmin,
		"deleteCourses":  isAdmin,
	}
}

func sanitizeInput(input string) string {
	return html.EscapeString(strings.TrimSpace(input))
}

func (s *Service) PDFPathFilename(path string) string {
	return filepath.Base(path)
}

func (s *Service) Logf(format string, args ...any) {
	log.Printf(format, args...)
}
