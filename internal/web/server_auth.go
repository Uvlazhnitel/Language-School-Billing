package web

import (
	"errors"
	"net/http"

	"langschool/internal/auth"
)

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
