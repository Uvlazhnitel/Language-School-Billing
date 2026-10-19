package web

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"langschool/internal/app"
	"langschool/internal/backend"
)

func TestInactiveDebtorHTTPWorkflow(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()
	ctx := context.Background()
	student := env.Runtime.DB.Ent.Student.Create().SetFullName("Inactive debtor HTTP").SaveX(ctx)
	env.Runtime.DB.Ent.Invoice.Create().SetStudentID(student.ID).
		SetPeriodYear(2026).SetPeriodMonth(10).SetTotalAmountCents(10000).
		SetStatus(app.InvoiceStatusIssuedPendingPDF).SaveX(ctx)
	assertDebtor := func(active bool) {
		t.Helper()
		items := getJSON[[]map[string]any](t, env.Client, env.Server.URL, "/api/debtors")
		if len(items) != 1 || items[0]["isActive"] != active || items[0]["debt"] != float64(100) || items[0]["studentId"] != float64(student.ID) {
			t.Fatalf("debtor active=%v: %+v", active, items)
		}
	}
	assertDebtor(true)
	postJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/students/"+strconv.Itoa(student.ID)+"/active", map[string]any{
		"active": false, "version": student.Version,
	})
	assertDebtor(false)
	before := getJSON[backend.MonthOverviewDTO](t, env.Client, env.Server.URL, "/api/dashboard/month-overview?year=2026&month=10")
	if before.TotalDebt != 100 || before.DebtorsCount != 1 || before.ActiveStudents != 0 {
		t.Fatalf("inactive overview: %+v", before)
	}
	request := map[string]any{"operationId": uuid.NewString(), "studentId": student.ID, "amount": 100, "method": "cash", "paidAt": "2026-10-19", "note": "pay inactive debtor"}
	first := postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", request)
	repeated := postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", request)
	if !reflect.DeepEqual(first, repeated) {
		t.Fatalf("payment replay differs: %+v %+v", first, repeated)
	}
	items := getJSON[[]backend.DebtorDTO](t, env.Client, env.Server.URL, "/api/debtors")
	if len(items) != 0 {
		t.Fatalf("paid inactive debtor still listed: %+v", items)
	}
	after := getJSON[backend.MonthOverviewDTO](t, env.Client, env.Server.URL, "/api/dashboard/month-overview?year=2026&month=10")
	if after.TotalDebt != 0 || after.PaymentsMonthTotal != 100 || after.DebtorsCount != 0 {
		t.Fatalf("inactive payment missing from overview: %+v", after)
	}
}
