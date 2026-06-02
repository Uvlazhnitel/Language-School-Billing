package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	invsvc "langschool/internal/app/invoice"
	"langschool/internal/backend"
	appruntime "langschool/internal/runtime"
)

func TestHealthAndMeta(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	health := getJSON[map[string]bool](t, env.Server, "/healthz")
	if !health["ready"] {
		t.Fatalf("healthz ready = false, want true")
	}

	meta := getJSON[backend.Meta](t, env.Server, "/api/meta")
	if !meta.Ready {
		t.Fatalf("meta ready = false, want true")
	}
	if meta.Locale == "" {
		t.Fatal("meta locale is empty")
	}
}

func TestStudentCourseEnrollmentCRUD(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Server, "/api/students", map[string]any{
		"fullName": "Alice Student",
		"phone":    "123",
		"email":    "alice@example.com",
	})
	if st.FullName != "Alice Student" {
		t.Fatalf("student fullName = %q", st.FullName)
	}

	st = putJSON[backend.StudentDTO](t, env.Server, "/api/students/"+strconv.Itoa(st.ID), map[string]any{
		"fullName": "Alice Updated",
		"phone":    "555",
		"email":    "alice@example.com",
	})
	if st.FullName != "Alice Updated" {
		t.Fatalf("student fullName = %q, want updated", st.FullName)
	}

	teacher := postJSON[backend.TeacherDTO](t, env.Server, "/api/teachers", map[string]any{
		"fullName": "Teacher One",
	})

	course := postJSON[backend.CourseDTO](t, env.Server, "/api/courses", map[string]any{
		"name":              "Drawing",
		"teacherId":         teacher.ID,
		"type":              "group",
		"lessonPrice":       20,
		"subscriptionPrice": 80,
	})
	if course.TeacherName != "Teacher One" {
		t.Fatalf("course teacherName = %q", course.TeacherName)
	}

	enrollment := postJSON[backend.EnrollmentDTO](t, env.Server, "/api/enrollments", map[string]any{
		"studentId":   st.ID,
		"courseId":    course.ID,
		"billingMode": "per_lesson",
		"discountPct": 0,
		"note":        "",
	})
	if enrollment.StudentID != st.ID {
		t.Fatalf("enrollment studentID = %d, want %d", enrollment.StudentID, st.ID)
	}

	items := getJSON[[]backend.EnrollmentDTO](t, env.Server, "/api/enrollments?studentId="+strconv.Itoa(st.ID))
	if len(items) != 1 {
		t.Fatalf("enrollment count = %d, want 1", len(items))
	}
}

func TestInvoicePDFAndPaymentWorkflow(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	st := postJSON[backend.StudentDTO](t, env.Server, "/api/students", map[string]any{"fullName": "Billing Student"})
	course := postJSON[backend.CourseDTO](t, env.Server, "/api/courses", map[string]any{
		"name":              "Painting",
		"type":              "group",
		"lessonPrice":       25,
		"subscriptionPrice": 90,
	})
	postJSON[backend.EnrollmentDTO](t, env.Server, "/api/enrollments", map[string]any{
		"studentId":   st.ID,
		"courseId":    course.ID,
		"billingMode": "per_lesson",
		"discountPct": 0,
		"note":        "",
	})

	putJSON[map[string]bool](t, env.Server, "/api/attendance", map[string]any{
		"studentId": st.ID,
		"courseId":  course.ID,
		"year":      2026,
		"month":     6,
		"hours":     1.0,
	})
	postJSON[map[string]any](t, env.Server, "/api/invoices/generate-drafts", map[string]any{"year": 2026, "month": 6})

	invoices := getJSON[[]backend.InvoiceListItem](t, env.Server, "/api/invoices?year=2026&month=6&status=all")
	if len(invoices) != 1 {
		t.Fatalf("invoice count = %d, want 1", len(invoices))
	}
	invoiceID := invoices[0].ID

	issue := postJSON[backend.IssueResult](t, env.Server, "/api/invoices/"+strconv.Itoa(invoiceID)+"/issue", map[string]any{})
	pdfPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, 2026, 6, issue.Number, st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n%test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status := getJSON[map[string]bool](t, env.Server, "/api/invoices/"+strconv.Itoa(invoiceID)+"/pdf-status")
	if !status["ready"] {
		t.Fatal("expected pdf-status ready=true")
	}

	pdfMeta := postJSON[map[string]string](t, env.Server, "/api/invoices/"+strconv.Itoa(invoiceID)+"/pdf", map[string]any{})
	if pdfMeta["downloadUrl"] == "" {
		t.Fatal("missing pdf downloadUrl")
	}

	res, err := http.Get(env.Server.URL + "/api/invoices/" + strconv.Itoa(invoiceID) + "/pdf")
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

	payment := postJSON[backend.PaymentDTO](t, env.Server, "/api/payments", map[string]any{
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

	summary := getJSON[backend.InvoiceSummaryDTO](t, env.Server, "/api/invoices/"+strconv.Itoa(invoiceID)+"/payment-summary")
	if summary.Paid <= 0 {
		t.Fatalf("summary paid = %f, want > 0", summary.Paid)
	}
}

func TestLocaleBackupAndErrors(t *testing.T) {
	env := newTestServer(t)
	defer env.Close()

	locale := postJSON[map[string]string](t, env.Server, "/api/settings/locale", map[string]any{"locale": "ru-RU"})
	if locale["locale"] != "ru-RU" {
		t.Fatalf("locale response = %q", locale["locale"])
	}

	current := getJSON[map[string]string](t, env.Server, "/api/settings/locale")
	if current["locale"] != "ru-RU" {
		t.Fatalf("current locale = %q", current["locale"])
	}

	backup := postJSON[map[string]string](t, env.Server, "/api/backups", map[string]any{})
	if backup["filename"] == "" {
		t.Fatal("backup filename is empty")
	}

	resp, _ := rawRequest(t, http.MethodGet, env.Server.URL+"/api/students/not-a-number", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad path status = %d, want 400", resp.StatusCode)
	}

	resp, _ = rawRequest(t, http.MethodGet, env.Server.URL+"/api/students/999999", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing student status = %d, want 404", resp.StatusCode)
	}

	activeStudent := postJSON[backend.StudentDTO](t, env.Server, "/api/students", map[string]any{"fullName": "Still Active"})
	resp, _ = rawRequest(t, http.MethodDelete, env.Server.URL+"/api/students/"+strconv.Itoa(activeStudent.ID), nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("delete active student status = %d, want 409", resp.StatusCode)
	}
}

type testServerEnv struct {
	Server  *httptest.Server
	Runtime *appruntime.Runtime
}

func (e *testServerEnv) Close() {
	e.Server.Close()
}

func newTestServer(t *testing.T) *testServerEnv {
	t.Helper()
	root := t.TempDir()
	fontsDir, err := makeTestFontsDir(t)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := appruntime.Start(context.Background(), appruntime.Config{
		BaseDir:     filepath.Join(root, "base"),
		DataDir:     filepath.Join(root, "data"),
		BackupsDir:  filepath.Join(root, "backups"),
		InvoicesDir: filepath.Join(root, "invoices"),
		ExportsDir:  filepath.Join(root, "exports"),
		FontsDir:    fontsDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = rt.Close()
	})
	return &testServerEnv{
		Server:  httptest.NewServer(NewHandler(backend.New(rt))),
		Runtime: rt,
	}
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

func getJSON[T any](t *testing.T, ts *httptest.Server, path string) T {
	t.Helper()
	resp, err := http.Get(ts.URL + path)
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

func postJSON[T any](t *testing.T, ts *httptest.Server, path string, payload any) T {
	t.Helper()
	return doJSON[T](t, http.MethodPost, ts.URL+path, payload, http.StatusCreated, http.StatusOK)
}

func putJSON[T any](t *testing.T, ts *httptest.Server, path string, payload any) T {
	t.Helper()
	return doJSON[T](t, http.MethodPut, ts.URL+path, payload, http.StatusOK)
}

func doJSON[T any](t *testing.T, method, url string, payload any, wantStatuses ...int) T {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, body := rawRequest(t, method, url, bytes.NewReader(data))
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

func rawRequest(t *testing.T, method, url string, body io.Reader) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
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
