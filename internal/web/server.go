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
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("POST /api/auth/login", s.handleAuthLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleAuthLogout)
	s.mux.HandleFunc("GET /api/auth/session", s.handleAuthSession)
	s.mux.HandleFunc("GET /api/meta", s.handleMeta)
	s.mux.HandleFunc("GET /api/me/locale", s.handleCurrentUserGetLocale)
	s.mux.HandleFunc("POST /api/me/locale", s.handleCurrentUserSetLocale)
	s.mux.HandleFunc("POST /api/backups", s.handleBackupsCreate)
	s.mux.HandleFunc("GET /api/invoice-archive", s.handleInvoiceArchiveList)
	s.mux.HandleFunc("GET /api/invoice-archive/{year}/{month}/{filename}/open", s.handleInvoiceArchiveOpen)
	s.mux.HandleFunc("GET /api/invoice-archive/{year}/{month}/{filename}/download", s.handleInvoiceArchiveDownload)

	s.mux.HandleFunc("GET /api/students", s.handleStudentsList)
	s.mux.HandleFunc("POST /api/students", s.handleStudentsCreate)
	s.mux.HandleFunc("GET /api/students/{id}", s.handleStudentsGet)
	s.mux.HandleFunc("PUT /api/students/{id}", s.handleStudentsUpdate)
	s.mux.HandleFunc("DELETE /api/students/{id}", s.handleStudentsDelete)
	s.mux.HandleFunc("POST /api/students/{id}/active", s.handleStudentsActive)
	s.mux.HandleFunc("GET /api/students/{id}/debt-details", s.handleStudentDebtDetails)

	s.mux.HandleFunc("GET /api/teachers", s.handleTeachersList)
	s.mux.HandleFunc("POST /api/teachers", s.handleTeachersCreate)

	s.mux.HandleFunc("GET /api/courses", s.handleCoursesList)
	s.mux.HandleFunc("POST /api/courses", s.handleCoursesCreate)
	s.mux.HandleFunc("GET /api/courses/{id}", s.handleCoursesGet)
	s.mux.HandleFunc("PUT /api/courses/{id}", s.handleCoursesUpdate)
	s.mux.HandleFunc("DELETE /api/courses/{id}", s.handleCoursesDelete)

	s.mux.HandleFunc("GET /api/enrollments", s.handleEnrollmentsList)
	s.mux.HandleFunc("POST /api/enrollments", s.handleEnrollmentsCreate)
	s.mux.HandleFunc("PUT /api/enrollments/{id}", s.handleEnrollmentsUpdate)
	s.mux.HandleFunc("DELETE /api/enrollments/{id}", s.handleEnrollmentsDelete)

	s.mux.HandleFunc("GET /api/attendance/per-lesson", s.handleAttendanceList)
	s.mux.HandleFunc("PUT /api/attendance", s.handleAttendanceUpsert)
	s.mux.HandleFunc("POST /api/attendance/add-one", s.handleAttendanceAddOne)
	s.mux.HandleFunc("GET /api/attendance/subscription-month", s.handleAttendanceSubscriptionMonthList)
	s.mux.HandleFunc("PUT /api/attendance/subscription-month", s.handleAttendanceSubscriptionMonthUpsert)

	s.mux.HandleFunc("GET /api/invoices", s.handleInvoicesList)
	s.mux.HandleFunc("GET /api/invoices/{id}", s.handleInvoicesGet)
	s.mux.HandleFunc("DELETE /api/invoices/{id}/draft", s.handleInvoicesDeleteDraft)
	s.mux.HandleFunc("POST /api/invoices/generate-drafts", s.handleInvoicesGenerateDrafts)
	s.mux.HandleFunc("POST /api/invoices/rebuild-student-draft", s.handleInvoicesRebuildStudentDraft)
	s.mux.HandleFunc("POST /api/invoices/{id}/reopen-draft", s.handleInvoicesReopenDraft)
	s.mux.HandleFunc("POST /api/invoices/{id}/issue", s.handleInvoicesIssue)
	s.mux.HandleFunc("POST /api/invoices/issue-all", s.handleInvoicesIssueAll)
	s.mux.HandleFunc("GET /api/invoices/{id}/pdf-status", s.handleInvoicesPDFStatus)
	s.mux.HandleFunc("POST /api/invoices/{id}/pdf", s.handleInvoicesEnsurePDF)
	s.mux.HandleFunc("GET /api/invoices/{id}/pdf", s.handleInvoicesDownloadPDF)
	s.mux.HandleFunc("POST /api/invoices/{id}/email-preview", s.handleInvoicesEmailPreview)
	s.mux.HandleFunc("POST /api/invoices/{id}/send-email", s.handleInvoicesSendEmail)
	s.mux.HandleFunc("GET /api/invoices/{id}/payment-summary", s.handleInvoicePaymentSummary)

	s.mux.HandleFunc("GET /api/settings/locale", s.handleSettingsGetLocale)
	s.mux.HandleFunc("POST /api/settings/locale", s.handleSettingsSetLocale)
	s.mux.HandleFunc("GET /api/settings/invoice-email", s.handleSettingsGetInvoiceEmail)
	s.mux.HandleFunc("POST /api/settings/invoice-email", s.handleSettingsSetInvoiceEmail)
	s.mux.HandleFunc("GET /api/audit-logs", s.handleAuditLogsList)
	s.mux.HandleFunc("GET /api/users", s.handleUsersList)
	s.mux.HandleFunc("POST /api/users", s.handleUsersCreate)
	s.mux.HandleFunc("PUT /api/users/{id}", s.handleUsersUpdate)
	s.mux.HandleFunc("DELETE /api/users/{id}", s.handleUsersDelete)
	s.mux.HandleFunc("POST /api/users/{id}/password", s.handleUsersSetPassword)
	s.mux.HandleFunc("POST /api/users/{id}/active", s.handleUsersSetActive)

	s.mux.HandleFunc("POST /api/payments", s.handlePaymentsCreate)
	s.mux.HandleFunc("DELETE /api/payments/{id}", s.handlePaymentsDelete)
	s.mux.HandleFunc("POST /api/payments/quick-cash", s.handlePaymentsQuickCash)
	s.mux.HandleFunc("GET /api/payments/student/{studentId}", s.handlePaymentsListForStudent)
	s.mux.HandleFunc("GET /api/payments/student/{studentId}/balance", s.handleStudentBalance)

	s.mux.HandleFunc("GET /api/debtors", s.handleDebtorsList)
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

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ready": s.svc.Ready()})
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RememberMe bool   `json:"rememberMe"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	currentUser, signedToken, expiresAt, persistent, err := s.svc.Login(r.Context(), req.Username, req.Password, req.RememberMe)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			writeUnauthorized(w, "invalid username or password")
			return
		}
		writeError(w, err)
		return
	}

	http.SetCookie(w, s.svc.SessionCookie(signedToken, expiresAt, persistent))

	session, err := s.svc.SessionState(r.Context(), currentUser)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		_ = s.svc.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, s.svc.ClearSessionCookie())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAuthSession(w http.ResponseWriter, r *http.Request) {
	currentUser, err := s.userFromRequest(r)
	if err != nil && !errors.Is(err, auth.ErrUnauthorized) {
		writeError(w, err)
		return
	}
	if errors.Is(err, auth.ErrUnauthorized) {
		currentUser = nil
	}

	session, err := s.svc.SessionState(r.Context(), currentUser)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	meta, err := s.svc.Meta(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (s *Server) handleAuditLogsList(w http.ResponseWriter, r *http.Request) {
	filter := backend.AuditLogListFilter{
		Query:      strings.TrimSpace(r.URL.Query().Get("q")),
		ActorLabel: strings.TrimSpace(r.URL.Query().Get("actor")),
		EntityType: strings.TrimSpace(r.URL.Query().Get("entityType")),
		Action:     strings.TrimSpace(r.URL.Query().Get("action")),
		DateFrom:   strings.TrimSpace(r.URL.Query().Get("dateFrom")),
		DateTo:     strings.TrimSpace(r.URL.Query().Get("dateTo")),
		Page:       intQuery(r, "page", 1),
		PageSize:   intQuery(r, "pageSize", 50),
	}
	result, err := s.svc.AuditLogList(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleBackupsCreate(w http.ResponseWriter, r *http.Request) {
	path, err := s.svc.BackupNow()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"filename": filepath.Base(path),
	})
}

func (s *Server) handleStudentsList(w http.ResponseWriter, r *http.Request) {
	includeInactive, err := parseBoolDefault(r.URL.Query().Get("includeInactive"), false)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	items, err := s.svc.StudentList(r.Context(), r.URL.Query().Get("q"), includeInactive)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleStudentsGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.StudentGet(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleStudentsCreate(w http.ResponseWriter, r *http.Request) {
	var req studentUpsertRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.StudentCreate(r.Context(), req.FullName, req.PersonalCode, req.Phone, req.Email, req.Note, req.IsMinor, req.PayerName, req.PayerRole)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleStudentsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req studentUpsertRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.StudentUpdateWithVersion(r.Context(), id, req.Version, req.FullName, req.PersonalCode, req.Phone, req.Email, req.Note, req.IsMinor, req.PayerName, req.PayerRole)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleStudentsActive(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Active  bool `json:"active"`
		Version int  `json:"version"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.svc.StudentSetActiveWithVersion(r.Context(), id, req.Version, req.Active); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleStudentsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	version, err := parseRequiredVersionQuery(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.svc.StudentDeleteWithVersion(r.Context(), id, version); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStudentDebtDetails(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	items, err := s.svc.StudentDebtDetails(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleTeachersList(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.TeacherList(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleTeachersCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName string `json:"fullName"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.TeacherCreate(r.Context(), req.FullName)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleCoursesList(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.CourseList(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleCoursesGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.CourseGet(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleCoursesCreate(w http.ResponseWriter, r *http.Request) {
	var req courseUpsertRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.CourseCreate(r.Context(), req.Name, req.TeacherID, req.Type, req.LessonPrice, req.SubscriptionPrice)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleCoursesUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req courseUpsertRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.CourseUpdateWithVersion(r.Context(), id, req.Version, req.Name, req.TeacherID, req.Type, req.LessonPrice, req.SubscriptionPrice)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleCoursesDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	version, err := parseRequiredVersionQuery(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.svc.CourseDeleteWithVersion(r.Context(), id, version); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEnrollmentsList(w http.ResponseWriter, r *http.Request) {
	studentID, err := parseOptionalInt(r.URL.Query().Get("studentId"))
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	courseID, err := parseOptionalInt(r.URL.Query().Get("courseId"))
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	items, err := s.svc.EnrollmentList(r.Context(), studentID, courseID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleEnrollmentsCreate(w http.ResponseWriter, r *http.Request) {
	var req enrollmentCreateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.EnrollmentCreate(r.Context(), req.StudentID, req.CourseID, req.BillingMode, req.ChargeMaterials, req.DiscountPct, req.SubscriptionLessonPrice, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleEnrollmentsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req enrollmentUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.EnrollmentUpdateWithVersion(r.Context(), id, req.Version, req.BillingMode, req.ChargeMaterials, req.DiscountPct, req.SubscriptionLessonPrice, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleEnrollmentsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	version, err := parseRequiredVersionQuery(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.svc.EnrollmentDeleteWithVersion(r.Context(), id, version); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAttendanceList(w http.ResponseWriter, r *http.Request) {
	year, err := parseRequiredQueryInt(r, "year")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	month, err := parseRequiredQueryInt(r, "month")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	courseID, err := parseOptionalInt(r.URL.Query().Get("courseId"))
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	items, err := s.svc.AttendanceListPerLesson(r.Context(), year, month, courseID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleAttendanceUpsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int     `json:"studentId"`
		CourseID  int     `json:"courseId"`
		Year      int     `json:"year"`
		Month     int     `json:"month"`
		Hours     float64 `json:"hours"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.svc.AttendanceUpsert(r.Context(), req.StudentID, req.CourseID, req.Year, req.Month, req.Hours); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAttendanceAddOne(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Year     int  `json:"year"`
		Month    int  `json:"month"`
		CourseID *int `json:"courseId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	count, err := s.svc.AttendanceAddOne(r.Context(), req.Year, req.Month, req.CourseID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (s *Server) handleAttendanceSubscriptionMonthList(w http.ResponseWriter, r *http.Request) {
	year, err := parseRequiredQueryInt(r, "year")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	month, err := parseRequiredQueryInt(r, "month")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	courseID, err := parseOptionalInt(r.URL.Query().Get("courseId"))
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	items, err := s.svc.CourseMonthSubscriptionList(r.Context(), year, month, courseID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleAttendanceSubscriptionMonthUpsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CourseID    int     `json:"courseId"`
		Year        int     `json:"year"`
		Month       int     `json:"month"`
		LessonsHeld float64 `json:"lessonsHeld"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.CourseMonthSubscriptionUpsert(r.Context(), req.CourseID, req.Year, req.Month, req.LessonsHeld)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesList(w http.ResponseWriter, r *http.Request) {
	year, err := parseRequiredQueryInt(r, "year")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	month, err := parseRequiredQueryInt(r, "month")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	status := r.URL.Query().Get("status")
	items, err := s.svc.InvoiceList(r.Context(), year, month, status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleInvoicesGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.InvoiceGet(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesGenerateDrafts(w http.ResponseWriter, r *http.Request) {
	var req periodRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceGenerateDrafts(r.Context(), req.Year, req.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesRebuildStudentDraft(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int `json:"studentId"`
		Year      int `json:"year"`
		Month     int `json:"month"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceRebuildStudentDraft(r.Context(), req.StudentID, req.Year, req.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesDeleteDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	version, err := parseRequiredVersionQuery(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.svc.InvoiceDeleteDraftWithVersion(r.Context(), id, version); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleInvoicesReopenDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req versionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.svc.InvoiceReopenDraftWithVersion(r.Context(), id, req.Version); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleInvoicesIssue(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req versionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceIssueWithVersion(r.Context(), id, req.Version)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesIssueAll(w http.ResponseWriter, r *http.Request) {
	var req periodRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceIssueAll(r.Context(), req.Year, req.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesPDFStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	ready, err := s.svc.InvoiceHasPDF(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ready": ready})
}

func (s *Server) handleInvoicesEnsurePDF(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	path, err := s.svc.InvoiceEnsurePDF(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"filename":    filepath.Base(path),
		"downloadUrl": fmt.Sprintf("/api/invoices/%d/pdf", id),
	})
}

func (s *Server) handleInvoicesDownloadPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	path, err := s.svc.InvoiceEnsurePDF(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func (s *Server) handleInvoiceArchiveList(w http.ResponseWriter, r *http.Request) {
	item, err := s.svc.InvoiceArchiveList(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoiceArchiveOpen(w http.ResponseWriter, r *http.Request) {
	s.handleInvoiceArchiveFile(w, r, "inline")
}

func (s *Server) handleInvoiceArchiveDownload(w http.ResponseWriter, r *http.Request) {
	s.handleInvoiceArchiveFile(w, r, "attachment")
}

func (s *Server) handleInvoiceArchiveFile(w http.ResponseWriter, r *http.Request, disposition string) {
	year, ok := pathInt(w, r, "year")
	if !ok {
		return
	}
	month, ok := pathInt(w, r, "month")
	if !ok {
		return
	}
	filename := r.PathValue("filename")
	path, err := s.svc.InvoiceArchiveFilePath(year, month, filename)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func (s *Server) handleInvoicesEmailPreview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.InvoiceEmailPreview(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesSendEmail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req backend.InvoiceEmailRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceSendEmail(r.Context(), id, req.To, req.Subject, req.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleSettingsGetLocale(w http.ResponseWriter, r *http.Request) {
	locale, err := s.svc.SettingsGetLocale(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"locale": locale})
}

func (s *Server) handleSettingsSetLocale(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Locale string `json:"locale"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.svc.SettingsSetLocale(r.Context(), req.Locale); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"locale": req.Locale})
}

func (s *Server) handleSettingsGetInvoiceEmail(w http.ResponseWriter, r *http.Request) {
	item, err := s.svc.SettingsGetInvoiceEmail(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleSettingsSetInvoiceEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubjectTemplate string `json:"subjectTemplate"`
		BodyTemplate    string `json:"bodyTemplate"`
		ReplyTo         string `json:"replyTo"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.SettingsSetInvoiceEmail(r.Context(), req.SubjectTemplate, req.BodyTemplate, req.ReplyTo)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleCurrentUserGetLocale(w http.ResponseWriter, r *http.Request) {
	currentUser := currentUserFromContext(r.Context())
	if currentUser == nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	locale, err := s.svc.UserGetLocale(r.Context(), currentUser.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"locale": locale})
}

func (s *Server) handleCurrentUserSetLocale(w http.ResponseWriter, r *http.Request) {
	currentUser := currentUserFromContext(r.Context())
	if currentUser == nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	var req struct {
		Locale string `json:"locale"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	locale, err := s.svc.UserSetLocale(r.Context(), currentUser.ID, req.Locale)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"locale": locale})
}

func (s *Server) handleUsersList(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.UserList(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleUsersCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.UserCreate(r.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleUsersUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		IsActive bool   `json:"isActive"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.UserUpdate(r.Context(), id, req.Username, req.Role, req.IsActive)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleUsersSetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.svc.UserSetPassword(r.Context(), id, req.Password); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleUsersDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	currentUser := currentUserFromContext(r.Context())
	if currentUser == nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := s.svc.UserDelete(r.Context(), currentUser.ID, id); err != nil {
		switch {
		case errors.Is(err, auth.ErrDeleteSelf), errors.Is(err, auth.ErrDeleteLastAdmin):
			writeBadRequest(w, err.Error())
		default:
			writeError(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUsersSetActive(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Active bool `json:"active"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.UserSetActive(r.Context(), id, req.Active)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handlePaymentsCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int     `json:"studentId"`
		InvoiceID *int    `json:"invoiceId"`
		Amount    float64 `json:"amount"`
		Method    string  `json:"method"`
		PaidAt    string  `json:"paidAt"`
		Note      string  `json:"note"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.PaymentCreate(r.Context(), req.StudentID, req.InvoiceID, req.Amount, req.Method, req.PaidAt, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handlePaymentsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	if err := s.svc.PaymentDelete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePaymentsQuickCash(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int     `json:"studentId"`
		Amount    float64 `json:"amount"`
		Note      string  `json:"note"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.PaymentQuickCash(r.Context(), req.StudentID, req.Amount, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handlePaymentsListForStudent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "studentId")
	if !ok {
		return
	}
	items, err := s.svc.PaymentListForStudent(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleStudentBalance(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "studentId")
	if !ok {
		return
	}
	item, err := s.svc.StudentBalance(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDebtorsList(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.DebtorsList(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleDashboardMonthOverview(w http.ResponseWriter, r *http.Request) {
	year, err := parseRequiredQueryInt(r, "year")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	month, err := parseRequiredQueryInt(r, "month")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	item, err := s.svc.MonthOverview(r.Context(), year, month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDashboardRecentPayments(w http.ResponseWriter, r *http.Request) {
	limit, err := parseQueryIntDefault(r.URL.Query().Get("limit"), 8)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	items, err := s.svc.RecentPayments(r.Context(), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleInvoicePaymentSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.InvoicePaymentSummary(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
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
	DiscountPct             float64 `json:"discountPct"`
	SubscriptionLessonPrice float64 `json:"subscriptionLessonPrice"`
	Note                    string  `json:"note"`
}

type enrollmentUpdateRequest struct {
	Version                 int     `json:"version"`
	BillingMode             string  `json:"billingMode"`
	ChargeMaterials         bool    `json:"chargeMaterials"`
	DiscountPct             float64 `json:"discountPct"`
	SubscriptionLessonPrice float64 `json:"subscriptionLessonPrice"`
	Note                    string  `json:"note"`
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
		strings.Contains(msg, "заблокирована") ||
		strings.Contains(msg, "нельзя")
}

func isBadRequestError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "must be") ||
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
