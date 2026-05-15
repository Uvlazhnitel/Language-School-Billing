// Package attendance provides services for tracking student attendance.
// Attendance is tracked monthly per student-course pair.
package attendance

import (
	"context"
	"errors"
	"fmt"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/internal/app"
	"langschool/internal/app/utils"
)

// Service provides attendance tracking functionality.
// It manages monthly attendance records for students enrolled in courses.
type Service struct{ db *ent.Client }

// New creates a new attendance service with the given database client.
func New(db *ent.Client) *Service { return &Service{db: db} }

// Row represents a single attendance record in the attendance sheet.
// It combines enrollment, student, and course information with the
// attendance count for a specific month.
type Row struct {
	EnrollmentID     int     `json:"enrollmentId"`     // ID of the enrollment
	StudentID        int     `json:"studentId"`        // ID of the student
	StudentName      string  `json:"studentName"`      // Student's full name
	CourseID         int     `json:"courseId"`         // ID of the course
	CourseName       string  `json:"courseName"`       // Course name
	CourseType       string  `json:"courseType"`       // Course type: "group" or "individual"
	BillingMode      string  `json:"billingMode"`      // Enrollment billing mode
	LessonPrice      float64 `json:"lessonPrice"`      // Price per lesson for this enrollment
	Count            int     `json:"count"`            // Number of lessons attended in the month
	HasRecord        bool    `json:"hasRecord"`        // Whether an AttendanceMonth record exists for this month
	CanDelete        bool    `json:"canDelete"`        // Whether enrollment can be safely deleted
	AttendanceLocked bool    `json:"attendanceLocked"` // Whether attendance is locked by a non-draft invoice
	InvoiceStatus    string  `json:"invoiceStatus"`    // Invoice status for the student/month, if any
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

func lockReason(invoiceStatus string) bool {
	return invoiceStatus != "" && invoiceStatus != app.InvoiceStatusDraft
}

// ListPerLesson retrieves attendance sheet rows for all enrollments
// for the specified year and month. Optionally filters by courseID.
// Returns a list of rows with student, course, and attendance information.
// If no attendance record exists for a student-course pair, the count defaults to 0.
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

		cnt := 0
		hasRecord := false
		if am != nil {
			cnt = am.LessonsCount
			hasRecord = true
		}

		hasInvoiceHistory, err := s.db.InvoiceLine.Query().
			Where(invoiceline.EnrollmentIDEQ(e.ID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}

		invoiceStatus, _, err := s.getMonthInvoiceStatus(ctx, e.StudentID, y, m)
		if err != nil {
			return nil, err
		}

		rows = append(rows, Row{
			EnrollmentID:     e.ID,
			StudentID:        e.StudentID,
			StudentName:      sname,
			CourseID:         e.CourseID,
			CourseName:       c.Name,
			CourseType:       string(c.Type),
			BillingMode:      string(e.BillingMode),
			LessonPrice:      utils.Round2(c.LessonPrice),
			Count:            cnt,
			HasRecord:        hasRecord,
			CanDelete:        !hasInvoiceHistory,
			AttendanceLocked: lockReason(invoiceStatus),
			InvoiceStatus:    invoiceStatus,
		})
	}
	return rows, nil
}

// Upsert creates or updates an attendance record for a student-course pair
// for a specific month. If no record exists, a new one is created. If a record exists, it is updated.
func (s *Service) Upsert(ctx context.Context, studentID, courseID, y, m, count int) error {
	invoiceStatus, _, err := s.getMonthInvoiceStatus(ctx, studentID, y, m)
	if err != nil {
		return err
	}
	if lockReason(invoiceStatus) {
		return fmt.Errorf("attendance is locked because the invoice for %04d-%02d is %s; reopen it to draft first", y, m, invoiceStatus)
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
		_, err = am.Update().SetLessonsCount(count).Save(ctx)
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
		SetLessonsCount(count).
		Save(ctx)
	return err
}

// AddOneForFilter increments the lesson count by 1 for all attendance
// records matching the filter (year, month, optional courseID).
// This is useful for bulk operations like "add one lesson to all students in a course".
// Returns the number of records that were successfully updated.
func (s *Service) AddOneForFilter(ctx context.Context, y, m int, courseID *int) (int, error) {
	rows, err := s.ListPerLesson(ctx, y, m, courseID)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, r := range rows {
		if err := s.Upsert(ctx, r.StudentID, r.CourseID, y, m, r.Count+1); err == nil {
			changed++
		}
	}
	return changed, nil
}

// DeleteEnrollment deletes an enrollment and all associated attendance records.
// This ensures that when an enrollment is removed, all attendance history
// for that student-course pair is also cleaned up.
func (s *Service) DeleteEnrollment(ctx context.Context, enrollmentID int) error {
	en, err := s.db.Enrollment.Get(ctx, enrollmentID)
	if err != nil {
		return err
	}
	hasInvoiceHistory, err := s.db.InvoiceLine.Query().
		Where(invoiceline.EnrollmentIDEQ(enrollmentID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasInvoiceHistory {
		return errors.New("cannot delete enrollment: it is already used in invoice history")
	}
	if _, err := s.db.AttendanceMonth.
		Delete().
		Where(
			attendancemonth.StudentIDEQ(en.StudentID),
			attendancemonth.CourseIDEQ(en.CourseID),
		).Exec(ctx); err != nil {
		return err
	}
	return s.db.Enrollment.DeleteOneID(enrollmentID).Exec(ctx)
}
