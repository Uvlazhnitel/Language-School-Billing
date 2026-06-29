package backend

import (
	"context"
	"fmt"

	auditsvc "langschool/internal/app/audit"
)

func (s *Service) PaymentCreate(ctx context.Context, studentID int, invoiceID *int, amount float64, method string, paidAt string, note string) (*PaymentDTO, error) {
	before, _, err := s.auditStudentFinanceSnapshot(ctx, studentID)
	if err != nil {
		return nil, err
	}
	item, err := s.rt.Payment.Create(ctx, studentID, invoiceID, amount, method, paidAt, note)
	if err != nil {
		return nil, err
	}
	after, studentMeta, err := s.auditStudentFinanceSnapshot(ctx, studentID)
	if err == nil {
		action := "payment.create"
		if invoiceID == nil {
			action = "payment.allocate_or_credit"
		}
		summary := fmt.Sprintf("Recorded payment of %.2f for %s", item.Amount, studentMeta.StudentName)
		if item.InvoiceID != nil {
			summary = fmt.Sprintf("%s on invoice %d", summary, *item.InvoiceID)
		} else {
			summary = fmt.Sprintf("%s as student credit", summary)
		}
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "payment",
			EntityID:   intPtr(item.ID),
			Action:     action,
			Summary:    summary,
			Before:     before,
			After:      after,
			StudentID:  intPtr(studentID),
			InvoiceID:  item.InvoiceID,
		})
	}
	return item, nil
}

func (s *Service) PaymentDelete(ctx context.Context, paymentID int) error {
	before, meta, err := s.auditPaymentDeleteSnapshot(ctx, paymentID)
	if err != nil {
		return err
	}
	if err := s.rt.Payment.Delete(ctx, paymentID); err != nil {
		return err
	}
	after, _, err := s.auditStudentFinanceSnapshot(ctx, meta.StudentID)
	if err == nil {
		s.recordAudit(ctx, auditsvc.RecordEvent{
			EntityType: "payment",
			EntityID:   intPtr(paymentID),
			Action:     "payment.delete",
			Summary:    fmt.Sprintf("Deleted payment %d for %s, amount %.2f", paymentID, meta.StudentName, meta.Amount),
			Before:     before,
			After:      after,
			StudentID:  intPtr(meta.StudentID),
			InvoiceID:  meta.InvoiceID,
		})
	}
	return nil
}

func (s *Service) PaymentListForStudent(ctx context.Context, studentID int) ([]PaymentDTO, error) {
	return s.rt.Payment.ListForStudent(ctx, studentID)
}

func (s *Service) StudentBalance(ctx context.Context, studentID int) (*BalanceDTO, error) {
	return s.rt.Payment.StudentBalance(ctx, studentID)
}

func (s *Service) DebtorsList(ctx context.Context) ([]DebtorDTO, error) {
	return s.rt.Payment.ListDebtors(ctx)
}

func (s *Service) MonthOverview(ctx context.Context, year, month int) (*MonthOverviewDTO, error) {
	return s.rt.Payment.MonthOverview(ctx, year, month)
}

func (s *Service) RecentPayments(ctx context.Context, limit int) ([]RecentPaymentDTO, error) {
	return s.rt.Payment.ListRecent(ctx, limit)
}

func (s *Service) StudentDebtDetails(ctx context.Context, studentID int) ([]DebtInvoiceDTO, error) {
	return s.rt.Payment.StudentDebtDetails(ctx, studentID)
}

func (s *Service) InvoicePaymentSummary(ctx context.Context, invoiceID int) (*InvoiceSummaryDTO, error) {
	return s.rt.Payment.InvoiceSummary(ctx, invoiceID)
}

func (s *Service) PaymentQuickCash(ctx context.Context, studentID int, amount float64, note string) (*PaymentDTO, error) {
	return s.rt.Payment.QuickCash(ctx, studentID, amount, note)
}
