package main

import (
	"errors"

	sharedapp "langschool/internal/app"
	"langschool/internal/backend"
)

const (
	CourseTypeGroup         = sharedapp.CourseTypeGroup
	CourseTypeIndividual    = sharedapp.CourseTypeIndividual
	BillingModeSubscription = sharedapp.BillingModeSubscription
	BillingModePerLesson    = sharedapp.BillingModePerLesson
)

type StudentDTO = backend.StudentDTO
type CourseDTO = backend.CourseDTO
type EnrollmentDTO = backend.EnrollmentDTO
type CourseMonthSubscriptionDTO = backend.CourseMonthSubscriptionDTO
type TeacherDTO = backend.TeacherDTO

func (a *App) crudService() (*backend.Service, error) {
	svc := a.ensureBackendService()
	if svc == nil {
		return nil, errors.New("backend service is not initialized")
	}
	return svc, nil
}

func (a *App) StudentList(q string, includeInactive bool) ([]StudentDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.StudentList(a.appContext(), q, includeInactive)
}

func (a *App) StudentGet(id int) (*StudentDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.StudentGet(a.appContext(), id)
}

func (a *App) StudentCreate(fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.StudentCreate(a.appContext(), fullName, personalCode, phone, email, note, isMinor, payerName, payerRole)
}

func (a *App) StudentUpdate(id int, fullName, personalCode, phone, email, note string, isMinor bool, payerName, payerRole string) (*StudentDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.StudentUpdate(a.appContext(), id, fullName, personalCode, phone, email, note, isMinor, payerName, payerRole)
}

func (a *App) StudentSetActive(id int, active bool) error {
	svc, err := a.crudService()
	if err != nil {
		return err
	}
	return svc.StudentSetActive(a.appContext(), id, active)
}

func (a *App) StudentDelete(id int) error {
	svc, err := a.crudService()
	if err != nil {
		return err
	}
	return svc.StudentDelete(a.appContext(), id)
}

func (a *App) TeacherList(q string) ([]TeacherDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.TeacherList(a.appContext(), q)
}

func (a *App) TeacherCreate(fullName string) (*TeacherDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.TeacherCreate(a.appContext(), fullName)
}

func (a *App) CourseList(q string) ([]CourseDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.CourseList(a.appContext(), q)
}

func (a *App) CourseGet(id int) (*CourseDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.CourseGet(a.appContext(), id)
}

func (a *App) CourseCreate(name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.CourseCreate(a.appContext(), name, teacherID, courseType, lessonPrice, subscriptionPrice)
}

func (a *App) CourseUpdate(id int, name string, teacherID *int, courseType string, lessonPrice, subscriptionPrice float64) (*CourseDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.CourseUpdate(a.appContext(), id, name, teacherID, courseType, lessonPrice, subscriptionPrice)
}

func (a *App) CourseDelete(id int) error {
	svc, err := a.crudService()
	if err != nil {
		return err
	}
	return svc.CourseDelete(a.appContext(), id)
}

func (a *App) EnrollmentList(studentID *int, courseID *int) ([]EnrollmentDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.EnrollmentList(a.appContext(), studentID, courseID)
}

func (a *App) EnrollmentCreate(studentID, courseID int, billingMode string, chargeMaterials bool, discountPct, subscriptionDiscountPct float64, note string) (*EnrollmentDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.EnrollmentCreate(a.appContext(), studentID, courseID, billingMode, chargeMaterials, discountPct, subscriptionDiscountPct, note)
}

func (a *App) EnrollmentUpdate(enrollmentID int, billingMode string, chargeMaterials bool, discountPct, subscriptionDiscountPct float64, note string) (*EnrollmentDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.EnrollmentUpdate(a.appContext(), enrollmentID, billingMode, chargeMaterials, discountPct, subscriptionDiscountPct, note)
}

func (a *App) CourseMonthSubscriptionList(year, month int, courseID *int) ([]CourseMonthSubscriptionDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.CourseMonthSubscriptionList(a.appContext(), year, month, courseID)
}

func (a *App) CourseMonthSubscriptionUpsert(courseID, year, month int, lessonsHeld float64) (*CourseMonthSubscriptionDTO, error) {
	svc, err := a.crudService()
	if err != nil {
		return nil, err
	}
	return svc.CourseMonthSubscriptionUpsert(a.appContext(), courseID, year, month, lessonsHeld)
}
