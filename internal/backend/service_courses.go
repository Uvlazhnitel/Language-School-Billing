package backend

import (
	"context"
	"errors"
	"strings"

	"langschool/ent"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/teacher"
	"langschool/internal/money"
)

func (s *Service) TeacherList(ctx context.Context, q string) ([]TeacherDTO, error) {
	q = strings.TrimSpace(q)
	query := s.rt.DB.Ent.Teacher.Query().Where(teacher.IsActiveEQ(true))
	if q != "" {
		query = query.Where(teacher.FullNameContainsFold(q))
	}
	items, err := query.Order(ent.Asc(teacher.FieldFullName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TeacherDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toTeacherDTO(item))
	}
	return out, nil
}

func (s *Service) TeacherCreate(ctx context.Context, fullName string) (*TeacherDTO, error) {
	fullName = sanitizeInput(normalizePersonNameInput(fullName))
	if err := validatePersonName(fullName, "fullName", true); err != nil {
		return nil, err
	}
	existing, err := s.rt.DB.Ent.Teacher.Query().Where(teacher.FullNameEqualFold(fullName)).Only(ctx)
	if err == nil {
		dto := toTeacherDTO(existing)
		return &dto, nil
	}
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	item, err := s.rt.DB.Ent.Teacher.Create().SetFullName(fullName).SetIsActive(true).Save(ctx)
	if err != nil {
		return nil, err
	}
	dto := toTeacherDTO(item)
	return &dto, nil
}

func (s *Service) CourseList(ctx context.Context, q string) ([]CourseDTO, error) {
	q = strings.TrimSpace(q)
	query := s.rt.DB.Ent.Course.Query().WithTeacher()
	if q != "" {
		query = query.Where(course.Or(
			course.NameContainsFold(q),
			course.TeacherNameContainsFold(q),
			course.HasTeacherWith(teacher.FullNameContainsFold(q)),
		))
	}
	items, err := query.Order(ent.Asc(course.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CourseDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toCourseDTO(item))
	}
	return out, nil
}

func (s *Service) CourseGet(ctx context.Context, id int) (*CourseDTO, error) {
	item, err := s.rt.DB.Ent.Course.Query().Where(course.IDEQ(id)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseCreate(ctx context.Context, name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = sanitizeInput(name)
	courseType = strings.TrimSpace(courseType)
	if err := validateNonEmpty(name, "name"); err != nil {
		return nil, err
	}
	if err := validateCourseType(courseType); err != nil {
		return nil, err
	}
	if err := validatePrices(lessonPrice, subscriptionPrice); err != nil {
		return nil, err
	}
	selectedTeacher, err := s.resolveTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	create := s.rt.DB.Ent.Course.Create().
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPriceCents(money.EurosToCents(lessonPrice)).
		SetSubscriptionPriceCents(money.EurosToCents(subscriptionPrice))
	if selectedTeacher != nil {
		create = create.SetTeacherID(selectedTeacher.ID).SetTeacherName(selectedTeacher.FullName)
	} else {
		create = create.SetTeacherName("")
	}
	item, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	item, err = s.rt.DB.Ent.Course.Query().Where(course.IDEQ(item.ID)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseUpdate(ctx context.Context, id int, name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = sanitizeInput(name)
	courseType = strings.TrimSpace(courseType)
	if err := validateNonEmpty(name, "name"); err != nil {
		return nil, err
	}
	if err := validateCourseType(courseType); err != nil {
		return nil, err
	}
	if err := validatePrices(lessonPrice, subscriptionPrice); err != nil {
		return nil, err
	}
	selectedTeacher, err := s.resolveTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	update := s.rt.DB.Ent.Course.UpdateOneID(id).
		AddVersion(1).
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPriceCents(money.EurosToCents(lessonPrice)).
		SetSubscriptionPriceCents(money.EurosToCents(subscriptionPrice))
	if selectedTeacher != nil {
		update = update.SetTeacherID(selectedTeacher.ID).SetTeacherName(selectedTeacher.FullName)
	} else {
		update = update.ClearTeacher().SetTeacherName("")
	}
	item, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	item, err = s.rt.DB.Ent.Course.Query().Where(course.IDEQ(item.ID)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseUpdateWithVersion(ctx context.Context, id, version int, name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = sanitizeInput(name)
	courseType = strings.TrimSpace(courseType)
	if err := validateVersion(version); err != nil {
		return nil, err
	}
	if err := validateNonEmpty(name, "name"); err != nil {
		return nil, err
	}
	if err := validateCourseType(courseType); err != nil {
		return nil, err
	}
	if err := validatePrices(lessonPrice, subscriptionPrice); err != nil {
		return nil, err
	}
	selectedTeacher, err := s.resolveTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	update := s.rt.DB.Ent.Course.UpdateOneID(id).
		Where(course.VersionEQ(version)).
		SetVersion(version + 1).
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPriceCents(money.EurosToCents(lessonPrice)).
		SetSubscriptionPriceCents(money.EurosToCents(subscriptionPrice))
	if selectedTeacher != nil {
		update = update.SetTeacherID(selectedTeacher.ID).SetTeacherName(selectedTeacher.FullName)
	} else {
		update = update.ClearTeacher().SetTeacherName("")
	}
	item, err := update.Save(ctx)
	if err != nil {
		return nil, staleOnNotFound(err)
	}
	item, err = s.rt.DB.Ent.Course.Query().Where(course.IDEQ(item.ID)).WithTeacher().Only(ctx)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(item)
	return &dto, nil
}

func (s *Service) CourseDelete(ctx context.Context, id int) error {
	enrollmentCount, err := s.rt.DB.Ent.Enrollment.Query().Where(enrollment.CourseIDEQ(id)).Count(ctx)
	if err != nil {
		return err
	}
	if enrollmentCount > 0 {
		return errors.New("cannot delete course: it has enrollments; remove enrollments first or keep course")
	}
	err = s.rt.DB.Ent.Course.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return errors.New("cannot delete course: it is still referenced by existing records")
		}
		return err
	}
	return nil
}

func (s *Service) CourseDeleteWithVersion(ctx context.Context, id, version int) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	enrollmentCount, err := s.rt.DB.Ent.Enrollment.Query().Where(enrollment.CourseIDEQ(id)).Count(ctx)
	if err != nil {
		return err
	}
	if enrollmentCount > 0 {
		return errors.New("cannot delete course: it has enrollments; remove enrollments first or keep course")
	}
	err = s.rt.DB.Ent.Course.DeleteOneID(id).Where(course.VersionEQ(version)).Exec(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return errors.New("cannot delete course: it is still referenced by existing records")
		}
		return staleOnNotFound(err)
	}
	return nil
}

func (s *Service) resolveTeacher(ctx context.Context, teacherID *int) (*ent.Teacher, error) {
	if teacherID == nil {
		return nil, nil
	}
	if *teacherID <= 0 {
		return nil, errors.New("teacherID must be > 0 when provided")
	}
	return s.rt.DB.Ent.Teacher.Get(ctx, *teacherID)
}

func toCourseDTO(c *ent.Course) CourseDTO {
	dto := CourseDTO{
		ID:                c.ID,
		Version:           c.Version,
		Name:              c.Name,
		TeacherName:       c.TeacherName,
		Type:              string(c.Type),
		LessonPrice:       money.CentsToEuros(c.LessonPriceCents),
		SubscriptionPrice: money.CentsToEuros(c.SubscriptionPriceCents),
	}
	if c.TeacherID != nil {
		id := *c.TeacherID
		dto.TeacherID = &id
	}
	if c.Edges.Teacher != nil {
		dto.TeacherName = c.Edges.Teacher.FullName
		id := c.Edges.Teacher.ID
		dto.TeacherID = &id
	}
	return dto
}

func toTeacherDTO(t *ent.Teacher) TeacherDTO {
	return TeacherDTO{
		ID:       t.ID,
		FullName: t.FullName,
		IsActive: t.IsActive,
	}
}
