package attendance

import (
	"context"
	"errors"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/enrollment"
	"langschool/internal/app"
)

type Service struct{ db *ent.Client }

func New(db *ent.Client) *Service { return &Service{db: db} }

type Row struct {
	EnrollmentID int     `json:"enrollmentId"`
	StudentID    int     `json:"studentId"`
	StudentName  string  `json:"studentName"`
	CourseID     int     `json:"courseId"`
	CourseName   string  `json:"courseName"`
	CourseType   string  `json:"courseType"` // group|individual
	LessonPrice  float64 `json:"lessonPrice"`
	Count        int     `json:"count"`
	Locked       bool    `json:"locked"`
}

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

		cnt, locked := 0, false
		if am != nil {
			cnt, locked = am.LessonsCount, am.Locked
		}

		rows = append(rows, Row{
			EnrollmentID: e.ID,
			StudentID:    e.StudentID, StudentName: sname,
			CourseID: e.CourseID, CourseName: c.Name, CourseType: string(c.Type),
			LessonPrice: c.LessonPrice, Count: cnt, Locked: locked,
		})
	}
	return rows, nil
}

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
		if am.Locked {
			return errors.New("month locked for this pair")
		}
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

func (s *Service) AddOneForFilter(ctx context.Context, y, m int, courseID *int) (int, error) {
	rows, err := s.ListPerLesson(ctx, y, m, courseID)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, r := range rows {
		if r.Locked {
			continue
		}
		if err := s.Upsert(ctx, r.StudentID, r.CourseID, y, m, r.Count+1); err == nil {
			changed++
		}
	}
	return changed, nil
}

func (s *Service) SetLocked(ctx context.Context, y, m int, courseID *int, lock bool) (int, error) {
	rows, err := s.ListPerLesson(ctx, y, m, courseID)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, r := range rows {
		am, err := s.db.AttendanceMonth.
			Query().
			Where(
				attendancemonth.StudentIDEQ(r.StudentID),
				attendancemonth.CourseIDEQ(r.CourseID),
				attendancemonth.YearEQ(y),
				attendancemonth.MonthEQ(m),
			).Only(ctx)
		if err == nil && am.Locked != lock {
			if _, e := am.Update().SetLocked(lock).Save(ctx); e == nil {
				changed++
			}
		}
	}
	return changed, nil
}

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
