package web

import (
	"errors"
	"net/http"

	paysvc "langschool/internal/app/payment"
)

func (s *Server) handlePaymentsCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OperationID string  `json:"operationId"`
		StudentID   int     `json:"studentId"`
		InvoiceID   *int    `json:"invoiceId"`
		Amount      float64 `json:"amount"`
		Method      string  `json:"method"`
		PaidAt      string  `json:"paidAt"`
		Note        string  `json:"note"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.svc.PaymentCreateWithOperation(r.Context(), req.OperationID, req.StudentID, req.InvoiceID, req.Amount, req.Method, req.PaidAt, req.Note)
	if err != nil {
		if errors.Is(err, paysvc.ErrInvalidOperationID) {
			writeBadRequest(w, err.Error())
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handlePaymentsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	if err := s.svc.PaymentDelete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePaymentsListForStudent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "studentId")
	if !ok {
		return
	}
	items, err := s.svc.PaymentListForStudent(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleStudentBalance(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "studentId")
	if !ok {
		return
	}
	item, err := s.svc.StudentBalance(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDebtorsList(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.DebtorsList(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
