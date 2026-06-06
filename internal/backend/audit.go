package backend

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"langschool/ent"
	"langschool/ent/invoice"
	auditsvc "langschool/internal/app/audit"
	"langschool/internal/auth"
)

type AuditLogListItem = auditsvc.ListItem
type AuditLogListResult = auditsvc.ListResult
type AuditLogListFilter = auditsvc.ListFilter

func (s *Service) AuditLogList(ctx context.Context, filter AuditLogListFilter) (*AuditLogListResult, error) {
	return s.rt.Audit.List(ctx, auditsvc.ListFilter(filter))
}

type auditInvoiceMeta struct {
	StudentID   *int
	StudentName string
	Year        int
	Month       int
	Number      *string
}

type auditPaymentMeta struct {
	StudentID   int
	StudentName string
	Amount      float64
	InvoiceID   *int
}

type backendStudentRef struct {
	StudentID   int
	StudentName string
}

func (s *Service) recordAudit(ctx context.Context, event auditsvc.RecordEvent) {
	if s == nil || s.rt == nil || s.rt.Audit == nil {
		return
	}
	actor := actorFromContext(ctx)
	if actor != nil {
		event.ActorUserID = intPtr(actor.ID)
		event.ActorLabel = actor.Username
	}
	if strings.TrimSpace(event.ActorLabel) == "" {
		event.ActorLabel = "system"
	}
	if err := s.rt.Audit.Record(ctx, event); err != nil {
		log.Printf("audit record failed: %v", err)
	}
}

func (s *Service) auditInvoiceSnapshot(ctx context.Context, invoiceID int) (map[string]any, auditInvoiceMeta, error) {
	dto, err := s.rt.Invoice.Get(ctx, invoiceID)
	if err != nil {
		return nil, auditInvoiceMeta{}, err
	}
	summary, err := s.rt.Payment.InvoiceSummary(ctx, invoiceID)
	if err != nil {
		return nil, auditInvoiceMeta{}, err
	}
	return map[string]any{
			"invoice": dto,
			"payment": summary,
		}, auditInvoiceMeta{
			StudentID:   intPtr(dto.StudentID),
			StudentName: dto.StudentName,
			Year:        dto.Year,
			Month:       dto.Month,
			Number:      dto.Number,
		}, nil
}

func (s *Service) auditStudentFinanceSnapshotByInvoice(ctx context.Context, invoiceID int) (map[string]any, auditInvoiceMeta, error) {
	invoiceSnapshot, meta, err := s.auditInvoiceSnapshot(ctx, invoiceID)
	if err != nil {
		return nil, meta, err
	}
	if meta.StudentID == nil {
		return invoiceSnapshot, meta, nil
	}
	financeSnapshot, _, err := s.auditStudentFinanceSnapshot(ctx, *meta.StudentID)
	if err != nil {
		return invoiceSnapshot, meta, nil
	}
	return financeSnapshot, meta, nil
}

func (s *Service) auditStudentFinanceSnapshot(ctx context.Context, studentID int) (map[string]any, backendStudentRef, error) {
	studentItem, err := s.rt.DB.Ent.Student.Get(ctx, studentID)
	if err != nil {
		return nil, backendStudentRef{}, err
	}
	payments, err := s.rt.Payment.ListForStudent(ctx, studentID)
	if err != nil {
		return nil, backendStudentRef{}, err
	}
	debts, err := s.rt.Payment.StudentDebtDetails(ctx, studentID)
	if err != nil {
		return nil, backendStudentRef{}, err
	}
	balance, err := s.rt.Payment.StudentBalance(ctx, studentID)
	if err != nil {
		return nil, backendStudentRef{}, err
	}
	invoices, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.StudentIDEQ(studentID)).
		Order(ent.Asc(invoice.FieldPeriodYear), ent.Asc(invoice.FieldPeriodMonth), ent.Asc(invoice.FieldID)).
		All(ctx)
	if err != nil {
		return nil, backendStudentRef{}, err
	}
	invoiceSnapshots := make([]map[string]any, 0, len(invoices))
	for _, item := range invoices {
		summary, summaryErr := s.rt.Payment.InvoiceSummary(ctx, item.ID)
		if summaryErr != nil {
			return nil, backendStudentRef{}, summaryErr
		}
		number := ""
		if item.Number != nil {
			number = *item.Number
		}
		invoiceSnapshots = append(invoiceSnapshots, map[string]any{
			"id":      item.ID,
			"year":    item.PeriodYear,
			"month":   item.PeriodMonth,
			"number":  number,
			"status":  string(item.Status),
			"total":   item.TotalAmount,
			"summary": summary,
		})
	}
	return map[string]any{
		"student": map[string]any{
			"id":       studentItem.ID,
			"fullName": studentItem.FullName,
		},
		"balance":  balance,
		"debts":    debts,
		"payments": payments,
		"invoices": invoiceSnapshots,
	}, backendStudentRef{StudentID: studentItem.ID, StudentName: studentItem.FullName}, nil
}

func (s *Service) auditPaymentDeleteSnapshot(ctx context.Context, paymentID int) (map[string]any, auditPaymentMeta, error) {
	paymentItem, err := s.rt.DB.Ent.Payment.Get(ctx, paymentID)
	if err != nil {
		return nil, auditPaymentMeta{}, err
	}
	before, studentMeta, err := s.auditStudentFinanceSnapshot(ctx, paymentItem.StudentID)
	if err != nil {
		return nil, auditPaymentMeta{}, err
	}
	var paymentInvoiceID *int
	if paymentItem.InvoiceID != nil {
		paymentInvoiceID = intPtr(*paymentItem.InvoiceID)
	}
	before["deletedPayment"] = toAuditPaymentDTO(paymentItem)
	return before, auditPaymentMeta{
		StudentID:   paymentItem.StudentID,
		StudentName: studentMeta.StudentName,
		Amount:      paymentItem.Amount,
		InvoiceID:   paymentInvoiceID,
	}, nil
}

func (s *Service) auditSnapshotForPeriod(ctx context.Context, year, month int) (map[string]any, error) {
	invoices, err := s.rt.Invoice.List(ctx, year, month, "all")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"year":     year,
		"month":    month,
		"invoices": invoices,
	}, nil
}

func (s *Service) auditSnapshotForStudentMonth(ctx context.Context, studentID, year, month int) (map[string]any, error) {
	invoices, err := s.rt.Invoice.List(ctx, year, month, "all")
	if err != nil {
		return nil, err
	}
	filtered := make([]InvoiceListItem, 0, len(invoices))
	for _, item := range invoices {
		if item.StudentID == studentID {
			filtered = append(filtered, item)
		}
	}
	return map[string]any{
		"studentId": studentID,
		"year":      year,
		"month":     month,
		"invoices":  filtered,
	}, nil
}

func toAuditPaymentDTO(item *ent.Payment) map[string]any {
	var invoiceID *int
	if item.InvoiceID != nil {
		invoiceID = intPtr(*item.InvoiceID)
	}
	return map[string]any{
		"id":        item.ID,
		"studentId": item.StudentID,
		"invoiceId": invoiceID,
		"amount":    item.Amount,
		"method":    string(item.Method),
		"note":      item.Note,
		"paidAt":    item.PaidAt.Format(time.RFC3339),
		"createdAt": item.CreatedAt.Format(time.RFC3339),
	}
}

func invoiceLabel(number *string, id int) string {
	if number != nil && strings.TrimSpace(*number) != "" {
		return *number
	}
	return fmt.Sprintf("#%d", id)
}

type contextKey string

const actorKey contextKey = "auditActor"

func WithActor(ctx context.Context, currentUser *auth.UserInfo) context.Context {
	return context.WithValue(ctx, actorKey, currentUser)
}

func actorFromContext(ctx context.Context) *auth.UserInfo {
	currentUser, _ := ctx.Value(actorKey).(*auth.UserInfo)
	return currentUser
}

func intPtr(value int) *int { return &value }
