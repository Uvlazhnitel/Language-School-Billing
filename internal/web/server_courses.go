package web

import "net/http"

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
