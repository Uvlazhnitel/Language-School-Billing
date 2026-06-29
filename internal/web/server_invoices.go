package web

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"langschool/internal/backend"
)

func (s *Server) handleInvoicesList(w http.ResponseWriter, r *http.Request) {
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
	status := r.URL.Query().Get("status")
	items, err := s.svc.InvoiceList(r.Context(), year, month, status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleInvoicesGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.InvoiceGet(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesGenerateDrafts(w http.ResponseWriter, r *http.Request) {
	var req periodRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceGenerateDrafts(r.Context(), req.Year, req.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesRebuildStudentDraft(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int `json:"studentId"`
		Year      int `json:"year"`
		Month     int `json:"month"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceRebuildStudentDraft(r.Context(), req.StudentID, req.Year, req.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesDeleteDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	version, err := parseRequiredVersionQuery(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if err := s.svc.InvoiceDeleteDraftWithVersion(r.Context(), id, version); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleInvoicesReopenDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req versionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.svc.InvoiceReopenDraftWithVersion(r.Context(), id, req.Version); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleInvoicesIssue(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req versionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceIssueWithVersion(r.Context(), id, req.Version)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesIssueAll(w http.ResponseWriter, r *http.Request) {
	var req periodRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceIssueAll(r.Context(), req.Year, req.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesEnsurePDFAll(w http.ResponseWriter, r *http.Request) {
	var req periodRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceEnsurePDFAll(r.Context(), req.Year, req.Month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesPDFStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	ready, err := s.svc.InvoiceHasPDF(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ready": ready})
}

func (s *Server) handleInvoicesEnsurePDF(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	path, err := s.svc.InvoiceEnsurePDF(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"filename":    filepath.Base(path),
		"downloadUrl": fmt.Sprintf("/api/invoices/%d/pdf", id),
	})
}

func (s *Server) handleInvoicesDownloadPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	path, err := s.svc.InvoiceEnsurePDF(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func (s *Server) handleInvoiceArchiveList(w http.ResponseWriter, r *http.Request) {
	item, err := s.svc.InvoiceArchiveList(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoiceArchiveOpen(w http.ResponseWriter, r *http.Request) {
	s.handleInvoiceArchiveFile(w, r, "inline")
}

func (s *Server) handleInvoiceArchiveZip(w http.ResponseWriter, r *http.Request) {
	year, ok := pathInt(w, r, "year")
	if !ok {
		return
	}
	month, ok := pathInt(w, r, "month")
	if !ok {
		return
	}

	entries, filename, err := s.svc.InvoiceArchiveZIPEntries(r.Context(), year, month)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	zw := zip.NewWriter(w)
	for _, entry := range entries {
		fileWriter, err := zw.Create(entry.Name)
		if err != nil {
			writeError(w, err)
			return
		}
		file, err := os.Open(entry.Path)
		if err != nil {
			writeError(w, err)
			return
		}
		if _, err := io.Copy(fileWriter, file); err != nil {
			_ = file.Close()
			writeError(w, err)
			return
		}
		if err := file.Close(); err != nil {
			writeError(w, err)
			return
		}
	}
	if err := zw.Close(); err != nil {
		writeError(w, err)
		return
	}
}

func (s *Server) handleInvoiceArchiveDownload(w http.ResponseWriter, r *http.Request) {
	s.handleInvoiceArchiveFile(w, r, "attachment")
}

func (s *Server) handleInvoiceArchiveFile(w http.ResponseWriter, r *http.Request, disposition string) {
	year, ok := pathInt(w, r, "year")
	if !ok {
		return
	}
	month, ok := pathInt(w, r, "month")
	if !ok {
		return
	}
	filename := r.PathValue("filename")
	path, err := s.svc.InvoiceArchiveFilePath(year, month, filename)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filepath.Base(path)))
	http.ServeFile(w, r, path)
}

func (s *Server) handleInvoicesEmailPreview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.InvoiceEmailPreview(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicesSendEmail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var req backend.InvoiceEmailRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.InvoiceSendEmail(r.Context(), id, req.To, req.Subject, req.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleInvoicePaymentSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	item, err := s.svc.InvoicePaymentSummary(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
