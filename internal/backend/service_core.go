package backend

import (
	"errors"
	"fmt"
	"html"
	"log"
	"math"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"langschool/ent"
	sharedapp "langschool/internal/app"
	"langschool/internal/app/attendance"
	invsvc "langschool/internal/app/invoice"
	paysvc "langschool/internal/app/payment"
	"langschool/internal/apperrors"
	"langschool/internal/auth"
	"langschool/internal/email"
	appruntime "langschool/internal/runtime"
)

const (
	CourseTypeGroup         = sharedapp.CourseTypeGroup
	CourseTypeIndividual    = sharedapp.CourseTypeIndividual
	BillingModeSubscription = sharedapp.BillingModeSubscription
	BillingModePerLesson    = sharedapp.BillingModePerLesson

	invoiceArchivePDFStatusReady    = invsvc.PDFStatusReady
	invoiceArchivePDFStatusMissing  = invsvc.PDFStatusMissing
	invoiceArchivePDFStatusOutdated = invsvc.PDFStatusOutdated
	invoiceArchivePDFStatusError    = invsvc.PDFStatusError
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

type StudentDuplicateCheckResult struct {
	ExactMatch      *StudentDTO  `json:"exactMatch,omitempty"`
	PossibleMatches []StudentDTO `json:"possibleMatches"`
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
	ChargeMaterials         bool    `json:"chargeMaterials"`
	LessonPriceOverride     float64 `json:"lessonPriceOverride"`
	SubscriptionLessonPrice float64 `json:"subscriptionLessonPrice"`
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
	Number    string `json:"number"`
	PDFReady  bool   `json:"pdfReady"`
	PDFStatus string `json:"pdfStatus"`
}

type IssueAllResult struct {
	Count          int      `json:"count"`
	PdfPaths       []string `json:"pdfPaths"`
	GeneratedCount int      `json:"generatedCount"`
	PendingCount   int      `json:"pendingCount"`
}

type EnsureAllPDFsItemResult struct {
	InvoiceID   int    `json:"invoiceId"`
	Number      string `json:"number"`
	StudentName string `json:"studentName"`
	Status      string `json:"status"`
	Result      string `json:"result"`
	Message     string `json:"message,omitempty"`
}

type EnsureAllPDFsResult struct {
	Year              int                       `json:"year"`
	Month             int                       `json:"month"`
	Processed         int                       `json:"processed"`
	GeneratedCount    int                       `json:"generatedCount"`
	AlreadyReadyCount int                       `json:"alreadyReadyCount"`
	FailedCount       int                       `json:"failedCount"`
	Items             []EnsureAllPDFsItemResult `json:"items"`
}

type InvoiceEmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
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

type InvoiceEmailSettingsDTO struct {
	SubjectTemplate       string   `json:"subjectTemplate"`
	BodyTemplate          string   `json:"bodyTemplate"`
	ReplyTo               string   `json:"replyTo"`
	AvailablePlaceholders []string `json:"availablePlaceholders"`
}

type InvoiceArchiveInvoiceDTO struct {
	InvoiceID     int     `json:"invoiceId"`
	Year          int     `json:"year"`
	Month         int     `json:"month"`
	Number        string  `json:"number"`
	StudentName   string  `json:"studentName"`
	RecipientName string  `json:"recipientName"`
	Total         float64 `json:"total"`
	Status        string  `json:"status"`
	PDFStatus     string  `json:"pdfStatus"`
	PDFFilename   string  `json:"pdfFilename,omitempty"`
	PDFUpdatedAt  string  `json:"pdfUpdatedAt,omitempty"`
	OpenURL       string  `json:"openUrl,omitempty"`
	DownloadURL   string  `json:"downloadUrl,omitempty"`
}

type InvoiceArchiveMonthDTO struct {
	Month             int                        `json:"month"`
	Count             int                        `json:"count"`
	ReadyPDFCount     int                        `json:"readyPdfCount"`
	MissingPDFCount   int                        `json:"missingPdfCount"`
	ZipDownloadURL    string                     `json:"zipDownloadUrl,omitempty"`
	ExpandedByDefault bool                       `json:"expandedByDefault"`
	Invoices          []InvoiceArchiveInvoiceDTO `json:"invoices"`
}

type InvoiceArchiveYearDTO struct {
	Year              int                      `json:"year"`
	Count             int                      `json:"count"`
	ExpandedByDefault bool                     `json:"expandedByDefault"`
	Months            []InvoiceArchiveMonthDTO `json:"months"`
}

type InvoiceArchiveResult struct {
	Years []InvoiceArchiveYearDTO `json:"years"`
}

type Service struct {
	rt          *appruntime.Runtime
	emailSender email.Sender
}

func New(rt *appruntime.Runtime) *Service {
	return NewWithEmailSender(rt, email.NewService(email.Config{
		Host:      rt.Config.SMTPHost,
		Port:      rt.Config.SMTPPort,
		Username:  rt.Config.SMTPUsername,
		Password:  rt.Config.SMTPPassword,
		FromEmail: rt.Config.SMTPFromEmail,
		FromName:  rt.Config.SMTPFromName,
	}))
}

func NewWithEmailSender(rt *appruntime.Runtime, sender email.Sender) *Service {
	return &Service{
		rt:          rt,
		emailSender: sender,
	}
}

func (s *Service) Ready() bool {
	return s != nil && s.rt != nil && s.rt.DB != nil && s.rt.DB.Ent != nil && s.rt.Attendance != nil && s.rt.Invoice != nil && s.rt.Payment != nil
}

type studentBalanceSummary struct {
	Balance float64
	Debt    float64
}

type studentMoneyAggregate struct {
	StudentID int   `json:"student_id"`
	Total     int64 `json:"total"`
}

func issuePDFStatus(pdfReady bool) string {
	if pdfReady {
		return "ready"
	}
	return "pending"
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatOptionalTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
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

func validateLessonPriceOverride(lessonPriceOverride float64) error {
	if math.IsNaN(lessonPriceOverride) || math.IsInf(lessonPriceOverride, 0) || lessonPriceOverride < 0 {
		return errors.New("lessonPriceOverride must be >= 0")
	}
	return nil
}

func validateSubscriptionLessonPrice(subscriptionLessonPrice float64) error {
	if math.IsNaN(subscriptionLessonPrice) || math.IsInf(subscriptionLessonPrice, 0) || subscriptionLessonPrice < 0 {
		return errors.New("subscriptionLessonPrice must be >= 0")
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

func capabilitiesForCurrentUser(currentUser *auth.UserInfo) map[string]bool {
	if currentUser == nil {
		return capabilitiesForRole("")
	}
	return capabilitiesForRole(currentUser.Role)
}

func capabilitiesForRole(role string) map[string]bool {
	isAdmin := role == auth.RoleAdmin
	return map[string]bool{
		"backups":        isAdmin,
		"emailSend":      true,
		"invoiceArchive": true,
		"pdfDownload":    true,
		"pdfGenerate":    true,
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

func parseArchiveYear(name string) (int, bool) {
	if len(name) != 4 {
		return 0, false
	}
	year := 0
	for _, ch := range name {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		year = year*10 + int(ch-'0')
	}
	return year, year > 0
}

func parseArchiveMonth(name string) (int, bool) {
	if len(name) != 2 {
		return 0, false
	}
	month := 0
	for _, ch := range name {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		month = month*10 + int(ch-'0')
	}
	return month, month >= 1 && month <= 12
}

func isArchivePDFName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return strings.EqualFold(filepath.Ext(name), ".pdf")
}

func invoiceArchiveFileURL(year, month int, filename, mode string) string {
	return fmt.Sprintf("/api/invoice-archive/%04d/%02d/%s/%s", year, month, url.PathEscape(filename), mode)
}

func invoiceArchiveMonthZipURL(year, month int) string {
	return fmt.Sprintf("/api/invoice-archive/%04d/%02d/zip", year, month)
}
