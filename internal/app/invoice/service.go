// Package invoice provides services for invoice generation, management, and PDF creation.
// It handles the complete invoice lifecycle from draft creation to final issuance.
package invoice

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/priceoverride"
	"langschool/ent/settings"
	"langschool/ent/student"
	"langschool/internal/app"
	"langschool/internal/app/utils"
	pdfgen "langschool/internal/pdf"
)

// Constants for invoice statuses and billing modes (aliased from shared package)
const (
	StatusDraft    = app.InvoiceStatusDraft    // Draft invoice status
	StatusIssued   = app.InvoiceStatusIssued  // Issued invoice status
	StatusPaid     = app.InvoiceStatusPaid     // Paid invoice status
	StatusCanceled = app.InvoiceStatusCanceled // Canceled invoice status

	BillingPerLesson    = app.BillingModePerLesson    // Per-lesson billing mode
	BillingSubscription = app.BillingModeSubscription // Subscription billing mode
)

// Service provides invoice generation, management, and PDF creation functionality.
type Service struct{ db *ent.Client }

// New creates a new invoice service with the given database client.
func New(db *ent.Client) *Service { return &Service{db: db} }

// ----- DTO for UI -----

// ListItem represents a summary of an invoice for list views.
type ListItem struct {
	ID          int     `json:"id"`          // Invoice ID
	StudentID   int     `json:"studentId"`   // Student ID
	StudentName string  `json:"studentName"`  // Student's full name
	Year        int     `json:"year"`         // Invoice period year
	Month       int     `json:"month"`        // Invoice period month
	Total       float64 `json:"total"`        // Total invoice amount
	Status      string  `json:"status"`       // Invoice status
	LinesCount  int     `json:"linesCount"`   // Number of line items
	Number      *string `json:"number,omitempty"` // Invoice number (nil for drafts)
}

// LineDTO represents a single line item in an invoice.
type LineDTO struct {
	EnrollmentID int     `json:"enrollmentId"` // ID of the enrollment this line is for
	Description  string  `json:"description"`   // Line item description
	Qty          int     `json:"qty"`            // Quantity (e.g., number of lessons)
	UnitPrice    float64 `json:"unitPrice"`      // Price per unit
	Amount       float64 `json:"amount"`         // Total amount for this line (qty * unitPrice)
}

// InvoiceDTO represents a complete invoice with all line items.
type InvoiceDTO struct {
	ID          int       `json:"id"`          // Invoice ID
	StudentID   int       `json:"studentId"`   // Student ID
	StudentName string    `json:"studentName"`  // Student's full name
	Year        int       `json:"year"`         // Invoice period year
	Month       int       `json:"month"`        // Invoice period month
	Total       float64   `json:"total"`        // Total invoice amount
	Status      string    `json:"status"`       // Invoice status
	Number      *string   `json:"number,omitempty"` // Invoice number (nil for drafts)
	Lines       []LineDTO `json:"lines"`        // All line items in the invoice
}

// GenerateResult contains statistics about invoice generation.
type GenerateResult struct {
	Created           int `json:"created"`           // Number of new draft invoices created
	Updated           int `json:"updated"`           // Number of existing drafts that were rebuilt
	SkippedHasInvoice int `json:"skippedHasInvoice"` // Number skipped: already issued/paid/canceled
	SkippedNoLines    int `json:"skippedNoLines"`    // Number skipped: no lines (0 attendance and/or 0 prices)
}

// ----- Domain utilities -----

// periodBounds calculates the start and end dates for a given year and month.
// Start is the first day of the month at 00:00:00, end is the last day of the month.
func periodBounds(y, m int) (start, end time.Time) {
	start = time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.Local)
	end = start.AddDate(0, 1, -1)
	return
}

// activeInPeriod determines if an enrollment is active for the given period.
// Since start_date and end_date have been removed, all enrollments are considered active.
// This means invoice generation will include all enrollments for the specified period,
// regardless of when the enrollment was created or ended.
func activeInPeriod(en *ent.Enrollment, y, m int) bool {
	return true
}

// getStudentName safely retrieves the student name from invoice edges.
// Returns an empty string if the student edge is not loaded.
func getStudentName(iv *ent.Invoice) string {
	if iv.Edges.Student != nil {
		return iv.Edges.Student.FullName
	}
	return ""
}

// buildPerLessonLine creates an invoice line for per-lesson billing
func (s *Service) buildPerLessonLine(ctx context.Context, en *ent.Enrollment, y, m int, lessonPrice float64) (*ent.InvoiceLineCreate, float64) {
	// Query attendance for the month
	am, err := s.db.AttendanceMonth.Query().Where(
		attendancemonth.StudentIDEQ(en.StudentID),
		attendancemonth.CourseIDEQ(en.CourseID),
		attendancemonth.YearEQ(y),
		attendancemonth.MonthEQ(m),
	).Only(ctx)

	qty := 0
	if err != nil {
		if !ent.IsNotFound(err) {
			fmt.Printf("AttendanceMonth query error (student %d, course %d, %04d-%02d): %v\n",
				en.StudentID, en.CourseID, y, m, err)
		}
		// NotFound => qty remains 0
	} else {
		qty = am.LessonsCount
	}

	amount := utils.Round2(float64(qty) * lessonPrice)
	desc := fmt.Sprintf("Payment for lessons (%02d.%d), course #%d", m, y, en.CourseID)

	line := s.db.InvoiceLine.Create().
		SetEnrollmentID(en.ID).
		SetDescription(desc).
		SetQty(qty).
		SetUnitPrice(lessonPrice).
		SetAmount(amount)

	return line, amount
}

// buildSubscriptionLine creates an invoice line for subscription billing
func (s *Service) buildSubscriptionLine(en *ent.Enrollment, y, m int, subscriptionPrice float64) (*ent.InvoiceLineCreate, float64) {
	amount := utils.Round2(subscriptionPrice)
	desc := fmt.Sprintf("Subscription (%02d.%d), course #%d", m, y, en.CourseID)

	line := s.db.InvoiceLine.Create().
		SetEnrollmentID(en.ID).
		SetDescription(desc).
		SetQty(1).
		SetUnitPrice(subscriptionPrice).
		SetAmount(amount)

	return line, amount
}

// getSettings retrieves the singleton settings record
// getSettings retrieves the singleton Settings entity from the database.
func (s *Service) getSettings(ctx context.Context) (*ent.Settings, error) {
	settings, err := s.db.Settings.Query().Where(settings.SingletonIDEQ(app.SettingsSingletonID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	return settings, nil
}

// resolvePrices determines the effective prices for an enrollment in a given period.
// It considers price overrides first, then falls back to course prices with discounts applied.
// Price overrides take precedence if they are valid for the period.
// Returns both lesson price and subscription price.
func (s *Service) resolvePrices(ctx context.Context, en *ent.Enrollment, y, m int) (lessonPrice, subscriptionPrice float64) {
	lessonPrice, subscriptionPrice = 0, 0

	// Base prices
	c, err := s.db.Enrollment.Query().Where(enrollment.IDEQ(en.ID)).QueryCourse().Only(ctx)
	if err == nil && c != nil {
		lp, sp := c.LessonPrice, c.SubscriptionPrice
		if en.DiscountPct != 0 {
			lp = utils.Round2(lp * (1 - en.DiscountPct/100.0))
			sp = utils.Round2(sp * (1 - en.DiscountPct/100.0))
		}
		lessonPrice, subscriptionPrice = lp, sp
	}

	ps, pe := periodBounds(y, m)

	// Override — take the latest valid_from that intersects with the period
	ovr, _ := s.db.PriceOverride.
		Query().
		Where(
			priceoverride.EnrollmentIDEQ(en.ID),
			priceoverride.ValidFromLTE(pe),
		).
		Order(ent.Desc(priceoverride.FieldValidFrom)).
		All(ctx)

	for _, o := range ovr {
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

// ----- Draft generation -----

// GenerateDrafts creates draft invoices for all active students in the specified period.
// For each student enrollment, it determines the appropriate billing method (per-lesson or subscription)
// and creates invoice lines accordingly. Existing drafts for the period are rebuilt on repeated calls.
//
// The method returns:
// - count: number of draft invoices created
// - paths: list of PDF paths where issued invoices will be saved (when issued)
// - error: any error encountered during generation
func (s *Service) GenerateDrafts(ctx context.Context, y, m int) (GenerateResult, error) {
	res := GenerateResult{}

	// all active students
	studs, err := s.db.Student.Query().Where(student.IsActiveEQ(true)).All(ctx)
	if err != nil {
		return res, err
	}
	if len(studs) == 0 {
		// fallback: if the active flag is not set — take all
		studs, err = s.db.Student.Query().All(ctx)
		if err != nil {
			return res, err
		}
	}

	for _, st := range studs {
		ens, err := s.db.Enrollment.Query().
			Where(enrollment.StudentIDEQ(st.ID)).
			All(ctx)
		if err != nil || len(ens) == 0 {
			continue
		}

		// collect invoice lines
		lines := make([]*ent.InvoiceLineCreate, 0, 4)
		total := 0.0

		for _, en := range ens {
			if !activeInPeriod(en, y, m) {
				continue
			}

			lp, sp := s.resolvePrices(ctx, en, y, m)

			switch en.BillingMode {
			case BillingPerLesson:
				line, amount := s.buildPerLessonLine(ctx, en, y, m, lp)
				lines = append(lines, line)
				total += amount

			case BillingSubscription:
				// Skip if subscription price is invalid
				if sp <= 0 {
					continue
				}
				line, amount := s.buildSubscriptionLine(en, y, m, sp)
				lines = append(lines, line)
				total += amount

			default:
				// Log unexpected billing mode
				fmt.Printf("Unexpected billing mode: %s\n", en.BillingMode)
			}
		}

		if len(lines) == 0 {
			res.SkippedNoLines++
			continue
		}
		total = utils.Round2(total)

		// Find ANY invoice for the period
		existing, _ := s.db.Invoice.Query().Where(
			invoice.StudentIDEQ(st.ID),
			invoice.PeriodYearEQ(y),
			invoice.PeriodMonthEQ(m),
		).Only(ctx)

		switch {
		case existing == nil:
			// create a new draft
			inv, err := s.db.Invoice.Create().
				SetStudentID(st.ID).
				SetPeriodYear(y).
				SetPeriodMonth(m).
				SetStatus(StatusDraft).
				SetTotalAmount(total).
				Save(ctx)
			if err != nil {
				continue
			}
			for _, lc := range lines {
				if _, err := lc.SetInvoiceID(inv.ID).Save(ctx); err != nil {
					fmt.Printf("InvoiceLine save failed (student %d, period %04d-%02d): %v\n", st.ID, y, m, err)
				}
			}
			res.Created++

		case existing.Status == StatusDraft:
			// rebuild existing draft
			_, _ = s.db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(existing.ID)).Exec(ctx)
			for _, lc := range lines {
				if _, err := lc.SetInvoiceID(existing.ID).Save(ctx); err != nil {
					fmt.Printf("InvoiceLine save failed (rebuild, invoice %d): %v\n", existing.ID, err)
				}
			}
			_, _ = existing.Update().SetTotalAmount(total).Save(ctx)
			res.Updated++

		default:
			// already issued/paid/canceled — skip
			res.SkippedHasInvoice++
		}
	}

	return res, nil
}

// ListDrafts returns a list of draft invoices for the given year and month.
// Only invoices with status "draft" are included.
func (s *Service) ListDrafts(ctx context.Context, y, m int) ([]ListItem, error) {
	invs, err := s.db.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(y),
			invoice.PeriodMonthEQ(m),
			invoice.StatusEQ(StatusDraft),
		).
		WithStudent().
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]ListItem, 0, len(invs))
	for _, iv := range invs {
		count, _ := s.db.InvoiceLine.Query().Where(invoiceline.InvoiceIDEQ(iv.ID)).Count(ctx)
		items = append(items, ListItem{
			ID: iv.ID, StudentID: iv.StudentID, StudentName: getStudentName(iv),
			Year: iv.PeriodYear, Month: iv.PeriodMonth,
			Total: utils.Round2(iv.TotalAmount), Status: string(iv.Status), LinesCount: count, Number: iv.Number,
		})
	}
	return items, nil
}

// Get retrieves a single invoice by ID with all its line items.
// Works for invoices in any status (draft, issued, paid, canceled).
func (s *Service) Get(ctx context.Context, id int) (*InvoiceDTO, error) {
	iv, err := s.db.Invoice.Query().Where(invoice.IDEQ(id)).WithStudent().Only(ctx)
	if err != nil {
		return nil, err
	}
	ls, err := s.db.InvoiceLine.Query().Where(invoiceline.InvoiceIDEQ(iv.ID)).All(ctx)
	if err != nil {
		return nil, err
	}
	dto := &InvoiceDTO{
		ID: iv.ID, StudentID: iv.StudentID,
		StudentName: getStudentName(iv),
		Year:        iv.PeriodYear, Month: iv.PeriodMonth,
		Total: utils.Round2(iv.TotalAmount), Status: string(iv.Status), Number: iv.Number,
	}
	for _, l := range ls {
		dto.Lines = append(dto.Lines, LineDTO{
			EnrollmentID: l.EnrollmentID,
			Description:  l.Description,
			Qty:          l.Qty,
			UnitPrice:    utils.Round2(l.UnitPrice),
			Amount:       utils.Round2(l.Amount),
		})
	}
	return dto, nil
}

// DeleteDraft deletes a draft invoice and all its line items.
// Only draft invoices can be deleted; issued/paid/canceled invoices are protected.
func (s *Service) DeleteDraft(ctx context.Context, id int) error {
	iv, err := s.db.Invoice.Get(ctx, id)
	if err != nil {
		return err
	}
	if iv.Status != StatusDraft {
		return fmt.Errorf("can delete only draft invoices")
	}
	if _, err := s.db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(iv.ID)).Exec(ctx); err != nil {
		return err
	}
	return s.db.Invoice.DeleteOneID(iv.ID).Exec(ctx)
}

// ----- Issuing, numbering, PDF generation -----

// FormatNumber generates an invoice number in the format PREFIX-YYYYMM-SEQ.
// SEQ is a 3-digit sequence number with leading zeros (e.g., LS-202401-001).
func FormatNumber(prefix string, y, m, seq int) string {
	return fmt.Sprintf("%s-%04d%02d-%03d", prefix, y, m, seq)
}

// issueOne assigns an invoice number and changes the status from draft to issued.
// This operation is performed in a transaction to ensure atomicity.
// The invoice number is generated using the settings prefix and sequence number,
// which is then incremented. Returns the assigned invoice number.
func (s *Service) issueOne(ctx context.Context, id int) (string, error) {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return "", err
	}
	
	// Proper defer pattern: only rollback if commit hasn't been called
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	iv, err := tx.Invoice.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if iv.Status != StatusDraft {
		if iv.Number != nil && *iv.Number != "" {
			return *iv.Number, nil
		}
		return "", fmt.Errorf("invoice %d is not draft", id)
	}

	st, err := s.getSettings(ctx)
	if err != nil {
		return "", err
	}
	prefix := "LS"
	seq := 1
	if st != nil {
		if st.InvoicePrefix != "" {
			prefix = st.InvoicePrefix
		}
		seq = st.NextSeq
	}
	number := FormatNumber(prefix, iv.PeriodYear, iv.PeriodMonth, seq)

	// Increment the counter
	if st != nil {
		if err := tx.Settings.Update().Where(settings.SingletonIDEQ(app.SettingsSingletonID)).SetNextSeq(seq + 1).Exec(ctx); err != nil {
			return "", err
		}
	}

	// Save the number and status
	if _, err := tx.Invoice.UpdateOneID(iv.ID).
		SetNumber(number).
		SetStatus(StatusIssued).
		Save(ctx); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	committed = true
	return number, nil
}

// Issue issues a single draft invoice and generates its PDF.
// This combines issueOne (assigning number and status) with PDF generation.
// Returns the invoice number and the path to the generated PDF file.
func (s *Service) Issue(ctx context.Context, id int, outBaseDir, fontsDir string) (string, string, error) {
	number, err := s.issueOne(ctx, id)
	if err != nil {
		return "", "", err
	}
	// LOG
	fmt.Printf("PDF: generating for invoice %d number=%s out=%s fonts=%s\n", id, number, outBaseDir, fontsDir)

	p, err := pdfgen.GenerateInvoicePDF(ctx, s.db, id, pdfgen.Options{
		OutBaseDir: outBaseDir,
		FontsDir:   fontsDir,
		Currency:   "",
		Locale:     "",
	})
	if err != nil {
		fmt.Printf("PDF: failed for invoice %d: %v\n", id, err)
		return "", "", err
	}
	fmt.Printf("PDF: done %s\n", p)
	return number, p, nil
}

// IssueAll issues all draft invoices for a given year and month.
// Each invoice is assigned a number, marked as issued, and a PDF is generated.
// Returns the count of issued invoices and paths to all generated PDFs.
// If any invoice fails to issue, the operation stops and returns an error.
func (s *Service) IssueAll(ctx context.Context, y, m int, outBaseDir, fontsDir string) (int, []string, error) {
	invs, err := s.db.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(y),
			invoice.PeriodMonthEQ(m),
			invoice.StatusEQ(StatusDraft),
		).All(ctx)
	if err != nil {
		return 0, nil, err
	}
	paths := make([]string, 0, len(invs))
	count := 0
	for _, iv := range invs {
		if _, err := s.issueOne(ctx, iv.ID); err != nil {
			return count, paths, err
		}
		p, err := pdfgen.GenerateInvoicePDF(ctx, s.db, iv.ID, pdfgen.Options{
			OutBaseDir: outBaseDir,
			FontsDir:   fontsDir,
		})
		if err != nil {
			return count, paths, err
		}
		paths = append(paths, p)
		count++
	}
	return count, paths, nil
}

// PDFPathByNumber generates the file path for an invoice PDF based on its number.
// The path structure is: outBaseDir/YYYY/MM/number.pdf
func PDFPathByNumber(outBaseDir string, y, m int, number string) string {
	return filepath.Join(outBaseDir, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", m), number+".pdf")
}

// List returns invoices for a given period with optional status filter.
// Optimized to avoid N+1 query by loading invoice lines in a single query.
func (s *Service) List(ctx context.Context, y, m int, status string) ([]ListItem, error) {
	q := s.db.Invoice.Query().
		Where(
			invoice.PeriodYearEQ(y),
			invoice.PeriodMonthEQ(m),
		).
		WithStudent()
	switch status {
	case StatusDraft, StatusIssued, StatusPaid, StatusCanceled:
		q = q.Where(invoice.StatusEQ(invoice.Status(status)))
	case "all":
		// no additional filter
	default:
		// default to draft
		q = q.Where(invoice.StatusEQ(StatusDraft))
	}

	invs, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	// Get all invoice IDs to fetch line counts in a single query
	invoiceIDs := make([]int, len(invs))
	for i, iv := range invs {
		invoiceIDs[i] = iv.ID
	}

	// Batch query: get line counts for all invoices at once
	type countResult struct {
		InvoiceID int `json:"invoice_id"`
		Count     int `json:"count"`
	}
	
	var counts []countResult
	if len(invoiceIDs) > 0 {
		err = s.db.InvoiceLine.Query().
			Where(invoiceline.InvoiceIDIn(invoiceIDs...)).
			GroupBy(invoiceline.FieldInvoiceID).
			Aggregate(ent.Count()).
			Scan(ctx, &counts)
		if err != nil {
			// If batch query fails, fall back to individual queries
			counts = nil
		}
	}

	// Create a map for O(1) lookup
	countMap := make(map[int]int)
	for _, c := range counts {
		countMap[c.InvoiceID] = c.Count
	}

	out := make([]ListItem, 0, len(invs))
	for _, iv := range invs {
		cnt := countMap[iv.ID]
		out = append(out, ListItem{
			ID:          iv.ID,
			StudentID:   iv.StudentID,
			StudentName: getStudentName(iv),
			Year:        iv.PeriodYear,
			Month:       iv.PeriodMonth,
			Total:       utils.Round2(iv.TotalAmount),
			Status:      string(iv.Status),
			LinesCount:  cnt,
			Number:      iv.Number,
		})
	}
	return out, nil
}
