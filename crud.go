package main

import (
	"errors"
	"fmt"
	"strings"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/payment"
	"langschool/ent/student"
)

// -------------------- Constants --------------------

const (
	// Course types
	CourseTypeGroup      = "group"
	CourseTypeIndividual = "individual"

	// Billing modes
	BillingModeSubscription = "subscription"
	BillingModePerLesson    = "per_lesson"
)

// -------------------- DTOs for Wails --------------------

type StudentDTO struct {
	ID       int    `json:"id"`
	FullName string `json:"fullName"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Note     string `json:"note"`
	IsActive bool   `json:"isActive"`
}

type CourseDTO struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Type              string  `json:"type"` // group|individual
	LessonPrice       float64 `json:"lessonPrice"`
	SubscriptionPrice float64 `json:"subscriptionPrice"`
}

type EnrollmentDTO struct {
	ID          int     `json:"id"`
	StudentID   int     `json:"studentId"`
	StudentName string  `json:"studentName"`
	CourseID    int     `json:"courseId"`
	CourseName  string  `json:"courseName"`
	BillingMode string  `json:"billingMode"` // subscription|per_lesson
	DiscountPct float64 `json:"discountPct"`
	Note        string  `json:"note"`
}

// -------------------- Validation helpers --------------------

// validateNonEmpty checks if a string is non-empty after trimming.
func validateNonEmpty(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// validatePrices checks if prices are non-negative.
func validatePrices(lessonPrice, subscriptionPrice float64) error {
	if lessonPrice < 0 || subscriptionPrice < 0 {
		return errors.New("prices must be >= 0")
	}
	return nil
}

// validateCourseType checks if the course type is valid.
func validateCourseType(courseType string) error {
	if courseType != CourseTypeGroup && courseType != CourseTypeIndividual {
		return fmt.Errorf("courseType must be '%s' or '%s'", CourseTypeGroup, CourseTypeIndividual)
	}
	return nil
}

// validateBillingMode checks if the billing mode is valid.
func validateBillingMode(billingMode string) error {
	if billingMode != BillingModeSubscription && billingMode != BillingModePerLesson {
		return fmt.Errorf("billingMode must be '%s' or '%s'", BillingModeSubscription, BillingModePerLesson)
	}
	return nil
}

// validateDiscountPct checks if the discount percentage is within valid range.
func validateDiscountPct(discountPct float64) error {
	if discountPct < 0 || discountPct > 100 {
		return errors.New("discountPct must be between 0 and 100")
	}
	return nil
}

// -------------------- DTO conversion helpers --------------------

// toStudentDTO converts an ent.Student to StudentDTO.
func toStudentDTO(s *ent.Student) StudentDTO {
	return StudentDTO{
		ID:       s.ID,
		FullName: s.FullName,
		Phone:    s.Phone,
		Email:    s.Email,
		Note:     s.Note,
		IsActive: s.IsActive,
	}
}

// toCourseDTO converts an ent.Course to CourseDTO.
func toCourseDTO(c *ent.Course) CourseDTO {
	return CourseDTO{
		ID:                c.ID,
		Name:              c.Name,
		Type:              string(c.Type),
		LessonPrice:       c.LessonPrice,
		SubscriptionPrice: c.SubscriptionPrice,
	}
}

// toEnrollmentDTO converts an ent.Enrollment to EnrollmentDTO.
// It extracts student and course names from edges if available.
func toEnrollmentDTO(e *ent.Enrollment) EnrollmentDTO {
	dto := EnrollmentDTO{
		ID:          e.ID,
		StudentID:   e.StudentID,
		CourseID:    e.CourseID,
		BillingMode: string(e.BillingMode),
		DiscountPct: e.DiscountPct,
		Note:        e.Note,
	}
	if e.Edges.Student != nil {
		dto.StudentName = e.Edges.Student.FullName
	}
	if e.Edges.Course != nil {
		dto.CourseName = e.Edges.Course.Name
	}
	return dto
}

// -------------------- Students CRUD --------------------

// StudentList returns active students by default. If includeInactive=true, returns all.
// If q is not empty, filters by name/phone/email (case-insensitive where supported).
func (a *App) StudentList(q string, includeInactive bool) ([]StudentDTO, error) {
	ctx := a.ctx
	q = strings.TrimSpace(q)

	query := a.db.Ent.Student.Query()

	if !includeInactive {
		query = query.Where(student.IsActiveEQ(true))
	}

	if q != "" {
		// ContainsFold is supported for SQLite in ent.
		// We apply OR across a few fields.
		query = query.Where(
			student.Or(
				student.FullNameContainsFold(q),
				student.PhoneContainsFold(q),
				student.EmailContainsFold(q),
			),
		)
	}

	studs, err := query.Order(ent.Asc(student.FieldFullName)).All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]StudentDTO, 0, len(studs))
	for _, s := range studs {
		out = append(out, toStudentDTO(s))
	}
	return out, nil
}

func (a *App) StudentGet(id int) (*StudentDTO, error) {
	s, err := a.db.Ent.Student.Get(a.ctx, id)
	if err != nil {
		return nil, err
	}
	dto := toStudentDTO(s)
	return &dto, nil
}

func (a *App) StudentCreate(fullName, phone, email, note string) (*StudentDTO, error) {
	fullName = strings.TrimSpace(fullName)
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}

	s, err := a.db.Ent.Student.Create().
		SetFullName(fullName).
		SetPhone(strings.TrimSpace(phone)).
		SetEmail(strings.TrimSpace(email)).
		SetNote(strings.TrimSpace(note)).
		SetIsActive(true).
		Save(a.ctx)
	if err != nil {
		return nil, err
	}

	dto := toStudentDTO(s)
	return &dto, nil
}

func (a *App) StudentUpdate(id int, fullName, phone, email, note string) (*StudentDTO, error) {
	fullName = strings.TrimSpace(fullName)
	if err := validateNonEmpty(fullName, "fullName"); err != nil {
		return nil, err
	}

	s, err := a.db.Ent.Student.UpdateOneID(id).
		SetFullName(fullName).
		SetPhone(strings.TrimSpace(phone)).
		SetEmail(strings.TrimSpace(email)).
		SetNote(strings.TrimSpace(note)).
		Save(a.ctx)
	if err != nil {
		return nil, err
	}

	dto := toStudentDTO(s)
	return &dto, nil
}

// StudentSetActive deactivates (or activates) a student without deleting history.
func (a *App) StudentSetActive(id int, active bool) error {
	_, err := a.db.Ent.Student.UpdateOneID(id).
		SetIsActive(active).
		Save(a.ctx)
	return err
}

// StudentDelete deletes a student. If inactive, automatically removes enrollments and attendance.
// Prevents deletion if student has invoices or payments (financial records).
func (a *App) StudentDelete(id int) error {
	ctx := a.ctx

	// Check if student is active
	st, err := a.db.Ent.Student.Get(ctx, id)
	if err != nil {
		return err
	}
	if st.IsActive {
		return errors.New("cannot delete active student; deactivate first")
	}

	// Check for invoices - these are financial records and should not be auto-deleted
	hasInvoices, err := a.db.Ent.Invoice.Query().
		Where(invoice.StudentIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasInvoices {
		return errors.New("cannot delete student: has invoices (financial records)")
	}

	// Check for payments - these are financial records and should not be auto-deleted
	hasPayments, err := a.db.Ent.Payment.Query().
		Where(payment.StudentIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasPayments {
		return errors.New("cannot delete student: has payments (financial records)")
	}

	// Use a transaction to ensure all deletions succeed or fail together
	tx, err := a.db.Ent.Tx(ctx)
	if err != nil {
		return err
	}
	// Defer rollback - will be a no-op if commit succeeds
	defer tx.Rollback()

	// Auto-delete attendance records (these are draft data that can be safely removed)
	_, err = tx.AttendanceMonth.Delete().
		Where(attendancemonth.StudentIDEQ(id)).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Auto-delete enrollments (these are relationships that can be safely removed)
	_, err = tx.Enrollment.Delete().
		Where(enrollment.StudentIDEQ(id)).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Finally, delete the student
	err = tx.Student.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit()
}

// -------------------- Courses CRUD --------------------

// CourseList lists courses. Optional search by name.
func (a *App) CourseList(q string) ([]CourseDTO, error) {
	ctx := a.ctx
	q = strings.TrimSpace(q)

	query := a.db.Ent.Course.Query()
	if q != "" {
		query = query.Where(course.NameContainsFold(q))
	}

	cs, err := query.Order(ent.Asc(course.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]CourseDTO, 0, len(cs))
	for _, c := range cs {
		out = append(out, toCourseDTO(c))
	}
	return out, nil
}

func (a *App) CourseGet(id int) (*CourseDTO, error) {
	c, err := a.db.Ent.Course.Get(a.ctx, id)
	if err != nil {
		return nil, err
	}
	dto := toCourseDTO(c)
	return &dto, nil
}

func (a *App) CourseCreate(name, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = strings.TrimSpace(name)
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

	c, err := a.db.Ent.Course.Create().
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPrice(lessonPrice).
		SetSubscriptionPrice(subscriptionPrice).
		Save(a.ctx)
	if err != nil {
		return nil, err
	}

	dto := toCourseDTO(c)
	return &dto, nil
}

func (a *App) CourseUpdate(id int, name, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = strings.TrimSpace(name)
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

	c, err := a.db.Ent.Course.UpdateOneID(id).
		SetName(name).
		SetType(course.Type(courseType)).
		SetLessonPrice(lessonPrice).
		SetSubscriptionPrice(subscriptionPrice).
		Save(a.ctx)
	if err != nil {
		return nil, err
	}

	dto := toCourseDTO(c)
	return &dto, nil
}

// CourseDelete deletes a course only if no enrollments reference it.
// This keeps history safe without adding is_active field right now.
func (a *App) CourseDelete(id int) error {
	ctx := a.ctx

	used, err := a.db.Ent.Enrollment.Query().
		Where(enrollment.CourseIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if used {
		return errors.New("cannot delete course: it has enrollments; remove enrollments first or keep course")
	}

	return a.db.Ent.Course.DeleteOneID(id).Exec(ctx)
}

// -------------------- Enrollments CRUD --------------------

// EnrollmentList lists enrollments, optionally filtered by studentID or courseID.
// NOTE: activeOnly/date filters were removed; API now uses only (studentID, courseID).
func (a *App) EnrollmentList(studentID *int, courseID *int) ([]EnrollmentDTO, error) {
	ctx := a.ctx

	q := a.db.Ent.Enrollment.Query().WithStudent().WithCourse()

	if studentID != nil {
		q = q.Where(enrollment.StudentIDEQ(*studentID))
	}
	if courseID != nil {
		q = q.Where(enrollment.CourseIDEQ(*courseID))
	}

	ens, err := q.Order(ent.Asc(enrollment.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]EnrollmentDTO, 0, len(ens))
	for _, e := range ens {
		out = append(out, toEnrollmentDTO(e))
	}
	return out, nil
}

// EnrollmentCreate creates an enrollment (student -> course).
// To prevent duplicates, it rejects if an enrollment for the same pair already exists.
func (a *App) EnrollmentCreate(studentID, courseID int, billingMode string, discountPct float64, note string) (*EnrollmentDTO, error) {
	ctx := a.ctx

	if studentID <= 0 || courseID <= 0 {
		return nil, errors.New("studentID and courseID must be > 0")
	}
	billingMode = strings.TrimSpace(billingMode)
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(discountPct); err != nil {
		return nil, err
	}

	// Ensure referenced entities exist and student is active.
	st, err := a.db.Ent.Student.Get(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if !st.IsActive {
		return nil, errors.New("cannot enroll a deactivated student")
	}
	if _, err := a.db.Ent.Course.Get(ctx, courseID); err != nil {
		return nil, err
	}

	// Prevent duplicate enrollment for the same student-course pair.
	exists, err := a.db.Ent.Enrollment.Query().
		Where(
			enrollment.StudentIDEQ(studentID),
			enrollment.CourseIDEQ(courseID),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("an enrollment for this student and course already exists")
	}

	e, err := a.db.Ent.Enrollment.Create().
		SetStudentID(studentID).
		SetCourseID(courseID).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetDiscountPct(discountPct).
		SetNote(strings.TrimSpace(note)).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Return full DTO with names.
	e2, err := a.db.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(e.ID)).
		WithStudent().
		WithCourse().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	dto := toEnrollmentDTO(e2)
	return &dto, nil
}

func (a *App) EnrollmentUpdate(enrollmentID int, billingMode string, discountPct float64, note string) (*EnrollmentDTO, error) {
	ctx := a.ctx

	billingMode = strings.TrimSpace(billingMode)
	if err := validateBillingMode(billingMode); err != nil {
		return nil, err
	}
	if err := validateDiscountPct(discountPct); err != nil {
		return nil, err
	}

	_, err := a.db.Ent.Enrollment.UpdateOneID(enrollmentID).
		SetBillingMode(enrollment.BillingMode(billingMode)).
		SetDiscountPct(discountPct).
		SetNote(strings.TrimSpace(note)).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	// Return full DTO with names.
	e2, err := a.db.Ent.Enrollment.Query().
		Where(enrollment.IDEQ(enrollmentID)).
		WithStudent().
		WithCourse().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	dto := toEnrollmentDTO(e2)
	return &dto, nil
}
