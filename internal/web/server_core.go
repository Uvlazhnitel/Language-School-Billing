package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"langschool/ent"
	"langschool/internal/apperrors"
	"langschool/internal/auth"
	"langschool/internal/backend"
)

type HandlerOptions struct {
	DistDir string
}

type Server struct {
	svc      *backend.Service
	mux      *http.ServeMux
	distDir  string
	hasIndex bool
}

func NewHandler(svc *backend.Service, opts HandlerOptions) http.Handler {
	server := &Server{
		svc:     svc,
		mux:     http.NewServeMux(),
		distDir: normalizeDistDir(opts.DistDir),
	}
	server.hasIndex = fileExists(filepath.Join(server.distDir, "index.html"))
	server.routes()
	return http.HandlerFunc(server.serve)
}

func (s *Server) routes() {
	s.registerAuthRoutes()
	s.registerMetaRoutes()
	s.registerStudentRoutes()
	s.registerCourseRoutes()
	s.registerEnrollmentRoutes()
	s.registerAttendanceRoutes()
	s.registerInvoiceRoutes()
	s.registerSettingsRoutes()
	s.registerUserRoutes()
	s.registerPaymentRoutes()
	s.registerDashboardRoutes()
}

func (s *Server) registerAuthRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("POST /api/auth/login", s.handleAuthLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleAuthLogout)
	s.mux.HandleFunc("GET /api/auth/session", s.handleAuthSession)
}

func (s *Server) registerMetaRoutes() {
	s.mux.HandleFunc("GET /api/meta", s.handleMeta)
	s.mux.HandleFunc("GET /api/audit-logs", s.handleAuditLogsList)
	s.mux.HandleFunc("POST /api/backups", s.handleBackupsCreate)
}

func (s *Server) registerStudentRoutes() {
	s.mux.HandleFunc("GET /api/students", s.handleStudentsList)
	s.mux.HandleFunc("POST /api/students", s.handleStudentsCreate)
	s.mux.HandleFunc("POST /api/students/onboard", s.handleStudentsOnboard)
	s.mux.HandleFunc("POST /api/students/duplicate-check", s.handleStudentsDuplicateCheck)
	s.mux.HandleFunc("GET /api/students/{id}", s.handleStudentsGet)
	s.mux.HandleFunc("PUT /api/students/{id}", s.handleStudentsUpdate)
	s.mux.HandleFunc("DELETE /api/students/{id}", s.handleStudentsDelete)
	s.mux.HandleFunc("POST /api/students/{id}/active", s.handleStudentsActive)
	s.mux.HandleFunc("GET /api/students/{id}/debt-details", s.handleStudentDebtDetails)
	s.mux.HandleFunc("GET /api/teachers", s.handleTeachersList)
	s.mux.HandleFunc("POST /api/teachers", s.handleTeachersCreate)
}

func (s *Server) registerCourseRoutes() {
	s.mux.HandleFunc("GET /api/courses", s.handleCoursesList)
	s.mux.HandleFunc("POST /api/courses", s.handleCoursesCreate)
	s.mux.HandleFunc("GET /api/courses/{id}", s.handleCoursesGet)
	s.mux.HandleFunc("PUT /api/courses/{id}", s.handleCoursesUpdate)
	s.mux.HandleFunc("DELETE /api/courses/{id}", s.handleCoursesDelete)
}

func (s *Server) registerEnrollmentRoutes() {
	s.mux.HandleFunc("GET /api/enrollments", s.handleEnrollmentsList)
	s.mux.HandleFunc("POST /api/enrollments", s.handleEnrollmentsCreate)
	s.mux.HandleFunc("POST /api/enrollments/bulk", s.handleEnrollmentsBulkCreate)
	s.mux.HandleFunc("PUT /api/enrollments/{id}", s.handleEnrollmentsUpdate)
	s.mux.HandleFunc("DELETE /api/enrollments/{id}", s.handleEnrollmentsDelete)
}

func (s *Server) registerAttendanceRoutes() {
	s.mux.HandleFunc("GET /api/attendance/per-lesson", s.handleAttendanceList)
	s.mux.HandleFunc("PUT /api/attendance", s.handleAttendanceUpsert)
	s.mux.HandleFunc("POST /api/attendance/add-one", s.handleAttendanceAddOne)
	s.mux.HandleFunc("GET /api/attendance/subscription-month", s.handleAttendanceSubscriptionMonthList)
	s.mux.HandleFunc("PUT /api/attendance/subscription-month", s.handleAttendanceSubscriptionMonthUpsert)
}

func (s *Server) registerInvoiceRoutes() {
	s.mux.HandleFunc("GET /api/invoice-archive", s.handleInvoiceArchiveList)
	s.mux.HandleFunc("GET /api/invoice-archive/{year}/{month}/zip", s.handleInvoiceArchiveZip)
	s.mux.HandleFunc("GET /api/invoice-archive/{year}/{month}/{filename}/open", s.handleInvoiceArchiveOpen)
	s.mux.HandleFunc("GET /api/invoice-archive/{year}/{month}/{filename}/download", s.handleInvoiceArchiveDownload)
	s.mux.HandleFunc("GET /api/invoices", s.handleInvoicesList)
	s.mux.HandleFunc("GET /api/invoices/{id}", s.handleInvoicesGet)
	s.mux.HandleFunc("DELETE /api/invoices/{id}/draft", s.handleInvoicesDeleteDraft)
	s.mux.HandleFunc("POST /api/invoices/generate-drafts", s.handleInvoicesGenerateDrafts)
	s.mux.HandleFunc("POST /api/invoices/rebuild-student-draft", s.handleInvoicesRebuildStudentDraft)
	s.mux.HandleFunc("POST /api/invoices/{id}/reopen-draft", s.handleInvoicesReopenDraft)
	s.mux.HandleFunc("POST /api/invoices/{id}/issue", s.handleInvoicesIssue)
	s.mux.HandleFunc("POST /api/invoices/issue-all", s.handleInvoicesIssueAll)
	s.mux.HandleFunc("POST /api/invoices/ensure-pdf-all", s.handleInvoicesEnsurePDFAll)
	s.mux.HandleFunc("GET /api/invoices/{id}/pdf-status", s.handleInvoicesPDFStatus)
	s.mux.HandleFunc("POST /api/invoices/{id}/pdf", s.handleInvoicesEnsurePDF)
	s.mux.HandleFunc("GET /api/invoices/{id}/pdf", s.handleInvoicesDownloadPDF)
	s.mux.HandleFunc("POST /api/invoices/{id}/email-preview", s.handleInvoicesEmailPreview)
	s.mux.HandleFunc("POST /api/invoices/{id}/send-email", s.handleInvoicesSendEmail)
	s.mux.HandleFunc("GET /api/invoices/{id}/payment-summary", s.handleInvoicePaymentSummary)
}

func (s *Server) registerSettingsRoutes() {
	s.mux.HandleFunc("GET /api/me/locale", s.handleCurrentUserGetLocale)
	s.mux.HandleFunc("POST /api/me/locale", s.handleCurrentUserSetLocale)
	s.mux.HandleFunc("GET /api/settings/locale", s.handleSettingsGetLocale)
	s.mux.HandleFunc("POST /api/settings/locale", s.handleSettingsSetLocale)
	s.mux.HandleFunc("GET /api/settings/invoice-email", s.handleSettingsGetInvoiceEmail)
	s.mux.HandleFunc("POST /api/settings/invoice-email", s.handleSettingsSetInvoiceEmail)
}

func (s *Server) registerUserRoutes() {
	s.mux.HandleFunc("GET /api/users", s.handleUsersList)
	s.mux.HandleFunc("POST /api/users", s.handleUsersCreate)
	s.mux.HandleFunc("PUT /api/users/{id}", s.handleUsersUpdate)
	s.mux.HandleFunc("DELETE /api/users/{id}", s.handleUsersDelete)
	s.mux.HandleFunc("POST /api/users/{id}/password", s.handleUsersSetPassword)
	s.mux.HandleFunc("POST /api/users/{id}/active", s.handleUsersSetActive)
}

func (s *Server) registerPaymentRoutes() {
	s.mux.HandleFunc("POST /api/payments", s.handlePaymentsCreate)
	s.mux.HandleFunc("DELETE /api/payments/{id}", s.handlePaymentsDelete)
	s.mux.HandleFunc("POST /api/payments/quick-cash", s.handlePaymentsQuickCash)
	s.mux.HandleFunc("GET /api/payments/student/{studentId}", s.handlePaymentsListForStudent)
	s.mux.HandleFunc("GET /api/payments/student/{studentId}/balance", s.handleStudentBalance)
	s.mux.HandleFunc("GET /api/debtors", s.handleDebtorsList)
}

func (s *Server) registerDashboardRoutes() {
	s.mux.HandleFunc("GET /api/dashboard/month-overview", s.handleDashboardMonthOverview)
	s.mux.HandleFunc("GET /api/dashboard/recent-payments", s.handleDashboardRecentPayments)
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		s.mux.ServeHTTP(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.serveAPI(w, r)
		return
	}

	if s.distDir == "" || !s.hasIndex {
		http.NotFound(w, r)
		return
	}

	if path, ok := s.staticPath(r.URL.Path); ok {
		http.ServeFile(w, r, path)
		return
	}

	if looksLikeAssetRequest(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filepath.Join(s.distDir, "index.html"))
}

func (s *Server) serveAPI(w http.ResponseWriter, r *http.Request) {
	if isPublicAPIPath(r.URL.Path) {
		s.mux.ServeHTTP(w, r)
		return
	}

	currentUser, err := s.userFromRequest(r)
	if err != nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if isAdminOnlyAPIPath(r.Method, r.URL.Path) && currentUser.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
		return
	}

	ctx := withCurrentUser(r.Context(), currentUser)
	ctx = backend.WithActor(ctx, currentUser)
	s.mux.ServeHTTP(w, r.WithContext(ctx))
}

type studentUpsertRequest struct {
	Version      int    `json:"version"`
	FullName     string `json:"fullName"`
	PersonalCode string `json:"personalCode"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Note         string `json:"note"`
	IsMinor      bool   `json:"isMinor"`
	PayerName    string `json:"payerName"`
	PayerRole    string `json:"payerRole"`
}

type courseUpsertRequest struct {
	Version           int     `json:"version"`
	Name              string  `json:"name"`
	TeacherID         *int    `json:"teacherId"`
	Type              string  `json:"type"`
	LessonPrice       float64 `json:"lessonPrice"`
	SubscriptionPrice float64 `json:"subscriptionPrice"`
}

type enrollmentCreateRequest struct {
	StudentID               int     `json:"studentId"`
	CourseID                int     `json:"courseId"`
	BillingMode             string  `json:"billingMode"`
	ChargeMaterials         bool    `json:"chargeMaterials"`
	LessonPriceOverride     float64 `json:"lessonPriceOverride"`
	SubscriptionLessonPrice float64 `json:"subscriptionLessonPrice"`
	Note                    string  `json:"note"`
}

type enrollmentUpdateRequest struct {
	Version                 int     `json:"version"`
	BillingMode             string  `json:"billingMode"`
	ChargeMaterials         bool    `json:"chargeMaterials"`
	LessonPriceOverride     float64 `json:"lessonPriceOverride"`
	SubscriptionLessonPrice float64 `json:"subscriptionLessonPrice"`
	Note                    string  `json:"note"`
}

type studentOnboardRequest struct {
	Student     studentUpsertRequest      `json:"student"`
	Enrollment  *enrollmentCreateRequest  `json:"enrollment"`
	Enrollments []enrollmentCreateRequest `json:"enrollments"`
}

type enrollmentBulkCreateRequest struct {
	StudentID   int                       `json:"studentId"`
	Enrollments []enrollmentCreateRequest `json:"enrollments"`
}

type periodRequest struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

type versionRequest struct {
	Version int `json:"version"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": message})
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": message})
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case ent.IsNotFound(err):
		status = http.StatusNotFound
	case errors.Is(err, os.ErrNotExist):
		status = http.StatusNotFound
	case ent.IsConstraintError(err):
		status = http.StatusConflict
	case errors.Is(err, auth.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, auth.ErrForbidden):
		status = http.StatusForbidden
	case apperrors.IsConflict(err):
		status = http.StatusConflict
	case isConflictError(err):
		status = http.StatusConflict
	case isBadRequestError(err):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func isConflictError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cannot ") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "already") ||
		strings.Contains(msg, "заблокирован") ||
		strings.Contains(msg, "нельзя")
}

func isBadRequestError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "must be") ||
		strings.Contains(msg, "must contain") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "not configured") ||
		strings.Contains(msg, "некоррект") ||
		strings.Contains(msg, "долж")
}

func pathInt(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	value := r.PathValue(name)
	id, err := strconv.Atoi(value)
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("invalid %s", name))
		return 0, false
	}
	return id, true
}

func parseOptionalInt(raw string) (*int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid integer %q", raw)
	}
	return &value, nil
}

func parseRequiredVersionQuery(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("version"))
	if raw == "" {
		return 0, errors.New("missing version")
	}
	version, err := strconv.Atoi(raw)
	if err != nil || version <= 0 {
		return 0, errors.New("invalid version")
	}
	return version, nil
}

func parseRequiredQueryInt(r *http.Request, name string) (int, error) {
	raw := r.URL.Query().Get(name)
	if strings.TrimSpace(raw) == "" {
		return 0, fmt.Errorf("missing query parameter %s", name)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid query parameter %s", name)
	}
	return value, nil
}

func parseBoolDefault(raw string, fallback bool) (bool, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, errors.New("invalid boolean value")
	}
	return value, nil
}

func parseQueryIntDefault(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("invalid integer value")
	}
	return value, nil
}

func intQuery(r *http.Request, name string, fallback int) int {
	value, err := parseQueryIntDefault(r.URL.Query().Get(name), fallback)
	if err != nil {
		return fallback
	}
	return value
}

func normalizeDistDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	return abs
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (s *Server) staticPath(requestPath string) (string, bool) {
	cleanPath := filepath.Clean("/" + strings.TrimSpace(requestPath))
	if cleanPath == "/" {
		indexPath := filepath.Join(s.distDir, "index.html")
		return indexPath, fileExists(indexPath)
	}

	relPath := strings.TrimPrefix(cleanPath, "/")
	fullPath := filepath.Join(s.distDir, relPath)
	relToBase, err := filepath.Rel(s.distDir, fullPath)
	if err != nil || relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
		return "", false
	}
	return fullPath, fileExists(fullPath)
}

func looksLikeAssetRequest(requestPath string) bool {
	base := filepath.Base(strings.TrimSpace(requestPath))
	return strings.Contains(base, ".")
}

func isPublicAPIPath(path string) bool {
	switch path {
	case "/api/auth/login", "/api/auth/logout", "/api/auth/session":
		return true
	default:
		return false
	}
}

func isAdminOnlyAPIPath(method, path string) bool {
	switch {
	case method == http.MethodPost && path == "/api/backups":
		return true
	case method == http.MethodPost && path == "/api/settings/locale":
		return true
	case (method == http.MethodGet || method == http.MethodPost) && path == "/api/settings/invoice-email":
		return true
	case method == http.MethodGet && path == "/api/audit-logs":
		return true
	case strings.HasPrefix(path, "/api/users"):
		return true
	case method == http.MethodDelete && strings.HasPrefix(path, "/api/payments/"):
		return true
	case method == http.MethodDelete && strings.HasPrefix(path, "/api/students/"):
		return true
	case method == http.MethodDelete && strings.HasPrefix(path, "/api/courses/"):
		return true
	default:
		return false
	}
}

type contextKey string

const currentUserKey contextKey = "currentUser"

func withCurrentUser(ctx context.Context, currentUser *auth.UserInfo) context.Context {
	return context.WithValue(ctx, currentUserKey, currentUser)
}

func currentUserFromContext(ctx context.Context) *auth.UserInfo {
	currentUser, _ := ctx.Value(currentUserKey).(*auth.UserInfo)
	return currentUser
}

func (s *Server) userFromRequest(r *http.Request) (*auth.UserInfo, error) {
	cookie, err := r.Cookie(auth.CookieName)
	if err != nil {
		return nil, auth.ErrUnauthorized
	}
	return s.svc.Session(r.Context(), cookie.Value)
}
