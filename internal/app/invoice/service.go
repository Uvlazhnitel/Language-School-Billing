package invoice

import (
	"context"
	"fmt"
	"math"
	"time"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/priceoverride"
	"langschool/ent/student"
)

type Service struct{ db *ent.Client }

func New(db *ent.Client) *Service { return &Service{db: db} }

// --- DTO for UI ---
type ListItem struct {
	ID          int     `json:"id"`
	StudentID   int     `json:"studentId"`
	StudentName string  `json:"studentName"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	Total       float64 `json:"total"`
	Status      string  `json:"status"`
	LinesCount  int     `json:"linesCount"`
	Number      *string `json:"number,omitempty"`
}

type LineDTO struct {
	EnrollmentID int     `json:"enrollmentId"`
	Description  string  `json:"description"`
	Qty          int     `json:"qty"`
	UnitPrice    float64 `json:"unitPrice"`
	Amount       float64 `json:"amount"`
}

type InvoiceDTO struct {
	ID          int       `json:"id"`
	StudentID   int       `json:"studentId"`
	StudentName string    `json:"studentName"`
	Year        int       `json:"year"`
	Month       int       `json:"month"`
	Total       float64   `json:"total"`
	Status      string    `json:"status"`
	Number      *string   `json:"number,omitempty"`
	Lines       []LineDTO `json:"lines"`
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// --- Helper for period dates ---
func periodBounds(y, m int) (start, end time.Time) {
	start = time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.Local)
	end = start.AddDate(0, 1, -1) // last day of the month
	return
}

// Select override for the period. Take the latest valid_from that falls within the period.
func (s *Service) resolvePrices(ctx context.Context, en *ent.Enrollment, y, m int) (lessonPrice, subscriptionPrice float64) {
	lessonPrice = 0
	subscriptionPrice = 0

	// base prices from the course and discount
	c, err := s.db.Enrollment.Query().
		Where(enrollment.IDEQ(en.ID)).
		QueryCourse().Only(ctx)
	if err == nil && c != nil {
		lp := c.LessonPrice
		sp := c.SubscriptionPrice
		if en.DiscountPct != 0 {
			lp = round2(lp * (1 - en.DiscountPct/100.0))
			sp = round2(sp * (1 - en.DiscountPct/100.0))
		}
		lessonPrice, subscriptionPrice = lp, sp
	}

	ps, pe := periodBounds(y, m)

	// override
	ovr, _ := s.db.PriceOverride.
		Query().
		Where(
			priceoverride.EnrollmentIDEQ(en.ID),
			priceoverride.ValidFromLTE(pe),
		).
		Order(ent.Desc(priceoverride.FieldValidFrom)).
		All(ctx)

	for _, o := range ovr {
		// Does the period fall within valid_from..valid_to?
		if o.ValidTo == nil || !o.ValidTo.Before(ps) {
			if o.LessonPrice != nil {
				lessonPrice = *o.LessonPrice
			}
			if o.SubscriptionPrice != nil {
				subscriptionPrice = *o.SubscriptionPrice
			}
			break
		}
	}
	return
}

// Is the enrollment active in the period
func activeInPeriod(en *ent.Enrollment, y, m int) bool {
	ps, pe := periodBounds(y, m)
	if en.StartDate.After(pe) {
		return false
	}
	if en.EndDate != nil && en.EndDate.Before(ps) {
		return false
	}
	return true
}

// Generate draft invoices for students for (y, m). Rebuilds if drafts already exist.
func (s *Service) GenerateDrafts(ctx context.Context, y, m int) (int, error) {
	// fetch all students with enrollments
	studs, err := s.db.Student.Query().Where(student.IsActiveEQ(true)).All(ctx)
	if err != nil {
		return 0, err
	}

	created := 0

	for _, st := range studs {
		ens, err := s.db.Enrollment.Query().
			Where(
				enrollment.StudentIDEQ(st.ID),
			).
			All(ctx)
		if err != nil || len(ens) == 0 {
			continue
		}

		lines := make([]*ent.InvoiceLineCreate, 0, 4)
		total := 0.0

		for _, en := range ens {
			if !activeInPeriod(en, y, m) {
				continue
			}

			lp, sp := s.resolvePrices(ctx, en, y, m)
			switch en.BillingMode {
			case "per_lesson":
				am, _ := s.db.AttendanceMonth.Query().Where(
					attendancemonth.StudentIDEQ(en.StudentID),
					attendancemonth.CourseIDEQ(en.CourseID),
					attendancemonth.YearEQ(y),
					attendancemonth.MonthEQ(m),
				).Only(ctx)
				qty := 0
				if am != nil {
					qty = am.LessonsCount
				}
				if qty <= 0 || lp <= 0 {
					continue
				}
				amount := round2(float64(qty) * lp)
				desc := fmt.Sprintf("Payment for lessons (%02d.%d), course #%d", m, y, en.CourseID)
				lines = append(lines, s.db.InvoiceLine.Create().
					SetEnrollmentID(en.ID).
					SetDescription(desc).
					SetQty(qty).
					SetUnitPrice(lp).
					SetAmount(amount))
				total += amount

			case "subscription":
				if sp <= 0 {
					continue
				}
				amount := round2(sp)
				desc := fmt.Sprintf("Subscription (%02d.%d), course #%d", m, y, en.CourseID)
				lines = append(lines, s.db.InvoiceLine.Create().
					SetEnrollmentID(en.ID).
					SetDescription(desc).
					SetQty(1).
					SetUnitPrice(sp).
					SetAmount(amount))
				total += amount
			}
		}

		if len(lines) == 0 {
			// if the student has no lines for this month — skip
			continue
		}

		total = round2(total)

		// if a draft already exists for the period — rebuild it
		existing, _ := s.db.Invoice.Query().Where(
			invoice.StudentIDEQ(st.ID),
			invoice.PeriodYearEQ(y),
			invoice.PeriodMonthEQ(m),
			invoice.StatusEQ("draft"),
		).Only(ctx)

		if existing != nil {
			// delete old lines, update total
			_, _ = s.db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(existing.ID)).Exec(ctx)
			for _, lc := range lines {
				_, _ = lc.SetInvoiceID(existing.ID).Save(ctx)
			}
			_, _ = existing.Update().SetTotalAmount(total).Save(ctx)
		} else {
			// create a new invoice and lines
			inv, err := s.db.Invoice.Create().
				SetStudentID(st.ID).
				SetPeriodYear(y).
				SetPeriodMonth(m).
				SetStatus("draft").
				SetTotalAmount(total).
				Save(ctx)
			if err != nil {
				continue
			}
			for _, lc := range lines {
				_, _ = lc.SetInvoiceID(inv.ID).Save(ctx)
			}
			created++
		}
	}
	return created, nil
}

// List drafts for the month
func (s *Service) ListDrafts(ctx context.Context, y, m int) ([]ListItem, error) {
	invs, err := s.db.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(y),
			invoice.PeriodMonthEQ(m),
			invoice.StatusEQ("draft"),
		).
		WithStudent().
		All(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, 0, len(invs))
	for _, iv := range invs {
		count, _ := s.db.InvoiceLine.Query().Where(invoiceline.InvoiceIDEQ(iv.ID)).Count(ctx)
		name := ""
		if iv.Edges.Student != nil {
			name = iv.Edges.Student.FullName
		}
		items = append(items, ListItem{
			ID: iv.ID, StudentID: iv.StudentID, StudentName: name,
			Year: iv.PeriodYear, Month: iv.PeriodMonth,
			Total: round2(iv.TotalAmount), Status: string(iv.Status), LinesCount: count, Number: iv.Number,
		})
	}
	return items, nil
}

// Get invoice with lines
func (s *Service) Get(ctx context.Context, id int) (*InvoiceDTO, error) {
	iv, err := s.db.Invoice.Query().
		Where(invoice.IDEQ(id)).
		WithStudent().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	ls, err := s.db.InvoiceLine.Query().Where(invoiceline.InvoiceIDEQ(iv.ID)).All(ctx)
	if err != nil {
		return nil, err
	}
	dto := &InvoiceDTO{
		ID: iv.ID, StudentID: iv.StudentID,
		StudentName: func() string {
			if iv.Edges.Student != nil {
				return iv.Edges.Student.FullName
			}
			return ""
		}(),
		Year: iv.PeriodYear, Month: iv.PeriodMonth,
		Total: round2(iv.TotalAmount), Status: string(iv.Status), Number: iv.Number,
	}
	for _, l := range ls {
		dto.Lines = append(dto.Lines, LineDTO{
			EnrollmentID: l.EnrollmentID,
			Description:  l.Description,
			Qty:          l.Qty,
			UnitPrice:    round2(l.UnitPrice),
			Amount:       round2(l.Amount),
		})
	}
	return dto, nil
}

// Delete draft (if needed)
func (s *Service) DeleteDraft(ctx context.Context, id int) error {
	iv, err := s.db.Invoice.Get(ctx, id)
	if err != nil {
		return err
	}
	if iv.Status != "draft" {
		return fmt.Errorf("can delete only draft invoices")
	}
	if _, err := s.db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(iv.ID)).Exec(ctx); err != nil {
		return err
	}
	return s.db.Invoice.DeleteOneID(iv.ID).Exec(ctx)
}
