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
	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/payment"
	"langschool/ent/student"
	"langschool/internal/app"
	"langschool/internal/money"
)

// Service provides payment processing and financial tracking functionality.
type Service struct{ db *ent.Client }

// New creates a new payment service with the given database client.
func New(db *ent.Client) *Service { return &Service{db: db} }

// PaymentDTO represents a payment record in the frontend.
type PaymentDTO struct {
	ID        int     `json:"id"`                  // Payment ID
	StudentID int     `json:"studentId"`           // ID of the student who made the payment
	InvoiceID *int    `json:"invoiceId,omitempty"` // Optional: ID of invoice this payment is for
	PaidAt    string  `json:"paidAt"`              // Payment date in RFC3339 format
	Amount    float64 `json:"amount"`              // Payment amount
	Method    string  `json:"method"`              // Payment method: "cash" or "bank"
	Note      string  `json:"note"`                // Optional notes about the payment
	CreatedAt string  `json:"createdAt"`           // Record creation date in RFC3339 format
}

// BalanceDTO represents a student's financial balance.
type BalanceDTO struct {
	StudentID     int     `json:"studentId"`     // Student ID
	StudentName   string  `json:"studentName"`   // Student's full name
	TotalInvoiced float64 `json:"totalInvoiced"` // Total amount invoiced (issued + paid invoices)
	TotalPaid     float64 `json:"totalPaid"`     // Total amount paid (all payments)
	Balance       float64 `json:"balance"`       // Balance: paid - invoiced (negative => student owes)
	Debt          float64 `json:"debt"`          // Debt: max(0, -balance), amount student owes
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
	InvoiceID int     `json:"invoiceId"`        // Invoice ID
	Total     float64 `json:"total"`            // Total invoice amount
	Paid      float64 `json:"paid"`             // Amount paid so far
	Remaining float64 `json:"remaining"`        // Remaining amount to pay
	Status    string  `json:"status"`           // Invoice status
	Number    *string `json:"number,omitempty"` // Invoice number
}

// DebtInvoiceDTO represents one open invoice in a student's debt breakdown.
type DebtInvoiceDTO struct {
	InvoiceID int     `json:"invoiceId"`
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	Number    *string `json:"number,omitempty"`
	Total     float64 `json:"total"`
	Paid      float64 `json:"paid"`
	Remaining float64 `json:"remaining"`
	Status    string  `json:"status"`
}

// MonthOverviewDTO represents a read-only monthly dashboard snapshot.
type MonthOverviewDTO struct {
	Year  int `json:"year"`
	Month int `json:"month"`

	ActiveStudents int `json:"activeStudents"`
	ActiveCourses  int `json:"activeCourses"`
	Enrollments    int `json:"enrollments"`

	PerLessonEnrollments int `json:"perLessonEnrollments"`
	AttendanceFilled     int `json:"attendanceFilled"`
	AttendanceMissing    int `json:"attendanceMissing"`

	DraftInvoices   int `json:"draftInvoices"`
	IssuedInvoices  int `json:"issuedInvoices"`
	PaidInvoices    int `json:"paidInvoices"`
	OverdueInvoices int `json:"overdueInvoicesCount"`

	TotalIssued            float64 `json:"totalIssued"`
	TotalPaid              float64 `json:"totalPaid"`
	PaymentsMonthTotal     float64 `json:"paymentsMonthTotal"`
	PaymentsMonthCashTotal float64 `json:"paymentsMonthCashTotal"`
	PaymentsMonthBankTotal float64 `json:"paymentsMonthBankTotal"`
	UnlinkedCreditTotal    float64 `json:"unlinkedCreditTotal"`
	MonthDebtTotal         float64 `json:"monthDebtTotal"`
	HistoricalDebtTotal    float64 `json:"historicalDebtTotal"`
	ActionQueueCount       int     `json:"actionQueueCount"`

	DebtorsCount int     `json:"debtorsCount"`
	TotalDebt    float64 `json:"totalDebt"`
}

// RecentPaymentDTO represents a recent payment for dashboard activity feeds.
type RecentPaymentDTO struct {
	ID          int     `json:"id"`
	StudentID   int     `json:"studentId"`
	StudentName string  `json:"studentName"`
	InvoiceID   *int    `json:"invoiceId,omitempty"`
	Amount      float64 `json:"amount"`
	Method      string  `json:"method"`
	PaidAt      string  `json:"paidAt"`
	Note        string  `json:"note"`
}

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
	tx, err := s.db.Tx(ctx)
	if err != nil {
		if err == ent.ErrTxStarted {
			return s.createInStore(ctx, studentID, invoiceID, amount, method, paidAt, note)
		}
		return nil, err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	dto, err := (&Service{db: tx.Client()}).createInStore(ctx, studentID, invoiceID, amount, method, paidAt, note)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return dto, nil
}

func (s *Service) createInStore(ctx context.Context, studentID int, invoiceID *int, amount float64, method string, paidAt string, note string) (*PaymentDTO, error) {
	amountCents := money.EurosToCents(amount)
	if studentID <= 0 {
		return nil, errors.New("studentID должен быть больше 0")
	}
	if amountCents <= 0 {
		return nil, errors.New("сумма должна быть больше 0")
	}
	if method != app.PaymentMethodCash && method != app.PaymentMethodBank {
		return nil, errors.New("способ оплаты должен быть 'cash' или 'bank'")
	}
	t, err := parseDate(paidAt)
	if err != nil {
		return nil, fmt.Errorf("некорректная дата оплаты paidAt: %w", err)
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
			return nil, errors.New("счёт не принадлежит этому ученику")
		}
		if iv.Status == app.InvoiceStatusDraft {
			return nil, errors.New("нельзя привязать оплату к черновику счёта; сначала выставьте счёт")
		}
		if iv.Status == app.InvoiceStatusCanceled {
			return nil, errors.New("нельзя привязать оплату к отменённому счёту")
		}
	}

	// Global debtor payments should reduce the oldest open invoices first so
	// invoice-level "Paid" values and statuses stay in sync with the debt view.
	if invoiceID == nil {
		return s.allocateToOldestInvoices(ctx, studentID, amountCents, method, t, note)
	}

	p, err := s.db.Payment.Create().
		SetStudentID(studentID).
		SetAmountCents(amountCents).
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

func (s *Service) allocateToOldestInvoices(ctx context.Context, studentID int, amountCents int64, method string, paidAt time.Time, note string) (*PaymentDTO, error) {
	remaining := amountCents

	invoices, err := s.db.Invoice.Query().
		Where(
			invoice.StudentIDEQ(studentID),
			invoice.StatusIn(app.InvoiceStatusIssued, app.InvoiceStatusPaid),
		).
		Order(
			ent.Asc(invoice.FieldPeriodYear),
			ent.Asc(invoice.FieldPeriodMonth),
			ent.Asc(invoice.FieldID),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var firstCreated *ent.Payment

	for _, iv := range invoices {
		if remaining <= 0 {
			break
		}

		_, _, invoiceRemaining, err := s.invoiceBalanceCents(ctx, iv)
		if err != nil {
			return nil, err
		}
		if invoiceRemaining <= 0 {
			continue
		}

		applied := invoiceRemaining
		if remaining < applied {
			applied = remaining
		}

		invoiceID := iv.ID
		p, err := s.db.Payment.Create().
			SetStudentID(studentID).
			SetAmountCents(applied).
			SetMethod(payment.Method(method)).
			SetPaidAt(paidAt).
			SetNote(note).
			SetNillableInvoiceID(&invoiceID).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		if firstCreated == nil {
			firstCreated = p
		}

		if err := s.recomputeInvoiceStatus(ctx, iv.ID); err != nil {
			return nil, err
		}

		remaining -= applied
	}

	// If payment is larger than the current debt, keep the extra part as student credit.
	if remaining > 0 || firstCreated == nil {
		creditAmount := remaining
		if creditAmount <= 0 {
			creditAmount = amountCents
		}
		p, err := s.db.Payment.Create().
			SetStudentID(studentID).
			SetAmountCents(creditAmount).
			SetMethod(payment.Method(method)).
			SetPaidAt(paidAt).
			SetNote(note).
			Save(ctx)
		if err != nil {
			return nil, err
		}
		if firstCreated == nil {
			firstCreated = p
		}
	}

	return toDTO(firstCreated), nil
}

// ApplyCreditToOldestInvoices finds all unlinked payments (student credit) for
// the given student and applies them to open invoices, oldest-first.
// If credit partially covers an invoice, the invoice remains "issued" with reduced remaining.
// If credit fully covers an invoice, the invoice becomes "paid".
// Remaining credit after all invoices are covered stays as unlinked payments.
func (s *Service) ApplyCreditToOldestInvoices(ctx context.Context, studentID int) error {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		if err == ent.ErrTxStarted {
			return s.applyCreditToOldestInvoicesInStore(ctx, studentID)
		}
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := (&Service{db: tx.Client()}).applyCreditToOldestInvoicesInStore(ctx, studentID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *Service) applyCreditToOldestInvoicesInStore(ctx context.Context, studentID int) error {
	// Find all unlinked credit payments for this student, ordered oldest first.
	credits, err := s.db.Payment.Query().
		Where(
			payment.StudentIDEQ(studentID),
			payment.InvoiceIDIsNil(),
		).
		Order(ent.Asc(payment.FieldPaidAt), ent.Asc(payment.FieldID)).
		All(ctx)
	if err != nil {
		return err
	}
	if len(credits) == 0 {
		return nil
	}

	// Find open invoices for this student, oldest first.
	invoices, err := s.db.Invoice.Query().
		Where(
			invoice.StudentIDEQ(studentID),
			invoice.StatusIn(app.InvoiceStatusIssued, app.InvoiceStatusPaid),
		).
		Order(
			ent.Asc(invoice.FieldPeriodYear),
			ent.Asc(invoice.FieldPeriodMonth),
			ent.Asc(invoice.FieldID),
		).
		All(ctx)
	if err != nil {
		return err
	}

	for _, iv := range invoices {
		if len(credits) == 0 {
			break
		}

		_, _, invoiceRemaining, err := s.invoiceBalanceCents(ctx, iv)
		if err != nil {
			return err
		}
		if invoiceRemaining <= 0 {
			continue
		}

		toApply := invoiceRemaining

		for len(credits) > 0 && toApply > 0 {
			cr := credits[0]
			creditRemaining := cr.AmountCents

			applied := creditRemaining
			if toApply < applied {
				applied = toApply
			}

			invoiceID := iv.ID
			note := cr.Note
			if note != "" {
				note = fmt.Sprintf("%s (applied from credit)", note)
			} else {
				note = "Applied from student credit"
			}

			// Create new linked payment for the applied amount.
			if _, err := s.db.Payment.Create().
				SetStudentID(studentID).
				SetAmountCents(applied).
				SetMethod(cr.Method).
				SetPaidAt(cr.PaidAt).
				SetNote(note).
				SetNillableInvoiceID(&invoiceID).
				Save(ctx); err != nil {
				return err
			}

			// Reduce or delete original credit payment.
			newCreditAmount := creditRemaining - applied
			if newCreditAmount <= 0 {
				if err := s.db.Payment.DeleteOneID(cr.ID).Exec(ctx); err != nil {
					return err
				}
				credits = credits[1:]
			} else {
				updated, err := cr.Update().SetAmountCents(newCreditAmount).Save(ctx)
				if err != nil {
					return err
				}
				credits[0] = updated
			}

			toApply -= applied
		}

		if err := s.recomputeInvoiceStatus(ctx, iv.ID); err != nil {
			return err
		}
	}

	return nil
}

// Delete removes a payment record. If the payment was linked to an invoice,
// the invoice status will be recomputed (e.g., from "paid" back to "issued"
// if the invoice is no longer fully paid).
func (s *Service) Delete(ctx context.Context, paymentID int) error {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		if err == ent.ErrTxStarted {
			return s.deleteInStore(ctx, paymentID)
		}
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := (&Service{db: tx.Client()}).deleteInStore(ctx, paymentID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *Service) deleteInStore(ctx context.Context, paymentID int) error {
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

// ListRecent returns the most recent payments across all students for dashboard feeds.
func (s *Service) ListRecent(ctx context.Context, limit int) ([]RecentPaymentDTO, error) {
	if limit <= 0 {
		limit = 8
	}

	ps, err := s.db.Payment.Query().
		WithStudent().
		Order(ent.Desc(payment.FieldPaidAt), ent.Desc(payment.FieldID)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]RecentPaymentDTO, 0, len(ps))
	for _, p := range ps {
		studentName := ""
		if p.Edges.Student != nil {
			studentName = p.Edges.Student.FullName
		}
		var invoiceID *int
		if p.InvoiceID != nil {
			id := *p.InvoiceID
			invoiceID = &id
		}
		out = append(out, RecentPaymentDTO{
			ID:          p.ID,
			StudentID:   p.StudentID,
			StudentName: studentName,
			InvoiceID:   invoiceID,
			Amount:      money.CentsToEuros(p.AmountCents),
			Method:      string(p.Method),
			PaidAt:      p.PaidAt.Format(time.RFC3339),
			Note:        p.Note,
		})
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
	total, paid, remaining, err := s.invoiceBalanceCents(ctx, iv)
	if err != nil {
		return nil, err
	}
	return &InvoiceSummaryDTO{
		InvoiceID: invoiceID,
		Total:     money.CentsToEuros(total),
		Paid:      money.CentsToEuros(paid),
		Remaining: money.CentsToEuros(remaining),
		Status:    string(iv.Status),
		Number:    iv.Number,
	}, nil
}

// StudentDebtDetails returns open invoice balances for a student, oldest first.
// It is read-only and includes only issued/paid invoices with remaining debt.
func (s *Service) StudentDebtDetails(ctx context.Context, studentID int) ([]DebtInvoiceDTO, error) {
	invs, err := s.db.Invoice.Query().
		Where(
			invoice.StudentIDEQ(studentID),
			invoice.StatusIn(app.InvoiceStatusIssued, app.InvoiceStatusPaid),
		).
		Order(
			ent.Asc(invoice.FieldPeriodYear),
			ent.Asc(invoice.FieldPeriodMonth),
			ent.Asc(invoice.FieldID),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]DebtInvoiceDTO, 0, len(invs))
	for _, iv := range invs {
		total, paid, remaining, err := s.invoiceBalanceCents(ctx, iv)
		if err != nil {
			return nil, err
		}
		if remaining <= 0 {
			continue
		}

		out = append(out, DebtInvoiceDTO{
			InvoiceID: iv.ID,
			Year:      iv.PeriodYear,
			Month:     iv.PeriodMonth,
			Number:    iv.Number,
			Total:     money.CentsToEuros(total),
			Paid:      money.CentsToEuros(paid),
			Remaining: money.CentsToEuros(remaining),
			Status:    string(iv.Status),
		})
	}

	return out, nil
}

// MonthOverview returns a read-only monthly dashboard snapshot.
func (s *Service) MonthOverview(ctx context.Context, year, month int) (*MonthOverviewDTO, error) {
	activeStudents, err := s.db.Student.Query().
		Where(student.IsActiveEQ(true)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	activeCourses, err := s.db.Course.Query().
		Where(course.IsActiveEQ(true)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	totalEnrollments, err := s.db.Enrollment.Query().
		Where(
			enrollment.HasStudentWith(student.IsActiveEQ(true)),
			enrollment.HasCourseWith(course.IsActiveEQ(true)),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	perLessonEnrollments, err := s.db.Enrollment.Query().
		Where(
			enrollment.BillingModeEQ(enrollment.BillingModePerLesson),
			enrollment.HasStudentWith(student.IsActiveEQ(true)),
			enrollment.HasCourseWith(course.IsActiveEQ(true)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	perLessonKeys := make(map[string]struct{}, len(perLessonEnrollments))
	for _, enr := range perLessonEnrollments {
		perLessonKeys[overviewEnrollmentKey(enr.StudentID, enr.CourseID)] = struct{}{}
	}

	filledRows, err := s.db.AttendanceMonth.Query().
		Where(
			attendancemonth.YearEQ(year),
			attendancemonth.MonthEQ(month),
			attendancemonth.HoursGT(0),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	attendanceFilled := 0
	seenAttendance := make(map[string]struct{}, len(filledRows))
	for _, row := range filledRows {
		key := overviewEnrollmentKey(row.StudentID, row.CourseID)
		if _, ok := perLessonKeys[key]; !ok {
			continue
		}
		if _, ok := seenAttendance[key]; ok {
			continue
		}
		seenAttendance[key] = struct{}{}
		attendanceFilled++
	}

	attendanceMissing := len(perLessonEnrollments) - attendanceFilled
	if attendanceMissing < 0 {
		attendanceMissing = 0
	}

	monthInvoices, err := s.db.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(year),
			invoice.PeriodMonthEQ(month),
			invoice.HasStudentWith(student.IsActiveEQ(true)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	monthInvoiceIDs := make([]int, 0, len(monthInvoices))
	draftInvoices := 0
	issuedInvoices := 0
	paidInvoices := 0
	var totalIssuedCents int64
	var monthDebtCents int64
	debtStudentIDs := make(map[int]struct{})

	for _, iv := range monthInvoices {
		monthInvoiceIDs = append(monthInvoiceIDs, iv.ID)
		switch iv.Status {
		case app.InvoiceStatusDraft:
			draftInvoices++
		case app.InvoiceStatusIssued:
			issuedInvoices++
			totalIssuedCents += iv.TotalAmountCents
			_, _, remaining, err := s.invoiceBalanceCents(ctx, iv)
			if err != nil {
				return nil, err
			}
			if remaining > 0 {
				monthDebtCents += remaining
				debtStudentIDs[iv.StudentID] = struct{}{}
			}
		case app.InvoiceStatusPaid:
			paidInvoices++
			totalIssuedCents += iv.TotalAmountCents
			_, _, remaining, err := s.invoiceBalanceCents(ctx, iv)
			if err != nil {
				return nil, err
			}
			if remaining > 0 {
				monthDebtCents += remaining
				debtStudentIDs[iv.StudentID] = struct{}{}
			}
		}
	}

	var totalPaidCents int64
	if len(monthInvoiceIDs) > 0 {
		linkedPayments, err := s.db.Payment.Query().
			Where(payment.InvoiceIDIn(monthInvoiceIDs...)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range linkedPayments {
			totalPaidCents += p.AmountCents
		}
	}

	historicalInvoices, err := s.db.Invoice.Query().
		Where(
			invoice.HasStudentWith(student.IsActiveEQ(true)),
			invoice.StatusIn(app.InvoiceStatusIssued, app.InvoiceStatusPaid),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var historicalDebtCents int64
	overdueInvoices := 0
	for _, iv := range historicalInvoices {
		if !isBeforePeriod(iv.PeriodYear, iv.PeriodMonth, year, month) {
			continue
		}
		_, _, remaining, err := s.invoiceBalanceCents(ctx, iv)
		if err != nil {
			return nil, err
		}
		if remaining <= 0 {
			continue
		}
		historicalDebtCents += remaining
		overdueInvoices++
		debtStudentIDs[iv.StudentID] = struct{}{}
	}

	rangeStart, rangeEnd := monthBounds(year, month)
	monthPayments, err := s.db.Payment.Query().
		Where(
			payment.HasStudentWith(student.IsActiveEQ(true)),
			payment.PaidAtGTE(rangeStart),
			payment.PaidAtLT(rangeEnd),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var paymentsMonthCents int64
	var paymentsMonthCashCents int64
	var paymentsMonthBankCents int64
	for _, p := range monthPayments {
		paymentsMonthCents += p.AmountCents
		switch p.Method {
		case payment.MethodCash:
			paymentsMonthCashCents += p.AmountCents
		case payment.MethodBank:
			paymentsMonthBankCents += p.AmountCents
		}
	}

	credits, err := s.db.Payment.Query().
		Where(
			payment.InvoiceIDIsNil(),
			payment.HasStudentWith(student.IsActiveEQ(true)),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var unlinkedCreditCents int64
	for _, credit := range credits {
		unlinkedCreditCents += credit.AmountCents
	}

	debtorsCount := len(debtStudentIDs)
	totalDebtCents := monthDebtCents + historicalDebtCents
	actionQueueCount := debtorsCount + draftInvoices + attendanceMissing

	return &MonthOverviewDTO{
		Year:  year,
		Month: month,

		ActiveStudents: activeStudents,
		ActiveCourses:  activeCourses,
		Enrollments:    totalEnrollments,

		PerLessonEnrollments: len(perLessonEnrollments),
		AttendanceFilled:     attendanceFilled,
		AttendanceMissing:    attendanceMissing,

		DraftInvoices:   draftInvoices,
		IssuedInvoices:  issuedInvoices,
		PaidInvoices:    paidInvoices,
		OverdueInvoices: overdueInvoices,

		TotalIssued:            money.CentsToEuros(totalIssuedCents),
		TotalPaid:              money.CentsToEuros(totalPaidCents),
		PaymentsMonthTotal:     money.CentsToEuros(paymentsMonthCents),
		PaymentsMonthCashTotal: money.CentsToEuros(paymentsMonthCashCents),
		PaymentsMonthBankTotal: money.CentsToEuros(paymentsMonthBankCents),
		UnlinkedCreditTotal:    money.CentsToEuros(unlinkedCreditCents),
		MonthDebtTotal:         money.CentsToEuros(monthDebtCents),
		HistoricalDebtTotal:    money.CentsToEuros(historicalDebtCents),
		ActionQueueCount:       actionQueueCount,

		DebtorsCount: debtorsCount,
		TotalDebt:    money.CentsToEuros(totalDebtCents),
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

	invoicedCents, err := s.sumInvoicesForStudentCents(ctx, studentID)
	if err != nil {
		return nil, err
	}
	paidCents, err := s.sumPaymentsForStudentCents(ctx, studentID)
	if err != nil {
		return nil, err
	}

	invoiced := money.CentsToEuros(invoicedCents)
	paid := money.CentsToEuros(paidCents)
	balanceCents := paidCents - invoicedCents
	bal := money.CentsToEuros(balanceCents)
	debt := 0.0
	if balanceCents < 0 {
		debt = money.CentsToEuros(-balanceCents)
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
		if b.Debt > 0 {
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
	if _, err := s.db.Student.Get(ctx, studentID); err != nil {
		return nil, err
	}

	p, err := s.db.Payment.Create().
		SetStudentID(studentID).
		SetAmountCents(money.EurosToCents(amount)).
		SetMethod(payment.Method(app.PaymentMethodCash)).
		SetPaidAt(t).
		SetNote(note).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return toDTO(p), nil
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

	total, paid, remaining, err := s.invoiceBalanceCents(ctx, iv)
	if err != nil {
		return err
	}

	if paid >= total || remaining <= 0 {
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

func (s *Service) invoiceBalanceCents(ctx context.Context, iv *ent.Invoice) (total, paid, remaining int64, err error) {
	paid, err = s.sumPaymentsForInvoiceCents(ctx, iv.ID)
	if err != nil {
		return 0, 0, 0, err
	}

	total = iv.TotalAmountCents
	remaining = total - paid
	if remaining < 0 {
		remaining = 0
	}

	return total, paid, remaining, nil
}

func (s *Service) invoiceBalance(ctx context.Context, iv *ent.Invoice) (total, paid, remaining float64, err error) {
	totalCents, paidCents, remainingCents, err := s.invoiceBalanceCents(ctx, iv)
	if err != nil {
		return 0, 0, 0, err
	}
	return money.CentsToEuros(totalCents), money.CentsToEuros(paidCents), money.CentsToEuros(remainingCents), nil
}

// sumPaymentsForInvoice calculates the total amount of all payments
// linked to a specific invoice.
func (s *Service) sumPaymentsForInvoiceCents(ctx context.Context, invoiceID int) (int64, error) {
	ps, err := s.db.Payment.Query().
		Where(payment.InvoiceIDEQ(invoiceID)).
		All(ctx)
	if err != nil {
		return 0, err
	}
	var sum int64
	for _, p := range ps {
		sum += p.AmountCents
	}
	return sum, nil
}

// sumPaymentsForStudent calculates the total amount of all payments
// made by a specific student (both linked and unlinked to invoices).
func (s *Service) sumPaymentsForStudentCents(ctx context.Context, studentID int) (int64, error) {
	ps, err := s.db.Payment.Query().
		Where(payment.StudentIDEQ(studentID)).
		All(ctx)
	if err != nil {
		return 0, err
	}
	var sum int64
	for _, p := range ps {
		sum += p.AmountCents
	}
	return sum, nil
}

// sumInvoicesForStudent calculates the total amount of all invoices
// for a student that are in "issued" or "paid" status. Draft and canceled
// invoices are not included in the calculation.
func (s *Service) sumInvoicesForStudentCents(ctx context.Context, studentID int) (int64, error) {
	invs, err := s.db.Invoice.Query().
		Where(
			invoice.StudentIDEQ(studentID),
			invoice.StatusIn(app.InvoiceStatusIssued, app.InvoiceStatusPaid),
		).
		All(ctx)
	if err != nil {
		return 0, err
	}
	var sum int64
	for _, iv := range invs {
		sum += iv.TotalAmountCents
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
		Amount:    money.CentsToEuros(p.AmountCents),
		Method:    string(p.Method),
		Note:      p.Note,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
	}
}

func overviewEnrollmentKey(studentID, courseID int) string {
	return fmt.Sprintf("%d:%d", studentID, courseID)
}

func isBeforePeriod(year, month, targetYear, targetMonth int) bool {
	if year != targetYear {
		return year < targetYear
	}
	return month < targetMonth
}

func monthBounds(year, month int) (time.Time, time.Time) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}
