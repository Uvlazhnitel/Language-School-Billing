// Package invoice provides services for invoice generation, management, and PDF creation.
// It handles the complete invoice lifecycle from draft creation to final issuance.
package invoice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/coursemonthstat"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/payment"
	"langschool/ent/settings"
	"langschool/ent/student"
	"langschool/internal/app"
	paysvc "langschool/internal/app/payment"
	"langschool/internal/app/recipient"
	"langschool/internal/apperrors"
	"langschool/internal/money"
	pdfgen "langschool/internal/pdf"
)

// Constants for invoice statuses and billing modes (aliased from shared package)
const (
	StatusDraft            = app.InvoiceStatusDraft            // Draft invoice status
	StatusIssuedPendingPDF = app.InvoiceStatusIssuedPendingPDF // Issued without a ready PDF
	StatusIssued           = app.InvoiceStatusIssued           // Issued invoice status
	StatusPaidPendingPDF   = app.InvoiceStatusPaidPendingPDF   // Paid without a ready PDF
	StatusPaid             = app.InvoiceStatusPaid             // Paid invoice status
	StatusCanceled         = app.InvoiceStatusCanceled         // Canceled invoice status

	BillingPerLesson    = app.BillingModePerLesson    // Per-lesson billing mode
	BillingSubscription = app.BillingModeSubscription // Subscription billing mode

	materialsLineDescription = "Mācību materiāli"
	materialsLineAmountCents = int64(500)
)

var currentTime = time.Now

// Service provides invoice generation, management, and PDF creation functionality.
type Service struct {
	db          *ent.Client
	invoicesDir string
}

// New creates a new invoice service with the given database client.
func New(db *ent.Client) *Service { return &Service{db: db} }

// NewWithInvoicesDir creates a service that can invalidate generated invoice
// PDFs after successful rebuilds of issued or paid invoices.
func NewWithInvoicesDir(db *ent.Client, invoicesDir string) *Service {
	return &Service{db: db, invoicesDir: invoicesDir}
}

// ----- DTO for UI -----

// ListItem represents a summary of an invoice for list views.
type ListItem struct {
	ID            int     `json:"id"`               // Invoice ID
	Version       int     `json:"version"`          // Optimistic-lock revision
	StudentID     int     `json:"studentId"`        // Student ID
	StudentName   string  `json:"studentName"`      // Student's full name
	Year          int     `json:"year"`             // Invoice period year
	Month         int     `json:"month"`            // Invoice period month
	Total         float64 `json:"total"`            // Total invoice amount
	Status        string  `json:"status"`           // Invoice status
	PDFReady      bool    `json:"pdfReady"`         // Whether canonical PDF is ready
	LinesCount    int     `json:"linesCount"`       // Number of line items
	Number        *string `json:"number,omitempty"` // Invoice number (nil for drafts)
	EventDate     string  `json:"eventDate"`        // Real timeline event date for draft/update/issue/pay
	LastEmailedAt string  `json:"lastEmailedAt,omitempty"`
	LastEmailedTo string  `json:"lastEmailedTo,omitempty"`
}

// LineDTO represents a single line item in an invoice.
type LineDTO struct {
	EnrollmentID int     `json:"enrollmentId"` // ID of the enrollment this line is for
	Description  string  `json:"description"`  // Line item description
	Qty          float64 `json:"qty"`          // Quantity (e.g., number of hours)
	UnitPrice    float64 `json:"unitPrice"`    // Price per unit
	Amount       float64 `json:"amount"`       // Total amount for this line (qty * unitPrice)
}

// InvoiceDTO represents a complete invoice with all line items.
type InvoiceDTO struct {
	ID                  int       `json:"id"`                  // Invoice ID
	Version             int       `json:"version"`             // Optimistic-lock revision
	StudentID           int       `json:"studentId"`           // Student ID
	StudentName         string    `json:"studentName"`         // Student's full name
	RecipientName       string    `json:"recipientName"`       // Visible invoice recipient
	RecipientPhone      string    `json:"recipientPhone"`      // Optional recipient phone
	RecipientEmail      string    `json:"recipientEmail"`      // Optional recipient email
	ChildName           string    `json:"childName"`           // Child/student name
	StudentPersonalCode string    `json:"studentPersonalCode"` // Student's own personal code
	IsMinor             bool      `json:"isMinor"`             // Whether invoice is for a minor student
	Year                int       `json:"year"`                // Invoice period year
	Month               int       `json:"month"`               // Invoice period month
	Total               float64   `json:"total"`               // Total invoice amount
	Status              string    `json:"status"`              // Invoice status
	PDFReady            bool      `json:"pdfReady"`            // Whether canonical PDF is ready
	Number              *string   `json:"number,omitempty"`    // Invoice number (nil for drafts)
	LastEmailedAt       string    `json:"lastEmailedAt,omitempty"`
	LastEmailedTo       string    `json:"lastEmailedTo,omitempty"`
	Lines               []LineDTO `json:"lines"` // All line items in the invoice
}

// GenerateResult contains statistics about invoice generation.
type GenerateResult struct {
	Created           int `json:"created"`           // Number of new draft invoices created
	Updated           int `json:"updated"`           // Number of existing drafts that were rebuilt
	SkippedHasInvoice int `json:"skippedHasInvoice"` // Number skipped: already issued/paid/canceled
	SkippedNoLines    int `json:"skippedNoLines"`    // Number skipped: no lines (0 attendance and/or 0 prices)
}

// ----- Domain utilities -----

// getStudentName safely retrieves the student name from invoice edges.
// Returns an empty string if the student edge is not loaded.
func getStudentName(iv *ent.Invoice) string {
	if iv.Edges.Student != nil {
		return iv.Edges.Student.FullName
	}
	return ""
}

func optionalRFC3339(ts *time.Time) string {
	if ts == nil || ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func buildCourseLineDescription(courseName string) string {
	courseName = strings.TrimSpace(courseName)
	if courseName == "" {
		return "Dalības maksa par mācību pakalpojumiem"
	}
	return fmt.Sprintf("Dalības maksa par %s", courseName)
}

// buildPerLessonLine creates an invoice line for per-lesson billing
func (s *Service) buildPerLessonLine(ctx context.Context, en *ent.Enrollment, y, m int, lessonPriceCents int64) (*ent.InvoiceLineCreate, int64) {
	// Query attendance for the month
	am, err := s.db.AttendanceMonth.Query().Where(
		attendancemonth.StudentIDEQ(en.StudentID),
		attendancemonth.CourseIDEQ(en.CourseID),
		attendancemonth.YearEQ(y),
		attendancemonth.MonthEQ(m),
	).Only(ctx)

	qty := 0.0
	if err != nil {
		if !ent.IsNotFound(err) {
			fmt.Printf("AttendanceMonth query error (student %d, course %d, %04d-%02d): %v\n",
				en.StudentID, en.CourseID, y, m, err)
		}
		// NotFound => qty remains 0
	} else {
		qty = am.Hours
	}

	amountCents := money.MulFloatToCents(qty, lessonPriceCents)
	courseName := ""
	if c, err := s.db.Course.Get(ctx, en.CourseID); err == nil {
		courseName = c.Name
	}
	desc := buildCourseLineDescription(courseName)

	line := s.db.InvoiceLine.Create().
		SetEnrollmentID(en.ID).
		SetDescription(desc).
		SetQty(qty).
		SetUnitPriceCents(lessonPriceCents).
		SetAmountCents(amountCents)

	return line, amountCents
}

// buildSubscriptionLine creates an invoice line for subscription billing
func (s *Service) buildSubscriptionLine(ctx context.Context, en *ent.Enrollment, y, m int, lessonPriceCents int64, lessonsHeld float64) (*ent.InvoiceLineCreate, int64) {
	unitPriceCents := en.SubscriptionLessonPriceCents
	if unitPriceCents < 0 {
		unitPriceCents = lessonPriceCents
	}
	amountCents := money.MulFloatToCents(lessonsHeld, unitPriceCents)
	courseName := ""
	if c, err := s.db.Course.Get(ctx, en.CourseID); err == nil {
		courseName = c.Name
	}
	desc := buildCourseLineDescription(courseName)

	line := s.db.InvoiceLine.Create().
		SetEnrollmentID(en.ID).
		SetDescription(desc).
		SetQty(lessonsHeld).
		SetUnitPriceCents(unitPriceCents).
		SetAmountCents(amountCents)

	return line, amountCents
}

// buildMaterialsLine creates a fixed invoice line for monthly learning materials.
// InvoiceLine requires an enrollment reference, so this line is attached to one of
// the student's enrollments for the billing period.
func (s *Service) buildMaterialsLine(enrollmentID int) (*ent.InvoiceLineCreate, int64) {
	line := s.db.InvoiceLine.Create().
		SetEnrollmentID(enrollmentID).
		SetDescription(materialsLineDescription).
		SetQty(1).
		SetUnitPriceCents(materialsLineAmountCents).
		SetAmountCents(materialsLineAmountCents)

	return line, materialsLineAmountCents
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

func (s *Service) getSettingsOrDefaults(ctx context.Context) (string, int, bool, error) {
	st, err := s.db.Settings.Query().Where(settings.SingletonIDEQ(app.SettingsSingletonID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "LS", 1, false, nil
		}
		return "", 0, false, fmt.Errorf("failed to get settings: %w", err)
	}

	prefix := strings.TrimSpace(st.InvoicePrefix)
	if prefix == "" {
		prefix = "LS"
	}
	seq := st.NextSeq
	if seq <= 0 {
		seq = 1
	}
	return prefix, seq, true, nil
}

// resolvePrices determines the effective prices for an enrollment in a given period.
// It uses the course prices and applies the enrollment discount, if any.
// Returns both lesson price and subscription price.
func (s *Service) resolvePrices(ctx context.Context, en *ent.Enrollment, y, m int) (lessonPriceCents, subscriptionPriceCents int64) {
	lessonPriceCents, subscriptionPriceCents = 0, 0

	// Base prices
	c, err := s.db.Enrollment.Query().Where(enrollment.IDEQ(en.ID)).QueryCourse().Only(ctx)
	if err == nil && c != nil {
		lp, sp := c.LessonPriceCents, c.SubscriptionPriceCents
		if en.DiscountPct != 0 {
			lp = int64(float64(lp) * (1 - en.DiscountPct/100.0))
			sp = int64(float64(sp) * (1 - en.DiscountPct/100.0))
		}
		lessonPriceCents, subscriptionPriceCents = lp, sp
	}

	return lessonPriceCents, subscriptionPriceCents
}

func (s *Service) subscriptionLessonsHeld(ctx context.Context, courseID, y, m int) float64 {
	item, err := s.db.CourseMonthStat.Query().
		Where(
			coursemonthstat.CourseIDEQ(courseID),
			coursemonthstat.YearEQ(y),
			coursemonthstat.MonthEQ(m),
		).
		Only(ctx)
	if err != nil {
		return 0
	}
	return item.SubscriptionLessonsHeld
}

func (s *Service) hasAnyLessonsInMonth(ctx context.Context, ens []*ent.Enrollment, y, m int) bool {
	for _, en := range ens {
		switch en.BillingMode {
		case BillingSubscription:
			if s.subscriptionLessonsHeld(ctx, en.CourseID, y, m) > 0 {
				return true
			}
		default:
			count, err := s.db.AttendanceMonth.Query().
				Where(
					attendancemonth.StudentIDEQ(en.StudentID),
					attendancemonth.CourseIDEQ(en.CourseID),
					attendancemonth.YearEQ(y),
					attendancemonth.MonthEQ(m),
					attendancemonth.HoursGT(0),
				).
				Count(ctx)
			if err == nil && count > 0 {
				return true
			}
		}
	}
	return false
}

func firstMaterialsEnrollment(ens []*ent.Enrollment) *ent.Enrollment {
	for _, en := range ens {
		if en.ChargeMaterials {
			return en
		}
	}
	return nil
}

func mergeGenerateResult(dst *GenerateResult, src GenerateResult) {
	dst.Created += src.Created
	dst.Updated += src.Updated
	dst.SkippedHasInvoice += src.SkippedHasInvoice
	dst.SkippedNoLines += src.SkippedNoLines
}

type rebuildInvoiceResult struct {
	stats            GenerateResult
	pdfsToInvalidate []invoicePDFRef
}

type invoicePDFRef struct {
	invoiceID int
	number    string
	studentID int
	year      int
	month     int
}

func isCurrentEditableMonth(y, m int) bool {
	now := currentTime()
	return now.Year() == y && int(now.Month()) == m
}

func (s *Service) rebuildDraftForStudent(ctx context.Context, studentID, y, m int) (GenerateResult, error) {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		if err == ent.ErrTxStarted {
			result, err := s.rebuildDraftForStudentInStore(ctx, studentID, y, m)
			return result.stats, err
		}
		return GenerateResult{}, err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, err := (&Service{db: tx.Client(), invoicesDir: s.invoicesDir}).rebuildDraftForStudentInStore(ctx, studentID, y, m)
	if err != nil {
		return GenerateResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return GenerateResult{}, err
	}
	committed = true
	if err := s.invalidateInvoicePDFs(ctx, res.pdfsToInvalidate); err != nil {
		return GenerateResult{}, err
	}
	return res.stats, nil
}

func (s *Service) rebuildDraftForStudentInStore(ctx context.Context, studentID, y, m int) (rebuildInvoiceResult, error) {
	res := rebuildInvoiceResult{}

	ens, err := s.db.Enrollment.Query().
		Where(enrollment.StudentIDEQ(studentID)).
		All(ctx)
	if err != nil {
		return res, err
	}

	lineCapacity := len(ens) + 1
	if lineCapacity < 1 {
		lineCapacity = 1
	}
	lines := make([]*ent.InvoiceLineCreate, 0, lineCapacity)
	var totalCents int64

	for _, en := range ens {
		lp, _ := s.resolvePrices(ctx, en, y, m)

		switch en.BillingMode {
		case BillingPerLesson:
			line, amount := s.buildPerLessonLine(ctx, en, y, m, lp)
			lines = append(lines, line)
			totalCents += amount

		case BillingSubscription:
			lessonsHeld := s.subscriptionLessonsHeld(ctx, en.CourseID, y, m)
			if lp <= 0 || lessonsHeld <= 0 {
				continue
			}
			line, amount := s.buildSubscriptionLine(ctx, en, y, m, lp, lessonsHeld)
			lines = append(lines, line)
			totalCents += amount

		default:
			fmt.Printf("Unexpected billing mode: %s\n", en.BillingMode)
		}
	}

	if materialsEnrollment := firstMaterialsEnrollment(ens); materialsEnrollment != nil && s.hasAnyLessonsInMonth(ctx, ens, y, m) {
		materialsLine, materialsAmount := s.buildMaterialsLine(materialsEnrollment.ID)
		lines = append(lines, materialsLine)
		totalCents += materialsAmount
	}

	existing, err := s.db.Invoice.Query().Where(
		invoice.StudentIDEQ(studentID),
		invoice.PeriodYearEQ(y),
		invoice.PeriodMonthEQ(m),
	).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return res, err
	}

	if totalCents <= 0 {
		if err == nil && existing.Status == StatusDraft {
			if _, err := s.db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(existing.ID)).Exec(ctx); err != nil {
				return res, err
			}
			if err := s.db.Invoice.DeleteOneID(existing.ID).Exec(ctx); err != nil {
				return res, err
			}
		} else if err == nil && isCurrentEditableMonth(y, m) && existing.Status != StatusCanceled {
			if _, err := s.db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(existing.ID)).Exec(ctx); err != nil {
				return res, err
			}
			update := existing.Update().SetTotalAmountCents(0)
			if existing.Status != StatusDraft {
				update = update.ClearPdfRevision().ClearPdfGeneratedAt()
			}
			if _, err := update.Save(ctx); err != nil {
				return res, err
			}
			if existing.Status != StatusDraft {
				if err := paysvc.New(s.db).RecomputeInvoiceStatus(ctx, existing.ID); err != nil {
					return res, err
				}
			}
			res.stats.Updated++
		}
		res.stats.SkippedNoLines++
		return res, nil
	}

	switch {
	case ent.IsNotFound(err):
		inv, createErr := s.db.Invoice.Create().
			SetStudentID(studentID).
			SetPeriodYear(y).
			SetPeriodMonth(m).
			SetStatus(StatusDraft).
			SetTotalAmountCents(totalCents).
			Save(ctx)
		if createErr != nil {
			return res, createErr
		}
		for _, lc := range lines {
			if _, saveErr := lc.SetInvoiceID(inv.ID).Save(ctx); saveErr != nil {
				return res, saveErr
			}
		}
		res.stats.Created++

	case existing.Status == StatusDraft || (isCurrentEditableMonth(y, m) &&
		(existing.Status == StatusIssuedPendingPDF || existing.Status == StatusIssued || existing.Status == StatusPaidPendingPDF || existing.Status == StatusPaid)):
		pdfRef, err := s.invoicePDFRef(existing)
		if err != nil {
			return res, err
		}
		if _, err := s.db.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(existing.ID)).Exec(ctx); err != nil {
			return res, err
		}
		for _, lc := range lines {
			if _, saveErr := lc.SetInvoiceID(existing.ID).Save(ctx); saveErr != nil {
				return res, saveErr
			}
		}
		update := existing.Update().SetTotalAmountCents(totalCents)
		if existing.Status != StatusDraft {
			update = update.ClearPdfRevision().ClearPdfGeneratedAt()
		}
		if _, err := update.Save(ctx); err != nil {
			return res, err
		}
		if existing.Status != StatusDraft {
			if err := paysvc.New(s.db).RecomputeInvoiceStatus(ctx, existing.ID); err != nil {
				return res, err
			}
			if pdfRef != nil {
				res.pdfsToInvalidate = append(res.pdfsToInvalidate, *pdfRef)
			}
		}
		res.stats.Updated++

	default:
		res.stats.SkippedHasInvoice++
	}

	return res, nil
}

func (s *Service) invoicePDFRef(iv *ent.Invoice) (*invoicePDFRef, error) {
	if strings.TrimSpace(s.invoicesDir) == "" || iv == nil || iv.Number == nil || strings.TrimSpace(*iv.Number) == "" {
		return nil, nil
	}
	return &invoicePDFRef{
		invoiceID: iv.ID,
		number:    *iv.Number,
		studentID: iv.StudentID,
		year:      iv.PeriodYear,
		month:     iv.PeriodMonth,
	}, nil
}

func (s *Service) invalidateInvoicePDFs(ctx context.Context, refs []invoicePDFRef) error {
	if strings.TrimSpace(s.invoicesDir) == "" || len(refs) == 0 {
		return nil
	}
	locator := NewPDFLocator(s.invoicesDir)
	for _, ref := range refs {
		recipientInfo, err := recipient.ResolveInvoiceRecipient(ctx, s.db, ref.studentID)
		if err != nil {
			return err
		}
		paths := []string{
			locator.PathByNumberAndName(ref.year, ref.month, ref.number, recipientInfo.InvoiceSubjectName()),
			locator.PathByNumber(ref.year, ref.month, ref.number),
		}
		for _, pdfPath := range paths {
			if err := os.Remove(pdfPath); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
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
		ens, err := s.db.Enrollment.Query().Where(enrollment.StudentIDEQ(st.ID)).All(ctx)
		if err != nil {
			return res, err
		}
		if len(ens) == 0 {
			continue
		}
		studentRes, rebuildErr := s.rebuildDraftForStudent(ctx, st.ID, y, m)
		if rebuildErr != nil {
			return res, rebuildErr
		}
		mergeGenerateResult(&res, studentRes)
	}

	return res, nil
}

// RebuildStudentDraft rebuilds or removes the draft invoice for one student in the given month.
// Issued, paid, and canceled invoices are left untouched.
func (s *Service) RebuildStudentDraft(ctx context.Context, studentID, y, m int) (GenerateResult, error) {
	return s.rebuildDraftForStudent(ctx, studentID, y, m)
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
			Total: money.CentsToEuros(iv.TotalAmountCents), Status: string(iv.Status), PDFReady: CanonicalPDFReady(iv), LinesCount: count, Number: iv.Number,
			EventDate:     invoiceEventDate(iv),
			LastEmailedAt: optionalRFC3339(iv.LastEmailedAt),
			LastEmailedTo: optionalString(iv.LastEmailedTo),
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
	recipientInfo, err := recipient.ResolveInvoiceRecipient(ctx, s.db, iv.StudentID)
	if err != nil {
		return nil, err
	}
	dto := &InvoiceDTO{
		ID:                  iv.ID,
		Version:             iv.Version,
		StudentID:           iv.StudentID,
		StudentName:         getStudentName(iv),
		RecipientName:       recipientInfo.RecipientName,
		RecipientPhone:      recipientInfo.RecipientPhone,
		RecipientEmail:      recipientInfo.RecipientEmail,
		ChildName:           recipientInfo.ChildName,
		StudentPersonalCode: recipientInfo.StudentPersonalCode,
		IsMinor:             recipientInfo.IsMinor,
		Year:                iv.PeriodYear,
		Month:               iv.PeriodMonth,
		Total:               money.CentsToEuros(iv.TotalAmountCents),
		Status:              string(iv.Status),
		PDFReady:            CanonicalPDFReady(iv),
		Number:              iv.Number,
		LastEmailedAt:       optionalRFC3339(iv.LastEmailedAt),
		LastEmailedTo:       optionalString(iv.LastEmailedTo),
	}
	for _, l := range ls {
		dto.Lines = append(dto.Lines, LineDTO{
			EnrollmentID: l.EnrollmentID,
			Description:  l.Description,
			Qty:          l.Qty,
			UnitPrice:    money.CentsToEuros(l.UnitPriceCents),
			Amount:       money.CentsToEuros(l.AmountCents),
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
	return s.DeleteDraftWithVersion(ctx, id, iv.Version)
}

func (s *Service) DeleteDraftWithVersion(ctx context.Context, id, version int) error {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	iv, err := tx.Invoice.Get(ctx, id)
	if err != nil {
		return err
	}
	if iv.Version != version {
		return apperrors.StaleRevision()
	}
	if iv.Status != StatusDraft {
		return fmt.Errorf("можно удалять только счета в статусе черновика")
	}
	if _, err := tx.InvoiceLine.Delete().Where(invoiceline.InvoiceIDEQ(iv.ID)).Exec(ctx); err != nil {
		return err
	}
	if err := tx.Invoice.DeleteOneID(iv.ID).Where(invoice.VersionEQ(version)).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return apperrors.StaleRevision()
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// ReopenDraft moves an issued invoice with no payments back to draft state.
// The invoice lines and total remain unchanged, but the assigned number is cleared.
// If a PDF exists for the old issued number, it is removed to prevent stale documents.
func (s *Service) ReopenDraft(ctx context.Context, id int, outBaseDir string) error {
	iv, err := s.db.Invoice.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.ReopenDraftWithVersion(ctx, id, iv.Version, outBaseDir)
}

func (s *Service) ReopenDraftWithVersion(ctx context.Context, id, version int, outBaseDir string) error {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	iv, err := tx.Invoice.Get(ctx, id)
	if err != nil {
		return err
	}
	if iv.Version != version {
		return apperrors.StaleRevision()
	}
	if iv.Status != StatusIssued && iv.Status != StatusIssuedPendingPDF {
		return fmt.Errorf("вернуть в черновик можно только выставленные счета без оплат")
	}

	paymentCount, err := tx.Payment.Query().Where(payment.InvoiceIDEQ(iv.ID)).Count(ctx)
	if err != nil {
		return err
	}
	if paymentCount > 0 {
		return fmt.Errorf("нельзя вернуть в черновик счёт, по которому уже есть оплаты")
	}

	oldNumber := ""
	if iv.Number != nil {
		oldNumber = *iv.Number
	}
	recipientInfo, err := recipient.ResolveInvoiceRecipient(ctx, tx.Client(), iv.StudentID)
	if err != nil {
		return err
	}

	if _, err := tx.Invoice.UpdateOneID(iv.ID).
		Where(invoice.VersionEQ(version)).
		SetVersion(version + 1).
		SetStatus(StatusDraft).
		ClearNumber().
		ClearPdfFilename().
		ClearPdfRevision().
		ClearPdfGeneratedAt().
		Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return apperrors.StaleRevision()
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	if oldNumber != "" && outBaseDir != "" {
		locator := NewPDFLocator(outBaseDir)
		paths := []string{
			locator.PathByNumberAndName(iv.PeriodYear, iv.PeriodMonth, oldNumber, recipientInfo.InvoiceSubjectName()),
			locator.PathByNumber(iv.PeriodYear, iv.PeriodMonth, oldNumber),
		}
		for _, pdfPath := range paths {
			if err := os.Remove(pdfPath); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
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
	iv, err := s.db.Invoice.Get(ctx, id)
	if err != nil {
		return "", err
	}
	number, _, err := s.issueOneWithVersion(ctx, id, iv.Version)
	return number, err
}

func (s *Service) issueOneWithVersion(ctx context.Context, id, version int) (string, int, error) {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		if err == ent.ErrTxStarted {
			return s.issueOneInStore(ctx, id, version)
		}
		return "", 0, err
	}

	// Proper defer pattern: only rollback if commit hasn't been called
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	number, studentID, err := (&Service{db: tx.Client()}).issueOneInStore(ctx, id, version)
	if err != nil {
		return "", 0, err
	}
	if err := tx.Commit(); err != nil {
		return "", 0, err
	}
	committed = true
	return number, studentID, nil
}

func (s *Service) issueOneInStore(ctx context.Context, id, version int) (string, int, error) {
	iv, err := s.db.Invoice.Get(ctx, id)
	if err != nil {
		return "", 0, err
	}
	if iv.Version != version {
		return "", 0, apperrors.StaleRevision()
	}
	if iv.Status != StatusDraft {
		if iv.Number != nil && *iv.Number != "" {
			return *iv.Number, iv.StudentID, nil
		}
		return "", 0, fmt.Errorf("счёт %d не находится в статусе черновика", id)
	}
	if iv.TotalAmountCents <= 0 {
		return "", 0, fmt.Errorf("нельзя выставить пустой счёт или счёт с нулевой суммой")
	}

	prefix, seq, hasSettings, err := s.getSettingsOrDefaults(ctx)
	if err != nil {
		return "", 0, err
	}
	number := FormatNumber(prefix, iv.PeriodYear, iv.PeriodMonth, seq)

	// Increment the counter
	if hasSettings {
		if err := s.db.Settings.Update().Where(settings.SingletonIDEQ(app.SettingsSingletonID)).SetNextSeq(seq + 1).Exec(ctx); err != nil {
			return "", 0, err
		}
	}

	// Save the number and provisional status. A successful PDF generation will
	// promote the invoice to issued/paid, while a failed generation leaves it pending.
	if _, err := s.db.Invoice.UpdateOneID(iv.ID).
		Where(invoice.VersionEQ(version)).
		SetVersion(version + 1).
		SetNumber(number).
		SetStatus(StatusIssuedPendingPDF).
		Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return "", 0, apperrors.StaleRevision()
		}
		return "", 0, err
	}
	return number, iv.StudentID, nil
}

// Issue issues a single draft invoice and generates its PDF.
// This combines issueOne (assigning number and status) with PDF generation.
// Returns the invoice number and the path to the generated PDF file.
func (s *Service) IssueOne(ctx context.Context, id int) (string, error) {
	return s.issueOne(ctx, id)
}

func (s *Service) IssueAndApplyCreditWithVersion(ctx context.Context, id, version int) (string, int, error) {
	tx, err := s.db.Tx(ctx)
	if err != nil {
		return "", 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	number, studentID, err := (&Service{db: tx.Client()}).issueOneWithVersion(ctx, id, version)
	if err != nil {
		return "", 0, err
	}
	if err := paysvc.New(tx.Client()).ApplyCreditToOldestInvoices(ctx, studentID); err != nil {
		return "", 0, err
	}
	if err := tx.Commit(); err != nil {
		return "", 0, err
	}
	committed = true
	return number, studentID, nil
}

// Issue issues a single invoice and generates its PDF.
// Draft invoices are issued first; already-issued invoices keep their number.
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
	if err := s.persistPDFMetadata(ctx, id, p); err != nil {
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
			continue
		}
		if err := s.persistPDFMetadata(ctx, iv.ID, p); err != nil {
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
	return NewPDFLocator(outBaseDir).PathByNumber(y, m, number)
}

// PDFPathByNumberAndName generates the file path for an invoice PDF based on
// its number and student-facing subject name.
func PDFPathByNumberAndName(outBaseDir string, y, m int, number, subjectName string) string {
	return NewPDFLocator(outBaseDir).PathByNumberAndName(y, m, number, subjectName)
}

func invoiceFileStem(number, subjectName string) string {
	subjectName = strings.TrimSpace(subjectName)
	if subjectName == "" {
		return sanitizeInvoiceFileName(number)
	}
	return sanitizeInvoiceFileName(fmt.Sprintf("%s - %s", number, subjectName))
}

func sanitizeInvoiceFileName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return replacer.Replace(name)
}

func (s *Service) persistPDFMetadata(ctx context.Context, id int, pdfPath string) error {
	iv, err := s.db.Invoice.Get(ctx, id)
	if err != nil {
		return err
	}
	filename := strings.TrimSpace(filepath.Base(pdfPath))
	if filename == "" || filename == "." {
		return fmt.Errorf("invalid generated PDF path: %q", pdfPath)
	}
	generatedAt := currentTime().UTC()
	_, err = s.db.Invoice.UpdateOneID(id).
		SetPdfFilename(filename).
		SetPdfGeneratedAt(generatedAt).
		SetPdfRevision(iv.Version).
		Save(ctx)
	if err != nil {
		return err
	}
	return paysvc.New(s.db).RecomputeInvoiceStatus(ctx, id)
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
	case StatusDraft, StatusIssuedPendingPDF, StatusIssued, StatusPaidPendingPDF, StatusPaid, StatusCanceled:
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
			ID:            iv.ID,
			Version:       iv.Version,
			StudentID:     iv.StudentID,
			StudentName:   getStudentName(iv),
			Year:          iv.PeriodYear,
			Month:         iv.PeriodMonth,
			Total:         money.CentsToEuros(iv.TotalAmountCents),
			Status:        string(iv.Status),
			PDFReady:      CanonicalPDFReady(iv),
			LinesCount:    cnt,
			Number:        iv.Number,
			EventDate:     invoiceEventDate(iv),
			LastEmailedAt: optionalRFC3339(iv.LastEmailedAt),
			LastEmailedTo: optionalString(iv.LastEmailedTo),
		})
	}
	return out, nil
}

func invoiceEventDate(iv *ent.Invoice) string {
	if iv.UpdatedAt != nil && !iv.UpdatedAt.IsZero() {
		return iv.UpdatedAt.Format(time.RFC3339)
	}
	if iv.CreatedAt != nil && !iv.CreatedAt.IsZero() {
		return iv.CreatedAt.Format(time.RFC3339)
	}
	return ""
}
