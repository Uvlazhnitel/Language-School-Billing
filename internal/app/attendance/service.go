package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
)

type Service struct{ db *ent.Client }

func New(db *ent.Client) *Service { return &Service{db: db} }

type Row struct {
	StudentID   int     `json:"studentId"`
	StudentName string  `json:"studentName"`
	CourseID    int     `json:"courseId"`
	CourseName  string  `json:"courseName"`
	CourseType  string  `json:"courseType"`
	LessonPrice float64 `json:"lessonPrice"`
	Count       int     `json:"count"`
	Locked      bool    `json:"locked"`
}

// ListPerLesson lists attendance rows for "per lesson" billing mode enrollments.
func (s *Service) ListPerLesson(ctx context.Context, y, m int, courseID *int) ([]Row, error) {
	q := s.db.Enrollment.
		Query().
		Where(enrollment.BillingModeEQ("per_lesson")).
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) {
			if courseID != nil && *courseID > 0 {
				cq.Where(course.IDEQ(*courseID))
			}
		})

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
			StudentID: e.StudentID, StudentName: sname,
			CourseID: e.CourseID, CourseName: c.Name, CourseType: string(c.Type),
			LessonPrice: c.LessonPrice, Count: cnt, Locked: locked,
		})
	}
	return rows, nil
}

// Upsert updates or creates an attendance month record.
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

// AddOneForFilter increments attendance count by one for all unlocked records matching the filter.
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
		_ = s.Upsert(ctx, r.StudentID, r.CourseID, y, m, r.Count+1)
		changed++
	}
	return changed, nil
}

type schedule struct {
	DaysOfWeek []int `json:"daysOfWeek"` // 1=Mon..7=Sun
}

func countMatches(y int, m time.Month, days []int) int {
	if len(days) == 0 {
		return 0
	}
	set := map[int]bool{}
	for _, d := range days {
		set[d] = true
	}
	t := time.Date(y, m, 1, 0, 0, 0, 0, time.Local)
	cnt := 0
	for t.Month() == m {
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		if set[wd] {
			cnt++
		}
		t = t.AddDate(0, 0, 1)
	}
	return cnt
}

// EstimateBySchedule estimates attendance counts based on course schedules.
func (s *Service) EstimateBySchedule(ctx context.Context, y, m int, courseID *int) (map[string]int, error) {
	rows, err := s.ListPerLesson(ctx, y, m, courseID)
	if err != nil {
		return nil, err
	}
	result := map[string]int{}
	for _, r := range rows {
		c, err := s.db.Course.Get(ctx, r.CourseID)
		if err != nil {
			continue
		}
		var sch schedule
		_ = json.Unmarshal([]byte(c.ScheduleJSON), &sch)
		hint := 0
		if c.Type == course.TypeGroup && len(sch.DaysOfWeek) > 0 {
			hint = countMatches(y, time.Month(m), sch.DaysOfWeek)
		}
		key :=
			fmt.Sprintf("%d-%d", r.StudentID, r.CourseID)
		result[key] = hint
	}
	return result, nil
}

// SetLocked sets or unsets the "locked" status for all records matching the filter.q
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
		if err == nil {
			if am.Locked != lock {
				_, _ = am.Update().SetLocked(lock).Save(ctx)
				changed++
			}
		}
	}
	return changed, nil
}
