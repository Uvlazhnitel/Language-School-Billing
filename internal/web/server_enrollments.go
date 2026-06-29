package web

import "net/http"

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
	item, err := s.svc.EnrollmentCreate(r.Context(), req.StudentID, req.CourseID, req.BillingMode, req.ChargeMaterials, req.LessonPriceOverride, req.SubscriptionLessonPrice, req.Note)
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
	item, err := s.svc.EnrollmentUpdateWithVersion(r.Context(), id, req.Version, req.BillingMode, req.ChargeMaterials, req.LessonPriceOverride, req.SubscriptionLessonPrice, req.Note)
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
