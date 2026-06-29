package web

import (
	"errors"
	"net/http"

	"langschool/internal/auth"
)

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
