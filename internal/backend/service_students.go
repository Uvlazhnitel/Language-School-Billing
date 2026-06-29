package backend

import (
	"context"
	"errors"
	"strings"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/payment"
	"langschool/ent/student"
	sharedapp "langschool/internal/app"
	"langschool/internal/apperrors"
	"langschool/internal/money"
)

func (s *Service) StudentList(ctx context.Context, q string, includeInactive bool) ([]StudentDTO, error) {
	q = strings.TrimSpace(q)
	query := s.rt.DB.Ent.Student.Query()
	if !includeInactive {
		query = query.Where(student.IsActiveEQ(true))
	}
	if q != "" {
		query = query.Where(student.Or(
			student.FullNameContainsFold(q),
			student.PhoneContainsFold(q),
			student.EmailContainsFold(q),
		))
	}
	studs, err := query.Order(ent.Asc(student.FieldFullName)).All(ctx)
	if err != nil {
		return nil, err
	}
	summaries, err := s.studentBalanceSummaries(ctx, studs)
	if err != nil {
		return nil, err
	}
	out := make([]StudentDTO, 0, len(studs))
	for _, item := range studs {
		out = append(out, toStudentDTO(item, summaries[item.ID]))
	}
	return out, nil
}

func (s *Service) StudentGet(ctx context.Context, id int) (*StudentDTO, error) {
	item, err := s.rt.DB.Ent.Student.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := toStudentDTO(item, s.studentBalanceSummary(ctx, item.ID))
	return &dto, nil
}

func (s *Service) studentBalanceSummaries(ctx context.Context, studs []*ent.Student) (map[int]studentBalanceSummary, error) {
	summaries := make(map[int]studentBalanceSummary, len(studs))
	if len(studs) == 0 {
		return summaries, nil
	}
	studentIDs := make([]int, 0, len(studs))
	for _, st := range studs {
		studentIDs = append(studentIDs, st.ID)
	}
	invoicedByStudent, err := s.aggregateInvoiceTotalsByStudent(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	paidByStudent, err := s.aggregatePaymentTotalsByStudent(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	for _, st := range studs {
		summaries[st.ID] = makeStudentBalanceSummary(invoicedByStudent[st.ID], paidByStudent[st.ID])
	}
	return summaries, nil
}

func (s *Service) studentBalanceSummary(ctx context.Context, studentID int) studentBalanceSummary {
	invoicedByStudent, err := s.aggregateInvoiceTotalsByStudent(ctx, []int{studentID})
	if err != nil {
		return studentBalanceSummary{}
	}
	paidByStudent, err := s.aggregatePaymentTotalsByStudent(ctx, []int{studentID})
	if err != nil {
		return studentBalanceSummary{}
	}
	return makeStudentBalanceSummary(invoicedByStudent[studentID], paidByStudent[studentID])
}

func makeStudentBalanceSummary(invoicedCents, paidCents int64) studentBalanceSummary {
	balanceCents := paidCents - invoicedCents
	summary := studentBalanceSummary{
		Balance: money.CentsToEuros(balanceCents),
	}
	if balanceCents < 0 {
		summary.Debt = money.CentsToEuros(-balanceCents)
	}
	return summary
}

func (s *Service) aggregateInvoiceTotalsByStudent(ctx context.Context, studentIDs []int) (map[int]int64, error) {
	rows := []studentMoneyAggregate{}
	err := s.rt.DB.Ent.Invoice.Query().
		Where(
			invoice.StudentIDIn(studentIDs...),
			invoice.StatusIn(
				invoice.Status(sharedapp.InvoiceStatusIssuedPendingPDF),
				invoice.StatusIssued,
				invoice.Status(sharedapp.InvoiceStatusPaidPendingPDF),
				invoice.StatusPaid,
			),
		).
		GroupBy(invoice.FieldStudentID).
		Aggregate(ent.As(ent.Sum(invoice.FieldTotalAmountCents), "total")).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, row := range rows {
		out[row.StudentID] = row.Total
	}
	return out, nil
}

func (s *Service) aggregatePaymentTotalsByStudent(ctx context.Context, studentIDs []int) (map[int]int64, error) {
	rows := []studentMoneyAggregate{}
	err := s.rt.DB.Ent.Payment.Query().
		Where(payment.StudentIDIn(studentIDs...)).
		GroupBy(payment.FieldStudentID).
		Aggregate(ent.As(ent.Sum(payment.FieldAmountCents), "total")).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, row := range rows {
		out[row.StudentID] = row.Total
	}
	return out, nil
}

func (s *Service) StudentCreate(ctx context.Context, fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	fullName = sanitizeInput(fullName)
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}
	personalCode = sanitizeInput(personalCode)
	payerName = sanitizeInput(payerName)
	payerRole = normalizePayerRole(payerRole)
	if err := validateMinorPayer(isMinor, payerName, payerRole); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Student.Create().
		SetFullName(fullName).
		SetPersonalCode(personalCode).
		SetPhone(sanitizeInput(phone)).
		SetEmail(sanitizeInput(email)).
		SetNote(sanitizeInput(note)).
		SetIsMinor(isMinor).
		SetPayerName(payerName).
		SetPayerRole(payerRole).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	dto := toStudentDTO(item, studentBalanceSummary{})
	return &dto, nil
}

func (s *Service) StudentUpdate(ctx context.Context, id int, fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	fullName = sanitizeInput(fullName)
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}
	personalCode = sanitizeInput(personalCode)
	payerName = sanitizeInput(payerName)
	payerRole = normalizePayerRole(payerRole)
	if err := validateMinorPayer(isMinor, payerName, payerRole); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Student.UpdateOneID(id).
		AddVersion(1).
		SetFullName(fullName).
		SetPersonalCode(personalCode).
		SetPhone(sanitizeInput(phone)).
		SetEmail(sanitizeInput(email)).
		SetNote(sanitizeInput(note)).
		SetIsMinor(isMinor).
		SetPayerName(payerName).
		SetPayerRole(payerRole).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	dto := toStudentDTO(item, s.studentBalanceSummary(ctx, item.ID))
	return &dto, nil
}

func (s *Service) StudentUpdateWithVersion(ctx context.Context, id, version int, fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	fullName = sanitizeInput(fullName)
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}
	personalCode = sanitizeInput(personalCode)
	payerName = sanitizeInput(payerName)
	payerRole = normalizePayerRole(payerRole)
	if err := validateMinorPayer(isMinor, payerName, payerRole); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Student.UpdateOneID(id).
		Where(student.VersionEQ(version)).
		SetVersion(version + 1).
		SetFullName(fullName).
		SetPersonalCode(personalCode).
		SetPhone(sanitizeInput(phone)).
		SetEmail(sanitizeInput(email)).
		SetNote(sanitizeInput(note)).
		SetIsMinor(isMinor).
		SetPayerName(payerName).
		SetPayerRole(payerRole).
		Save(ctx)
	if err != nil {
		return nil, staleOnNotFound(err)
	}
	dto := toStudentDTO(item, s.studentBalanceSummary(ctx, item.ID))
	return &dto, nil
}

func (s *Service) StudentSetActive(ctx context.Context, id int, active bool) error {
	_, err := s.rt.DB.Ent.Student.UpdateOneID(id).AddVersion(1).SetIsActive(active).Save(ctx)
	return err
}

func (s *Service) StudentSetActiveWithVersion(ctx context.Context, id, version int, active bool) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	_, err := s.rt.DB.Ent.Student.UpdateOneID(id).
		Where(student.VersionEQ(version)).
		SetVersion(version + 1).
		SetIsActive(active).
		Save(ctx)
	return staleOnNotFound(err)
}

func (s *Service) StudentDelete(ctx context.Context, id int) error {
	st, err := s.rt.DB.Ent.Student.Get(ctx, id)
	if err != nil {
		return err
	}
	if st.IsActive {
		return errors.New("cannot delete active student; deactivate first")
	}
	hasPayments, err := s.rt.DB.Ent.Payment.Query().Where(payment.StudentIDEQ(id)).Exist(ctx)
	if err != nil {
		return err
	}
	if hasPayments {
		return errors.New("cannot delete student: has payments (financial records)")
	}
	hasProtectedInvoices, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusNEQ(sharedapp.InvoiceStatusDraft)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasProtectedInvoices {
		return errors.New("cannot delete student: has issued, paid, or canceled invoices")
	}

	tx, err := s.rt.DB.Ent.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	draftInvoiceIDs, err := tx.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusEQ(sharedapp.InvoiceStatusDraft)).
		IDs(ctx)
	if err != nil {
		return err
	}
	if len(draftInvoiceIDs) > 0 {
		if _, err := tx.InvoiceLine.Delete().Where(invoiceline.InvoiceIDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.Invoice.Delete().Where(invoice.IDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
	}
	if _, err := tx.AttendanceMonth.Delete().Where(attendancemonth.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.Enrollment.Delete().Where(enrollment.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if err := tx.Student.DeleteOneID(id).Exec(ctx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) StudentDeleteWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	st, err := s.rt.DB.Ent.Student.Get(ctx, id)
	if err != nil {
		return err
	}
	if st.Version != version {
		return apperrors.StaleRevision()
	}
	if st.IsActive {
		return errors.New("cannot delete active student; deactivate first")
	}
	hasPayments, err := s.rt.DB.Ent.Payment.Query().Where(payment.StudentIDEQ(id)).Exist(ctx)
	if err != nil {
		return err
	}
	if hasPayments {
		return errors.New("cannot delete student: has payments (financial records)")
	}
	hasProtectedInvoices, err := s.rt.DB.Ent.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusNEQ(sharedapp.InvoiceStatusDraft)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasProtectedInvoices {
		return errors.New("cannot delete student: has issued, paid, or canceled invoices")
	}

	tx, err := s.rt.DB.Ent.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	draftInvoiceIDs, err := tx.Invoice.Query().
		Where(invoice.StudentIDEQ(id), invoice.StatusEQ(sharedapp.InvoiceStatusDraft)).
		IDs(ctx)
	if err != nil {
		return err
	}
	if len(draftInvoiceIDs) > 0 {
		if _, err := tx.InvoiceLine.Delete().Where(invoiceline.InvoiceIDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.Invoice.Delete().Where(invoice.IDIn(draftInvoiceIDs...)).Exec(ctx); err != nil {
			return err
		}
	}
	if _, err := tx.AttendanceMonth.Delete().Where(attendancemonth.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.Enrollment.Delete().Where(enrollment.StudentIDEQ(id)).Exec(ctx); err != nil {
		return err
	}
	if err := tx.Student.DeleteOneID(id).Where(student.VersionEQ(version)).Exec(ctx); err != nil {
		return staleOnNotFound(err)
	}
	return tx.Commit()
}

func toStudentDTO(s *ent.Student, summary studentBalanceSummary) StudentDTO {
	return StudentDTO{
		ID:           s.ID,
		Version:      s.Version,
		FullName:     s.FullName,
		PersonalCode: s.PersonalCode,
		Phone:        s.Phone,
		Email:        s.Email,
		Note:         s.Note,
		IsMinor:      s.IsMinor,
		PayerName:    s.PayerName,
		PayerRole:    s.PayerRole,
		IsActive:     s.IsActive,
		Balance:      summary.Balance,
		Debt:         summary.Debt,
	}
}
