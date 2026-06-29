package web

import "net/http"

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
