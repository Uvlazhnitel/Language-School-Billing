package backend

import "context"

func (s *Service) StudentOnboard(ctx context.Context, studentInput StudentCreateInput, enrollmentInput *EnrollmentCreateInput) (*StudentOnboardingResult, error) {
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

	result := &StudentOnboardingResult{Student: *studentDTO}
	if enrollmentInput != nil {
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
		result.Enrollment = enrollmentDTO
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
