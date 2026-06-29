package web

import "net/http"

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
