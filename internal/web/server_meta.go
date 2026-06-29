package web

import (
	"net/http"
	"path/filepath"
	"strings"

	"langschool/internal/backend"
)

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
