package invoice

import (
	"context"
	"errors"
	"langschool/ent"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"testing"
)

func TestRebuildPropagatesFinancialReadFailures(t *testing.T) {
	for _, entity := range []string{"attendance", "course"} {
		t.Run(entity, func(t *testing.T) {
			ctx := context.Background()
			db := enttest.Open(t, "sqlite3", "file:invoice-read-error-"+entity+"?mode=memory&_fk=1")
			defer db.Close()
			st := db.Student.Create().SetFullName("Student").SaveX(ctx)
			c := db.Course.Create().SetName("Course").SetType(course.TypeGroup).SetLessonPriceCents(1000).SetSubscriptionPriceCents(0).SaveX(ctx)
			db.Enrollment.Create().SetStudentID(st.ID).SetCourseID(c.ID).SetBillingMode(enrollment.BillingModeSubscription).SetSubscriptionLessonPriceCents(1500).SaveX(ctx)
			db.AttendanceMonth.Create().SetStudentID(st.ID).SetCourseID(c.ID).SetYear(2026).SetMonth(9).SetHours(4).SaveX(ctx)
			svc := New(db)
			if _, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 9); err != nil {
				t.Fatal(err)
			}
			before := db.Invoice.Query().OnlyX(ctx).TotalAmountCents
			injected := errors.New("financial read failed")
			interceptor := ent.InterceptFunc(func(next ent.Querier) ent.Querier {
				return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) { return nil, injected })
			})
			if entity == "attendance" {
				db.AttendanceMonth.Intercept(interceptor)
			} else {
				db.Course.Intercept(interceptor)
			}
			if _, err := svc.RebuildStudentDraft(ctx, st.ID, 2026, 9); !errors.Is(err, injected) {
				t.Fatalf("error = %v", err)
			}
			if got := db.Invoice.Query().OnlyX(ctx).TotalAmountCents; got != before {
				t.Fatalf("total changed: %d", got)
			}
			if count := db.InvoiceLine.Query().CountX(ctx); count == 0 {
				t.Fatal("invoice lines lost")
			}
		})
	}
}
