package backend

import (
	"context"
	"errors"
	"strings"
	"time"

	"langschool/ent"
	"langschool/ent/enrollment"
	entinvoice "langschool/ent/invoice"
	"langschool/internal/money"
)

var enrollmentCurrentTime = time.Now

func (s *Service) EnrollmentList(ctx context.Context, studentID *int, courseID *int) ([]EnrollmentDTO, error) {
	q := s.rt.DB.Ent.Enrollment.Query().
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) {
			cq.WithTeacher()
		})
	if studentID != nil {
		q = q.Where(enrollment.StudentIDEQ(*studentID))
	}
	if courseID != nil {
		q = q.Where(enrollment.CourseIDEQ(*courseID))
	}
	items, err := q.Order(ent.Asc(enrollment.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]EnrollmentDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toEnrollmentDTO(item))
	}
	return out, nil
}

func (s *Service) EnrollmentCreate(ctx context.Context, studentID, courseID int, billingMode string, chargeMaterials bool, lessonPriceOverride, subscriptionLessonPrice float64, note string) (*EnrollmentDTO, error) {
	if studentID <= 0 || courseID <= 0 {
		return nil, errors.New("studentID and courseID must be > 0")
	}
	billingMode = strings.TrimSpace(billingMode)
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateLessonPriceOverride(lessonPriceOverride); err != nil {
		return nil, err
	}
	if err := validateSubscriptionLessonPrice(subscriptionLessonPrice); err != nil {
		return nil, err
	}
	if billingMode != BillingModePerLesson {
		lessonPriceOverride = -1
	} else if lessonPriceOverride == 0 {
		lessonPriceOverride = -1
	}
	if billingMode != BillingModeSubscription {
		subscriptionLessonPrice = 0
	}
	st, err := s.rt.DB.Ent.Student.Get(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if !st.IsActive {
		return nil, errors.New("cannot enroll a deactivated student")
	}
	if _, err := s.rt.DB.Ent.Course.Get(ctx, courseID); err != nil {
		return nil, err
	}
	exists, err := s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.StudentIDEQ(studentID), enrollment.CourseIDEQ(courseID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("an enrollment for this student and course already exists")
	}
	item, err := s.rt.DB.Ent.Enrollment.Create().
		SetStudentID(studentID).
		SetCourseID(courseID).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetChargeMaterials(chargeMaterials).
		SetLessonPriceOverrideCents(money.EurosToCents(lessonPriceOverride)).
		SetSubscriptionLessonPriceCents(money.EurosToCents(subscriptionLessonPrice)).
		SetNote(sanitizeInput(note)).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	item, err = s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(item.ID)).
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) { cq.WithTeacher() }).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toEnrollmentDTO(item)
	return &dto, nil
}

func (s *Service) EnrollmentUpdate(ctx context.Context, enrollmentID int, billingMode string, chargeMaterials bool, lessonPriceOverride, subscriptionLessonPrice float64, note string) (*EnrollmentDTO, error) {
	billingMode = strings.TrimSpace(billingMode)
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateLessonPriceOverride(lessonPriceOverride); err != nil {
		return nil, err
	}
	if err := validateSubscriptionLessonPrice(subscriptionLessonPrice); err != nil {
		return nil, err
	}
	if billingMode != BillingModePerLesson {
		lessonPriceOverride = -1
	} else if lessonPriceOverride == 0 {
		lessonPriceOverride = -1
	}
	if billingMode != BillingModeSubscription {
		subscriptionLessonPrice = 0
	}
	before, err := s.rt.DB.Ent.Enrollment.Get(ctx, enrollmentID)
	if err != nil {
		return nil, err
	}
	if _, err := s.rt.DB.Ent.Enrollment.UpdateOneID(enrollmentID).
		AddVersion(1).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetChargeMaterials(chargeMaterials).
		SetLessonPriceOverrideCents(money.EurosToCents(lessonPriceOverride)).
		SetSubscriptionLessonPriceCents(money.EurosToCents(subscriptionLessonPrice)).
		SetNote(sanitizeInput(note)).
		Save(ctx); err != nil {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(enrollmentID)).
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) { cq.WithTeacher() }).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.rebuildInvoiceForEnrollmentChange(ctx, before, billingMode, chargeMaterials, lessonPriceOverride, subscriptionLessonPrice); err != nil {
		return nil, err
	}
	dto := toEnrollmentDTO(item)
	return &dto, nil
}

func (s *Service) EnrollmentUpdateWithVersion(ctx context.Context, enrollmentID, version int, billingMode string, chargeMaterials bool, lessonPriceOverride, subscriptionLessonPrice float64, note string) (*EnrollmentDTO, error) {
	billingMode = strings.TrimSpace(billingMode)
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateLessonPriceOverride(lessonPriceOverride); err != nil {
		return nil, err
	}
	if err := validateSubscriptionLessonPrice(subscriptionLessonPrice); err != nil {
		return nil, err
	}
	if billingMode != BillingModePerLesson {
		lessonPriceOverride = -1
	} else if lessonPriceOverride == 0 {
		lessonPriceOverride = -1
	}
	if billingMode != BillingModeSubscription {
		subscriptionLessonPrice = 0
	}
	before, err := s.rt.DB.Ent.Enrollment.Get(ctx, enrollmentID)
	if err != nil {
		return nil, err
	}
	if _, err := s.rt.DB.Ent.Enrollment.UpdateOneID(enrollmentID).
		Where(enrollment.VersionEQ(version)).
		SetVersion(version + 1).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetChargeMaterials(chargeMaterials).
		SetLessonPriceOverrideCents(money.EurosToCents(lessonPriceOverride)).
		SetSubscriptionLessonPriceCents(money.EurosToCents(subscriptionLessonPrice)).
		SetNote(sanitizeInput(note)).
		Save(ctx); err != nil {
		return nil, staleOnNotFound(err)
	}
	item, err := s.rt.DB.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(enrollmentID)).
		WithStudent().
		WithCourse(func(cq *ent.CourseQuery) { cq.WithTeacher() }).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.rebuildInvoiceForEnrollmentChange(ctx, before, billingMode, chargeMaterials, lessonPriceOverride, subscriptionLessonPrice); err != nil {
		return nil, err
	}
	dto := toEnrollmentDTO(item)
	return &dto, nil
}

func (s *Service) rebuildInvoiceForEnrollmentChange(ctx context.Context, before *ent.Enrollment, billingMode string, chargeMaterials bool, lessonPriceOverride, subscriptionLessonPrice float64) error {
	if before == nil || !enrollmentInvoiceAffectingFieldsChanged(before, billingMode, chargeMaterials, lessonPriceOverride, subscriptionLessonPrice) {
		return nil
	}
	if s.rt == nil || s.rt.Invoice == nil || s.rt.DB.Ent == nil {
		return nil
	}
	year, month, err := s.resolveEnrollmentInvoiceRebuildMonth(ctx, before.StudentID)
	if err != nil {
		return err
	}
	existing, err := s.rt.DB.Ent.Invoice.Query().
		Where(
			entinvoice.StudentIDEQ(before.StudentID),
			entinvoice.PeriodYearEQ(year),
			entinvoice.PeriodMonthEQ(month),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.Status != entinvoice.StatusDraft {
		return nil
	}
	_, err = s.rt.Invoice.RebuildStudentDraft(ctx, before.StudentID, year, month)
	return err
}

func (s *Service) resolveEnrollmentInvoiceRebuildMonth(ctx context.Context, studentID int) (int, int, error) {
	now := enrollmentCurrentTime()
	year, month := now.Year(), int(now.Month())
	if s.rt == nil || s.rt.DB.Ent == nil {
		return year, month, nil
	}
	currentInvoice, err := s.rt.DB.Ent.Invoice.Query().
		Where(
			entinvoice.StudentIDEQ(studentID),
			entinvoice.PeriodYearEQ(year),
			entinvoice.PeriodMonthEQ(month),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return year, month, nil
	}
	if err != nil {
		return 0, 0, err
	}
	if invoiceLocksEnrollmentChangesToNextMonth(currentInvoice.Status) {
		nextYear, nextMonth := nextBillingMonth(year, month)
		return nextYear, nextMonth, nil
	}
	return year, month, nil
}

func enrollmentInvoiceAffectingFieldsChanged(before *ent.Enrollment, billingMode string, chargeMaterials bool, lessonPriceOverride, subscriptionLessonPrice float64) bool {
	if before == nil {
		return false
	}
	nextLessonPriceOverrideCents := normalizedLessonPriceOverrideCents(billingMode, lessonPriceOverride)
	nextSubscriptionLessonPriceCents := normalizedSubscriptionLessonPriceCents(billingMode, subscriptionLessonPrice)
	return before.BillingMode != enrollment.BillingMode(billingMode) ||
		before.ChargeMaterials != chargeMaterials ||
		before.LessonPriceOverrideCents != nextLessonPriceOverrideCents ||
		before.SubscriptionLessonPriceCents != nextSubscriptionLessonPriceCents
}

func normalizedLessonPriceOverrideCents(billingMode string, lessonPriceOverride float64) int64 {
	if billingMode != BillingModePerLesson {
		return -1
	}
	if lessonPriceOverride == 0 {
		return -1
	}
	return money.EurosToCents(lessonPriceOverride)
}

func normalizedSubscriptionLessonPriceCents(billingMode string, subscriptionLessonPrice float64) int64 {
	if billingMode != BillingModeSubscription {
		return 0
	}
	return money.EurosToCents(subscriptionLessonPrice)
}

func invoiceLocksEnrollmentChangesToNextMonth(status entinvoice.Status) bool {
	return status == entinvoice.StatusIssued ||
		status == entinvoice.StatusIssuedPendingPdf ||
		status == entinvoice.StatusPaid ||
		status == entinvoice.StatusPaidPendingPdf
}

func nextBillingMonth(year, month int) (int, int) {
	if month == 12 {
		return year + 1, 1
	}
	return year, month + 1
}

func toEnrollmentDTO(e *ent.Enrollment) EnrollmentDTO {
	lessonPriceOverride := 0.0
	if e.BillingMode == enrollment.BillingModePerLesson {
		if e.LessonPriceOverrideCents >= 0 {
			lessonPriceOverride = money.CentsToEuros(e.LessonPriceOverrideCents)
		} else if e.Edges.Course != nil {
			lessonPriceOverride = money.CentsToEuros(e.Edges.Course.LessonPriceCents)
		}
	}
	dto := EnrollmentDTO{
		ID:                      e.ID,
		Version:                 e.Version,
		StudentID:               e.StudentID,
		CourseID:                e.CourseID,
		BillingMode:             string(e.BillingMode),
		ChargeMaterials:         e.ChargeMaterials,
		LessonPriceOverride:     lessonPriceOverride,
		SubscriptionLessonPrice: money.CentsToEuros(e.SubscriptionLessonPriceCents),
		Note:                    e.Note,
		CreatedAt:               formatOptionalTime(e.CreatedAt),
	}
	if e.Edges.Student != nil {
		dto.StudentName = e.Edges.Student.FullName
	}
	if e.Edges.Course != nil {
		dto.CourseName = e.Edges.Course.Name
		dto.CourseType = string(e.Edges.Course.Type)
		dto.TeacherName = e.Edges.Course.TeacherName
		if e.Edges.Course.TeacherID != nil {
			id := *e.Edges.Course.TeacherID
			dto.TeacherID = &id
		}
		if e.Edges.Course.Edges.Teacher != nil {
			dto.TeacherName = e.Edges.Course.Edges.Teacher.FullName
			id := e.Edges.Course.Edges.Teacher.ID
			dto.TeacherID = &id
		}
	}
	return dto
}
