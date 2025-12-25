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

type Service struct{ db *ent.Client }

func New(db *ent.Client) *Service { return &Service{db: db} }

type PaymentDTO struct {
	ID        int     `json:"id"`
	StudentID int     `json:"studentId"`
	InvoiceID *int    `json:"invoiceId,omitempty"`
	PaidAt    string  `json:"paidAt"` // RFC3339
	Amount    float64 `json:"amount"`
	Method    string  `json:"method"` // cash|bank
	Note      string  `json:"note"`
	CreatedAt string  `json:"createdAt"` // RFC3339
}

type BalanceDTO struct {
	StudentID     int     `json:"studentId"`
	StudentName   string  `json:"studentName"`
	TotalInvoiced float64 `json:"totalInvoiced"` // issued+paid
	TotalPaid     float64 `json:"totalPaid"`     // all payments
	Balance       float64 `json:"balance"`       // paid - invoiced (negative => owes)
	Debt          float64 `json:"debt"`          // max(0, -balance)
}

type DebtorDTO struct {
	StudentID     int     `json:"studentId"`
	StudentName   string  `json:"studentName"`
	Debt          float64 `json:"debt"`
	TotalInvoiced float64 `json:"totalInvoiced"`
	TotalPaid     float64 `json:"totalPaid"`
}

type InvoiceSummaryDTO struct {
	InvoiceID int     `json:"invoiceId"`
	Total     float64 `json:"total"`
	Paid      float64 `json:"paid"`
	Remaining float64 `json:"remaining"`
	Status    string  `json:"status"`
	Number    *string `json:"number,omitempty"`
}


// eps returns the epsilon value used for floating-point comparisons.
// The value 0.009 is chosen to account for rounding errors in currency calculations
// where amounts are rounded to 2 decimal places (0.01). This epsilon is slightly
// smaller than 0.01 to allow for safe boundary checks without false positives,
// while being large enough to handle typical floating-point precision issues.
func eps() float64 { return 0.009 }

func parseDate(s string) (time.Time, error) {
	// Accept YYYY-MM-DD or RFC3339.
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

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
