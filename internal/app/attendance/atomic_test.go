package attendance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"langschool/ent"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"langschool/ent/invoice"
)

func TestUpsertRollsBackAttendanceAndInvoiceOnRebuildFailure(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing=%t", existing), func(t *testing.T) {
			ctx := context.Background()
			db := enttest.Open(t, "sqlite3", fmt.Sprintf("file:atomic-%t?mode=memory&_fk=1", existing))
			defer db.Close()
			st := db.Student.Create().SetFullName("Student").SaveX(ctx)
			crs := db.Course.Create().SetName("Course").SetType(course.TypeGroup).SetLessonPriceCents(2000).SetSubscriptionPriceCents(0).SaveX(ctx)
			db.Enrollment.Create().SetStudentID(st.ID).SetCourseID(crs.ID).SetBillingMode(enrollment.BillingModePerLesson).SetChargeMaterials(false).SaveX(ctx)
			svc := New(db)
			if existing {
				if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 9, 2); err != nil {
					t.Fatal(err)
				}
			}
			injected := errors.New("invoice line write failed")
			db.InvoiceLine.Use(func(next ent.Mutator) ent.Mutator {
				return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
					if mutation.Op().Is(ent.OpCreate) {
						return nil, injected
					}
					return next.Mutate(ctx, mutation)
				})
			})
			if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 9, 4); !errors.Is(err, injected) {
				t.Fatalf("error = %v", err)
			}
			if !existing {
				if count := db.AttendanceMonth.Query().CountX(ctx); count != 0 {
					t.Fatalf("attendance leaked: %d", count)
				}
				if count := db.Invoice.Query().CountX(ctx); count != 0 {
					t.Fatalf("invoice leaked: %d", count)
				}
				if count := db.InvoiceLine.Query().CountX(ctx); count != 0 {
					t.Fatalf("line leaked: %d", count)
				}
			} else {
				if got := db.AttendanceMonth.Query().OnlyX(ctx).Hours; got != 2 {
					t.Fatalf("hours = %v", got)
				}
				if got := db.Invoice.Query().OnlyX(ctx).TotalAmountCents; got != 4000 {
					t.Fatalf("total = %d", got)
				}
				line := db.InvoiceLine.Query().OnlyX(ctx)
				if line.Qty != 2 || line.AmountCents != 4000 {
					t.Fatalf("line changed: %+v", line)
				}
			}
		})
	}
}

func TestAttendanceReadErrorsAreNotMissingRecords(t *testing.T) {
	ctx := context.Background()
	db := enttest.Open(t, "sqlite3", "file:attendance-read-error?mode=memory&_fk=1")
	defer db.Close()
	st := db.Student.Create().SetFullName("Student").SaveX(ctx)
	crs := db.Course.Create().SetName("Course").SetType(course.TypeGroup).SetLessonPriceCents(1000).SetSubscriptionPriceCents(0).SaveX(ctx)
	db.Enrollment.Create().SetStudentID(st.ID).SetCourseID(crs.ID).SetBillingMode(enrollment.BillingModePerLesson).SaveX(ctx)
	injected := errors.New("attendance read failed")
	db.AttendanceMonth.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) { return nil, injected })
	}))
	if _, err := New(db).ListPerLesson(ctx, 2026, 9, nil); !errors.Is(err, injected) {
		t.Fatalf("List error = %v", err)
	}
	if err := New(db).Upsert(ctx, st.ID, crs.ID, 2026, 9, 2); !errors.Is(err, injected) {
		t.Fatalf("Upsert error = %v", err)
	}
}

func TestQuarterQuantitiesAndTeacherIdentity(t *testing.T) {
	for _, mode := range []enrollment.BillingMode{enrollment.BillingModePerLesson, enrollment.BillingModeSubscription} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := context.Background()
			db := enttest.Open(t, "sqlite3", "file:quarter-"+string(mode)+"?mode=memory&_fk=1")
			defer db.Close()
			teacher := db.Teacher.Create().SetFullName("Teacher").SaveX(ctx)
			st := db.Student.Create().SetFullName("Student").SaveX(ctx)
			crs := db.Course.Create().SetName("Course").SetTeacherID(teacher.ID).SetType(course.TypeGroup).SetLessonPriceCents(2000).SetSubscriptionPriceCents(0).SaveX(ctx)
			db.Enrollment.Create().SetStudentID(st.ID).SetCourseID(crs.ID).SetBillingMode(mode).SetLessonPriceOverrideCents(1000).SetSubscriptionLessonPriceCents(1200).SetChargeMaterials(false).SaveX(ctx)
			svc := New(db)
			if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 9, 0); err != nil {
				t.Fatal(err)
			}
			rows, err := svc.ListPerLesson(ctx, 2026, 9, nil)
			if err != nil || len(rows) != 1 || !rows[0].HasRecord || rows[0].Hours != 0 || rows[0].TeacherID == nil || *rows[0].TeacherID != teacher.ID {
				t.Fatalf("rows=%+v, err=%v", rows, err)
			}
			if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 9, 1.25); err != nil {
				t.Fatal(err)
			}
			want := int64(1250)
			if mode == enrollment.BillingModeSubscription {
				want = 1500
			}
			iv := db.Invoice.Query().OnlyX(ctx)
			if iv.TotalAmountCents != want {
				t.Fatalf("total = %d want %d", iv.TotalAmountCents, want)
			}
			db.Invoice.UpdateOneID(iv.ID).SetStatus(invoice.StatusIssued).ExecX(ctx)
			if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 9, 2); err == nil {
				t.Fatal("issued invoice must lock attendance")
			}
			if got := db.AttendanceMonth.Query().OnlyX(ctx).Hours; got != 1.25 {
				t.Fatalf("locked hours changed to %v", got)
			}
		})
	}
}

// A second SQLite writer must already be excluded when invoice checks begin.
func TestUpsertReservesWriterBeforeInvoiceRead(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "reservation.sqlite") + "?_fk=1"
	db := enttest.Open(t, "sqlite3", dsn)
	defer db.Close()
	other, err := ent.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	st := db.Student.Create().SetFullName("Student").SaveX(ctx)
	crs := db.Course.Create().SetName("Course").SetType(course.TypeGroup).SetLessonPriceCents(1000).SetSubscriptionPriceCents(0).SaveX(ctx)
	db.Enrollment.Create().SetStudentID(st.ID).SetCourseID(crs.ID).SetBillingMode(enrollment.BillingModePerLesson).SaveX(ctx)
	checked := false
	db.Invoice.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
			if !checked {
				checked = true
				timeout, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
				defer cancel()
				competing, err := other.BeginTx(timeout, &sql.TxOptions{Isolation: sql.LevelSerializable})
				if err == nil {
					competing.Rollback()
					t.Error("another writer acquired SQLite during invoice check")
				}
			}
			return next.Query(ctx, query)
		})
	}))
	if err := New(db).Upsert(ctx, st.ID, crs.ID, 2026, 9, 1); err != nil {
		t.Fatal(err)
	}
	if !checked {
		t.Fatal("invoice check did not run")
	}
}

func TestUpsertRollsBackWhenPriceReadFailsAfterAttendanceWrite(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			ctx := context.Background()
			db := enttest.Open(t, "sqlite3", fmt.Sprintf("file:price-read-rollback-%t?mode=memory&_fk=1", existing))
			defer db.Close()
			st := db.Student.Create().SetFullName("Student").SaveX(ctx)
			crs := db.Course.Create().SetName("Course").SetType(course.TypeGroup).SetLessonPriceCents(2000).SetSubscriptionPriceCents(0).SaveX(ctx)
			db.Enrollment.Create().SetStudentID(st.ID).SetCourseID(crs.ID).SetBillingMode(enrollment.BillingModeSubscription).SetSubscriptionLessonPriceCents(1500).SetChargeMaterials(false).SaveX(ctx)
			svc := New(db)
			if existing {
				if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 9, 2); err != nil {
					t.Fatal(err)
				}
			}
			injected := errors.New("price read failed")
			db.Course.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
				return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) { return nil, injected })
			}))
			if err := svc.Upsert(ctx, st.ID, crs.ID, 2026, 9, 4); !errors.Is(err, injected) {
				t.Fatalf("error = %v", err)
			}
			if existing {
				if got := db.AttendanceMonth.Query().OnlyX(ctx).Hours; got != 2 {
					t.Fatalf("hours changed: %v", got)
				}
				if got := db.Invoice.Query().OnlyX(ctx).TotalAmountCents; got != 3000 {
					t.Fatalf("invoice changed: %d", got)
				}
			} else if db.AttendanceMonth.Query().CountX(ctx) != 0 || db.Invoice.Query().CountX(ctx) != 0 {
				t.Fatal("failed write was not rolled back")
			}
		})
	}
}
