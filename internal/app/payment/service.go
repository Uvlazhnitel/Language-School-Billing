// Package payment provides services for payment processing and financial tracking.
// It handles payment creation, balance calculations, and invoice payment tracking.
package payment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"langschool/ent"
	"langschool/ent/invoice"
	"langschool/ent/payment"
	"langschool/ent/student"
	"langschool/internal/app"
	"langschool/internal/app/utils"
)

// Service provides payment processing and financial tracking functionality.
type Service struct{ db *ent.Client }

// New creates a new payment service with the given database client.
func New(db *ent.Client) *Service { return &Service{db: db} }

// PaymentDTO represents a payment record in the frontend.
type PaymentDTO struct {
	ID        int     `json:"id"`        // Payment ID
	StudentID int     `json:"studentId"` // ID of the student who made the payment
	InvoiceID *int    `json:"invoiceId,omitempty"` // Optional: ID of invoice this payment is for
	PaidAt    string  `json:"paidAt"`    // Payment date in RFC3339 format
	Amount    float64 `json:"amount"`    // Payment amount
	Method    string  `json:"method"`    // Payment method: "cash" or "bank"
	Note      string  `json:"note"`      // Optional notes about the payment
	CreatedAt string  `json:"createdAt"` // Record creation date in RFC3339 format
}

// BalanceDTO represents a student's financial balance.
type BalanceDTO struct {
	StudentID     int     `json:"studentId"`     // Student ID
	StudentName   string  `json:"studentName"`   // Student's full name
	TotalInvoiced float64 `json:"totalInvoiced"` // Total amount invoiced (issued + paid invoices)
	TotalPaid     float64 `json:"totalPaid"`     // Total amount paid (all payments)
	Balance       float64 `json:"balance"`       // Balance: paid - invoiced (negative => student owes)
	Debt          float64 `json:"debt"`           // Debt: max(0, -balance), amount student owes
}

// DebtorDTO represents a student with outstanding debt.
type DebtorDTO struct {
	StudentID     int     `json:"studentId"`     // Student ID
	StudentName   string  `json:"studentName"`   // Student's full name
	Debt          float64 `json:"debt"`          // Amount owed
	TotalInvoiced float64 `json:"totalInvoiced"` // Total amount invoiced
	TotalPaid     float64 `json:"totalPaid"`     // Total amount paid
}

// InvoiceSummaryDTO represents payment summary for a specific invoice.
type InvoiceSummaryDTO struct {
	InvoiceID int     `json:"invoiceId"` // Invoice ID
	Total     float64 `json:"total"`      // Total invoice amount
	Paid      float64 `json:"paid"`       // Amount paid so far
	Remaining float64 `json:"remaining"`  // Remaining amount to pay
	Status    string  `json:"status"`     // Invoice status
	Number    *string `json:"number,omitempty"` // Invoice number
}


// eps returns the epsilon value used for floating-point comparisons.
// The value 0.009 is chosen to account for rounding errors in currency calculations
// where amounts are rounded to 2 decimal places (0.01). This epsilon is slightly
// smaller than 0.01 to allow for safe boundary checks without false positives,
// while being large enough to handle typical floating-point precision issues.
func eps() float64 { return 0.009 }

// parseDate parses a date string in either "YYYY-MM-DD" or RFC3339 format.
// This flexible parsing allows the frontend to send dates in either format.
func parseDate(s string) (time.Time, error) {
	// Accept YYYY-MM-DD or RFC3339.
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

// Create creates a new payment record. If invoiceID is provided, the payment
// is linked to that invoice and the invoice status may be automatically updated
// (e.g., from "issued" to "paid" if fully paid). Validates that the student exists,
// the invoice belongs to the student (if provided), and the invoice is not in
// draft or canceled status.
func (s *Service) Create(ctx context.Context, studentID int, invoiceID *int, amount float64, method string, paidAt string, note string) (*PaymentDTO, error) {
	if studentID <= 0 {
		return nil, errors.New("studentID must be > 0")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}
	if method != app.PaymentMethodCash && method != app.PaymentMethodBank {
		return nil, errors.New("method must be 'cash' or 'bank'")
	}
	t, err := parseDate(paidAt)
	if err != nil {
		return nil, fmt.Errorf("invalid paidAt: %w", err)
	}

	// Ensure student exists.
	if _, err := s.db.Student.Get(ctx, studentID); err != nil {
		return nil, err
	}

	// If invoiceID is provided, validate invoice ownership and status.
	if invoiceID != nil {
		iv, err := s.db.Invoice.Get(ctx, *invoiceID)
		if err != nil {
			return nil, err
		}
		if iv.StudentID != studentID {
			return nil, errors.New("invoice does not belong to student")
		}
		if iv.Status == app.InvoiceStatusDraft {
			return nil, errors.New("cannot attach payment to a draft invoice; issue it first")
		}
		if iv.Status == app.InvoiceStatusCanceled {
			return nil, errors.New("cannot attach payment to a canceled invoice")
		}
	}

	p, err := s.db.Payment.Create().
		SetStudentID(studentID).
		SetAmount(utils.Round2(amount)).
		SetMethod(payment.Method(method)).
		SetPaidAt(t).
		SetNote(note).
		SetNillableInvoiceID(invoiceID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Recompute invoice status if payment is linked to an invoice.
	if invoiceID != nil {
		if err := s.recomputeInvoiceStatus(ctx, *invoiceID); err != nil {
			return nil, err
		}
	}

	return toDTO(p), nil
}

// Delete removes a payment record. If the payment was linked to an invoice,
// the invoice status will be recomputed (e.g., from "paid" back to "issued"
// if the invoice is no longer fully paid).
func (s *Service) Delete(ctx context.Context, paymentID int) error {
	p, err := s.db.Payment.Get(ctx, paymentID)
	if err != nil {
		return err
	}
	var invID *int
	if p.InvoiceID != nil {
		id := *p.InvoiceID
		invID = &id
	}
	if err := s.db.Payment.DeleteOneID(paymentID).Exec(ctx); err != nil {
		return err
	}
	if invID != nil {
		return s.recomputeInvoiceStatus(ctx, *invID)
	}
	return nil
}

// ListForStudent returns all payments for a specific student,
// ordered by payment date (most recent first).
func (s *Service) ListForStudent(ctx context.Context, studentID int) ([]PaymentDTO, error) {
	ps, err := s.db.Payment.Query().
		Where(payment.StudentIDEQ(studentID)).
		Order(ent.Desc(payment.FieldPaidAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PaymentDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, *toDTO(p))
	}
	return out, nil
}

// InvoiceSummary calculates the payment summary for a specific invoice,
// including total amount, amount paid, remaining balance, and current status.
func (s *Service) InvoiceSummary(ctx context.Context, invoiceID int) (*InvoiceSummaryDTO, error) {
	iv, err := s.db.Invoice.Get(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	paid, err := s.sumPaymentsForInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	total := utils.Round2(iv.TotalAmount)
	paid = utils.Round2(paid)
	rem := utils.Round2(total - paid)
	if rem < 0 {
		rem = 0
	}
	return &InvoiceSummaryDTO{
		InvoiceID: invoiceID,
		Total:     total,
		Paid:      paid,
		Remaining: rem,
		Status:    string(iv.Status),
		Number:    iv.Number,
	}, nil
}

// StudentBalance calculates the financial balance for a student, including
// total invoiced amount (from issued and paid invoices), total paid amount
// (all payments), current balance, and debt (if any). The balance is calculated
// as paid - invoiced, so a negative balance means the student owes money.
func (s *Service) StudentBalance(ctx context.Context, studentID int) (*BalanceDTO, error) {
	st, err := s.db.Student.Get(ctx, studentID)
	if err != nil {
		return nil, err
	}

	invoiced, err := s.sumInvoicesForStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	paid, err := s.sumPaymentsForStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}

	invoiced = utils.Round2(invoiced)
	paid = utils.Round2(paid)
	bal := utils.Round2(paid - invoiced)
	debt := 0.0
	if bal < 0 {
		debt = utils.Round2(-bal)
	}

	return &BalanceDTO{
		StudentID:     studentID,
		StudentName:   st.FullName,
		TotalInvoiced: invoiced,
		TotalPaid:     paid,
		Balance:       bal,
		Debt:          debt,
	}, nil
}

// ListDebtors returns a list of all active students who have outstanding debt,
// sorted by debt amount (highest first). Only students with debt greater than
// the epsilon threshold (0.009) are included to account for floating-point
// rounding errors.
func (s *Service) ListDebtors(ctx context.Context) ([]DebtorDTO, error) {
	studs, err := s.db.Student.Query().
		Where(student.IsActiveEQ(true)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DebtorDTO, 0, len(studs))
	for _, st := range studs {
		b, err := s.StudentBalance(ctx, st.ID)
		if err != nil {
			log.Printf("failed to calculate balance for student %d (%s): %v", st.ID, st.FullName, err)
			continue
		}
		if b.Debt > eps() {
			out = append(out, DebtorDTO{
				StudentID: st.ID, StudentName: st.FullName,
				Debt: b.Debt, TotalInvoiced: b.TotalInvoiced, TotalPaid: b.TotalPaid,
			})
		}
	}
	// Sort by debt descending.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Debt > out[j].Debt
	})
	return out, nil
}

// QuickCash creates an unlinked cash payment for "lesson now".
// If you want exact price-per-student later, we can resolve it via Enrollment/Course and overrides.
// For this session we accept a direct amount from UI, but this helper exists for convenience.
func (s *Service) QuickCash(ctx context.Context, studentID int, amount float64, note string) (*PaymentDTO, error) {
	t := time.Now()
	return s.Create(ctx, studentID, nil, amount, app.PaymentMethodCash, t.Format("2006-01-02"), note)
}

// recomputeInvoiceStatus updates an invoice's status based on payment amounts.
// If the invoice is fully paid (within epsilon), status is set to "paid".
// If it was previously "paid" but is no longer fully paid, status reverts to "issued".
// Draft and canceled invoices are not modified.
func (s *Service) recomputeInvoiceStatus(ctx context.Context, invoiceID int) error {
	iv, err := s.db.Invoice.Get(ctx, invoiceID)
	if err != nil {
		return err
	}
	if iv.Status == app.InvoiceStatusCanceled {
		return nil
	}
	if iv.Status == app.InvoiceStatusDraft {
		return nil
	}

	paid, err := s.sumPaymentsForInvoice(ctx, invoiceID)
	if err != nil {
		return err
	}

	total := iv.TotalAmount
	if paid+eps() >= total {
		if iv.Status != app.InvoiceStatusPaid {
			_, err := iv.Update().SetStatus(app.InvoiceStatusPaid).Save(ctx)
			return err
		}
		return nil
	}

	// Not fully paid. If invoice was marked paid earlier, revert to issued.
	if iv.Status == app.InvoiceStatusPaid {
		_, err := iv.Update().SetStatus(app.InvoiceStatusIssued).Save(ctx)
		return err
	}
	return nil
}

// sumPaymentsForInvoice calculates the total amount of all payments
// linked to a specific invoice.
func (s *Service) sumPaymentsForInvoice(ctx context.Context, invoiceID int) (float64, error) {
	ps, err := s.db.Payment.Query().
		Where(payment.InvoiceIDEQ(invoiceID)).
		All(ctx)
	if err != nil {
		return 0, err
	}
	sum := 0.0
	for _, p := range ps {
		sum += p.Amount
	}
	return sum, nil
}

// sumPaymentsForStudent calculates the total amount of all payments
// made by a specific student (both linked and unlinked to invoices).
func (s *Service) sumPaymentsForStudent(ctx context.Context, studentID int) (float64, error) {
	ps, err := s.db.Payment.Query().
		Where(payment.StudentIDEQ(studentID)).
		All(ctx)
	if err != nil {
		return 0, err
	}
	sum := 0.0
	for _, p := range ps {
		sum += p.Amount
	}
	return sum, nil
}

// sumInvoicesForStudent calculates the total amount of all invoices
// for a student that are in "issued" or "paid" status. Draft and canceled
// invoices are not included in the calculation.
func (s *Service) sumInvoicesForStudent(ctx context.Context, studentID int) (float64, error) {
	invs, err := s.db.Invoice.Query().
		Where(
			invoice.StudentIDEQ(studentID),
			invoice.StatusIn(app.InvoiceStatusIssued, app.InvoiceStatusPaid),
		).
		All(ctx)
	if err != nil {
		return 0, err
	}
	sum := 0.0
	for _, iv := range invs {
		sum += iv.TotalAmount
	}
	return sum, nil
}

// toDTO converts an ent.Payment entity to a PaymentDTO for frontend consumption.
func toDTO(p *ent.Payment) *PaymentDTO {
	var invID *int
	if p.InvoiceID != nil {
		id := *p.InvoiceID
		invID = &id
	}
	return &PaymentDTO{
		ID:        p.ID,
		StudentID: p.StudentID,
		InvoiceID: invID,
		PaidAt:    p.PaidAt.Format(time.RFC3339),
		Amount:    utils.Round2(p.Amount),
		Method:    string(p.Method),
		Note:      p.Note,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
	}
}
