package backend

import "context"

func (s *Service) AttendanceListPerLesson(ctx context.Context, year, month int, courseID *int) ([]AttendanceRow, error) {
	return s.rt.Attendance.ListPerLesson(ctx, year, month, courseID)
}

func (s *Service) AttendanceUpsert(ctx context.Context, studentID, courseID, year, month int, hours float64) error {
	return s.rt.Attendance.Upsert(ctx, studentID, courseID, year, month, hours)
}

func (s *Service) CourseMonthSubscriptionList(ctx context.Context, year, month int, courseID *int) ([]CourseMonthSubscriptionDTO, error) {
	items, err := s.rt.Attendance.ListCourseMonthSubscriptions(ctx, year, month, courseID)
	if err != nil {
		return nil, err
	}
	out := make([]CourseMonthSubscriptionDTO, 0, len(items))
	for _, item := range items {
		out = append(out, CourseMonthSubscriptionDTO{
			CourseID:    item.CourseID,
			Year:        item.Year,
			Month:       item.Month,
			LessonsHeld: item.LessonsHeld,
		})
	}
	return out, nil
}

func (s *Service) CourseMonthSubscriptionUpsert(ctx context.Context, courseID, year, month int, lessonsHeld float64) (*CourseMonthSubscriptionDTO, error) {
	item, err := s.rt.Attendance.UpsertCourseMonthSubscription(ctx, courseID, year, month, lessonsHeld)
	if err != nil {
		return nil, err
	}
	return &CourseMonthSubscriptionDTO{
		CourseID:    item.CourseID,
		Year:        item.Year,
		Month:       item.Month,
		LessonsHeld: item.LessonsHeld,
	}, nil
}

func (s *Service) AttendanceAddOne(ctx context.Context, year, month int, courseID *int) (int, error) {
	return s.rt.Attendance.AddOneForFilter(ctx, year, month, courseID)
}

func (s *Service) EnrollmentDelete(ctx context.Context, enrollmentID int) error {
	return s.rt.Attendance.DeleteEnrollment(ctx, enrollmentID)
}

func (s *Service) EnrollmentDeleteWithVersion(ctx context.Context, enrollmentID, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	return s.rt.Attendance.DeleteEnrollmentWithVersion(ctx, enrollmentID, version)
}
