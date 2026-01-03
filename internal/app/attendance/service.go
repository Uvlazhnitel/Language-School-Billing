// Package attendance provides services for tracking student attendance.
// Attendance is tracked monthly per student-course pair for per-lesson billing.
package attendance

import (
	"context"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/enrollment"
	"langschool/internal/app"
)

// Service provides attendance tracking functionality.
// It manages monthly attendance records for students enrolled in courses
// with per-lesson billing mode.
type Service struct{ db *ent.Client }

// New creates a new attendance service with the given database client.
func New(db *ent.Client) *Service { return &Service{db: db} }

// Row represents a single attendance record in the attendance sheet.
// It combines enrollment, student, and course information with the
// attendance count for a specific month.
type Row struct {
	EnrollmentID int     `json:"enrollmentId"` // ID of the enrollment
	StudentID    int     `json:"studentId"`    // ID of the student
	StudentName  string  `json:"studentName"`  // Student's full name
	CourseID     int     `json:"courseId"`     // ID of the course
	CourseName   string  `json:"courseName"`   // Course name
	CourseType   string  `json:"courseType"`    // Course type: "group" or "individual"
	LessonPrice  float64 `json:"lessonPrice"`  // Price per lesson for this enrollment
	Count        int     `json:"count"`         // Number of lessons attended in the month
}

// ListPerLesson retrieves attendance records for all enrollments with per-lesson billing
// for the specified year and month. Optionally filters by courseID.
// Returns a list of rows with student, course, and attendance information.
// If no attendance record exists for a student-course pair, the count defaults to 0.
func (s *Service) ListPerLesson(ctx context.Context, y, m int, courseID *int) ([]Row, error) {
	q := s.db.Enrollment.
		Query().
		Where(enrollment.BillingModeEQ(app.BillingModePerLesson)).
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
		if am != nil {
			cnt = am.LessonsCount
		}

		rows = append(rows, Row{
			EnrollmentID: e.ID,
			StudentID:    e.StudentID, StudentName: sname,
			CourseID: e.CourseID, CourseName: c.Name, CourseType: string(c.Type),
			LessonPrice: c.LessonPrice, Count: cnt,
		})
	}
	return rows, nil
}

// Upsert creates or updates an attendance record for a student-course pair
// for a specific month. If no record exists, a new one is created. If a record exists, it is updated.
func (s *Service) Upsert(ctx context.Context, studentID, courseID, y, m, count int) error {
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
