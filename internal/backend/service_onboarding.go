package backend

import (
	"context"
	"errors"

	"langschool/ent/enrollment"
	"langschool/ent/student"
)

func validateUniqueEnrollmentCourses(inputs []EnrollmentCreateInput) error {
	seen := make(map[int]struct{}, len(inputs))
	for _, input := range inputs {
		if input.CourseID <= 0 {
			return errors.New("courseID must be > 0")
		}
		if _, exists := seen[input.CourseID]; exists {
			return errors.New("invalid enrollment list: duplicate courseID")
		}
		seen[input.CourseID] = struct{}{}
	}
	return nil
}

// StudentOnboard keeps the legacy single-enrollment service call compatible.
func (s *Service) StudentOnboard(ctx context.Context, studentInput StudentCreateInput, enrollmentInput *EnrollmentCreateInput) (*StudentOnboardingResult, error) {
	if enrollmentInput == nil {
		return s.StudentOnboardMany(ctx, studentInput, nil)
	}
	return s.StudentOnboardMany(ctx, studentInput, []EnrollmentCreateInput{*enrollmentInput})
}

func (s *Service) StudentOnboardMany(ctx context.Context, studentInput StudentCreateInput, enrollmentInputs []EnrollmentCreateInput) (*StudentOnboardingResult, error) {
	if err := validateUniqueEnrollmentCourses(enrollmentInputs); err != nil {
		return nil, err
	}
	tx, err := s.rt.DB.Ent.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	client := tx.Client()
	studentDTO, err := studentCreateInStore(
		ctx,
		client,
		studentInput.FullName,
		studentInput.PersonalCode,
		studentInput.Phone,
		studentInput.Email,
		studentInput.Note,
		studentInput.IsMinor,
		studentInput.PayerName,
		studentInput.PayerRole,
	)
	if err != nil {
		return nil, err
	}

	result := &StudentOnboardingResult{
		Student:     *studentDTO,
		Enrollments: make([]EnrollmentDTO, 0, len(enrollmentInputs)),
	}
	for _, enrollmentInput := range enrollmentInputs {
		enrollmentDTO, err := enrollmentCreateInStore(
			ctx,
			client,
			studentDTO.ID,
			enrollmentInput.CourseID,
			enrollmentInput.BillingMode,
			enrollmentInput.ChargeMaterials,
			enrollmentInput.LessonPriceOverride,
			enrollmentInput.SubscriptionLessonPrice,
			enrollmentInput.Note,
		)
		if err != nil {
			return nil, err
		}
		result.Enrollments = append(result.Enrollments, *enrollmentDTO)
	}
	if len(result.Enrollments) > 0 {
		first := result.Enrollments[0]
		result.Enrollment = &first
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) EnrollmentCreateMany(ctx context.Context, studentID int, inputs []EnrollmentCreateInput) (*EnrollmentBulkCreateResult, error) {
	if studentID <= 0 {
		return nil, errors.New("studentID must be > 0")
	}
	if err := validateUniqueEnrollmentCourses(inputs); err != nil {
		return nil, err
	}

	tx, err := s.rt.DB.Ent.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	client := tx.Client()

	st, err := client.Student.Query().Where(student.IDEQ(studentID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	if !st.IsActive {
		return nil, errors.New("cannot enroll a deactivated student")
	}

	courseIDs := make([]int, 0, len(inputs))
	for _, input := range inputs {
		courseIDs = append(courseIDs, input.CourseID)
	}
	existingCourseIDs := make(map[int]struct{}, len(inputs))
	if len(courseIDs) > 0 {
		existing, err := client.Enrollment.Query().
			Where(enrollment.StudentIDEQ(studentID), enrollment.CourseIDIn(courseIDs...)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range existing {
			existingCourseIDs[item.CourseID] = struct{}{}
		}
	}

	result := &EnrollmentBulkCreateResult{
		Enrollments:      make([]EnrollmentDTO, 0, len(inputs)),
		SkippedCourseIDs: make([]int, 0),
	}
	for _, input := range inputs {
		if _, exists := existingCourseIDs[input.CourseID]; exists {
			result.SkippedCourseIDs = append(result.SkippedCourseIDs, input.CourseID)
			continue
		}
		created, err := enrollmentCreateInStore(
			ctx,
			client,
			studentID,
			input.CourseID,
			input.BillingMode,
			input.ChargeMaterials,
			input.LessonPriceOverride,
			input.SubscriptionLessonPrice,
			input.Note,
		)
		if err != nil {
			return nil, err
		}
		result.Enrollments = append(result.Enrollments, *created)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
