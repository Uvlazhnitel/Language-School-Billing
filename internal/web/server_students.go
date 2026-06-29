package web

import "net/http"

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
