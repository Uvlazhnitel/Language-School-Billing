package main

import (
	"context"
	"strings"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/embed"

	"langschool/ent"
	"langschool/ent/attendancemonth"
	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"langschool/ent/invoice"
	"langschool/ent/payment"
	"langschool/ent/student"
	"langschool/ent/teacher"
	appconst "langschool/internal/app"
	"langschool/internal/infra"
	"langschool/internal/money"
)

func newCRUDTestApp(t *testing.T, name string) (*App, *ent.Client) {
	t.Helper()

	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:"+name+"?mode=memory&_fk=1")

	app := &App{
		ctx: ctx,
		db:  &infra.DB{Ent: client},
	}
	return app, client
}

func TestTeacherCreateAndList(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudteacher")
	defer client.Close()

	created, err := app.TeacherCreate("Anna Petrova")
	if err != nil {
		t.Fatalf("TeacherCreate: %v", err)
	}

	dup, err := app.TeacherCreate("anna petrova")
	if err != nil {
		t.Fatalf("TeacherCreate duplicate: %v", err)
	}
	if dup.ID != created.ID {
		t.Fatalf("duplicate teacher id = %d, want %d", dup.ID, created.ID)
	}

	list, err := app.TeacherList("petro")
	if err != nil {
		t.Fatalf("TeacherList: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].FullName != "Anna Petrova" {
		t.Fatalf("FullName = %q, want %q", list[0].FullName, "Anna Petrova")
	}
}

func TestTeacherCreateRejectsInvalidName(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudteacher-invalid")
	defer client.Close()

	if _, err := app.TeacherCreate("Anna2 Petrova"); err == nil {
		t.Fatal("expected TeacherCreate to reject digits in name")
	} else if !strings.Contains(err.Error(), "fullName contains invalid characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseTeacherCRUDAndSearch(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudcourse")
	defer client.Close()

	anna, err := app.TeacherCreate("Anna Petrova")
	if err != nil {
		t.Fatalf("TeacherCreate Anna: %v", err)
	}
	elina, err := app.TeacherCreate("Elina Ozola")
	if err != nil {
		t.Fatalf("TeacherCreate Elina: %v", err)
	}

	created, err := app.CourseCreate("Conversation Club", &anna.ID, "group", 25, 90)
	if err != nil {
		t.Fatalf("CourseCreate: %v", err)
	}

	if created.TeacherID == nil || *created.TeacherID != anna.ID {
		t.Fatalf("created.TeacherID = %v, want %d", created.TeacherID, anna.ID)
	}
	if created.TeacherName != "Anna Petrova" {
		t.Fatalf("created.TeacherName = %q, want %q", created.TeacherName, "Anna Petrova")
	}

	got, err := app.CourseGet(created.ID)
	if err != nil {
		t.Fatalf("CourseGet: %v", err)
	}
	if got.TeacherID == nil || *got.TeacherID != anna.ID {
		t.Fatalf("got.TeacherID = %v, want %d", got.TeacherID, anna.ID)
	}

	updated, err := app.CourseUpdate(created.ID, "Conversation Club", &elina.ID, "group", 25, 90)
	if err != nil {
		t.Fatalf("CourseUpdate: %v", err)
	}
	if updated.TeacherID == nil || *updated.TeacherID != elina.ID {
		t.Fatalf("updated.TeacherID = %v, want %d", updated.TeacherID, elina.ID)
	}
	if updated.TeacherName != "Elina Ozola" {
		t.Fatalf("updated.TeacherName = %q, want %q", updated.TeacherName, "Elina Ozola")
	}

	found, err := app.CourseList("ozola")
	if err != nil {
		t.Fatalf("CourseList search by teacher: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("len(found) = %d, want 1", len(found))
	}
	if found[0].ID != created.ID {
		t.Fatalf("found[0].ID = %d, want %d", found[0].ID, created.ID)
	}
}

func TestEnrollmentListIncludesTeacherName(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudenrollment")
	defer client.Close()

	st, err := app.StudentCreate("Mila Test", "010101-12345", "", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}

	tch, err := app.TeacherCreate("Janis Kalnins")
	if err != nil {
		t.Fatalf("TeacherCreate: %v", err)
	}

	crs, err := app.CourseCreate("Grammar Lab", &tch.ID, "group", 20, 80)
	if err != nil {
		t.Fatalf("CourseCreate: %v", err)
	}

	created, err := app.EnrollmentCreate(st.ID, crs.ID, "per_lesson", true, 5, 0, "evening group")
	if err != nil {
		t.Fatalf("EnrollmentCreate: %v", err)
	}
	if !created.ChargeMaterials {
		t.Fatal("created.ChargeMaterials = false, want true")
	}

	list, err := app.EnrollmentList(&st.ID, nil)
	if err != nil {
		t.Fatalf("EnrollmentList: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].TeacherID == nil || *list[0].TeacherID != tch.ID {
		t.Fatalf("TeacherID = %v, want %d", list[0].TeacherID, tch.ID)
	}
	if list[0].TeacherName != "Janis Kalnins" {
		t.Fatalf("TeacherName = %q, want %q", list[0].TeacherName, "Janis Kalnins")
	}
	if list[0].CourseName != "Grammar Lab" {
		t.Fatalf("CourseName = %q, want %q", list[0].CourseName, "Grammar Lab")
	}
	if !list[0].ChargeMaterials {
		t.Fatal("ChargeMaterials = false, want true")
	}

	updated, err := app.EnrollmentUpdate(created.ID, "per_lesson", false, 10, 0, "online")
	if err != nil {
		t.Fatalf("EnrollmentUpdate: %v", err)
	}
	if updated.ChargeMaterials {
		t.Fatal("updated.ChargeMaterials = true, want false")
	}
}

func TestMigrateLegacyCourseTeachers(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:legacyteachers?mode=memory&_fk=1")
	defer client.Close()

	if _, err := client.Course.Create().
		SetName("Legacy Course").
		SetTeacherName("Ieva Liepa").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(15)).
		SetSubscriptionPriceCents(money.EurosToCents(60)).
		Save(ctx); err != nil {
		t.Fatalf("create legacy course: %v", err)
	}

	if err := migrateLegacyCourseTeachers(ctx, client); err != nil {
		t.Fatalf("migrateLegacyCourseTeachers: %v", err)
	}

	teachers, err := client.Teacher.Query().All(ctx)
	if err != nil {
		t.Fatalf("Teacher.Query: %v", err)
	}
	if len(teachers) != 1 {
		t.Fatalf("len(teachers) = %d, want 1", len(teachers))
	}
	if teachers[0].FullName != "Ieva Liepa" {
		t.Fatalf("teacher.FullName = %q, want %q", teachers[0].FullName, "Ieva Liepa")
	}

	crs, err := client.Course.Query().
		Where(course.NameEQ("Legacy Course")).
		Only(ctx)
	if err != nil {
		t.Fatalf("Course.Query: %v", err)
	}
	if crs.TeacherID == nil || *crs.TeacherID != teachers[0].ID {
		t.Fatalf("course.TeacherID = %v, want %d", crs.TeacherID, teachers[0].ID)
	}

	// idempotency
	if err := migrateLegacyCourseTeachers(ctx, client); err != nil {
		t.Fatalf("migrateLegacyCourseTeachers second run: %v", err)
	}

	count, err := client.Teacher.Query().Where(teacher.FullNameEQ("Ieva Liepa")).Count(ctx)
	if err != nil {
		t.Fatalf("Teacher.Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("teacher count = %d, want 1", count)
	}
}

func TestSettingsLocalePersists(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudsettingslocale")
	defer client.Close()

	if _, err := client.Settings.Create().
		SetSingletonID(appconst.SettingsSingletonID).
		SetOrgName("LangSchool").
		SetAddress("Test street 1").
		SetInvoicePrefix("LS").
		SetNextSeq(1).
		SetInvoiceDayOfMonth(1).
		SetCurrency("EUR").
		SetLocale("en-US").
		Save(app.ctx); err != nil {
		t.Fatalf("Settings.Create: %v", err)
	}

	if err := app.SettingsSetLocale("ru-RU"); err != nil {
		t.Fatalf("SettingsSetLocale: %v", err)
	}

	got, err := app.SettingsGetLocale()
	if err != nil {
		t.Fatalf("SettingsGetLocale: %v", err)
	}
	if got != "ru-RU" {
		t.Fatalf("locale = %q, want %q", got, "ru-RU")
	}
}

func TestStudentCreateAndUpdateIsMinor(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentminor")
	defer client.Close()

	created, err := app.StudentCreate("Nika Test", "020202-23456", "", "", "", true, "Anna Test", "mother")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}
	if !created.IsMinor {
		t.Fatalf("created.IsMinor = false, want true")
	}
	if created.PayerName != "Anna Test" {
		t.Fatalf("created.PayerName = %q, want %q", created.PayerName, "Anna Test")
	}
	if created.PersonalCode != "020202-23456" {
		t.Fatalf("created.PersonalCode = %q, want %q", created.PersonalCode, "020202-23456")
	}
	if created.CreatedAt == "" {
		t.Fatal("created.CreatedAt = empty, want RFC3339 timestamp")
	}

	updated, err := app.StudentUpdate(created.ID, "Nika Test", "030303-34567", "+371 22123", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentUpdate: %v", err)
	}
	if updated.IsMinor {
		t.Fatalf("updated.IsMinor = true, want false")
	}
	if updated.PayerName != "" {
		t.Fatalf("updated.PayerName = %q, want empty", updated.PayerName)
	}
	if updated.PersonalCode != "030303-34567" {
		t.Fatalf("updated.PersonalCode = %q, want %q", updated.PersonalCode, "030303-34567")
	}
	if updated.CreatedAt == "" {
		t.Fatal("updated.CreatedAt = empty, want original timestamp")
	}
}

func TestStudentCreateMinorRequiresPayerFields(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentvalidation")
	defer client.Close()

	if _, err := app.StudentCreate("Minor Missing Payer", "", "", "", "", true, "", ""); err == nil {
		t.Fatalf("expected StudentCreate to fail when minor payer fields are missing")
	}

	if _, err := app.StudentCreate("Minor Missing Role", "", "", "", "", true, "Anna Parent", ""); err == nil {
		t.Fatalf("expected StudentCreate to fail when minor payerRole is missing")
	}

	if _, err := app.StudentCreate("Minor Bad Role", "", "", "", "", true, "Anna Parent", "uncle"); err == nil {
		t.Fatalf("expected StudentCreate to fail for invalid payerRole")
	}
}

func TestStudentValidationRejectsInvalidPeopleAndContactFields(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudent-invalid-fields")
	defer client.Close()

	if _, err := app.StudentCreate("Žanna Petrova", "131008-22451", "+371 22137936", "valid@example.com", "", false, "", ""); err != nil {
		t.Fatalf("expected valid multilingual student create, got %v", err)
	}

	tests := []struct {
		name         string
		fullName     string
		personalCode string
		phone        string
		email        string
		isMinor      bool
		payerName    string
		payerRole    string
		wantErr      string
	}{
		{
			name:         "student name digits",
			fullName:     "Ivan2 Test",
			personalCode: "",
			wantErr:      "fullName contains invalid characters",
		},
		{
			name:         "invalid personal code",
			fullName:     "Ivan Test",
			personalCode: "abc",
			wantErr:      "personalCode must be in the format 123456-12345",
		},
		{
			name:         "invalid phone",
			fullName:     "Ivan Test",
			personalCode: "",
			phone:        "abc123",
			wantErr:      "phone contains invalid characters",
		},
		{
			name:         "invalid email",
			fullName:     "Ivan Test",
			personalCode: "",
			email:        "bad-email",
			wantErr:      "email is invalid",
		},
		{
			name:         "invalid payer name",
			fullName:     "Ivan Test",
			personalCode: "",
			isMinor:      true,
			payerName:    "Mama2",
			payerRole:    "mother",
			wantErr:      "payerName contains invalid characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := app.StudentCreate(tc.fullName, tc.personalCode, tc.phone, tc.email, "", tc.isMinor, tc.payerName, tc.payerRole)
			if err == nil {
				t.Fatalf("expected validation error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestStudentCreateRejectsDuplicatePersonalCodeForInactiveStudent(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentduplicate-create")
	defer client.Close()

	first, err := app.StudentCreate("Mila Test", "010101-12345", "", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate first: %v", err)
	}
	if err := app.StudentSetActive(first.ID, false); err != nil {
		t.Fatalf("StudentSetActive: %v", err)
	}

	if _, err := app.StudentCreate("Mila Clone", "010101-12345", "", "", "", false, "", ""); err == nil {
		t.Fatal("expected StudentCreate to reject duplicate personalCode")
	} else if !strings.Contains(err.Error(), "personal code already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStudentDuplicateCheck(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentduplicate-check")
	defer client.Close()

	exact, err := app.StudentCreate("Anna Student", "020202-23456", "+371 22111", "anna@example.com", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate exact: %v", err)
	}
	possibleByPhone, err := app.StudentCreate("Berta Student", "", "+371 22222", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate phone match: %v", err)
	}
	possibleByEmail, err := app.StudentCreate("Berta Student", "", "", "berta@example.com", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate email match: %v", err)
	}
	if err := app.StudentSetActive(possibleByEmail.ID, false); err != nil {
		t.Fatalf("StudentSetActive possibleByEmail: %v", err)
	}
	if _, err := app.StudentCreate("Other Name", "", "+371 22222", "berta@example.com", "", false, "", ""); err != nil {
		t.Fatalf("StudentCreate other name: %v", err)
	}

	svc := app.ensureBackendService()
	result, err := svc.StudentDuplicateCheck(app.ctx, "Anna Student", "020202-23456", "", "")
	if err != nil {
		t.Fatalf("StudentDuplicateCheck exact: %v", err)
	}
	if result.ExactMatch == nil || result.ExactMatch.ID != exact.ID {
		t.Fatalf("exactMatch = %+v, want student %d", result.ExactMatch, exact.ID)
	}
	if result.ExactMatch != nil && result.ExactMatch.CreatedAt == "" {
		t.Fatal("expected duplicate-check exactMatch to include createdAt")
	}
	if len(result.PossibleMatches) != 0 {
		t.Fatalf("possibleMatches len = %d, want 0", len(result.PossibleMatches))
	}

	result, err = svc.StudentDuplicateCheck(app.ctx, "Berta Student", "", "+371 22222", "")
	if err != nil {
		t.Fatalf("StudentDuplicateCheck phone: %v", err)
	}
	if result.ExactMatch != nil {
		t.Fatalf("exactMatch = %+v, want nil", result.ExactMatch)
	}
	if len(result.PossibleMatches) != 1 || result.PossibleMatches[0].ID != possibleByPhone.ID {
		t.Fatalf("possibleMatches = %+v, want [%d]", result.PossibleMatches, possibleByPhone.ID)
	}

	result, err = svc.StudentDuplicateCheck(app.ctx, "Berta Student", "", "", "berta@example.com")
	if err != nil {
		t.Fatalf("StudentDuplicateCheck email: %v", err)
	}
	if len(result.PossibleMatches) != 1 || result.PossibleMatches[0].ID != possibleByEmail.ID {
		t.Fatalf("possibleMatches = %+v, want [%d]", result.PossibleMatches, possibleByEmail.ID)
	}
	if result.PossibleMatches[0].IsActive {
		t.Fatal("expected inactive student to be included in possible matches")
	}

	result, err = svc.StudentDuplicateCheck(app.ctx, "Different Name", "", "+371 22222", "berta@example.com")
	if err != nil {
		t.Fatalf("StudentDuplicateCheck different name: %v", err)
	}
	if len(result.PossibleMatches) != 0 {
		t.Fatalf("possibleMatches len = %d, want 0", len(result.PossibleMatches))
	}

	result, err = svc.StudentDuplicateCheck(app.ctx, "Berta Student", "", "", "")
	if err != nil {
		t.Fatalf("StudentDuplicateCheck empty contacts: %v", err)
	}
	if len(result.PossibleMatches) != 0 || result.ExactMatch != nil {
		t.Fatalf("unexpected duplicate result for empty contacts: %+v", result)
	}
}

func TestStudentDeleteAllowsDraftInvoices(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentdelete-draft")
	defer client.Close()

	ctx := app.ctx

	st, err := app.StudentCreate("Draft Student", "", "", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}

	if err := app.StudentSetActive(st.ID, false); err != nil {
		t.Fatalf("StudentSetActive: %v", err)
	}

	crs, err := client.Course.Create().
		SetName("Draft Course").
		SetType(course.TypeGroup).
		SetLessonPriceCents(money.EurosToCents(20)).
		SetSubscriptionPriceCents(money.EurosToCents(0)).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := client.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	if _, err := client.AttendanceMonth.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetYear(2026).
		SetMonth(5).
		SetHours(2).
		Save(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Create: %v", err)
	}

	iv, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(5).
		SetStatus(appconst.InvoiceStatusDraft).
		SetTotalAmountCents(money.EurosToCents(40)).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := client.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("Draft invoice line").
		SetQty(2).
		SetUnitPriceCents(money.EurosToCents(20)).
		SetAmountCents(money.EurosToCents(40)).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	if err := app.StudentDelete(st.ID); err != nil {
		t.Fatalf("StudentDelete: %v", err)
	}

	if exists, err := client.Student.Query().Where(student.IDEQ(st.ID)).Exist(ctx); err != nil {
		t.Fatalf("Student.Exists: %v", err)
	} else if exists {
		t.Fatalf("student still exists after delete")
	}

	if count, err := client.Invoice.Query().Where(invoice.StudentIDEQ(st.ID)).Count(ctx); err != nil {
		t.Fatalf("Invoice.Count: %v", err)
	} else if count != 0 {
		t.Fatalf("invoice count = %d, want 0", count)
	}

	if count, err := client.InvoiceLine.Query().Count(ctx); err != nil {
		t.Fatalf("InvoiceLine.Count: %v", err)
	} else if count != 0 {
		t.Fatalf("invoice line count = %d, want 0", count)
	}

	if count, err := client.Enrollment.Query().Where(enrollment.StudentIDEQ(st.ID)).Count(ctx); err != nil {
		t.Fatalf("Enrollment.Count: %v", err)
	} else if count != 0 {
		t.Fatalf("enrollment count = %d, want 0", count)
	}

	if count, err := client.AttendanceMonth.Query().Where(attendancemonth.StudentIDEQ(st.ID)).Count(ctx); err != nil {
		t.Fatalf("AttendanceMonth.Count: %v", err)
	} else if count != 0 {
		t.Fatalf("attendance count = %d, want 0", count)
	}
}

func TestStudentDeleteRejectsIssuedInvoices(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentdelete-issued")
	defer client.Close()

	ctx := app.ctx

	st, err := app.StudentCreate("Issued Student", "", "", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}
	if err := app.StudentSetActive(st.ID, false); err != nil {
		t.Fatalf("StudentSetActive: %v", err)
	}

	if _, err := client.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(6).
		SetStatus(appconst.InvoiceStatusIssued).
		SetNumber("LS-202606-001").
		SetTotalAmountCents(money.EurosToCents(10)).
		Save(ctx); err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	err = app.StudentDelete(st.ID)
	if err == nil {
		t.Fatalf("expected StudentDelete to fail for issued invoice")
	}
	if !strings.Contains(err.Error(), "issued, paid, or canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStudentDeleteRejectsPayments(t *testing.T) {
	app, client := newCRUDTestApp(t, "crudstudentdelete-payment")
	defer client.Close()

	ctx := app.ctx

	st, err := app.StudentCreate("Paid Student", "", "", "", "", false, "", "")
	if err != nil {
		t.Fatalf("StudentCreate: %v", err)
	}
	if err := app.StudentSetActive(st.ID, false); err != nil {
		t.Fatalf("StudentSetActive: %v", err)
	}

	if _, err := client.Payment.Create().
		SetStudentID(st.ID).
		SetAmountCents(money.EurosToCents(15)).
		SetMethod(payment.MethodCash).
		SetPaidAt(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)).
		SetNote("").
		Save(ctx); err != nil {
		t.Fatalf("Payment.Create: %v", err)
	}

	err = app.StudentDelete(st.ID)
	if err == nil {
		t.Fatalf("expected StudentDelete to fail for payment history")
	}
	if !strings.Contains(err.Error(), "has payments") {
		t.Fatalf("unexpected error: %v", err)
	}
}
