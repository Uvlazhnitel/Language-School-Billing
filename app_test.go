package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"

	"langschool/ent/course"
	"langschool/ent/enrollment"
	"langschool/ent/enttest"
	"langschool/ent/settings"
	sharedapp "langschool/internal/app"
	invsvc "langschool/internal/app/invoice"
	paysvc "langschool/internal/app/payment"
	"langschool/internal/infra"
	"langschool/internal/paths"
)

func TestDefaultSettingsCreateWithoutAutoIssue(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:appsettings?mode=memory&_fk=1")
	defer client.Close()

	st, err := client.Settings.Create().
		SetSingletonID(sharedapp.SettingsSingletonID).
		SetOrgName("").
		SetAddress("").
		SetInvoicePrefix("LS").
		SetNextSeq(1).
		SetInvoiceDayOfMonth(1).
		SetCurrency("EUR").
		SetLocale("lv-LV").
		Save(ctx)
	if err != nil {
		t.Fatalf("create settings without auto_issue: %v", err)
	}

	got, err := client.Settings.Query().
		Where(settings.SingletonIDEQ(sharedapp.SettingsSingletonID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query settings: %v", err)
	}

	if got.ID != st.ID {
		t.Fatalf("got settings id %d, want %d", got.ID, st.ID)
	}
}

func TestResolveAppBaseDirMigratesLegacyDirectory(t *testing.T) {
	home := t.TempDir()
	legacyBase := filepath.Join(home, legacyAppDirName)
	legacyData := filepath.Join(legacyBase, "Data")

	if err := os.MkdirAll(legacyData, 0o755); err != nil {
		t.Fatalf("MkdirAll legacy data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyData, "app.sqlite"), []byte("db"), 0o644); err != nil {
		t.Fatalf("WriteFile legacy db: %v", err)
	}

	got := resolveAppBaseDir(home)
	want := filepath.Join(home, appDirName)
	if got != want {
		t.Fatalf("resolveAppBaseDir() = %q, want %q", got, want)
	}

	if _, err := os.Stat(filepath.Join(want, "Data", "app.sqlite")); err != nil {
		t.Fatalf("expected migrated database under new path: %v", err)
	}
	if _, err := os.Stat(legacyBase); !os.IsNotExist(err) {
		t.Fatalf("expected legacy directory to be moved away, err=%v", err)
	}
}

func TestInvoiceIssueDoesNotGeneratePDFUntilRequested(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	dirs, err := paths.Ensure(base)
	if err != nil {
		t.Fatalf("paths.Ensure: %v", err)
	}

	db, err := infra.Open(ctx, filepath.Join(dirs.Data, "app.sqlite"))
	if err != nil {
		t.Fatalf("infra.Open: %v", err)
	}
	defer db.Ent.Close()

	if _, err := db.Ent.Settings.Create().
		SetSingletonID(sharedapp.SettingsSingletonID).
		SetOrgName("ArtLab").
		SetAddress("Latgales iela 260, Rīga, Latvija").
		SetInvoicePrefix("AL").
		SetNextSeq(1).
		SetInvoiceDayOfMonth(1).
		SetCurrency("EUR").
		SetLocale("en-IE").
		Save(ctx); err != nil {
		t.Fatalf("Settings.Create: %v", err)
	}

	st, err := db.Ent.Student.Create().
		SetFullName("Cash Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	crs, err := db.Ent.Course.Create().
		SetName("Drawing").
		SetType(course.TypeGroup).
		SetLessonPrice(25).
		SetSubscriptionPrice(80).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Course.Create: %v", err)
	}

	enr, err := db.Ent.Enrollment.Create().
		SetStudentID(st.ID).
		SetCourseID(crs.ID).
		SetBillingMode(enrollment.BillingModePerLesson).
		SetDiscountPct(0).
		SetNote("").
		Save(ctx)
	if err != nil {
		t.Fatalf("Enrollment.Create: %v", err)
	}

	iv, err := db.Ent.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(5).
		SetTotalAmount(25).
		SetStatus(sharedapp.InvoiceStatusDraft).
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	if _, err := db.Ent.InvoiceLine.Create().
		SetInvoiceID(iv.ID).
		SetEnrollmentID(enr.ID).
		SetDescription("One lesson").
		SetQty(1).
		SetUnitPrice(25).
		SetAmount(25).
		Save(ctx); err != nil {
		t.Fatalf("InvoiceLine.Create: %v", err)
	}

	app := &App{
		ctx:  ctx,
		dirs: dirs,
		db:   db,
		inv:  invsvc.New(db.Ent),
		pay:  paysvc.New(db.Ent),
	}

	res, err := app.InvoiceIssue(iv.ID)
	if err != nil {
		t.Fatalf("InvoiceIssue: %v", err)
	}
	if res.Number != "AL-202605-001" {
		t.Fatalf("IssueResult.Number = %q, want %q", res.Number, "AL-202605-001")
	}

	got, err := db.Ent.Invoice.Get(ctx, iv.ID)
	if err != nil {
		t.Fatalf("Invoice.Get: %v", err)
	}
	if got.Status != sharedapp.InvoiceStatusIssued {
		t.Fatalf("Status = %q, want %q", got.Status, sharedapp.InvoiceStatusIssued)
	}
	if got.Number == nil || *got.Number != res.Number {
		t.Fatalf("Number = %v, want %q", got.Number, res.Number)
	}

	namedPath := invsvc.PDFPathByNumberAndName(dirs.Invoices, 2026, 5, res.Number, st.FullName)
	legacyPath := invsvc.PDFPathByNumber(dirs.Invoices, 2026, 5, res.Number)
	if _, err := os.Stat(namedPath); !os.IsNotExist(err) {
		t.Fatalf("named PDF should not exist after Issue: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy PDF should not exist after Issue: %v", err)
	}
	hasPDF, err := app.InvoiceHasPDF(iv.ID)
	if err != nil {
		t.Fatalf("InvoiceHasPDF before generation: %v", err)
	}
	if hasPDF {
		t.Fatalf("InvoiceHasPDF before generation = true, want false")
	}

	pdfPath, err := app.InvoiceEnsurePDF(iv.ID)
	if err != nil {
		t.Fatalf("InvoiceEnsurePDF: %v", err)
	}
	if pdfPath != namedPath {
		t.Fatalf("InvoiceEnsurePDF path = %q, want %q", pdfPath, namedPath)
	}
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("generated PDF missing: %v", err)
	}
	hasPDF, err = app.InvoiceHasPDF(iv.ID)
	if err != nil {
		t.Fatalf("InvoiceHasPDF after generation: %v", err)
	}
	if !hasPDF {
		t.Fatalf("InvoiceHasPDF after generation = false, want true")
	}
}

func TestInvoiceHasPDFRecognizesLegacyPath(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	dirs, err := paths.Ensure(base)
	if err != nil {
		t.Fatalf("paths.Ensure: %v", err)
	}

	db, err := infra.Open(ctx, filepath.Join(dirs.Data, "app.sqlite"))
	if err != nil {
		t.Fatalf("infra.Open: %v", err)
	}
	defer db.Ent.Close()

	if _, err := db.Ent.Settings.Create().
		SetSingletonID(sharedapp.SettingsSingletonID).
		SetOrgName("ArtLab").
		SetAddress("Latgales iela 260, Rīga, Latvija").
		SetInvoicePrefix("AL").
		SetNextSeq(1).
		SetInvoiceDayOfMonth(1).
		SetCurrency("EUR").
		SetLocale("en-IE").
		Save(ctx); err != nil {
		t.Fatalf("Settings.Create: %v", err)
	}

	st, err := db.Ent.Student.Create().
		SetFullName("Legacy Student").
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("Student.Create: %v", err)
	}

	iv, err := db.Ent.Invoice.Create().
		SetStudentID(st.ID).
		SetPeriodYear(2026).
		SetPeriodMonth(6).
		SetTotalAmount(10).
		SetStatus(sharedapp.InvoiceStatusIssued).
		SetNumber("AL-202606-001").
		Save(ctx)
	if err != nil {
		t.Fatalf("Invoice.Create: %v", err)
	}

	legacyPath := invsvc.PDFPathByNumber(dirs.Invoices, 2026, 6, "AL-202606-001")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	app := &App{
		ctx:  ctx,
		dirs: dirs,
		db:   db,
		inv:  invsvc.New(db.Ent),
		pay:  paysvc.New(db.Ent),
	}

	hasPDF, err := app.InvoiceHasPDF(iv.ID)
	if err != nil {
		t.Fatalf("InvoiceHasPDF legacy path: %v", err)
	}
	if !hasPDF {
		t.Fatalf("InvoiceHasPDF legacy path = false, want true")
	}
}
