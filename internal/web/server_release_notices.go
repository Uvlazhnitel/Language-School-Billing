package web

import "net/http"

func (s *Server) handleReleaseNoticeVisit(w http.ResponseWriter, r *http.Request) {
	user := currentUserFromContext(r.Context())
	if user == nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	result, err := s.svc.ReleaseNoticeVisit(r.Context(), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) handleReleaseNoticeDismiss(w http.ResponseWriter, r *http.Request) {
	user := currentUserFromContext(r.Context())
	if user == nil {
		writeUnauthorized(w, "authentication required")
		return
	}
	if err := s.svc.ReleaseNoticeDismiss(r.Context(), user.ID, r.PathValue("releaseId")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
