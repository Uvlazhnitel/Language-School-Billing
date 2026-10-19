package invoice

import (
	"context"
	"path/filepath"
	"testing"

	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
)

func TestSubscriptionDraftUsesEnrollmentPrice(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		coursePrice           int64
		enrollmentPrice       int64
		lessons               float64
		rebuildExisting       bool
		mixed                 bool
		wantTotal             int64
		wantSubscriptionPrice int64
	}{
		{name: "zero course rate", enrollmentPrice: 1500, lessons: 4, wantTotal: 6000, wantSubscriptionPrice: 1500},
		{name: "rebuild existing draft", enrollmentPrice: 1500, lessons: 4, rebuildExisting: true, wantTotal: 6000, wantSubscriptionPrice: 1500},
		{name: "mixed billing", enrollmentPrice: 1500, lessons: 4, mixed: true, wantTotal: 8000, wantSubscriptionPrice: 1500},
		{name: "zero attendance", enrollmentPrice: 1500},
		{name: "legacy fallback", coursePrice: 2000, enrollmentPrice: -1, lessons: 4, wantTotal: 8000, wantSubscriptionPrice: 2000},
		{name: "zero enrollment price", coursePrice: 2000, enrollmentPrice: 0, lessons: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := enttest.Open(t, "sqlite3", "file:"+filepath.Join(t.TempDir(), "invoice.sqlite"))
			defer client.Close()
			svc := New(client)
			st := client.Student.Create().SetFullName("Subscription price student").SaveX(ctx)
			crs := client.Course.Create().SetName("Subscription course").SetType(course.TypeGroup).
				SetLessonPriceCents(tc.coursePrice).SetSubscriptionPriceCents(9900).SaveX(ctx)
			enr := client.Enrollment.Create().SetStudentID(st.ID).SetCourseID(crs.ID).
				SetBillingMode(enrollment.BillingModeSubscription).SetChargeMaterials(false).
				SetSubscriptionLessonPriceCents(tc.enrollmentPrice).SaveX(ctx)
			client.AttendanceMonth.Create().SetStudentID(st.ID).SetCourseID(crs.ID).
				SetYear(2026).SetMonth(10).SetHours(tc.lessons).SaveX(ctx)

			if tc.mixed {
				other := client.Course.Create().SetName("Individual course").SetType(course.TypeIndividual).
					SetLessonPriceCents(2000).SaveX(ctx)
				client.Enrollment.Create().SetStudentID(st.ID).SetCourseID(other.ID).
					SetBillingMode(enrollment.BillingModePerLesson).SetChargeMaterials(false).SaveX(ctx)
				client.AttendanceMonth.Create().SetStudentID(st.ID).SetCourseID(other.ID).
					SetYear(2026).SetMonth(10).SetHours(1).SaveX(ctx)
			}

			var existingID int
			if tc.rebuildExisting {
				existing := client.Invoice.Create().SetStudentID(st.ID).SetPeriodYear(2026).SetPeriodMonth(10).
					SetStatus(invoice.StatusDraft).SetTotalAmountCents(1000).SaveX(ctx)
				existingID = existing.ID
				client.InvoiceLine.Create().SetInvoiceID(existingID).SetEnrollmentID(enr.ID).
					SetDescription("Outdated line").SetQty(1).SetUnitPriceCents(1000).SetAmountCents(1000).SaveX(ctx)
			}

			var result GenerateResult
			var err error
			if tc.rebuildExisting {
				result, err = svc.RebuildStudentDraft(ctx, st.ID, 2026, 10)
			} else {
				result, err = svc.GenerateDrafts(ctx, 2026, 10)
			}
			if err != nil {
				t.Fatalf("calculate draft: %v", err)
			}
			if tc.wantTotal == 0 {
				if count := client.Invoice.Query().CountX(ctx); count != 0 {
					t.Fatalf("invoice count = %d, want 0; result: %+v", count, result)
				}
				return
			}

			iv, err := client.Invoice.Query().Where(invoice.StudentIDEQ(st.ID), invoice.PeriodYearEQ(2026), invoice.PeriodMonthEQ(10)).Only(ctx)
			if err != nil {
				t.Fatalf("find calculated invoice: %v; result: %+v", err, result)
			}
			if iv.TotalAmountCents != tc.wantTotal || iv.Status != invoice.StatusDraft {
				t.Fatalf("invoice total/status = %d/%s, want %d/draft", iv.TotalAmountCents, iv.Status, tc.wantTotal)
			}
			if tc.rebuildExisting && (iv.ID != existingID || result.Updated != 1) {
				t.Fatalf("expected existing draft %d updated, got %d; result: %+v", existingID, iv.ID, result)
			}
			lines := client.InvoiceLine.Query().Where(invoiceline.InvoiceIDEQ(iv.ID)).AllX(ctx)
			wantLines := 1
			if tc.mixed {
				wantLines++
			}
			if len(lines) != wantLines {
				t.Fatalf("line count = %d, want %d", len(lines), wantLines)
			}
			var sum int64
			var foundSubscription bool
			for _, line := range lines {
				sum += line.AmountCents
				if line.EnrollmentID == enr.ID {
					foundSubscription = true
					if line.Qty != tc.lessons || line.UnitPriceCents != tc.wantSubscriptionPrice || line.AmountCents != int64(tc.lessons)*tc.wantSubscriptionPrice {
						t.Fatalf("incorrect subscription line: %+v", line)
					}
				}
			}
			if !foundSubscription || sum != tc.wantTotal {
				t.Fatalf("subscription present = %v, line sum = %d, want %d", foundSubscription, sum, tc.wantTotal)
			}
		})
	}
}
