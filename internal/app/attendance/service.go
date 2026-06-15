// Package attendance provides services for tracking student attendance.
// Attendance is tracked monthly per student-course pair.
package attendance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/coursemonthstat"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/student"
	"langschool/internal/app"
	invsvc "langschool/internal/app/invoice"
	"langschool/internal/app/utils"
	"langschool/internal/apperrors"
	"langschool/internal/money"
	"langschool/internal/validation"
)

// Service provides attendance tracking functionality.
// It manages monthly attendance records for students enrolled in courses.
type Service struct{ db *ent.Client }

var currentTime = time.Now

// New creates a new attendance service with the given database client.
func New(db *ent.Client) *Service { return &Service{db: db} }

// Row represents a single attendance record in the attendance sheet.
// It combines enrollment, student, and course information with the
// tracked hours for a specific month.
type Row struct {
	EnrollmentID            int     `json:"enrollmentId"`            // ID of the enrollment
	EnrollmentVersion       int     `json:"enrollmentVersion"`       // Optimistic-lock revision of the enrollment
	StudentID               int     `json:"studentId"`               // ID of the student
	StudentName             string  `json:"studentName"`             // Student's full name
	CourseID                int     `json:"courseId"`                // ID of the course
	CourseName              string  `json:"courseName"`              // Course name
	CourseType              string  `json:"courseType"`              // Course type: "group" or "individual"
	BillingMode             string  `json:"billingMode"`             // Enrollment billing mode
	LessonPrice             float64 `json:"lessonPrice"`             // Hourly rate for this enrollment
	DiscountPct             float64 `json:"discountPct"`             // Personal discount percentage for the enrollment
	SubscriptionLessonPrice float64 `json:"subscriptionLessonPrice"` // Explicit subscription lesson price for this enrollment
	Hours                   float64 `json:"hours"`                   // Hours attended in the month
	HasRecord               bool    `json:"hasRecord"`               // Whether an AttendanceMonth record exists for this month
	CanDelete               bool    `json:"canDelete"`               // Whether enrollment can be safely deleted
	AttendanceLocked        bool    `json:"attendanceLocked"`        // Whether attendance is locked for this month/invoice state
	InvoiceStatus           string  `json:"invoiceStatus"`           // Invoice status for the student/month, if any
}

type CourseMonthSubscription struct {
	CourseID    int     `json:"courseId"`
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	LessonsHeld float64 `json:"lessonsHeld"`
}

func (s *Service) getMonthInvoiceStatus(ctx context.Context, studentID, y, m int) (string, bool, error) {
	iv, err := s.db.Invoice.Query().
		Where(
			invoice.StudentIDEQ(studentID),
			invoice.PeriodYearEQ(y),
			invoice.PeriodMonthEQ(m),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(iv.Status), true, nil
}

func isCurrentEditableMonth(y, m int) bool {
	now := currentTime()
	return now.Year() == y && int(now.Month()) == m
}

func lockReason(y, m int, invoiceStatus string) bool {
	switch invoiceStatus {
	case "", app.InvoiceStatusDraft:
		return false
	case app.InvoiceStatusCanceled:
		return true
	case app.InvoiceStatusIssued, app.InvoiceStatusPaid:
		return !isCurrentEditableMonth(y, m)
	default:
		return invoiceStatus != ""
	}
}

func canDeleteEnrollmentWithInvoiceHistory(ctx context.Context, db *ent.Client, enrollmentID int) (bool, error) {
	hasProtectedInvoiceHistory, err := db.InvoiceLine.Query().
		Where(
			invoiceline.EnrollmentIDEQ(enrollmentID),
			invoiceline.HasInvoiceWith(invoice.StatusNEQ(app.InvoiceStatusDraft)),
		).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return !hasProtectedInvoiceHistory, nil
}

// ListPerLesson retrieves attendance sheet rows for all enrollments
// for the specified year and month. Optionally filters by courseID.
// Returns a list of rows with student, course, and attendance information.
// If no attendance record exists for a student-course pair, the hours default to 0.
func (s *Service) ListPerLesson(ctx context.Context, y, m int, courseID *int) ([]Row, error) {
	q := s.db.Enrollment.
		Query().
		WithStudent().
		WithCourse()
	if courseID != nil && *courseID > 0 {
		q = q.Where(enrollment.CourseIDEQ(*courseID))
	}
	ens, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(ens))
	for _, e := range ens {
		if e.Edges.Course == nil || e.Edges.Student == nil {
			continue
		}
		c := e.Edges.Course
		sname := e.Edges.Student.FullName

		am, _ := s.db.AttendanceMonth.
			Query().
			Where(
				attendancemonth.StudentIDEQ(e.StudentID),
				attendancemonth.CourseIDEQ(e.CourseID),
				attendancemonth.YearEQ(y),
				attendancemonth.MonthEQ(m),
			).Only(ctx)

		hours := 0.0
		hasRecord := false
		if am != nil {
			hours = am.Hours
			hasRecord = true
		}

		canDelete, err := canDeleteEnrollmentWithInvoiceHistory(ctx, s.db, e.ID)
		if err != nil {
			return nil, err
		}

		invoiceStatus, _, err := s.getMonthInvoiceStatus(ctx, e.StudentID, y, m)
		if err != nil {
			return nil, err
		}

		rows = append(rows, Row{
			EnrollmentID:            e.ID,
			EnrollmentVersion:       e.Version,
			StudentID:               e.StudentID,
			StudentName:             sname,
			CourseID:                e.CourseID,
			CourseName:              c.Name,
			CourseType:              string(c.Type),
			BillingMode:             string(e.BillingMode),
			LessonPrice:             money.CentsToEuros(c.LessonPriceCents),
			DiscountPct:             utils.Round2(e.DiscountPct),
			SubscriptionLessonPrice: utils.Round2(money.CentsToEuros(e.SubscriptionLessonPriceCents)),
			Hours:                   hours,
			HasRecord:               hasRecord,
			CanDelete:               canDelete,
			AttendanceLocked:        lockReason(y, m, invoiceStatus),
			InvoiceStatus:           invoiceStatus,
		})
	}
	return rows, nil
}

// Upsert creates or updates an attendance record for a student-course pair
// for a specific month. If no record exists, a new one is created. If a record exists, it is updated.
func (s *Service) Upsert(ctx context.Context, studentID, courseID, y, m int, hours float64) error {
	invoiceStatus, _, err := s.getMonthInvoiceStatus(ctx, studentID, y, m)
	if err != nil {
		return err
	}
	if err := validation.ValidateQuarterHours(hours); err != nil {
		return err
	}
	if lockReason(y, m, invoiceStatus) {
		return fmt.Errorf("посещаемость за %04d-%02d заблокирована, потому что счёт имеет статус %s", y, m, invoiceStatus)
	}

	am, err := s.db.AttendanceMonth.
		Query().
		Where(
			attendancemonth.StudentIDEQ(studentID),
			attendancemonth.CourseIDEQ(courseID),
			attendancemonth.YearEQ(y),
			attendancemonth.MonthEQ(m),
		).Only(ctx)
	if err == nil {
		_, err = am.Update().SetHours(hours).Save(ctx)
		return err
	}
	if !ent.IsNotFound(err) {
		return err
	}
	_, err = s.db.AttendanceMonth.
		Create().
		SetStudentID(studentID).
		SetCourseID(courseID).
		SetYear(y).SetMonth(m).
		SetHours(hours).
		Save(ctx)
	return err
}

// AddOneForFilter increments tracked hours by 0.25 for all attendance
// records matching the filter (year, month, optional courseID).
// This is useful for bulk operations like "add 0.25h to all students in a course".
// Returns the number of records that were successfully updated.
func (s *Service) AddOneForFilter(ctx context.Context, y, m int, courseID *int) (int, error) {
	rows, err := s.ListPerLesson(ctx, y, m, courseID)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, r := range rows {
		if err := s.Upsert(ctx, r.StudentID, r.CourseID, y, m, utils.Round2(r.Hours+0.25)); err == nil {
			changed++
		}
	}
	return changed, nil
}

func (s *Service) ListCourseMonthSubscriptions(ctx context.Context, y, m int, courseID *int) ([]CourseMonthSubscription, error) {
	q := s.db.CourseMonthStat.Query().
		Where(coursemonthstat.YearEQ(y), coursemonthstat.MonthEQ(m))
	if courseID != nil && *courseID > 0 {
		q = q.Where(coursemonthstat.CourseIDEQ(*courseID))
	}
	items, err := q.Order(ent.Asc(coursemonthstat.FieldCourseID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CourseMonthSubscription, 0, len(items))
	for _, item := range items {
		out = append(out, CourseMonthSubscription{
			CourseID:    item.CourseID,
			Year:        item.Year,
			Month:       item.Month,
			LessonsHeld: utils.Round2(item.SubscriptionLessonsHeld),
		})
	}
	return out, nil
}

func (s *Service) UpsertCourseMonthSubscription(ctx context.Context, courseID, y, m int, lessonsHeld float64) (*CourseMonthSubscription, error) {
	if courseID <= 0 {
		return nil, errors.New("courseID must be > 0")
	}
	if y <= 0 {
		return nil, errors.New("year must be > 0")
	}
	if m < 1 || m > 12 {
		return nil, errors.New("month must be between 1 and 12")
	}
	if lessonsHeld < 0 {
		return nil, errors.New("lessonsHeld must be >= 0")
	}
	if !isCurrentEditableMonth(y, m) {
		locked, err := s.db.Invoice.Query().
			Where(
				invoice.PeriodYearEQ(y),
				invoice.PeriodMonthEQ(m),
				invoice.StatusNEQ(app.InvoiceStatusDraft),
				invoice.StatusNEQ(app.InvoiceStatusCanceled),
				invoice.HasStudentWith(student.HasEnrollmentsWith(
					enrollment.CourseIDEQ(courseID),
					enrollment.BillingModeEQ(enrollment.BillingModeSubscription),
				)),
			).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if locked {
			return nil, fmt.Errorf("занятия по абонементу за %04d-%02d заблокированы выставленным или оплаченным счётом", y, m)
		}
	}
	lessonsHeld = utils.Round2(lessonsHeld)
	if _, err := s.db.Course.Get(ctx, courseID); err != nil {
		return nil, err
	}

	item, err := s.db.CourseMonthStat.Query().
		Where(
			coursemonthstat.CourseIDEQ(courseID),
			coursemonthstat.YearEQ(y),
			coursemonthstat.MonthEQ(m),
		).
		Only(ctx)
	if err == nil {
		item, err = item.Update().SetSubscriptionLessonsHeld(lessonsHeld).Save(ctx)
		if err != nil {
			return nil, err
		}
	} else if ent.IsNotFound(err) {
		item, err = s.db.CourseMonthStat.Create().
			SetCourseID(courseID).
			SetYear(y).
			SetMonth(m).
			SetSubscriptionLessonsHeld(lessonsHeld).
			Save(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}

	subscriptionEnrollments, err := s.db.Enrollment.Query().
		Where(
			enrollment.CourseIDEQ(courseID),
			enrollment.BillingModeEQ(enrollment.BillingModeSubscription),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	invoiceService := invsvc.New(s.db)
	for _, en := range subscriptionEnrollments {
		if _, err := invoiceService.RebuildStudentDraft(ctx, en.StudentID, y, m); err != nil {
			return nil, err
		}
	}

	return &CourseMonthSubscription{
		CourseID:    item.CourseID,
		Year:        item.Year,
		Month:       item.Month,
		LessonsHeld: utils.Round2(item.SubscriptionLessonsHeld),
	}, nil
}

// DeleteEnrollment deletes an enrollment and all associated attendance records.
// This ensures that when an enrollment is removed, all attendance history
// for that student-course pair is also cleaned up.
func (s *Service) DeleteEnrollment(ctx context.Context, enrollmentID int) error {
	return s.deleteEnrollment(ctx, enrollmentID, nil)
}

func (s *Service) DeleteEnrollmentWithVersion(ctx context.Context, enrollmentID, version int) error {
	return s.deleteEnrollment(ctx, enrollmentID, &version)
}

func (s *Service) deleteEnrollment(ctx context.Context, enrollmentID int, expectedVersion *int) error {
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

	en, err := tx.Enrollment.Get(ctx, enrollmentID)
	if err != nil {
		return err
	}
	if expectedVersion != nil && en.Version != *expectedVersion {
		return apperrors.StaleRevision()
	}

	canDelete, err := canDeleteEnrollmentWithInvoiceHistory(ctx, tx.Client(), enrollmentID)
	if err != nil {
		return err
	}
	if !canDelete {
		return errors.New("нельзя удалить зачисление: оно используется в выставленных, оплаченных или отменённых счетах")
	}

	draftLinks, err := tx.InvoiceLine.Query().
		Where(invoiceline.EnrollmentIDEQ(enrollmentID)).
		WithInvoice().
		All(ctx)
	if err != nil {
		return err
	}

	type affectedDraft struct {
		studentID int
		year      int
		month     int
	}
	affectedDrafts := make(map[string]affectedDraft, len(draftLinks))
	for _, line := range draftLinks {
		if line.Edges.Invoice == nil {
			continue
		}
		iv := line.Edges.Invoice
		key := fmt.Sprintf("%d-%04d-%02d", iv.StudentID, iv.PeriodYear, iv.PeriodMonth)
		affectedDrafts[key] = affectedDraft{
			studentID: iv.StudentID,
			year:      iv.PeriodYear,
			month:     iv.PeriodMonth,
		}
	}

	if len(draftLinks) > 0 {
		if _, err := tx.InvoiceLine.Delete().
			Where(invoiceline.EnrollmentIDEQ(enrollmentID)).
			Exec(ctx); err != nil {
			return err
		}
	}

	if _, err := tx.AttendanceMonth.
		Delete().
		Where(
			attendancemonth.StudentIDEQ(en.StudentID),
			attendancemonth.CourseIDEQ(en.CourseID),
		).Exec(ctx); err != nil {
		return err
	}
	deleteOne := tx.Enrollment.DeleteOneID(enrollmentID)
	if expectedVersion != nil {
		deleteOne = deleteOne.Where(enrollment.VersionEQ(*expectedVersion))
	}
	if err := deleteOne.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return apperrors.StaleRevision()
		}
		return err
	}

	invoiceService := invsvc.New(tx.Client())
	for _, draft := range affectedDrafts {
		if _, err := invoiceService.RebuildStudentDraft(ctx, draft.studentID, draft.year, draft.month); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
