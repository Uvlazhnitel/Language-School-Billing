package web

import "net/http"

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
