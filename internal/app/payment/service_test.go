package payment

import (
	"context"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"langschool/ent"
	entcourse "langschool/ent/course"
	entenrollment "langschool/ent/enrollment"
	"langschool/ent/enttest"
	entinvoice "langschool/ent/invoice"
	entpayment "langschool/ent/payment"
	"langschool/internal/app"
	"langschool/internal/money"
)

func TestCreateDirectPaymentUpdatesInvoiceSummaryAndStatus(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Arina Osipova")
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 3, 100, app.InvoiceStatusIssued)

	invoiceID := inv.ID
	if _, err := svc.Create(ctx, st.ID, &invoiceID, 30, app.PaymentMethodCash, "2026-03-12", "partial"); err != nil {
		t.Fatalf("create partial payment: %v", err)
	}

	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary after partial payment: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 30)
	assertFloatEqual(t, summary.Remaining, 70)
	assertEqual(t, summary.Status, app.InvoiceStatusIssued)

	if _, err := svc.Create(ctx, st.ID, &invoiceID, 70, app.PaymentMethodBank, "2026-03-15", "final"); err != nil {
		t.Fatalf("create final payment: %v", err)
	}

	summary, err = svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary after final payment: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 100)
	assertFloatEqual(t, summary.Remaining, 0)
	assertEqual(t, summary.Status, app.InvoiceStatusPaid)
}

func TestCreateGlobalPaymentAllocatesToOldestInvoices(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Mila Petrova")
	oldest := createTestInvoice(t, ctx, client, st.ID, 2026, 1, 50, app.InvoiceStatusIssued)
	newer := createTestInvoice(t, ctx, client, st.ID, 2026, 2, 30, app.InvoiceStatusIssued)

	dto, err := svc.Create(ctx, st.ID, nil, 60, app.PaymentMethodCash, "2026-02-20", "debtor payment")
	if err != nil {
		t.Fatalf("create global payment: %v", err)
	}
	if dto.InvoiceID == nil || *dto.InvoiceID != oldest.ID {
		t.Fatalf("expected first allocated payment to target oldest invoice %d, got %+v", oldest.ID, dto.InvoiceID)
	}

	payments, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID)).
		Order(ent.Asc(entpayment.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query payments: %v", err)
	}
	if len(payments) != 2 {
		t.Fatalf("expected 2 allocated payments, got %d", len(payments))
	}

	assertLinkedPayment(t, payments[0], oldest.ID, 50)
	assertLinkedPayment(t, payments[1], newer.ID, 10)

	oldestSummary, err := svc.InvoiceSummary(ctx, oldest.ID)
	if err != nil {
		t.Fatalf("oldest invoice summary: %v", err)
	}
	assertFloatEqual(t, oldestSummary.Paid, 50)
	assertFloatEqual(t, oldestSummary.Remaining, 0)
	assertEqual(t, oldestSummary.Status, app.InvoiceStatusPaid)

	newerSummary, err := svc.InvoiceSummary(ctx, newer.ID)
	if err != nil {
		t.Fatalf("newer invoice summary: %v", err)
	}
	assertFloatEqual(t, newerSummary.Paid, 10)
	assertFloatEqual(t, newerSummary.Remaining, 20)
	assertEqual(t, newerSummary.Status, app.InvoiceStatusIssued)

	balance, err := svc.StudentBalance(ctx, st.ID)
	if err != nil {
		t.Fatalf("student balance: %v", err)
	}
	assertFloatEqual(t, balance.TotalInvoiced, 80)
	assertFloatEqual(t, balance.TotalPaid, 60)
	assertFloatEqual(t, balance.Debt, 20)
}

func TestCreateGlobalPaymentLeavesExtraAsCredit(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Nikita Smirnov")
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 4, 50, app.InvoiceStatusIssued)

	if _, err := svc.Create(ctx, st.ID, nil, 80, app.PaymentMethodBank, "2026-04-25", "overpayment"); err != nil {
		t.Fatalf("create overpayment: %v", err)
	}

	payments, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID)).
		Order(ent.Asc(entpayment.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query payments: %v", err)
	}
	if len(payments) != 2 {
		t.Fatalf("expected 2 payments (invoice + credit), got %d", len(payments))
	}

	assertLinkedPayment(t, payments[0], inv.ID, 50)
	if payments[1].InvoiceID != nil {
		t.Fatalf("expected extra payment to stay unlinked as credit, got invoice_id=%v", *payments[1].InvoiceID)
	}
	assertFloatEqual(t, money.CentsToEuros(payments[1].AmountCents), 30)

	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 50)
	assertFloatEqual(t, summary.Remaining, 0)
	assertEqual(t, summary.Status, app.InvoiceStatusPaid)

	balance, err := svc.StudentBalance(ctx, st.ID)
	if err != nil {
		t.Fatalf("student balance: %v", err)
	}
	assertFloatEqual(t, balance.TotalInvoiced, 50)
	assertFloatEqual(t, balance.TotalPaid, 80)
	assertFloatEqual(t, balance.Balance, 30)
	assertFloatEqual(t, balance.Debt, 0)
}

func TestApplyCreditPartialCoversFutureInvoice(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Credit Test Partial")

	// Student has unlinked credit of €20.
	createUnlinkedPayment(t, ctx, client, st.ID, 20, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	// New invoice for €50.
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 4, 50, app.InvoiceStatusIssued)

	if err := svc.ApplyCreditToOldestInvoices(ctx, st.ID); err != nil {
		t.Fatalf("ApplyCreditToOldestInvoices: %v", err)
	}

	// Linked payment of €20 should have been created for the new invoice.
	payments, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID), entpayment.InvoiceIDEQ(inv.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query linked payments: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("expected 1 linked payment for invoice, got %d", len(payments))
	}
	assertFloatEqual(t, money.CentsToEuros(payments[0].AmountCents), 20)

	// Original credit payment should be gone.
	credits, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID), entpayment.InvoiceIDIsNil()).
		All(ctx)
	if err != nil {
		t.Fatalf("query credits: %v", err)
	}
	if len(credits) != 0 {
		t.Fatalf("expected no unlinked credit to remain, got %d", len(credits))
	}

	// Invoice should still be issued with €30 remaining.
	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 20)
	assertFloatEqual(t, summary.Remaining, 30)
	assertEqual(t, summary.Status, app.InvoiceStatusIssued)
}

func TestApplyCreditFullyCoversInvoiceLeavesLeftover(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Credit Test Full")

	// Student has unlinked credit of €80.
	createUnlinkedPayment(t, ctx, client, st.ID, 80, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))

	// New invoice for €50.
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 4, 50, app.InvoiceStatusIssued)

	if err := svc.ApplyCreditToOldestInvoices(ctx, st.ID); err != nil {
		t.Fatalf("ApplyCreditToOldestInvoices: %v", err)
	}

	// Invoice should be fully paid.
	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 50)
	assertFloatEqual(t, summary.Remaining, 0)
	assertEqual(t, summary.Status, app.InvoiceStatusPaid)

	// Remaining credit of €30 should still be unlinked.
	credits, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID), entpayment.InvoiceIDIsNil()).
		All(ctx)
	if err != nil {
		t.Fatalf("query credits: %v", err)
	}
	if len(credits) != 1 {
		t.Fatalf("expected 1 unlinked credit to remain, got %d", len(credits))
	}
	assertFloatEqual(t, money.CentsToEuros(credits[0].AmountCents), 30)
}

func TestApplyCreditMultipleCreditsOldestFirst(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Credit Test Multi")

	// Two unlinked credit payments: €10 cash, €15 bank.
	createUnlinkedPayment(t, ctx, client, st.ID, 10, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	createUnlinkedPayment(t, ctx, client, st.ID, 15, time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC))

	// New invoice for €20.
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 4, 20, app.InvoiceStatusIssued)

	if err := svc.ApplyCreditToOldestInvoices(ctx, st.ID); err != nil {
		t.Fatalf("ApplyCreditToOldestInvoices: %v", err)
	}

	// Total linked payments for the invoice should be €20.
	linkedPayments, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID), entpayment.InvoiceIDEQ(inv.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query linked payments: %v", err)
	}
	totalLinked := 0.0
	for _, p := range linkedPayments {
		totalLinked += money.CentsToEuros(p.AmountCents)
	}
	assertFloatEqual(t, totalLinked, 20)

	// Invoice should be paid.
	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary: %v", err)
	}
	assertEqual(t, summary.Status, app.InvoiceStatusPaid)

	// Remaining unlinked credit should be €5.
	credits, err := client.Payment.Query().
		Where(entpayment.StudentIDEQ(st.ID), entpayment.InvoiceIDIsNil()).
		All(ctx)
	if err != nil {
		t.Fatalf("query credits: %v", err)
	}
	if len(credits) != 1 {
		t.Fatalf("expected 1 unlinked credit to remain, got %d", len(credits))
	}
	assertFloatEqual(t, money.CentsToEuros(credits[0].AmountCents), 5)
}

func TestApplyCreditNoCreditNoOp(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Credit Test No-op")
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 4, 50, app.InvoiceStatusIssued)

	// No unlinked payments exist – should be a no-op.
	if err := svc.ApplyCreditToOldestInvoices(ctx, st.ID); err != nil {
		t.Fatalf("ApplyCreditToOldestInvoices: %v", err)
	}

	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 0)
	assertFloatEqual(t, summary.Remaining, 50)
	assertEqual(t, summary.Status, app.InvoiceStatusIssued)
}

func TestDeletePaymentRevertsPaidInvoiceBackToIssued(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Diana Volkova")
	inv := createTestInvoice(t, ctx, client, st.ID, 2026, 5, 40, app.InvoiceStatusIssued)

	invoiceID := inv.ID
	created, err := svc.Create(ctx, st.ID, &invoiceID, 40, app.PaymentMethodCash, "2026-05-10", "full payment")
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete payment: %v", err)
	}

	updatedInvoice, err := client.Invoice.Get(ctx, inv.ID)
	if err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	assertEqual(t, string(updatedInvoice.Status), app.InvoiceStatusIssued)

	summary, err := svc.InvoiceSummary(ctx, inv.ID)
	if err != nil {
		t.Fatalf("invoice summary after delete: %v", err)
	}
	assertFloatEqual(t, summary.Paid, 0)
	assertFloatEqual(t, summary.Remaining, 40)
	assertEqual(t, summary.Status, app.InvoiceStatusIssued)
}

func TestInvoiceBalance(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		payments      []float64
		wantTotal     float64
		wantPaid      float64
		wantRemaining float64
	}{
		{
			name:          "no payments",
			payments:      nil,
			wantTotal:     100,
			wantPaid:      0,
			wantRemaining: 100,
		},
		{
			name:          "partial payment",
			payments:      []float64{40},
			wantTotal:     100,
			wantPaid:      40,
			wantRemaining: 60,
		},
		{
			name:          "fully paid",
			payments:      []float64{40, 60},
			wantTotal:     100,
			wantPaid:      100,
			wantRemaining: 0,
		},
		{
			name:          "overpaid",
			payments:      []float64{120},
			wantTotal:     100,
			wantPaid:      120,
			wantRemaining: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			defer client.Close()

			svc := New(client)
			st := createTestStudent(t, ctx, client, "Balance Test")
			inv := createTestInvoice(t, ctx, client, st.ID, 2026, 6, 100, app.InvoiceStatusIssued)

			for i, amount := range tt.payments {
				createLinkedPayment(t, ctx, client, st.ID, inv.ID, amount, time.Date(2026, 6, i+1, 0, 0, 0, 0, time.UTC))
			}

			total, paid, remaining, err := svc.invoiceBalance(ctx, inv)
			if err != nil {
				t.Fatalf("invoiceBalance returned error: %v", err)
			}

			assertFloatEqual(t, total, tt.wantTotal)
			assertFloatEqual(t, paid, tt.wantPaid)
			assertFloatEqual(t, remaining, tt.wantRemaining)
		})
	}
}

func TestStudentDebtDetails(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)
	st := createTestStudent(t, ctx, client, "Debt Details")

	issuedUnpaid := createTestInvoiceWithNumber(t, ctx, client, st.ID, 2025, 12, 80, app.InvoiceStatusIssued, "LS-202512-001")
	draft := createTestInvoiceWithNumber(t, ctx, client, st.ID, 2026, 1, 30, app.InvoiceStatusDraft, nil)
	canceled := createTestInvoiceWithNumber(t, ctx, client, st.ID, 2026, 2, 40, app.InvoiceStatusCanceled, "LS-202602-099")
	paidPartial := createTestInvoiceWithNumber(t, ctx, client, st.ID, 2026, 3, 50, app.InvoiceStatusPaid, "LS-202603-001")
	issuedPartial := createTestInvoiceWithNumber(t, ctx, client, st.ID, 2026, 4, 70, app.InvoiceStatusIssued, "LS-202604-002")
	fullyPaid := createTestInvoiceWithNumber(t, ctx, client, st.ID, 2026, 5, 60, app.InvoiceStatusPaid, "LS-202605-001")

	createLinkedPayment(t, ctx, client, st.ID, paidPartial.ID, 20, time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC))
	createLinkedPayment(t, ctx, client, st.ID, issuedPartial.ID, 25, time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC))
	createLinkedPayment(t, ctx, client, st.ID, fullyPaid.ID, 60, time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC))

	got, err := svc.StudentDebtDetails(ctx, st.ID)
	if err != nil {
		t.Fatalf("StudentDebtDetails returned error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 debt rows, got %d", len(got))
	}

	assertEqual(t, got[0].InvoiceID, issuedUnpaid.ID)
	assertEqual(t, got[0].Year, 2025)
	assertEqual(t, got[0].Month, 12)
	assertStringPtrEqual(t, got[0].Number, "LS-202512-001")
	assertFloatEqual(t, got[0].Total, 80)
	assertFloatEqual(t, got[0].Paid, 0)
	assertFloatEqual(t, got[0].Remaining, 80)
	assertEqual(t, got[0].Status, app.InvoiceStatusIssued)

	assertEqual(t, got[1].InvoiceID, paidPartial.ID)
	assertEqual(t, got[1].Year, 2026)
	assertEqual(t, got[1].Month, 3)
	assertStringPtrEqual(t, got[1].Number, "LS-202603-001")
	assertFloatEqual(t, got[1].Total, 50)
	assertFloatEqual(t, got[1].Paid, 20)
	assertFloatEqual(t, got[1].Remaining, 30)
	assertEqual(t, got[1].Status, app.InvoiceStatusPaid)

	assertEqual(t, got[2].InvoiceID, issuedPartial.ID)
	assertEqual(t, got[2].Year, 2026)
	assertEqual(t, got[2].Month, 4)
	assertStringPtrEqual(t, got[2].Number, "LS-202604-002")
	assertFloatEqual(t, got[2].Total, 70)
	assertFloatEqual(t, got[2].Paid, 25)
	assertFloatEqual(t, got[2].Remaining, 45)
	assertEqual(t, got[2].Status, app.InvoiceStatusIssued)

	for _, dto := range got {
		if dto.InvoiceID == draft.ID {
			t.Fatalf("draft invoice %d should be excluded", draft.ID)
		}
		if dto.InvoiceID == canceled.ID {
			t.Fatalf("canceled invoice %d should be excluded", canceled.ID)
		}
		if dto.InvoiceID == fullyPaid.ID {
			t.Fatalf("fully paid invoice %d should be excluded", fullyPaid.ID)
		}
	}
}

func TestMonthOverview(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()

	svc := New(client)

	st1 := createTestStudent(t, ctx, client, "Overview Debtor One")
	st2 := createTestStudent(t, ctx, client, "Overview Debtor Two")
	st3 := createTestStudent(t, ctx, client, "Overview Credit Student")
	inactiveStudent := createTestStudentWithActive(t, ctx, client, "Overview Inactive Student", false)

	c1 := createTestCourse(t, ctx, client, "Group A", "group", true)
	c2 := createTestCourse(t, ctx, client, "Individual B", "individual", true)
	createTestCourse(t, ctx, client, "Inactive Course", "group", false)

	createTestEnrollment(t, ctx, client, st1.ID, c1.ID, "per_lesson")
	createTestEnrollment(t, ctx, client, st2.ID, c1.ID, "per_lesson")
	createTestEnrollment(t, ctx, client, st3.ID, c2.ID, "subscription")

	createTestAttendanceMonth(t, ctx, client, st1.ID, c1.ID, 2026, 4, 3)
	createTestAttendanceMonth(t, ctx, client, st2.ID, c1.ID, 2026, 4, 0)
	createTestAttendanceMonth(t, ctx, client, st3.ID, c2.ID, 2026, 4, 5)

	draftInvoice := createTestInvoice(t, ctx, client, st3.ID, 2026, 4, 10, app.InvoiceStatusDraft)
	issuedInvoice := createTestInvoice(t, ctx, client, st1.ID, 2026, 4, 70, app.InvoiceStatusIssued)
	paidInvoice := createTestInvoice(t, ctx, client, st2.ID, 2026, 4, 30, app.InvoiceStatusPaid)
	createTestInvoice(t, ctx, client, inactiveStudent.ID, 2026, 4, 25, app.InvoiceStatusCanceled)
	olderIssuedInvoice := createTestInvoice(t, ctx, client, st2.ID, 2026, 3, 40, app.InvoiceStatusIssued)

	createLinkedPayment(t, ctx, client, st1.ID, issuedInvoice.ID, 20, time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC))
	createLinkedPayment(t, ctx, client, st2.ID, paidInvoice.ID, 30, time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC))
	createLinkedPayment(t, ctx, client, st2.ID, olderIssuedInvoice.ID, 5, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))
	createUnlinkedPayment(t, ctx, client, st3.ID, 10, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))

	got, err := svc.MonthOverview(ctx, 2026, 4)
	if err != nil {
		t.Fatalf("MonthOverview returned error: %v", err)
	}

	assertEqual(t, got.Year, 2026)
	assertEqual(t, got.Month, 4)
	assertEqual(t, got.ActiveStudents, 3)
	assertEqual(t, got.ActiveCourses, 2)
	assertEqual(t, got.Enrollments, 3)
	assertEqual(t, got.PerLessonEnrollments, 2)
	assertEqual(t, got.AttendanceFilled, 1)
	assertEqual(t, got.AttendanceMissing, 1)
	assertEqual(t, got.DraftInvoices, 1)
	assertEqual(t, got.IssuedInvoices, 1)
	assertEqual(t, got.PaidInvoices, 1)
	assertFloatEqual(t, got.TotalIssued, 100)
	assertFloatEqual(t, got.TotalPaid, 50)
	assertEqual(t, got.DebtorsCount, 2)
	assertFloatEqual(t, got.TotalDebt, 85)

	if draftInvoice.Status != entinvoice.Status(app.InvoiceStatusDraft) {
		t.Fatalf("draft invoice status unexpectedly changed")
	}
}

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	return enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1")
}

func createTestStudent(t *testing.T, ctx context.Context, client *ent.Client, fullName string) *ent.Student {
	t.Helper()
	return createTestStudentWithActive(t, ctx, client, fullName, true)
}

func createTestStudentWithActive(t *testing.T, ctx context.Context, client *ent.Client, fullName string, isActive bool) *ent.Student {
	t.Helper()
	st, err := client.Student.Create().
		SetFullName(fullName).
		SetIsActive(isActive).
		Save(ctx)
	if err != nil {
		t.Fatalf("create student: %v", err)
	}
	return st
}

func createTestCourse(t *testing.T, ctx context.Context, client *ent.Client, name, courseType string, isActive bool) *ent.Course {
	t.Helper()
	c, err := client.Course.Create().
		SetName(name).
		SetType(entcourse.Type(courseType)).
		SetLessonPriceCents(money.EurosToCents(10)).
		SetSubscriptionPriceCents(money.EurosToCents(25)).
		SetIsActive(isActive).
		Save(ctx)
	if err != nil {
		t.Fatalf("create course: %v", err)
	}
	return c
}

func createTestEnrollment(t *testing.T, ctx context.Context, client *ent.Client, studentID, courseID int, billingMode string) *ent.Enrollment {
	t.Helper()
	enr, err := client.Enrollment.Create().
		SetStudentID(studentID).
		SetCourseID(courseID).
		SetBillingMode(entenrollment.BillingMode(billingMode)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create enrollment: %v", err)
	}
	return enr
}

func createTestAttendanceMonth(t *testing.T, ctx context.Context, client *ent.Client, studentID, courseID, year, month int, hours float64) *ent.AttendanceMonth {
	t.Helper()
	am, err := client.AttendanceMonth.Create().
		SetStudentID(studentID).
		SetCourseID(courseID).
		SetYear(year).
		SetMonth(month).
		SetHours(hours).
		Save(ctx)
	if err != nil {
		t.Fatalf("create attendance month: %v", err)
	}
	return am
}

func createTestInvoice(t *testing.T, ctx context.Context, client *ent.Client, studentID, year, month int, total float64, status string) *ent.Invoice {
	t.Helper()
	return createTestInvoiceWithNumber(t, ctx, client, studentID, year, month, total, status, nil)
}

func createTestInvoiceWithNumber(t *testing.T, ctx context.Context, client *ent.Client, studentID, year, month int, total float64, status string, number any) *ent.Invoice {
	t.Helper()
	create := client.Invoice.Create().
		SetStudentID(studentID).
		SetPeriodYear(year).
		SetPeriodMonth(month).
		SetTotalAmountCents(money.EurosToCents(total)).
		SetStatus(entinvoice.Status(status))
	switch n := number.(type) {
	case nil:
	case string:
		create.SetNumber(n)
	case *string:
		create.SetNillableNumber(n)
	default:
		t.Fatalf("unsupported invoice number type %T", number)
	}
	inv, err := create.Save(ctx)
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	return inv
}

func createLinkedPayment(t *testing.T, ctx context.Context, client *ent.Client, studentID, invoiceID int, amount float64, paidAt time.Time) *ent.Payment {
	t.Helper()
	p, err := client.Payment.Create().
		SetStudentID(studentID).
		SetInvoiceID(invoiceID).
		SetPaidAt(paidAt).
		SetAmountCents(money.EurosToCents(amount)).
		SetMethod(entpayment.Method(app.PaymentMethodCash)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create linked payment: %v", err)
	}
	return p
}

func createUnlinkedPayment(t *testing.T, ctx context.Context, client *ent.Client, studentID int, amount float64, paidAt time.Time) *ent.Payment {
	t.Helper()
	p, err := client.Payment.Create().
		SetStudentID(studentID).
		SetPaidAt(paidAt).
		SetAmountCents(money.EurosToCents(amount)).
		SetMethod(entpayment.Method(app.PaymentMethodCash)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create unlinked payment: %v", err)
	}
	return p
}

func assertLinkedPayment(t *testing.T, p *ent.Payment, invoiceID int, amount float64) {
	t.Helper()
	if p.InvoiceID == nil {
		t.Fatalf("expected payment to be linked to invoice %d, got nil", invoiceID)
	}
	if *p.InvoiceID != invoiceID {
		t.Fatalf("expected linked invoice %d, got %d", invoiceID, *p.InvoiceID)
	}
	assertFloatEqual(t, money.CentsToEuros(p.AmountCents), amount)
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertFloatEqual(t *testing.T, got, want float64) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.0001 {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}

func assertStringPtrEqual(t *testing.T, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("got nil, want %q", want)
	}
	if *got != want {
		t.Fatalf("got %q, want %q", *got, want)
	}
}
