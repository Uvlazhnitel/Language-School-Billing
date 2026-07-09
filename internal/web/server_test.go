package web

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"langschool/ent/invoice"
	"langschool/ent/invoiceline"
	"langschool/ent/user"
	invsvc "langschool/internal/app/invoice"
	"langschool/internal/backend"
	"langschool/internal/email"
	appruntime "langschool/internal/runtime"
)

func TestHealthAndMeta(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	health := getJSON[map[string]bool](t, http.DefaultClient, env.Server.URL, "/healthz")
	if !health["ready"] {
		t.Fatalf("healthz ready = false, want true")
	}

	meta := getJSON[backend.Meta](t, env.Client, env.Server.URL, "/api/meta")
	if !meta.Ready {
		t.Fatalf("meta ready = false, want true")
	}
	if meta.Locale != "lv-LV" {
		t.Fatalf("meta locale = %q, want lv-LV", meta.Locale)
	}
}

func TestStudentCourseEnrollmentCRUD(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Alice Student",
		"phone":    "+371 22123",
		"email":    "alice@example.com",
	})
	if st.FullName != "Alice Student" {
		t.Fatalf("student fullName = %q", st.FullName)
	}

	st = putJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students/"+strconv.Itoa(st.ID), map[string]any{
		"version":  st.Version,
		"fullName": "Alice Updated",
		"phone":    "+371 22555",
		"email":    "alice@example.com",
	})
	if st.FullName != "Alice Updated" {
		t.Fatalf("student fullName = %q, want updated", st.FullName)
	}

	teacher := postJSON[backend.TeacherDTO](t, env.Client, env.Server.URL, "/api/teachers", map[string]any{
		"fullName": "Teacher One",
	})

	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Drawing",
		"teacherId":         teacher.ID,
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 80,
	})
	if course.TeacherName != "Teacher One" {
		t.Fatalf("course teacherName = %q", course.TeacherName)
	}

	enrollment := postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	if enrollment.StudentID != st.ID {
		t.Fatalf("enrollment studentID = %d, want %d", enrollment.StudentID, st.ID)
	}
	if !enrollment.ChargeMaterials {
		t.Fatal("enrollment chargeMaterials = false, want true")
	}
	if enrollment.LessonPriceOverride != 20 {
		t.Fatalf("enrollment lessonPriceOverride = %v, want 20", enrollment.LessonPriceOverride)
	}

	items := getJSON[[]backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments?studentId="+strconv.Itoa(st.ID))
	if len(items) != 1 {
		t.Fatalf("enrollment count = %d, want 1", len(items))
	}
	if !items[0].ChargeMaterials {
		t.Fatal("listed chargeMaterials = false, want true")
	}
	if items[0].LessonPriceOverride != 20 {
		t.Fatalf("listed lessonPriceOverride = %v, want 20", items[0].LessonPriceOverride)
	}

	updated := putJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments/"+strconv.Itoa(enrollment.ID), map[string]any{
		"version":                 enrollment.Version,
		"billingMode":             "per_lesson",
		"chargeMaterials":         false,
		"lessonPriceOverride":     12.5,
		"subscriptionLessonPrice": 0,
		"note":                    "online",
	})
	if updated.ChargeMaterials {
		t.Fatal("updated chargeMaterials = true, want false")
	}
	if updated.LessonPriceOverride != 12.5 {
		t.Fatalf("updated lessonPriceOverride = %v, want 12.5", updated.LessonPriceOverride)
	}
}

func TestStudentDuplicateProtectionAPI(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	exact := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName":     "Anna Student",
		"personalCode": "020202-23456",
		"phone":        "+371 22111",
		"email":        "anna@example.com",
	})
	possible := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Berta Student",
		"phone":    "+371 22222",
	})
	possibleInactive := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Berta Student",
		"email":    "berta@example.com",
	})
	postJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/students/"+strconv.Itoa(possibleInactive.ID)+"/active", map[string]any{
		"version": possibleInactive.Version,
		"active":  false,
	})

	check := postJSON[backend.StudentDuplicateCheckResult](t, env.Client, env.Server.URL, "/api/students/duplicate-check", map[string]any{
		"fullName":     "Anna Student",
		"personalCode": "020202-23456",
	})
	if check.ExactMatch == nil || check.ExactMatch.ID != exact.ID {
		t.Fatalf("exactMatch = %+v, want %d", check.ExactMatch, exact.ID)
	}
	if len(check.PossibleMatches) != 0 {
		t.Fatalf("possibleMatches len = %d, want 0", len(check.PossibleMatches))
	}

	check = postJSON[backend.StudentDuplicateCheckResult](t, env.Client, env.Server.URL, "/api/students/duplicate-check", map[string]any{
		"fullName": "Berta Student",
		"phone":    "+371 22222",
	})
	if check.ExactMatch != nil {
		t.Fatalf("exactMatch = %+v, want nil", check.ExactMatch)
	}
	if len(check.PossibleMatches) != 1 || check.PossibleMatches[0].ID != possible.ID {
		t.Fatalf("possibleMatches = %+v, want [%d]", check.PossibleMatches, possible.ID)
	}

	check = postJSON[backend.StudentDuplicateCheckResult](t, env.Client, env.Server.URL, "/api/students/duplicate-check", map[string]any{
		"fullName": "Berta Student",
		"email":    "berta@example.com",
	})
	if len(check.PossibleMatches) != 1 || check.PossibleMatches[0].ID != possibleInactive.ID {
		t.Fatalf("possibleMatches = %+v, want [%d]", check.PossibleMatches, possibleInactive.ID)
	}
	if check.PossibleMatches[0].IsActive {
		t.Fatal("expected inactive student in duplicate check results")
	}

	resp, body := rawRequest(t, env.Client, http.MethodPost, env.Server.URL+"/api/students", bytes.NewReader(mustJSON(t, map[string]any{
		"fullName":     "Duplicate Student",
		"personalCode": "020202-23456",
	})))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate create status = %d body=%s, want 409", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "personal code already exists") {
		t.Fatalf("duplicate create body = %s, want personal code message", body)
	}
}

func TestInvoicePDFAndPaymentWorkflow(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Billing Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Painting",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 90,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     6,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{"year": 2026, "month": 6})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}
	invoiceID := invoices[0].ID

	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	if !issue.PDFReady {
		t.Fatal("expected PDFReady=true immediately after issue")
	}
	if issue.PDFStatus != "ready" {
		t.Fatalf("issue pdfStatus = %q, want ready", issue.PDFStatus)
	}

	status := getJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/pdf-status")
	if !status["ready"] {
		t.Fatal("expected pdf-status ready=true immediately after issue")
	}

	pdfMeta := postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/pdf", map[string]any{})
	if pdfMeta["downloadUrl"] == "" {
		t.Fatal("missing pdf downloadUrl")
	}

	status = getJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/pdf-status")
	if !status["ready"] {
		t.Fatal("expected pdf-status ready=true after ensure")
	}

	res, err := env.Client.Get(env.Server.URL + "/api/invoices/" + strconv.Itoa(invoiceID) + "/pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("pdf download status = %d, want 200", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("content type = %q, want application/pdf", got)
	}

	payment := postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", map[string]any{
		"studentId": st.ID,
		"invoiceId": invoiceID,
		"amount":    25,
		"method":    "cash",
		"paidAt":    "2026-06-02",
		"note":      "paid",
	})
	if payment.StudentID != st.ID {
		t.Fatalf("payment studentID = %d, want %d", payment.StudentID, st.ID)
	}

	summary := getJSON[backend.InvoiceSummaryDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/payment-summary")
	if summary.Paid <= 0 {
		t.Fatalf("summary paid = %f, want > 0", summary.Paid)
	}
}

func TestInvoiceIssueReturnsPendingWhenPDFFails(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	env.Runtime.Config.FontsDir = filepath.Join(t.TempDir(), "missing-fonts")
	t.Chdir(t.TempDir())

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Pending Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Painting",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 90,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     6,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{"year": 2026, "month": 6})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}

	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	if issue.PDFReady {
		t.Fatal("expected PDFReady=false when PDF generation fails")
	}
	if issue.PDFStatus != "pending" {
		t.Fatalf("issue pdfStatus = %q, want pending", issue.PDFStatus)
	}

	items := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if len(items) != 1 {
		t.Fatalf("invoice count after issue = %d, want 1", len(items))
	}
	if items[0].Status != "issued_pending_pdf" {
		t.Fatalf("status = %q, want issued_pending_pdf", items[0].Status)
	}
	if items[0].PDFReady {
		t.Fatal("expected listed invoice to have pdfReady=false")
	}
}

func TestInvoiceIssueAllReportsGeneratedAndPendingCounts(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st1 := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Ready Student"})
	st2 := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Pending Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Painting",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 90,
	})
	for _, studentID := range []int{st1.ID, st2.ID} {
		postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
			"studentId":           studentID,
			"courseId":            course.ID,
			"billingMode":         "per_lesson",
			"chargeMaterials":     true,
			"lessonPriceOverride": 0,
			"note":                "",
		})
		putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
			"studentId": studentID,
			"courseId":  course.ID,
			"year":      2026,
			"month":     6,
			"hours":     1.0,
		})
	}
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{"year": 2026, "month": 6})

	originalFontsDir := env.Runtime.Config.FontsDir
	env.Runtime.Config.FontsDir = filepath.Join(t.TempDir(), "missing-fonts")
	t.Chdir(t.TempDir())

	issueAll := postJSON[backend.IssueAllResult](t, env.Client, env.Server.URL, "/api/invoices/issue-all", map[string]any{
		"year":  2026,
		"month": 6,
	})
	if issueAll.Count != 2 {
		t.Fatalf("issue-all count = %d, want 2", issueAll.Count)
	}
	if issueAll.GeneratedCount != 0 {
		t.Fatalf("issue-all generatedCount = %d, want 0", issueAll.GeneratedCount)
	}
	if issueAll.PendingCount != 2 {
		t.Fatalf("issue-all pendingCount = %d, want 2", issueAll.PendingCount)
	}
	if len(issueAll.PdfPaths) != 0 {
		t.Fatalf("issue-all pdfPaths len = %d, want 0", len(issueAll.PdfPaths))
	}

	env.Runtime.Config.FontsDir = originalFontsDir

	items := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if len(items) != 2 {
		t.Fatalf("invoice count after issue-all = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.Status != "issued_pending_pdf" {
			t.Fatalf("status = %q, want issued_pending_pdf", item.Status)
		}
	}
}

func TestCurrentIssuedInvoiceLessonChangeInvalidatesPDFStatus(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Subscription Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Clay Club",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 90,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":               st.ID,
		"courseId":                course.ID,
		"billingMode":             "subscription",
		"chargeMaterials":         false,
		"lessonPriceOverride":     0,
		"subscriptionLessonPrice": 30,
		"note":                    "",
	})
	putJSON[backend.CourseMonthSubscriptionDTO](t, env.Client, env.Server.URL, "/api/attendance/subscription-month", map[string]any{
		"courseId":    course.ID,
		"year":        year,
		"month":       month,
		"lessonsHeld": 4,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  year,
		"month": month,
	})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(year)+"&month="+strconv.Itoa(month)+"&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}

	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	if issue.Number == "" {
		t.Fatal("missing issue number")
	}

	pdfMeta := postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/pdf", map[string]any{})
	if pdfMeta["downloadUrl"] == "" {
		t.Fatal("missing initial pdf downloadUrl")
	}

	status := getJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/pdf-status")
	if !status["ready"] {
		t.Fatal("expected initial pdf-status ready=true")
	}

	putJSON[backend.CourseMonthSubscriptionDTO](t, env.Client, env.Server.URL, "/api/attendance/subscription-month", map[string]any{
		"courseId":    course.ID,
		"year":        year,
		"month":       month,
		"lessonsHeld": 2,
	})

	status = getJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/pdf-status")
	if status["ready"] {
		t.Fatal("expected pdf-status ready=false after invoice rebuild")
	}

	pdfMeta = postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/pdf", map[string]any{})
	if pdfMeta["downloadUrl"] == "" {
		t.Fatal("missing pdf downloadUrl after regeneration")
	}
}

func TestInvoiceArchiveListAndFileAccess(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())
	olderYear := currentYear - 1
	olderMonth := 12

	student := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName":  "Archive Student",
		"isMinor":   true,
		"payerName": "Archive Parent",
		"payerRole": "mother",
	})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Archive Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 40,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":               student.ID,
		"courseId":                course.ID,
		"billingMode":             "subscription",
		"chargeMaterials":         false,
		"lessonPriceOverride":     0,
		"subscriptionLessonPrice": 30,
		"note":                    "",
	})

	type issuedInvoice struct {
		ID     int
		Number string
		Year   int
		Month  int
	}

	createIssuedInvoice := func(year, month int) issuedInvoice {
		putJSON[backend.CourseMonthSubscriptionDTO](t, env.Client, env.Server.URL, "/api/attendance/subscription-month", map[string]any{
			"courseId":    course.ID,
			"year":        year,
			"month":       month,
			"lessonsHeld": 2,
		})
		postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
			"year":  year,
			"month": month,
		})
		invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(year)+"&month="+strconv.Itoa(month)+"&status=all")
		if len(invoices) != 1 {
			t.Fatalf("invoice count for %04d-%02d = %d, want 1", year, month, len(invoices))
		}
		issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
			"version": invoices[0].Version,
		})
		return issuedInvoice{ID: invoices[0].ID, Number: issue.Number, Year: year, Month: month}
	}

	currentInvoice := createIssuedInvoice(currentYear, currentMonth)
	missingInvoice := createIssuedInvoice(olderYear, olderMonth)
	outdatedInvoice := createIssuedInvoice(olderYear, olderMonth-1)
	errorInvoice := createIssuedInvoice(olderYear, olderMonth-2)
	legacyInvoice := createIssuedInvoice(olderYear, olderMonth-3)

	missingPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, missingInvoice.Year, missingInvoice.Month, missingInvoice.Number, student.FullName)
	if err := os.Remove(missingPath); err != nil {
		t.Fatal(err)
	}

	outdatedRow, err := env.Runtime.DB.Ent.Invoice.Get(context.Background(), outdatedInvoice.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Runtime.DB.Ent.Invoice.UpdateOneID(outdatedInvoice.ID).
		SetPdfRevision(outdatedRow.Version - 1).
		Save(context.Background()); err != nil {
		t.Fatal(err)
	}

	errorPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, errorInvoice.Year, errorInvoice.Month, errorInvoice.Number, student.FullName)
	errorDir := filepath.Dir(errorPath)
	if err := os.Chmod(errorDir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(errorDir, 0o755)
	}()

	orphanDir := filepath.Join(env.Runtime.Dirs.Invoices, strconv.Itoa(currentYear), fmt.Sprintf("%02d", currentMonth))
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphanDir, "orphan.pdf"), []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	legacyCanonicalPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, legacyInvoice.Year, legacyInvoice.Month, legacyInvoice.Number, student.FullName)
	if err := os.Remove(legacyCanonicalPath); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Runtime.DB.Ent.Invoice.UpdateOneID(legacyInvoice.ID).
		ClearPdfFilename().
		ClearPdfGeneratedAt().
		ClearPdfRevision().
		Save(context.Background()); err != nil {
		t.Fatal(err)
	}

	legacyPath := invsvc.PDFPathByNumber(env.Runtime.Dirs.Invoices, legacyInvoice.Year, legacyInvoice.Month, legacyInvoice.Number)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	archive := getJSON[backend.InvoiceArchiveResult](t, env.Client, env.Server.URL, "/api/invoice-archive")
	if len(archive.Years) != 2 {
		t.Fatalf("archive years = %d, want 2", len(archive.Years))
	}
	if archive.Years[0].Year != currentYear || archive.Years[1].Year != olderYear {
		t.Fatalf("archive year order = %+v, want %d then %d", archive.Years, currentYear, olderYear)
	}
	if !archive.Years[0].ExpandedByDefault {
		t.Fatal("expected current year expanded by default")
	}
	if len(archive.Years[0].Months) != 1 || archive.Years[0].Months[0].Month != currentMonth {
		t.Fatalf("archive months[0] = %+v, want only current month %d", archive.Years[0].Months, currentMonth)
	}
	if !archive.Years[0].Months[0].ExpandedByDefault {
		t.Fatal("expected current month expanded by default")
	}
	if archive.Years[0].Count != 1 || archive.Years[1].Count != 4 {
		t.Fatalf("archive year counts = %+v, want 1 current and 4 older", archive.Years)
	}

	currentItem := archive.Years[0].Months[0].Invoices[0]
	if currentItem.InvoiceID != currentInvoice.ID {
		t.Fatalf("current archive invoice id = %d, want %d", currentItem.InvoiceID, currentInvoice.ID)
	}
	if currentItem.StudentName != "Archive Student" {
		t.Fatalf("student name = %q", currentItem.StudentName)
	}
	if currentItem.RecipientName != "Archive Parent" {
		t.Fatalf("recipient name = %q", currentItem.RecipientName)
	}
	if currentItem.PDFStatus != "ready" {
		t.Fatalf("current pdf status = %q, want ready", currentItem.PDFStatus)
	}
	if currentItem.OpenURL == "" || currentItem.DownloadURL == "" || currentItem.PDFUpdatedAt == "" {
		t.Fatalf("current item missing PDF metadata: %+v", currentItem)
	}

	olderItems := map[int]backend.InvoiceArchiveInvoiceDTO{}
	for _, monthGroup := range archive.Years[1].Months {
		for _, item := range monthGroup.Invoices {
			olderItems[item.InvoiceID] = item
		}
	}

	missingItem := olderItems[missingInvoice.ID]
	if missingItem.InvoiceID != missingInvoice.ID {
		t.Fatalf("missing archive invoice id = %d, want %d", missingItem.InvoiceID, missingInvoice.ID)
	}
	if missingItem.PDFStatus != "missing" {
		t.Fatalf("missing pdf status = %q, want missing", missingItem.PDFStatus)
	}
	if missingItem.OpenURL != "" || missingItem.DownloadURL != "" || missingItem.PDFUpdatedAt != "" {
		t.Fatalf("missing item should not expose PDF links: %+v", missingItem)
	}

	outdatedItem := olderItems[outdatedInvoice.ID]
	if outdatedItem.InvoiceID != outdatedInvoice.ID {
		t.Fatalf("outdated archive invoice id = %d, want %d", outdatedItem.InvoiceID, outdatedInvoice.ID)
	}
	if outdatedItem.PDFStatus != "outdated" {
		t.Fatalf("outdated pdf status = %q, want outdated", outdatedItem.PDFStatus)
	}
	if outdatedItem.OpenURL != "" || outdatedItem.DownloadURL != "" || outdatedItem.PDFUpdatedAt == "" {
		t.Fatalf("outdated item should expose only pdfUpdatedAt: %+v", outdatedItem)
	}

	errorItem := olderItems[errorInvoice.ID]
	if errorItem.InvoiceID != errorInvoice.ID {
		t.Fatalf("error archive invoice id = %d, want %d", errorItem.InvoiceID, errorInvoice.ID)
	}
	if errorItem.PDFStatus != "error" {
		t.Fatalf("error pdf status = %q, want error", errorItem.PDFStatus)
	}
	if errorItem.OpenURL != "" || errorItem.DownloadURL != "" || errorItem.PDFUpdatedAt != "" {
		t.Fatalf("error item should not expose PDF links: %+v", errorItem)
	}

	legacyItem := olderItems[legacyInvoice.ID]
	if legacyItem.InvoiceID != legacyInvoice.ID {
		t.Fatalf("legacy archive invoice id = %d, want %d", legacyItem.InvoiceID, legacyInvoice.ID)
	}
	if legacyItem.PDFStatus != "outdated" {
		t.Fatalf("legacy pdf status = %q, want outdated", legacyItem.PDFStatus)
	}
	if legacyItem.OpenURL != "" || legacyItem.DownloadURL != "" || legacyItem.PDFUpdatedAt != "" {
		t.Fatalf("legacy item should not expose canonical PDF metadata: %+v", legacyItem)
	}

	openResp, err := env.Client.Get(env.Server.URL + currentItem.OpenURL)
	if err != nil {
		t.Fatal(err)
	}
	defer openResp.Body.Close()
	if openResp.StatusCode != http.StatusOK {
		t.Fatalf("open status = %d, want 200", openResp.StatusCode)
	}
	if got := openResp.Header.Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("open content type = %q, want application/pdf", got)
	}
	if got := openResp.Header.Get("Content-Disposition"); !strings.HasPrefix(got, "inline;") {
		t.Fatalf("open content disposition = %q, want inline", got)
	}

	downloadResp, err := env.Client.Get(env.Server.URL + currentItem.DownloadURL)
	if err != nil {
		t.Fatal(err)
	}
	defer downloadResp.Body.Close()
	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d, want 200", downloadResp.StatusCode)
	}
	if got := downloadResp.Header.Get("Content-Disposition"); !strings.HasPrefix(got, "attachment;") {
		t.Fatalf("download content disposition = %q, want attachment", got)
	}

	resp, body := rawRequest(
		t,
		env.Client,
		http.MethodGet,
		env.Server.URL+"/api/invoice-archive/"+strconv.Itoa(currentYear)+"/"+fmt.Sprintf("%02d", currentMonth)+"/not-a-pdf.txt/open",
		nil,
	)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid file status = %d body=%s, want 400", resp.StatusCode, body)
	}

	resp, body = rawRequest(
		t,
		env.Client,
		http.MethodGet,
		env.Server.URL+"/api/invoice-archive/"+strconv.Itoa(currentYear)+"/"+fmt.Sprintf("%02d", currentMonth)+"/missing.pdf/download",
		nil,
	)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing file status = %d body=%s, want 404", resp.StatusCode, body)
	}
}

func TestEnsureAllPDFsBatchEndpoint(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	year, month := 2026, 6
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Batch PDF Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 50,
	})

	type issuedInvoice struct {
		ID int
	}

	createIssuedInvoice := func(studentName string, y, m int) issuedInvoice {
		student := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
			"fullName": studentName,
		})
		postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
			"studentId":           student.ID,
			"courseId":            course.ID,
			"billingMode":         "per_lesson",
			"chargeMaterials":     false,
			"lessonPriceOverride": 0,
			"note":                "",
		})
		putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
			"studentId": student.ID,
			"courseId":  course.ID,
			"year":      y,
			"month":     m,
			"hours":     1.0,
		})
		postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
			"year":  y,
			"month": m,
		})
		invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(y)+"&month="+strconv.Itoa(m)+"&status=all")
		var item backend.InvoiceListItem
		found := false
		for _, candidate := range invoices {
			if candidate.StudentName == studentName {
				item = candidate
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("invoice for %s not found", studentName)
		}
		postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(item.ID)+"/issue", map[string]any{
			"version": item.Version,
		})
		return issuedInvoice{ID: item.ID}
	}

	generatedInvoice := createIssuedInvoice("Generated Student", year, month)
	readyInvoice := createIssuedInvoice("Already Ready Student", year, month)
	failedInvoice := createIssuedInvoice("Failed Student", year, month)
	readyIssuedInvoice := createIssuedInvoice("Ready Issued Student", year, month)
	otherMonthInvoice := createIssuedInvoice("Other Month Student", year, month-1)

	postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(readyInvoice.ID)+"/pdf", map[string]any{})
	postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(readyIssuedInvoice.ID)+"/pdf", map[string]any{})

	if _, err := env.Runtime.DB.Ent.Invoice.UpdateOneID(generatedInvoice.ID).
		SetStatus(invoice.Status("issued_pending_pdf")).
		ClearPdfGeneratedAt().
		ClearPdfRevision().
		Save(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Runtime.DB.Ent.Invoice.UpdateOneID(readyInvoice.ID).
		SetStatus(invoice.Status("issued_pending_pdf")).
		Save(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Runtime.DB.Ent.Invoice.UpdateOneID(failedInvoice.ID).
		SetStatus(invoice.Status("issued_pending_pdf")).
		ClearNumber().
		ClearPdfFilename().
		ClearPdfGeneratedAt().
		ClearPdfRevision().
		Save(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Runtime.DB.Ent.Invoice.UpdateOneID(otherMonthInvoice.ID).
		SetStatus(invoice.Status("issued_pending_pdf")).
		ClearPdfGeneratedAt().
		ClearPdfRevision().
		Save(context.Background()); err != nil {
		t.Fatal(err)
	}

	result := postJSON[backend.EnsureAllPDFsResult](t, env.Client, env.Server.URL, "/api/invoices/ensure-pdf-all", map[string]any{
		"year":  year,
		"month": month,
	})

	if result.Year != year || result.Month != month {
		t.Fatalf("batch period = %d-%d, want %d-%d", result.Year, result.Month, year, month)
	}
	if result.Processed != 3 || result.GeneratedCount != 1 || result.AlreadyReadyCount != 1 || result.FailedCount != 1 {
		t.Fatalf("unexpected ensure-all result: %+v", result)
	}

	itemsByInvoiceID := make(map[int]backend.EnsureAllPDFsItemResult, len(result.Items))
	for _, item := range result.Items {
		itemsByInvoiceID[item.InvoiceID] = item
	}

	if itemsByInvoiceID[generatedInvoice.ID].Result != "generated" {
		t.Fatalf("generated invoice result = %q, want generated", itemsByInvoiceID[generatedInvoice.ID].Result)
	}
	if itemsByInvoiceID[generatedInvoice.ID].Status != "issued" {
		t.Fatalf("generated invoice status = %q, want issued", itemsByInvoiceID[generatedInvoice.ID].Status)
	}
	if itemsByInvoiceID[readyInvoice.ID].Result != "already_ready" {
		t.Fatalf("ready invoice result = %q, want already_ready", itemsByInvoiceID[readyInvoice.ID].Result)
	}
	if itemsByInvoiceID[readyInvoice.ID].Status != "issued" {
		t.Fatalf("ready invoice status = %q, want issued", itemsByInvoiceID[readyInvoice.ID].Status)
	}
	if itemsByInvoiceID[failedInvoice.ID].Result != "failed" {
		t.Fatalf("failed invoice result = %q, want failed", itemsByInvoiceID[failedInvoice.ID].Result)
	}
	if itemsByInvoiceID[failedInvoice.ID].Message == "" {
		t.Fatal("expected failed invoice message")
	}
	if _, ok := itemsByInvoiceID[readyIssuedInvoice.ID]; ok {
		t.Fatal("ready issued invoice should not be included in batch result")
	}
	if _, ok := itemsByInvoiceID[otherMonthInvoice.ID]; ok {
		t.Fatal("other-month invoice should not be included in batch result")
	}

	generatedRow := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(generatedInvoice.ID))
	if generatedRow.Status != "issued" || !generatedRow.PDFReady {
		t.Fatalf("generated invoice after batch = %+v", generatedRow)
	}
	readyRow := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(readyInvoice.ID))
	if readyRow.Status != "issued" || !readyRow.PDFReady {
		t.Fatalf("ready invoice after batch = %+v", readyRow)
	}
	otherMonthRow := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(otherMonthInvoice.ID))
	if otherMonthRow.Status != "issued_pending_pdf" {
		t.Fatalf("other month invoice status = %q, want issued_pending_pdf", otherMonthRow.Status)
	}

	audit := getJSON[backend.AuditLogListResult](t, env.Client, env.Server.URL, "/api/audit-logs?action=invoice.ensure_pdf_all&page=1&pageSize=10")
	if audit.Total < 1 {
		t.Fatal("expected invoice.ensure_pdf_all audit entry")
	}
	foundAudit := false
	for _, item := range audit.Items {
		if item.Action == "invoice.ensure_pdf_all" && strings.Contains(item.Summary, "generated 1") {
			foundAudit = true
			break
		}
	}
	if !foundAudit {
		t.Fatal("expected invoice.ensure_pdf_all audit summary with counts")
	}
}

func TestIssuedInvoiceCanReopenAfterEnsuringPDF(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "PDF Reopen Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Reopen Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 60,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     9,
		"hours":     2,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  2026,
		"month": 9,
	})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=9&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}

	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	if issue.Number == "" {
		t.Fatal("missing issue number")
	}

	pdfMeta := postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/pdf", map[string]any{})
	if pdfMeta["downloadUrl"] == "" {
		t.Fatal("missing pdf downloadUrl")
	}

	invoiceDetails := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID))
	resp, body := rawRequest(
		t,
		env.Client,
		http.MethodPost,
		env.Server.URL+"/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/reopen-draft",
		bytes.NewReader([]byte(fmt.Sprintf(`{"version":%d}`, invoiceDetails.Version))),
	)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reopen after pdf status = %d body=%s, want 200", resp.StatusCode, body)
	}

	reopened := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID))
	if reopened.Status != "draft" {
		t.Fatalf("invoice status = %q, want draft", reopened.Status)
	}
	if reopened.PDFReady {
		t.Fatal("expected reopened invoice to have pdfReady=false")
	}
	if reopened.Number != nil {
		t.Fatalf("expected reopened invoice number to be cleared, got %v", reopened.Number)
	}
}

func TestCurrentMonthInvoiceStaysLiveAfterFullPayment(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	year, month := now.Year(), int(now.Month())

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Live Month Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Live Month Course",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 80,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      year,
		"month":     month,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  year,
		"month": month,
	})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(year)+"&month="+strconv.Itoa(month)+"&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}

	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	if issue.Number == "" {
		t.Fatal("missing issue number")
	}

	postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", map[string]any{
		"studentId": st.ID,
		"invoiceId": invoices[0].ID,
		"amount":    30,
		"method":    "cash",
		"paidAt":    now.Format("2006-01-02"),
		"note":      "full payment before extra lesson",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      year,
		"month":     month,
		"hours":     2.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/rebuild-student-draft", map[string]any{
		"studentId": st.ID,
		"year":      year,
		"month":     month,
	})

	invoiceDetails := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID))
	if invoiceDetails.Number == nil || *invoiceDetails.Number != issue.Number {
		t.Fatalf("invoice number = %v, want %q", invoiceDetails.Number, issue.Number)
	}
	if invoiceDetails.Total != 55 {
		t.Fatalf("invoice total = %v, want 55", invoiceDetails.Total)
	}
	if invoiceDetails.Status != "issued_pending_pdf" {
		t.Fatalf("invoice status = %q, want issued_pending_pdf", invoiceDetails.Status)
	}

	summary := getJSON[backend.InvoiceSummaryDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/payment-summary")
	if summary.Paid != 30 {
		t.Fatalf("paid = %v, want 30", summary.Paid)
	}
	if summary.Remaining != 25 {
		t.Fatalf("remaining = %v, want 25", summary.Remaining)
	}
	if summary.Status != "issued_pending_pdf" {
		t.Fatalf("summary status = %q, want issued_pending_pdf", summary.Status)
	}

	pdfMeta := postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/pdf", map[string]any{})
	if pdfMeta["downloadUrl"] == "" {
		t.Fatal("missing pdf downloadUrl")
	}
}

func TestEnrollmentChargeMaterialsRebuildsCurrentMonthDraftOnly(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	currentYear, currentMonth := now.Year(), int(now.Month())
	previousYear, previousMonth := currentYear, currentMonth-1
	if previousMonth == 0 {
		previousMonth = 12
		previousYear--
	}

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Materials Toggle Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Materials Toggle Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 0,
	})
	enr := postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      previousYear,
		"month":     previousMonth,
		"hours":     1.0,
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      currentYear,
		"month":     currentMonth,
		"hours":     1.0,
	})

	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  previousYear,
		"month": previousMonth,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  currentYear,
		"month": currentMonth,
	})

	previousInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(previousYear)+"&month="+strconv.Itoa(previousMonth)+"&status=all")
	currentInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(currentYear)+"&month="+strconv.Itoa(currentMonth)+"&status=all")
	if len(previousInvoices) != 1 || len(currentInvoices) != 1 {
		t.Fatalf("invoice counts prev=%d current=%d, want 1/1", len(previousInvoices), len(currentInvoices))
	}

	beforeCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	beforePrevious := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(previousInvoices[0].ID))
	if beforeCurrent.Total != 25 || beforePrevious.Total != 25 {
		t.Fatalf("before totals current=%v previous=%v, want 25/25", beforeCurrent.Total, beforePrevious.Total)
	}

	updated := putJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments/"+strconv.Itoa(enr.ID), map[string]any{
		"version":                 enr.Version,
		"billingMode":             "per_lesson",
		"chargeMaterials":         false,
		"lessonPriceOverride":     0,
		"subscriptionLessonPrice": 0,
		"note":                    "",
	})
	if updated.ChargeMaterials {
		t.Fatal("updated chargeMaterials = true, want false")
	}

	afterCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	afterPrevious := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(previousInvoices[0].ID))
	if afterCurrent.Total != 20 {
		t.Fatalf("current invoice total = %v, want 20", afterCurrent.Total)
	}
	if afterPrevious.Total != 25 {
		t.Fatalf("previous invoice total = %v, want 25", afterPrevious.Total)
	}

	currentLines, err := env.Runtime.DB.Ent.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(currentInvoices[0].ID)).
		All(context.Background())
	if err != nil {
		t.Fatalf("InvoiceLine.Query current: %v", err)
	}
	for _, line := range currentLines {
		if line.Description == "Mācību materiāli" {
			t.Fatal("current invoice still has materials line after toggle off")
		}
	}
}

func TestEnrollmentChargeMaterialsDefersToNextMonthDraftWhenCurrentMonthIssued(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	currentYear, currentMonth := now.Year(), int(now.Month())
	nextYear, nextMonth := currentYear, currentMonth+1
	if nextMonth == 13 {
		nextMonth = 1
		nextYear++
	}

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Future Month Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Future Month Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 0,
	})
	enr := postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      currentYear,
		"month":     currentMonth,
		"hours":     1.0,
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      nextYear,
		"month":     nextMonth,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  currentYear,
		"month": currentMonth,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  nextYear,
		"month": nextMonth,
	})

	currentInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(currentYear)+"&month="+strconv.Itoa(currentMonth)+"&status=all")
	nextInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(nextYear)+"&month="+strconv.Itoa(nextMonth)+"&status=all")
	if len(currentInvoices) != 1 || len(nextInvoices) != 1 {
		t.Fatalf("invoice counts current=%d next=%d, want 1/1", len(currentInvoices), len(nextInvoices))
	}
	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID)+"/issue", map[string]any{
		"version": currentInvoices[0].Version,
	})
	if issue.Number == "" {
		t.Fatal("expected invoice number")
	}

	beforeCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	beforeNext := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(nextInvoices[0].ID))
	if beforeCurrent.Total != 25 || beforeNext.Total != 25 {
		t.Fatalf("before totals current=%v next=%v, want 25/25", beforeCurrent.Total, beforeNext.Total)
	}

	updated := putJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments/"+strconv.Itoa(enr.ID), map[string]any{
		"version":                 enr.Version,
		"billingMode":             "per_lesson",
		"chargeMaterials":         false,
		"lessonPriceOverride":     0,
		"subscriptionLessonPrice": 0,
		"note":                    "",
	})
	if updated.ChargeMaterials {
		t.Fatal("updated chargeMaterials = true, want false")
	}

	afterCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	afterNext := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(nextInvoices[0].ID))
	if afterCurrent.Total != 25 {
		t.Fatalf("current invoice total = %v, want 25", afterCurrent.Total)
	}
	if afterNext.Total != 20 {
		t.Fatalf("next invoice total = %v, want 20", afterNext.Total)
	}
	if afterCurrent.Status != beforeCurrent.Status {
		t.Fatalf("current invoice status = %q, want unchanged %q", afterCurrent.Status, beforeCurrent.Status)
	}

	nextLines, err := env.Runtime.DB.Ent.InvoiceLine.Query().
		Where(invoiceline.InvoiceIDEQ(nextInvoices[0].ID)).
		All(context.Background())
	if err != nil {
		t.Fatalf("InvoiceLine.Query next: %v", err)
	}
	for _, line := range nextLines {
		if line.Description == "Mācību materiāli" {
			t.Fatal("next invoice still has materials line after toggle off")
		}
	}
}

func TestEnrollmentChargeMaterialsAppliesOnNextGenerationWhenFutureDraftMissing(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	currentYear, currentMonth := now.Year(), int(now.Month())
	nextYear, nextMonth := currentYear, currentMonth+1
	if nextMonth == 13 {
		nextMonth = 1
		nextYear++
	}

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Generate Later Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Generate Later Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 0,
	})
	enr := postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      currentYear,
		"month":     currentMonth,
		"hours":     1.0,
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      nextYear,
		"month":     nextMonth,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  currentYear,
		"month": currentMonth,
	})

	currentInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(currentYear)+"&month="+strconv.Itoa(currentMonth)+"&status=all")
	if len(currentInvoices) != 1 {
		t.Fatalf("current invoice count = %d, want 1", len(currentInvoices))
	}
	postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID)+"/issue", map[string]any{
		"version": currentInvoices[0].Version,
	})

	updated := putJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments/"+strconv.Itoa(enr.ID), map[string]any{
		"version":                 enr.Version,
		"billingMode":             "per_lesson",
		"chargeMaterials":         false,
		"lessonPriceOverride":     0,
		"subscriptionLessonPrice": 0,
		"note":                    "",
	})
	if updated.ChargeMaterials {
		t.Fatal("updated chargeMaterials = true, want false")
	}

	nextBeforeGeneration := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(nextYear)+"&month="+strconv.Itoa(nextMonth)+"&status=all")
	if len(nextBeforeGeneration) != 1 {
		t.Fatalf("next invoice count before generation = %d, want 1", len(nextBeforeGeneration))
	}
	nextInvoiceBeforeGeneration := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(nextBeforeGeneration[0].ID))
	if nextInvoiceBeforeGeneration.Total != 20 {
		t.Fatalf("next invoice total before generation = %v, want 20", nextInvoiceBeforeGeneration.Total)
	}

	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  nextYear,
		"month": nextMonth,
	})

	nextAfterGeneration := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(nextYear)+"&month="+strconv.Itoa(nextMonth)+"&status=all")
	if len(nextAfterGeneration) != 1 {
		t.Fatalf("next invoice count after generation = %d, want 1", len(nextAfterGeneration))
	}
	nextInvoice := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(nextAfterGeneration[0].ID))
	if nextInvoice.Total != 20 {
		t.Fatalf("next invoice total = %v, want 20", nextInvoice.Total)
	}
}

func TestEnrollmentLessonPriceChangePreservesPreviousIssuedInvoiceAndRebuildsCurrentDraft(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	currentYear, currentMonth := now.Year(), int(now.Month())
	previousYear, previousMonth := currentYear, currentMonth-1
	if previousMonth == 0 {
		previousMonth = 12
		previousYear--
	}

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Historical Price Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Historical Price Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 0,
	})
	enr := postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     false,
		"lessonPriceOverride": 12.5,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      previousYear,
		"month":     previousMonth,
		"hours":     1.0,
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      currentYear,
		"month":     currentMonth,
		"hours":     1.0,
	})

	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  previousYear,
		"month": previousMonth,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  currentYear,
		"month": currentMonth,
	})

	previousInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(previousYear)+"&month="+strconv.Itoa(previousMonth)+"&status=all")
	currentInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(currentYear)+"&month="+strconv.Itoa(currentMonth)+"&status=all")
	if len(previousInvoices) != 1 || len(currentInvoices) != 1 {
		t.Fatalf("invoice counts prev=%d current=%d, want 1/1", len(previousInvoices), len(currentInvoices))
	}

	postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(previousInvoices[0].ID)+"/issue", map[string]any{
		"version": previousInvoices[0].Version,
	})

	beforePrevious := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(previousInvoices[0].ID))
	beforeCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	if len(beforePrevious.Lines) != 1 || len(beforeCurrent.Lines) != 1 {
		t.Fatalf("expected one invoice line in each invoice, got prev=%d current=%d", len(beforePrevious.Lines), len(beforeCurrent.Lines))
	}
	if beforePrevious.Lines[0].UnitPrice != 12.5 || beforeCurrent.Lines[0].UnitPrice != 12.5 {
		t.Fatalf("before line prices prev=%v current=%v, want 12.5/12.5", beforePrevious.Lines[0].UnitPrice, beforeCurrent.Lines[0].UnitPrice)
	}

	updated := putJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments/"+strconv.Itoa(enr.ID), map[string]any{
		"version":                 enr.Version,
		"billingMode":             "per_lesson",
		"chargeMaterials":         false,
		"lessonPriceOverride":     16,
		"subscriptionLessonPrice": 0,
		"note":                    "",
	})
	if updated.LessonPriceOverride != 16 {
		t.Fatalf("updated lessonPriceOverride = %v, want 16", updated.LessonPriceOverride)
	}

	afterPrevious := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(previousInvoices[0].ID))
	afterCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	if afterPrevious.Total != 12.5 {
		t.Fatalf("previous invoice total = %v, want 12.5", afterPrevious.Total)
	}
	if afterCurrent.Total != 16 {
		t.Fatalf("current invoice total = %v, want 16", afterCurrent.Total)
	}
	if afterPrevious.Lines[0].UnitPrice != 12.5 {
		t.Fatalf("previous invoice unitPrice = %v, want 12.5", afterPrevious.Lines[0].UnitPrice)
	}
	if afterCurrent.Lines[0].UnitPrice != 16 {
		t.Fatalf("current invoice unitPrice = %v, want 16", afterCurrent.Lines[0].UnitPrice)
	}
	if afterPrevious.Status != beforePrevious.Status {
		t.Fatalf("previous invoice status = %q, want unchanged %q", afterPrevious.Status, beforePrevious.Status)
	}
	if afterPrevious.Number == nil || beforePrevious.Number == nil || *afterPrevious.Number != *beforePrevious.Number {
		t.Fatalf("previous invoice number changed from %v to %v", beforePrevious.Number, afterPrevious.Number)
	}
}

func TestEnrollmentLessonPriceChangeDefersToNextMonthWhenCurrentMonthIssued(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	now := time.Now()
	currentYear, currentMonth := now.Year(), int(now.Month())
	nextYear, nextMonth := currentYear, currentMonth+1
	if nextMonth == 13 {
		nextMonth = 1
		nextYear++
	}

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Future Price Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Future Price Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 0,
	})
	enr := postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     false,
		"lessonPriceOverride": 12.5,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      currentYear,
		"month":     currentMonth,
		"hours":     1.0,
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      nextYear,
		"month":     nextMonth,
		"hours":     1.0,
	})

	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  currentYear,
		"month": currentMonth,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  nextYear,
		"month": nextMonth,
	})

	currentInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(currentYear)+"&month="+strconv.Itoa(currentMonth)+"&status=all")
	nextInvoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year="+strconv.Itoa(nextYear)+"&month="+strconv.Itoa(nextMonth)+"&status=all")
	if len(currentInvoices) != 1 || len(nextInvoices) != 1 {
		t.Fatalf("invoice counts current=%d next=%d, want 1/1", len(currentInvoices), len(nextInvoices))
	}

	postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID)+"/issue", map[string]any{
		"version": currentInvoices[0].Version,
	})

	beforeCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	beforeNext := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(nextInvoices[0].ID))
	if beforeCurrent.Lines[0].UnitPrice != 12.5 || beforeNext.Lines[0].UnitPrice != 12.5 {
		t.Fatalf("before line prices current=%v next=%v, want 12.5/12.5", beforeCurrent.Lines[0].UnitPrice, beforeNext.Lines[0].UnitPrice)
	}

	putJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments/"+strconv.Itoa(enr.ID), map[string]any{
		"version":                 enr.Version,
		"billingMode":             "per_lesson",
		"chargeMaterials":         false,
		"lessonPriceOverride":     16,
		"subscriptionLessonPrice": 0,
		"note":                    "",
	})

	afterCurrent := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(currentInvoices[0].ID))
	afterNext := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(nextInvoices[0].ID))
	if afterCurrent.Lines[0].UnitPrice != 12.5 {
		t.Fatalf("current invoice unitPrice = %v, want 12.5", afterCurrent.Lines[0].UnitPrice)
	}
	if afterNext.Lines[0].UnitPrice != 16 {
		t.Fatalf("next invoice unitPrice = %v, want 16", afterNext.Lines[0].UnitPrice)
	}
}

func TestAuditLogCapturesFinanceActions(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Audit Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Audit Course",
		"type":              "group",
		"lessonPrice":       30,
		"subscriptionPrice": 90,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     7,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{"year": 2026, "month": 7})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=7&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}
	invoiceID := invoices[0].ID

	postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	payment := postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", map[string]any{
		"studentId": st.ID,
		"invoiceId": invoiceID,
		"amount":    30,
		"method":    "cash",
		"paidAt":    "2026-07-02",
		"note":      "audit payment",
	})

	resp := getJSON[backend.AuditLogListResult](t, env.Client, env.Server.URL, "/api/audit-logs?page=1&pageSize=20")
	if resp.Total < 3 {
		t.Fatalf("audit total = %d, want at least 3", resp.Total)
	}

	var seenGenerate, seenIssue, seenPayment bool
	for _, item := range resp.Items {
		switch item.Action {
		case "invoice.generate_drafts":
			seenGenerate = item.ActorLabel == env.AdminUsername && item.AfterJSON != ""
		case "invoice.issue":
			seenIssue = item.InvoiceID != nil && *item.InvoiceID == invoiceID && item.StudentID != nil && *item.StudentID == st.ID
		case "payment.create":
			seenPayment = item.EntityID != nil && *item.EntityID == payment.ID && strings.Contains(item.AfterJSON, "audit payment")
		}
	}
	if !seenGenerate {
		t.Fatal("expected invoice.generate_drafts audit entry")
	}
	if !seenIssue {
		t.Fatal("expected invoice.issue audit entry")
	}
	if !seenPayment {
		t.Fatal("expected payment.create audit entry")
	}

	deleteResp, _ := rawRequest(t, env.Client, http.MethodDelete, env.Server.URL+"/api/payments/"+strconv.Itoa(payment.ID), nil)
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete payment status = %d, want 204", deleteResp.StatusCode)
	}

	filtered := getJSON[backend.AuditLogListResult](t, env.Client, env.Server.URL, "/api/audit-logs?action=payment.delete&page=1&pageSize=20")
	if filtered.Total < 1 {
		t.Fatal("expected payment.delete audit entry")
	}
	foundDelete := false
	for _, item := range filtered.Items {
		if item.Action == "payment.delete" && item.EntityID != nil && *item.EntityID == payment.ID {
			foundDelete = strings.Contains(item.BeforeJSON, "\"deletedPayment\"")
			break
		}
	}
	if !foundDelete {
		t.Fatal("expected payment.delete entry with deletedPayment snapshot")
	}
}

func TestInvoiceEmailPreviewAndSend(t *testing.T) {
	sender := &stubEmailSender{}
	env := newTestServerWithEmailSender(t, sender)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Email Student",
		"email":    "alice@example.com",
	})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Email Course",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 80,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     6,
		"hours":     1.5,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  2026,
		"month": 6,
	})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}

	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	pdfPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, 2026, 6, issue.Number, st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nemail test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	preview := postJSON[backend.InvoiceEmailPreviewResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/email-preview", map[string]any{})
	if preview.To != "alice@example.com" {
		t.Fatalf("preview to = %q, want alice@example.com", preview.To)
	}
	if preview.AttachmentFilename == "" {
		t.Fatal("preview attachment filename is empty")
	}
	if !strings.Contains(preview.Subject, issue.Number) {
		t.Fatalf("preview subject = %q, want issue number %q", preview.Subject, issue.Number)
	}

	savedSettings := postJSON[backend.InvoiceEmailSettingsDTO](t, env.Client, env.Server.URL, "/api/settings/invoice-email", map[string]any{
		"subjectTemplate": "Custom {invoice_number} {month_name}",
		"bodyTemplate":    "Sveiki, {recipient_name}! {amount} EUR / {foo}",
		"replyTo":         "reply@example.com",
	})
	if savedSettings.ReplyTo != "reply@example.com" {
		t.Fatalf("replyTo = %q, want reply@example.com", savedSettings.ReplyTo)
	}

	preview = postJSON[backend.InvoiceEmailPreviewResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/email-preview", map[string]any{})
	if preview.Subject != fmt.Sprintf("Custom %s jūniju", issue.Number) {
		t.Fatalf("custom preview subject = %q", preview.Subject)
	}
	if !strings.Contains(preview.Body, "Sveiki, Email Student!") {
		t.Fatalf("custom preview body = %q, want rendered recipient", preview.Body)
	}
	if !strings.Contains(preview.Body, "{foo}") {
		t.Fatalf("custom preview body = %q, want unknown placeholder preserved", preview.Body)
	}

	sent := postJSON[backend.InvoiceSendEmailResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/send-email", map[string]any{
		"to":      preview.To,
		"subject": preview.Subject,
		"body":    preview.Body,
	})
	if sent.To != preview.To {
		t.Fatalf("sent to = %q, want %q", sent.To, preview.To)
	}
	if sender.lastMessage.To != preview.To {
		t.Fatalf("sender to = %q, want %q", sender.lastMessage.To, preview.To)
	}
	if sender.lastMessage.AttachmentFilename != preview.AttachmentFilename {
		t.Fatalf("sender attachment = %q, want %q", sender.lastMessage.AttachmentFilename, preview.AttachmentFilename)
	}
	if len(sender.lastMessage.AttachmentData) == 0 {
		t.Fatal("expected attachment data to be sent")
	}
	if sender.lastMessage.ReplyTo != "reply@example.com" {
		t.Fatalf("sender replyTo = %q, want reply@example.com", sender.lastMessage.ReplyTo)
	}
	updatedList := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if updatedList[0].LastEmailedTo != preview.To {
		t.Fatalf("list lastEmailedTo = %q, want %q", updatedList[0].LastEmailedTo, preview.To)
	}
	if updatedList[0].LastEmailedAt == "" {
		t.Fatal("list lastEmailedAt is empty")
	}
	if updatedList[0].EmailCommunicationStatus != "sent" {
		t.Fatalf("list emailCommunicationStatus = %q, want sent", updatedList[0].EmailCommunicationStatus)
	}
	if updatedList[0].LastEmailError != "" {
		t.Fatalf("list lastEmailError = %q, want empty", updatedList[0].LastEmailError)
	}
	updatedInvoice := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID))
	if updatedInvoice.LastEmailedTo != preview.To {
		t.Fatalf("invoice lastEmailedTo = %q, want %q", updatedInvoice.LastEmailedTo, preview.To)
	}
	if updatedInvoice.LastEmailedAt == "" {
		t.Fatal("invoice lastEmailedAt is empty")
	}
	if updatedInvoice.EmailCommunicationStatus != "sent" {
		t.Fatalf("invoice emailCommunicationStatus = %q, want sent", updatedInvoice.EmailCommunicationStatus)
	}
	if updatedInvoice.LastEmailError != "" {
		t.Fatalf("invoice lastEmailError = %q, want empty", updatedInvoice.LastEmailError)
	}

	resp := getJSON[backend.AuditLogListResult](t, env.Client, env.Server.URL, "/api/audit-logs?action=invoice.send_email&page=1&pageSize=20")
	if resp.Total < 1 {
		t.Fatal("expected invoice.send_email audit entry")
	}
}

func TestInvoiceEmailStatusBecomesStaleAfterInvoiceVersionChanges(t *testing.T) {
	env := newTestServerWithEmailSender(t, &stubEmailSender{})
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Stale Email Student",
		"email":    "stale@example.com",
	})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Stale Email Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 60,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     6,
		"hours":     1,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  2026,
		"month": 6,
	})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	pdfPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, 2026, 6, issue.Number, st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nstale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	preview := postJSON[backend.InvoiceEmailPreviewResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/email-preview", map[string]any{})
	postJSON[backend.InvoiceSendEmailResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/send-email", map[string]any{
		"to":      preview.To,
		"subject": preview.Subject,
		"body":    preview.Body,
	})
	if _, err := env.Runtime.DB.Ent.Invoice.UpdateOneID(invoices[0].ID).
		SetVersion(invoices[0].Version + 2).
		Save(context.Background()); err != nil {
		t.Fatalf("Invoice.UpdateOneID: %v", err)
	}

	updatedList := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if updatedList[0].EmailCommunicationStatus != "stale" {
		t.Fatalf("list emailCommunicationStatus = %q, want stale", updatedList[0].EmailCommunicationStatus)
	}
	updatedInvoice := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID))
	if updatedInvoice.EmailCommunicationStatus != "stale" {
		t.Fatalf("invoice emailCommunicationStatus = %q, want stale", updatedInvoice.EmailCommunicationStatus)
	}
}

func TestInvoiceArchiveMonthZIPDownload(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "ZIP Student",
	})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "ZIP Course",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 80,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     6,
		"hours":     1.5,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  2026,
		"month": 6,
	})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}
	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	if !issue.PDFReady {
		t.Fatal("expected issued invoice pdfReady=true")
	}

	res, err := env.Client.Get(env.Server.URL + "/api/invoice-archive/2026/06/zip")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("zip status = %d body=%s, want 200", res.StatusCode, string(body))
	}
	if got := res.Header.Get("Content-Type"); got != "application/zip" {
		t.Fatalf("content type = %q, want application/zip", got)
	}
	if got := res.Header.Get("Content-Disposition"); !strings.Contains(got, `rekini-2026-06.zip`) {
		t.Fatalf("content disposition = %q, want rekini-2026-06.zip", got)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("zip reader: %v", err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("zip entries = %d, want 1", len(reader.File))
	}
	if !strings.HasSuffix(reader.File[0].Name, ".pdf") {
		t.Fatalf("zip entry name = %q, want pdf", reader.File[0].Name)
	}
}

func TestInvoiceArchiveMonthZIPDownloadRejectsMonthWithoutReadyPDFs(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Pending ZIP Student",
	})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Pending ZIP Course",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 80,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     6,
		"hours":     1.5,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  2026,
		"month": 6,
	})

	env.Runtime.Config.FontsDir = filepath.Join(t.TempDir(), "missing-fonts")
	t.Chdir(t.TempDir())

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=6&status=all")
	postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})

	res, body := rawRequest(t, env.Client, http.MethodGet, env.Server.URL+"/api/invoice-archive/2026/06/zip", nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("zip status = %d body=%s, want 400", res.StatusCode, body)
	}
	if !strings.Contains(string(body), "no ready pdfs for this month") {
		t.Fatalf("zip error body = %q", string(body))
	}
}

func TestInvoiceEmailSettingsDefaultsAndReset(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	settings := getJSON[backend.InvoiceEmailSettingsDTO](t, env.Client, env.Server.URL, "/api/settings/invoice-email")
	if settings.SubjectTemplate != appruntime.DefaultInvoiceEmailSubjectTemplate {
		t.Fatalf("default subject = %q", settings.SubjectTemplate)
	}
	if settings.BodyTemplate != appruntime.DefaultInvoiceEmailBodyTemplate {
		t.Fatalf("default body = %q", settings.BodyTemplate)
	}
	if settings.ReplyTo != "" {
		t.Fatalf("default replyTo = %q, want empty", settings.ReplyTo)
	}
	if len(settings.AvailablePlaceholders) == 0 {
		t.Fatal("expected available placeholders")
	}

	updated := postJSON[backend.InvoiceEmailSettingsDTO](t, env.Client, env.Server.URL, "/api/settings/invoice-email", map[string]any{
		"subjectTemplate": "Subject {invoice_number}",
		"bodyTemplate":    "Body {amount}",
		"replyTo":         "billing@example.com",
	})
	if updated.SubjectTemplate != "Subject {invoice_number}" {
		t.Fatalf("updated subject = %q", updated.SubjectTemplate)
	}
	if updated.ReplyTo != "billing@example.com" {
		t.Fatalf("updated replyTo = %q", updated.ReplyTo)
	}

	reset := postJSON[backend.InvoiceEmailSettingsDTO](t, env.Client, env.Server.URL, "/api/settings/invoice-email", map[string]any{
		"subjectTemplate": "",
		"bodyTemplate":    "",
		"replyTo":         "",
	})
	if reset.SubjectTemplate != appruntime.DefaultInvoiceEmailSubjectTemplate {
		t.Fatalf("reset subject = %q", reset.SubjectTemplate)
	}
	if reset.BodyTemplate != appruntime.DefaultInvoiceEmailBodyTemplate {
		t.Fatalf("reset body = %q", reset.BodyTemplate)
	}
	if reset.ReplyTo != "" {
		t.Fatalf("reset replyTo = %q, want empty", reset.ReplyTo)
	}
}

func TestInvoiceEmailSendRequiresRecipient(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Missing Email Student",
	})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Missing Email Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 60,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     7,
		"hours":     1,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  2026,
		"month": 7,
	})
	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=7&status=all")
	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	pdfPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, 2026, 7, issue.Number, st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, body := rawRequest(
		t,
		env.Client,
		http.MethodPost,
		env.Server.URL+"/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/send-email",
		bytes.NewReader([]byte(`{"to":"","subject":"Test","body":"Body"}`)),
	)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("send-email status = %d body=%s, want 400", resp.StatusCode, body)
	}
}

func TestInvoiceEmailSendRequiresSMTPConfiguration(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "No SMTP Student",
		"email":    "nosmtp@example.com",
	})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "No SMTP Course",
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 60,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})
	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     8,
		"hours":     1,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{
		"year":  2026,
		"month": 8,
	})
	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=8&status=all")
	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/issue", map[string]any{
		"version": invoices[0].Version,
	})
	pdfPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, 2026, 8, issue.Number, st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, body := rawRequest(
		t,
		env.Client,
		http.MethodPost,
		env.Server.URL+"/api/invoices/"+strconv.Itoa(invoices[0].ID)+"/send-email",
		bytes.NewReader([]byte(`{"to":"nosmtp@example.com","subject":"Test","body":"Body"}`)),
	)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("send-email status = %d body=%s, want 400", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "email sending is not configured") {
		t.Fatalf("send-email body = %s, want configuration error", body)
	}
	invoiceAfterFailure := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoices[0].ID))
	if invoiceAfterFailure.LastEmailedAt != "" {
		t.Fatalf("lastEmailedAt = %q, want empty", invoiceAfterFailure.LastEmailedAt)
	}
	if invoiceAfterFailure.LastEmailedTo != "" {
		t.Fatalf("lastEmailedTo = %q, want empty", invoiceAfterFailure.LastEmailedTo)
	}
	if invoiceAfterFailure.EmailCommunicationStatus != "failed" {
		t.Fatalf("emailCommunicationStatus = %q, want failed", invoiceAfterFailure.EmailCommunicationStatus)
	}
	if !strings.Contains(invoiceAfterFailure.LastEmailError, "not configured") {
		t.Fatalf("lastEmailError = %q, want configuration error", invoiceAfterFailure.LastEmailError)
	}
}

func TestLocaleBackupAndErrors(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	locale := postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/settings/locale", map[string]any{"locale": "ru-RU"})
	if locale["locale"] != "ru-RU" {
		t.Fatalf("locale response = %q", locale["locale"])
	}

	current := getJSON[map[string]string](t, env.Client, env.Server.URL, "/api/settings/locale")
	if current["locale"] != "ru-RU" {
		t.Fatalf("current locale = %q", current["locale"])
	}

	backup := postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/backups", map[string]any{})
	if backup["filename"] == "" {
		t.Fatal("backup filename is empty")
	}

	resp, _ := rawRequest(t, env.Client, http.MethodGet, env.Server.URL+"/api/students/not-a-number", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad path status = %d, want 400", resp.StatusCode)
	}

	resp, _ = rawRequest(t, env.Client, http.MethodGet, env.Server.URL+"/api/students/999999", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing student status = %d, want 404", resp.StatusCode)
	}

	activeStudent := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Still Active"})
	resp, _ = rawRequest(t, env.Client, http.MethodDelete, env.Server.URL+"/api/students/"+strconv.Itoa(activeStudent.ID)+"?version="+strconv.Itoa(activeStudent.Version), nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("delete active student status = %d, want 409", resp.StatusCode)
	}
}

func TestLocaleIsPerUserAndStaffCanChangeIt(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	adminLocale := getJSON[map[string]string](t, env.Client, env.Server.URL, "/api/me/locale")
	if adminLocale["locale"] != "lv-LV" {
		t.Fatalf("admin locale = %q, want lv-LV", adminLocale["locale"])
	}

	created := postJSON[backend.UserDTO](t, env.Client, env.Server.URL, "/api/users", map[string]any{
		"username": "staff-locale",
		"password": "staff-pass-123",
		"role":     "staff",
	})
	if created.Role != "staff" {
		t.Fatalf("created role = %q, want staff", created.Role)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	staffClient := &http.Client{Jar: jar}
	login := postJSON[backend.SessionDTO](t, staffClient, env.Server.URL, "/api/auth/login", map[string]any{
		"username": "staff-locale",
		"password": "staff-pass-123",
	})
	if !login.Authenticated || login.Locale != "lv-LV" {
		t.Fatalf("unexpected staff login locale: %+v", login)
	}

	staffLocale := postJSON[map[string]string](t, staffClient, env.Server.URL, "/api/me/locale", map[string]any{
		"locale": "ru-RU",
	})
	if staffLocale["locale"] != "ru-RU" {
		t.Fatalf("staff locale response = %q", staffLocale["locale"])
	}

	staffSession := getJSON[backend.SessionDTO](t, staffClient, env.Server.URL, "/api/auth/session")
	if staffSession.Locale != "ru-RU" {
		t.Fatalf("staff session locale = %q, want ru-RU", staffSession.Locale)
	}

	adminSession := getJSON[backend.SessionDTO](t, env.Client, env.Server.URL, "/api/auth/session")
	if adminSession.Locale != "lv-LV" {
		t.Fatalf("admin session locale = %q, want lv-LV", adminSession.Locale)
	}

	adminLocaleAfter := getJSON[map[string]string](t, env.Client, env.Server.URL, "/api/me/locale")
	if adminLocaleAfter["locale"] != "lv-LV" {
		t.Fatalf("admin locale after staff change = %q, want lv-LV", adminLocaleAfter["locale"])
	}
}

func TestAuthLoginProtectsAPIAndLogout(t *testing.T) {
	env := newAnonymousTestServer(t)
	defer env.Close()

	session := getJSON[backend.SessionDTO](t, env.Client, env.Server.URL, "/api/auth/session")
	if session.Authenticated {
		t.Fatal("expected anonymous session")
	}

	resp, _ := rawRequest(t, env.Client, http.MethodGet, env.Server.URL+"/api/students", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous students status = %d, want 401", resp.StatusCode)
	}

	resp, _ = rawRequest(
		t,
		env.Client,
		http.MethodPost,
		env.Server.URL+"/api/auth/login",
		bytes.NewReader([]byte(`{"username":"admin","password":"wrong"}`)),
	)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d, want 401", resp.StatusCode)
	}

	login := postJSON[backend.SessionDTO](t, env.Client, env.Server.URL, "/api/auth/login", map[string]any{
		"username": env.AdminUsername,
		"password": env.AdminPassword,
	})
	if !login.Authenticated || login.User == nil || login.User.Username != env.AdminUsername {
		t.Fatalf("unexpected login session: %+v", login)
	}
	if login.Locale != "lv-LV" {
		t.Fatalf("login locale = %q, want lv-LV", login.Locale)
	}

	_ = getJSON[[]backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students?q=&includeInactive=false")

	resp, _ = rawRequest(t, env.Client, http.MethodPost, env.Server.URL+"/api/auth/logout", bytes.NewReader([]byte("{}")))
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", resp.StatusCode)
	}

	resp, _ = rawRequest(t, env.Client, http.MethodGet, env.Server.URL+"/api/students", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout students status = %d, want 401", resp.StatusCode)
	}
}

func TestUserManagementAndStaffRestrictions(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	created := postJSON[backend.UserDTO](t, env.Client, env.Server.URL, "/api/users", map[string]any{
		"username": "staff",
		"password": "staff-pass-123",
		"role":     "staff",
	})
	if created.Role != "staff" {
		t.Fatalf("created role = %q, want staff", created.Role)
	}

	users := getJSON[[]backend.UserDTO](t, env.Client, env.Server.URL, "/api/users")
	if len(users) < 2 {
		t.Fatalf("users count = %d, want at least 2", len(users))
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	staffClient := &http.Client{Jar: jar}
	login := postJSON[backend.SessionDTO](t, staffClient, env.Server.URL, "/api/auth/login", map[string]any{
		"username": "staff",
		"password": "staff-pass-123",
	})
	if !login.Authenticated || login.User == nil || login.User.Role != "staff" {
		t.Fatalf("unexpected staff session: %+v", login)
	}
	if !login.Capabilities["invoiceArchive"] {
		t.Fatalf("staff invoiceArchive capability = false, want true")
	}

	staffStudent := postJSON[backend.StudentDTO](t, staffClient, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Staff Created Student",
	})
	if staffStudent.FullName != "Staff Created Student" {
		t.Fatalf("staff student fullName = %q", staffStudent.FullName)
	}

	resp, _ := rawRequest(t, staffClient, http.MethodGet, env.Server.URL+"/api/users", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff users status = %d, want 403", resp.StatusCode)
	}

	resp, _ = rawRequest(t, staffClient, http.MethodPost, env.Server.URL+"/api/backups", bytes.NewReader([]byte(`{}`)))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff backups status = %d, want 403", resp.StatusCode)
	}

	resp, _ = rawRequest(t, staffClient, http.MethodGet, env.Server.URL+"/api/settings/invoice-email", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff invoice email settings get status = %d, want 403", resp.StatusCode)
	}

	resp, _ = rawRequest(t, staffClient, http.MethodGet, env.Server.URL+"/api/invoice-archive", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff invoice archive status = %d, want 200", resp.StatusCode)
	}

	resp, _ = rawRequest(
		t,
		staffClient,
		http.MethodPost,
		env.Server.URL+"/api/settings/invoice-email",
		bytes.NewReader([]byte(`{"subjectTemplate":"x","bodyTemplate":"y","replyTo":""}`)),
	)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff invoice email settings post status = %d, want 403", resp.StatusCode)
	}

	resp, _ = rawRequest(t, staffClient, http.MethodDelete, env.Server.URL+"/api/students/"+strconv.Itoa(staffStudent.ID)+"?version="+strconv.Itoa(staffStudent.Version), nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff delete student status = %d, want 403", resp.StatusCode)
	}

	resp, _ = rawRequest(t, staffClient, http.MethodDelete, env.Server.URL+"/api/users/"+strconv.Itoa(created.ID), nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff delete user status = %d, want 403", resp.StatusCode)
	}

	resp, _ = rawRequest(t, staffClient, http.MethodGet, env.Server.URL+"/api/audit-logs", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff audit logs status = %d, want 403", resp.StatusCode)
	}

	updated := putJSON[backend.UserDTO](t, env.Client, env.Server.URL, "/api/users/"+strconv.Itoa(created.ID), map[string]any{
		"username": "staff",
		"role":     "staff",
		"isActive": false,
	})
	if updated.IsActive {
		t.Fatal("expected user to be inactive")
	}

	resp, _ = rawRequest(
		t,
		staffClient,
		http.MethodPost,
		env.Server.URL+"/api/auth/login",
		bytes.NewReader([]byte(`{"username":"staff","password":"staff-pass-123"}`)),
	)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("inactive staff login status = %d, want 401", resp.StatusCode)
	}

	deletable := postJSON[backend.UserDTO](t, env.Client, env.Server.URL, "/api/users", map[string]any{
		"username": "delete-me",
		"password": "delete-me-pass",
		"role":     "staff",
	})
	resp, _ = rawRequest(t, env.Client, http.MethodDelete, env.Server.URL+"/api/users/"+strconv.Itoa(deletable.ID), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete user status = %d, want 204", resp.StatusCode)
	}

	users = getJSON[[]backend.UserDTO](t, env.Client, env.Server.URL, "/api/users")
	for _, item := range users {
		if item.ID == deletable.ID {
			t.Fatalf("deleted user %d still present in list", deletable.ID)
		}
	}

	resp, _ = rawRequest(
		t,
		http.DefaultClient,
		http.MethodPost,
		env.Server.URL+"/api/auth/login",
		bytes.NewReader([]byte(`{"username":"delete-me","password":"delete-me-pass"}`)),
	)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("deleted user login status = %d, want 401", resp.StatusCode)
	}

	admin, err := env.Runtime.DB.Ent.User.Query().Where(user.UsernameEQ(env.AdminUsername)).Only(context.Background())
	if err != nil {
		t.Fatalf("admin query failed: %v", err)
	}
	resp, body := rawRequest(t, env.Client, http.MethodDelete, env.Server.URL+"/api/users/"+strconv.Itoa(admin.ID), nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("self delete status = %d, want 400", resp.StatusCode)
	}
	if !bytes.Contains(body, []byte("cannot delete your own account")) {
		t.Fatalf("self delete body = %q", string(body))
	}

}

func TestStaffCanManageEnrollmentMaterialsToggle(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	postJSON[backend.UserDTO](t, env.Client, env.Server.URL, "/api/users", map[string]any{
		"username": "staff-enrollment",
		"password": "staff-pass-123",
		"role":     "staff",
	})

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	staffClient := &http.Client{Jar: jar}
	login := postJSON[backend.SessionDTO](t, staffClient, env.Server.URL, "/api/auth/login", map[string]any{
		"username": "staff-enrollment",
		"password": "staff-pass-123",
	})
	if !login.Authenticated || login.User == nil || login.User.Role != "staff" {
		t.Fatalf("unexpected staff session: %+v", login)
	}

	st := postJSON[backend.StudentDTO](t, staffClient, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Staff Enrollment Student",
	})
	course := postJSON[backend.CourseDTO](t, staffClient, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Staff Enrollment Course",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 80,
	})

	enrollment := postJSON[backend.EnrollmentDTO](t, staffClient, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     false,
		"lessonPriceOverride": 0,
		"note":                "online",
	})
	if enrollment.ChargeMaterials {
		t.Fatal("created chargeMaterials = true, want false")
	}

	updated := putJSON[backend.EnrollmentDTO](t, staffClient, env.Server.URL, "/api/enrollments/"+strconv.Itoa(enrollment.ID), map[string]any{
		"version":                 enrollment.Version,
		"billingMode":             "per_lesson",
		"chargeMaterials":         true,
		"lessonPriceOverride":     0,
		"subscriptionLessonPrice": 0,
		"note":                    "offline",
	})
	if !updated.ChargeMaterials {
		t.Fatal("updated chargeMaterials = false, want true")
	}
}

func TestStudentUpdateRejectsStaleVersion(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Concurrent Student",
	})

	first := putJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students/"+strconv.Itoa(st.ID), map[string]any{
		"version":  st.Version,
		"fullName": "Concurrent Student A",
	})
	if first.Version <= st.Version {
		t.Fatalf("updated version = %d, want > %d", first.Version, st.Version)
	}

	resp, body := rawRequest(
		t,
		env.Client,
		http.MethodPut,
		env.Server.URL+"/api/students/"+strconv.Itoa(st.ID),
		bytes.NewReader([]byte(`{"version":1,"fullName":"Concurrent Student B"}`)),
	)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("stale update status = %d body=%s, want 409", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "record was changed or deleted by another user") {
		t.Fatalf("stale update body = %s", body)
	}
}

func TestInvoicePaymentDeleteThenReopenWorkflow(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{"fullName": "Workflow Student"})
	course := postJSON[backend.CourseDTO](t, env.Client, env.Server.URL, "/api/courses", map[string]any{
		"name":              "Workflow Course",
		"type":              "group",
		"lessonPrice":       40,
		"subscriptionPrice": 100,
	})
	postJSON[backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments", map[string]any{
		"studentId":           st.ID,
		"courseId":            course.ID,
		"billingMode":         "per_lesson",
		"chargeMaterials":     true,
		"lessonPriceOverride": 0,
		"note":                "",
	})

	putJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     8,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Client, env.Server.URL, "/api/invoices/generate-drafts", map[string]any{"year": 2026, "month": 8})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Client, env.Server.URL, "/api/invoices?year=2026&month=8&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}
	invoiceItem := invoices[0]

	postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceItem.ID)+"/issue", map[string]any{
		"version": invoiceItem.Version,
	})

	payment := postJSON[backend.PaymentDTO](t, env.Client, env.Server.URL, "/api/payments", map[string]any{
		"studentId": st.ID,
		"invoiceId": invoiceItem.ID,
		"amount":    40,
		"method":    "cash",
		"paidAt":    "2026-08-05",
	})

	invoiceDetails := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceItem.ID))
	resp, _ := rawRequest(
		t,
		env.Client,
		http.MethodPost,
		env.Server.URL+"/api/invoices/"+strconv.Itoa(invoiceItem.ID)+"/reopen-draft",
		bytes.NewReader([]byte(fmt.Sprintf(`{"version":%d}`, invoiceDetails.Version))),
	)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("reopen with payment status = %d, want 409", resp.StatusCode)
	}

	resp, _ = rawRequest(t, env.Client, http.MethodDelete, env.Server.URL+"/api/payments/"+strconv.Itoa(payment.ID), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete payment status = %d, want 204", resp.StatusCode)
	}

	invoiceDetails = getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceItem.ID))
	resp, body := rawRequest(
		t,
		env.Client,
		http.MethodPost,
		env.Server.URL+"/api/invoices/"+strconv.Itoa(invoiceItem.ID)+"/reopen-draft",
		bytes.NewReader([]byte(fmt.Sprintf(`{"version":%d}`, invoiceDetails.Version))),
	)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reopen after payment delete status = %d body=%s, want 200", resp.StatusCode, body)
	}

	reopened := getJSON[backend.InvoiceDTO](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceItem.ID))
	if reopened.Status != "draft" {
		t.Fatalf("invoice status = %q, want draft", reopened.Status)
	}
}

func TestStaticServingWithDist(t *testing.T) {
	distDir := writeTestDist(t)
	env := newTestServerWithDist(t, distDir)
	defer env.Close()

	resp, body := rawRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("root status = %d, want 200", resp.StatusCode)
	}
	if got := string(body); got != "<!doctype html><title>LangSchool</title>" {
		t.Fatalf("root body = %q", got)
	}

	resp, body = rawRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/students", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("spa route status = %d, want 200", resp.StatusCode)
	}
	if got := string(body); got != "<!doctype html><title>LangSchool</title>" {
		t.Fatalf("spa body = %q", got)
	}

	resp, body = rawRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/assets/app.js", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("asset status = %d, want 200", resp.StatusCode)
	}
	if got := string(body); got != "console.log('langschool');" {
		t.Fatalf("asset body = %q", got)
	}

	resp, _ = rawRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/assets/missing.js", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing asset status = %d, want 404", resp.StatusCode)
	}

	health := getJSON[map[string]bool](t, http.DefaultClient, env.Server.URL, "/healthz")
	if !health["ready"] {
		t.Fatal("healthz ready = false with dist")
	}
}

func TestStaticServingFallsBackToAPIOnlyWithoutDist(t *testing.T) {
	env := newTestServerWithDist(t, filepath.Join(t.TempDir(), "missing-dist"))
	defer env.Close()

	resp, _ := rawRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/students", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("spa route without dist status = %d, want 404", resp.StatusCode)
	}

	health := getJSON[map[string]bool](t, http.DefaultClient, env.Server.URL, "/healthz")
	if !health["ready"] {
		t.Fatal("healthz ready = false without dist")
	}
}

type testServerEnv struct {
	Server        *httptest.Server
	Runtime       *appruntime.Runtime
	Client        *http.Client
	AdminUsername string
	AdminPassword string
}

func (e *testServerEnv) Close() {
	e.Server.Close()
}

func newTestServer(t *testing.T) *testServerEnv {
	t.Helper()
	env := newAnonymousTestServerWithEmailSenderAndDist(t, nil, "")
	env.login(t)
	return env
}

func newAnonymousTestServer(t *testing.T) *testServerEnv {
	t.Helper()
	return newAnonymousTestServerWithEmailSenderAndDist(t, nil, "")
}

func newTestServerWithDist(t *testing.T, distDir string) *testServerEnv {
	t.Helper()
	env := newAnonymousTestServerWithEmailSenderAndDist(t, nil, distDir)
	env.login(t)
	return env
}

func newAnonymousTestServerWithDist(t *testing.T, distDir string) *testServerEnv {
	t.Helper()
	return newAnonymousTestServerWithEmailSenderAndDist(t, nil, distDir)
}

func newTestServerWithEmailSender(t *testing.T, sender email.Sender) *testServerEnv {
	t.Helper()
	env := newAnonymousTestServerWithEmailSenderAndDist(t, sender, "")
	env.login(t)
	return env
}

func newAnonymousTestServerWithEmailSenderAndDist(t *testing.T, sender email.Sender, distDir string) *testServerEnv {
	t.Helper()
	root := t.TempDir()
	fontsDir, err := makeTestFontsDir(t)
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	adminUsername := "admin"
	adminPassword := "test-password-123"
	rt, err := appruntime.Start(context.Background(), appruntime.Config{
		BaseDir:       filepath.Join(root, "base"),
		DataDir:       filepath.Join(root, "data"),
		BackupsDir:    filepath.Join(root, "backups"),
		InvoicesDir:   filepath.Join(root, "invoices"),
		ExportsDir:    filepath.Join(root, "exports"),
		FontsDir:      fontsDir,
		AdminUsername: adminUsername,
		AdminPassword: adminPassword,
		SessionSecret: "test-session-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = rt.Close()
	})
	return &testServerEnv{
		Server:        httptest.NewServer(NewHandler(backend.NewWithEmailSender(rt, sender), HandlerOptions{DistDir: distDir})),
		Runtime:       rt,
		Client:        &http.Client{Jar: jar},
		AdminUsername: adminUsername,
		AdminPassword: adminPassword,
	}
}

type stubEmailSender struct {
	lastMessage email.Message
}

func (s *stubEmailSender) Send(_ context.Context, msg email.Message) error {
	s.lastMessage = msg
	return nil
}

func (e *testServerEnv) login(t *testing.T) {
	t.Helper()
	_ = postJSON[backend.SessionDTO](t, e.Client, e.Server.URL, "/api/auth/login", map[string]any{
		"username": e.AdminUsername,
		"password": e.AdminPassword,
	})
}

func writeTestDist(t *testing.T) string {
	t.Helper()
	distDir := filepath.Join(t.TempDir(), "dist")
	if err := os.MkdirAll(filepath.Join(distDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<!doctype html><title>LangSchool</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "assets", "app.js"), []byte("console.log('langschool');"), 0o644); err != nil {
		t.Fatal(err)
	}
	return distDir
}

func makeTestFontsDir(t *testing.T) (string, error) {
	t.Helper()
	sourceDir, err := filepath.Abs(filepath.Join("..", "..", "Fonts"))
	if err != nil {
		return "", err
	}
	targetDir := filepath.Join(t.TempDir(), "fonts")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	files := map[string]string{
		"DejaVuSans.ttf":             "DejaVuSans.ttf",
		"DejaVuSans-Bold.ttf":        "DejaVuSans-Bold.ttf",
		"DejaVuSans-Oblique.ttf":     "DejaVuSans.ttf",
		"DejaVuSans-BoldOblique.ttf": "DejaVuSans-Bold.ttf",
	}

	for targetName, sourceName := range files {
		src := filepath.Join(sourceDir, sourceName)
		dst := filepath.Join(targetDir, targetName)
		data, err := os.ReadFile(src)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return "", err
		}
	}

	return targetDir, nil
}

func getJSON[T any](t *testing.T, client *http.Client, baseURL, path string) T {
	t.Helper()
	resp, err := client.Get(baseURL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status = %d body=%s", path, resp.StatusCode, body)
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func postJSON[T any](t *testing.T, client *http.Client, baseURL, path string, payload any) T {
	t.Helper()
	return doJSON[T](t, client, http.MethodPost, baseURL+path, payload, http.StatusCreated, http.StatusOK)
}

func putJSON[T any](t *testing.T, client *http.Client, baseURL, path string, payload any) T {
	t.Helper()
	return doJSON[T](t, client, http.MethodPut, baseURL+path, payload, http.StatusOK)
}

func doJSON[T any](t *testing.T, client *http.Client, method, url string, payload any, wantStatuses ...int) T {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, body := rawRequest(t, client, method, url, bytes.NewReader(data))
	ok := false
	for _, status := range wantStatuses {
		if resp.StatusCode == status {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("%s %s status = %d body=%s", method, url, resp.StatusCode, body)
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func rawRequest(t *testing.T, client *http.Client, method, url string, body io.Reader) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, data
}

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
