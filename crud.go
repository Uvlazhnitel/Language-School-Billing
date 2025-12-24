package main

import (
	"errors"
	"strings"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/invoice"
	"langschool/ent/payment"
	"langschool/ent/student"
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
		out = append(out, StudentDTO{
			ID: s.ID, FullName: s.FullName, Phone: s.Phone, Email: s.Email, Note: s.Note, IsActive: s.IsActive,
		})
	}
	return out, nil
}

func (a *App) StudentGet(id int) (*StudentDTO, error) {
	s, err := a.db.Ent.Student.Get(a.ctx, id)
	if err != nil {
		return nil, err
	}
	return &StudentDTO{
		ID: s.ID, FullName: s.FullName, Phone: s.Phone, Email: s.Email, Note: s.Note, IsActive: s.IsActive,
	}, nil
}

func (a *App) StudentCreate(fullName, phone, email, note string) (*StudentDTO, error) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return nil, errors.New("fullName is required")
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

	return &StudentDTO{
		ID: s.ID, FullName: s.FullName, Phone: s.Phone, Email: s.Email, Note: s.Note, IsActive: s.IsActive,
	}, nil
}

func (a *App) StudentUpdate(id int, fullName, phone, email, note string) (*StudentDTO, error) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return nil, errors.New("fullName is required")
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

	return &StudentDTO{
		ID: s.ID, FullName: s.FullName, Phone: s.Phone, Email: s.Email, Note: s.Note, IsActive: s.IsActive,
	}, nil
}

// StudentSetActive deactivates (or activates) a student without deleting history.
func (a *App) StudentSetActive(id int, active bool) error {
	_, err := a.db.Ent.Student.UpdateOneID(id).
		SetIsActive(active).
		Save(a.ctx)
	return err
}

// StudentDelete deletes a student only if inactive and no history exists.
// Checks for enrollments, attendance records, invoices, and payments before allowing deletion.
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

	// Check for enrollments
	hasEnrollments, err := a.db.Ent.Enrollment.Query().
		Where(enrollment.StudentIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasEnrollments {
		return errors.New("cannot delete student: has enrollments")
	}

	// Check for attendance records
	hasAttendance, err := a.db.Ent.AttendanceMonth.Query().
		Where(attendancemonth.StudentIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasAttendance {
		return errors.New("cannot delete student: has attendance records")
	}

	// Check for invoices
	hasInvoices, err := a.db.Ent.Invoice.Query().
		Where(invoice.StudentIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasInvoices {
		return errors.New("cannot delete student: has invoices")
	}

	// Check for payments
	hasPayments, err := a.db.Ent.Payment.Query().
		Where(payment.StudentIDEQ(id)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if hasPayments {
		return errors.New("cannot delete student: has payments")
	}

	return a.db.Ent.Student.DeleteOneID(id).Exec(ctx)
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
		out = append(out, CourseDTO{
			ID:                c.ID,
			Name:              c.Name,
			Type:              string(c.Type),
			LessonPrice:       c.LessonPrice,
			SubscriptionPrice: c.SubscriptionPrice,
		})
	}
	return out, nil
}

func (a *App) CourseGet(id int) (*CourseDTO, error) {
	c, err := a.db.Ent.Course.Get(a.ctx, id)
	if err != nil {
		return nil, err
	}
	return &CourseDTO{
		ID:                c.ID,
		Name:              c.Name,
		Type:              string(c.Type),
		LessonPrice:       c.LessonPrice,
		SubscriptionPrice: c.SubscriptionPrice,
	}, nil
}

func (a *App) CourseCreate(name, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = strings.TrimSpace(name)
	courseType = strings.TrimSpace(courseType)

	if name == "" {
		return nil, errors.New("name is required")
	}
	if courseType != "group" && courseType != "individual" {
		return nil, errors.New("courseType must be 'group' or 'individual'")
	}
	if lessonPrice < 0 || subscriptionPrice < 0 {
		return nil, errors.New("prices must be >= 0")
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

	return &CourseDTO{
		ID:                c.ID,
		Name:              c.Name,
		Type:              string(c.Type),
		LessonPrice:       c.LessonPrice,
		SubscriptionPrice: c.SubscriptionPrice,
	}, nil
}

func (a *App) CourseUpdate(id int, name, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	name = strings.TrimSpace(name)
	courseType = strings.TrimSpace(courseType)

	if name == "" {
		return nil, errors.New("name is required")
	}
	if courseType != "group" && courseType != "individual" {
		return nil, errors.New("courseType must be 'group' or 'individual'")
	}
	if lessonPrice < 0 || subscriptionPrice < 0 {
		return nil, errors.New("prices must be >= 0")
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

	return &CourseDTO{
		ID:                c.ID,
		Name:              c.Name,
		Type:              string(c.Type),
		LessonPrice:       c.LessonPrice,
		SubscriptionPrice: c.SubscriptionPrice,
	}, nil
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
		stName := ""
		if e.Edges.Student != nil {
			stName = e.Edges.Student.FullName
		}
		cName := ""
		if e.Edges.Course != nil {
			cName = e.Edges.Course.Name
		}
		out = append(out, EnrollmentDTO{
			ID:          e.ID,
			StudentID:   e.StudentID,
			StudentName: stName,
			CourseID:    e.CourseID,
			CourseName:  cName,
			BillingMode: string(e.BillingMode),
			DiscountPct: e.DiscountPct,
			Note:        e.Note,
		})
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
	if billingMode != "subscription" && billingMode != "per_lesson" {
		return nil, errors.New("billingMode must be 'subscription' or 'per_lesson'")
	}
	if discountPct < 0 || discountPct > 100 {
		return nil, errors.New("discountPct must be between 0 and 100")
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

	return &EnrollmentDTO{
		ID:          e2.ID,
		StudentID:   e2.StudentID,
		StudentName: e2.Edges.Student.FullName,
		CourseID:    e2.CourseID,
		CourseName:  e2.Edges.Course.Name,
		BillingMode: string(e2.BillingMode),
		DiscountPct: e2.DiscountPct,
		Note:        e2.Note,
	}, nil
}

func (a *App) EnrollmentUpdate(enrollmentID int, billingMode string, discountPct float64, note string) (*EnrollmentDTO, error) {
	ctx := a.ctx

	billingMode = strings.TrimSpace(billingMode)
	if billingMode != "subscription" && billingMode != "per_lesson" {
		return nil, errors.New("billingMode must be 'subscription' or 'per_lesson'")
	}
	if discountPct < 0 || discountPct > 100 {
		return nil, errors.New("discountPct must be between 0 and 100")
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

	return &EnrollmentDTO{
		ID:          e2.ID,
		StudentID:   e2.StudentID,
		StudentName: e2.Edges.Student.FullName,
		CourseID:    e2.CourseID,
		CourseName:  e2.Edges.Course.Name,
		BillingMode: string(e2.BillingMode),
		DiscountPct: e2.DiscountPct,
		Note:        e2.Note,
	}, nil
}

