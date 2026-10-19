package payment

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"langschool/internal/app"
)

func TestDebtorsSurviveActivityChangesAndAcceptPayment(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()
	svc := New(client)
	student := createTestStudent(t, ctx, client, "Leaving school")
	other := createTestStudentWithActive(t, ctx, client, "Inactive debtor", false)
	createTestStudentWithActive(t, ctx, client, "No debt", false)
	credit := createTestStudentWithActive(t, ctx, client, "Credit", false)
	createTestInvoice(t, ctx, client, student.ID, 2026, 10, 100, app.InvoiceStatusIssued)
	createTestInvoice(t, ctx, client, other.ID, 2026, 10, 50, app.InvoiceStatusIssued)
	if _, err := svc.Create(ctx, credit.ID, nil, 20, "cash", "2026-10-19", ""); err != nil {
		t.Fatal(err)
	}
	for _, active := range []bool{true, false, true, false} {
		if err := client.Student.UpdateOneID(student.ID).SetIsActive(active).Exec(ctx); err != nil {
			t.Fatal(err)
		}
		debtors, err := svc.ListDebtors(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(debtors) != 2 {
			t.Fatalf("active=%v: got %d debtors, want 2", active, len(debtors))
		}
		if debtors[0].StudentID != student.ID || debtors[0].Debt != 100 || debtors[0].IsActive != active || debtors[0].TotalInvoiced != 100 || debtors[0].TotalPaid != 0 {
			t.Fatalf("deactivation changed financial data: %+v", debtors[0])
		}
		if debtors[1].StudentID != other.ID || debtors[1].IsActive || debtors[1].Debt != 50 {
			t.Fatalf("inactive debtor: %+v", debtors[1])
		}
	}
	key := uuid.NewString()
	first, replayed, err := svc.CreateWithOperation(ctx, 1, key, student.ID, nil, 100, "cash", "2026-10-19", "inactive payment")
	if err != nil || replayed {
		t.Fatalf("pay inactive student: %v %v", replayed, err)
	}
	second, replayed, err := svc.CreateWithOperation(ctx, 1, key, student.ID, nil, 100, "cash", "2026-10-19", "inactive payment")
	if err != nil || !replayed || !reflect.DeepEqual(first, second) {
		t.Fatalf("duplicate payment: %v %v", replayed, err)
	}
	remaining, err := svc.ListDebtors(ctx)
	if err != nil || len(remaining) != 1 || remaining[0].StudentID != other.ID {
		t.Fatalf("paid debtor still listed: %+v %v", remaining, err)
	}
	if _, err := svc.Create(ctx, other.ID, nil, 60, "cash", "2026-10-19", "overpayment"); err != nil {
		t.Fatal(err)
	}
	remaining, err = svc.ListDebtors(ctx)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("zero/credit debtors included: %+v %v", remaining, err)
	}
}

func TestMonthOverviewFinancesSurviveStudentDeactivation(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	defer client.Close()
	svc := New(client)
	student := createTestStudent(t, ctx, client, "Historical finance")
	course := createTestCourse(t, ctx, client, "Active course", "group", true)
	createTestEnrollment(t, ctx, client, student.ID, course.ID, "per_lesson")
	createTestAttendanceMonth(t, ctx, client, student.ID, course.ID, 2026, 10, 4)
	current := createTestInvoice(t, ctx, client, student.ID, 2026, 10, 100, app.InvoiceStatusIssued)
	older := createTestInvoice(t, ctx, client, student.ID, 2026, 9, 40, app.InvoiceStatusIssued)
	paidStudent := createTestStudentWithActive(t, ctx, client, "Inactive paid", false)
	draftStudent := createTestStudentWithActive(t, ctx, client, "Inactive draft", false)
	canceledStudent := createTestStudentWithActive(t, ctx, client, "Inactive canceled", false)
	paid := createTestInvoice(t, ctx, client, paidStudent.ID, 2026, 10, 30, app.InvoiceStatusPaid)
	createTestInvoice(t, ctx, client, draftStudent.ID, 2026, 10, 10, app.InvoiceStatusDraft)
	createTestInvoice(t, ctx, client, canceledStudent.ID, 2026, 10, 99, app.InvoiceStatusCanceled)
	createTestInvoice(t, ctx, client, student.ID, 2026, 11, 200, app.InvoiceStatusIssued)
	date := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	createLinkedPaymentWithMethod(t, ctx, client, student.ID, current.ID, 20, app.PaymentMethodCash, date)
	createLinkedPaymentWithMethod(t, ctx, client, paidStudent.ID, paid.ID, 30, app.PaymentMethodBank, date)
	createLinkedPaymentWithMethod(t, ctx, client, student.ID, older.ID, 5, app.PaymentMethodBank, date.AddDate(0, -1, 0))
	createUnlinkedPaymentWithMethod(t, ctx, client, student.ID, 10, app.PaymentMethodCash, date)
	before, err := svc.MonthOverview(ctx, 2026, 10)
	if err != nil {
		t.Fatal(err)
	}
	assertFloatEqual(t, before.MonthDebtTotal, 80)
	assertFloatEqual(t, before.HistoricalDebtTotal, 35)
	assertFloatEqual(t, before.TotalDebt, 115)
	assertFloatEqual(t, before.PaymentsMonthTotal, 60)
	assertEqual(t, before.MonthInvoicesTotal, 4)
	if err := client.Student.UpdateOneID(student.ID).SetIsActive(false).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := svc.MonthOverview(ctx, 2026, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"DraftInvoices", "IssuedInvoices", "PaidInvoices", "PendingPdfInvoices", "ReadyPdfInvoices", "MonthInvoicesTotal", "EmailedInvoices", "NotEmailedInvoices", "OverdueInvoices",
		"TotalIssued", "TotalPaid", "PaymentsMonthTotal", "PaymentsMonthCashTotal", "PaymentsMonthBankTotal", "UnlinkedCreditTotal", "MonthDebtTotal", "HistoricalDebtTotal", "DebtorsCount", "TotalDebt",
	} {
		old := reflect.ValueOf(before).Elem().FieldByName(field).Interface()
		new := reflect.ValueOf(after).Elem().FieldByName(field).Interface()
		if !reflect.DeepEqual(old, new) {
			t.Errorf("%s changed after deactivation: %v -> %v", field, old, new)
		}
	}
	assertEqual(t, before.ActiveStudents, 1)
	assertEqual(t, after.ActiveStudents, 0)
	assertEqual(t, before.Enrollments, 1)
	assertEqual(t, after.Enrollments, 0)
	assertEqual(t, before.AttendanceFilled, 1)
	assertEqual(t, after.AttendanceFilled, 0)
	assertEqual(t, after.ActiveCourses, before.ActiveCourses)
	if err := client.Student.UpdateOneID(student.ID).SetIsActive(true).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	reactivated, err := svc.MonthOverview(ctx, 2026, 10)
	if err != nil || !reflect.DeepEqual(before, reactivated) {
		t.Fatalf("overview not restored: %+v, err=%v", reactivated, err)
	}
}
