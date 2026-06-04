package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
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

	health := getJSON[map[string]bool](t, http.DefaultClient, env.Server.URL, "/healthz")
	if !health["ready"] {
		t.Fatalf("healthz ready = false, want true")
	}

	meta := getJSON[backend.Meta](t, env.Client, env.Server.URL, "/api/meta")
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

	st := postJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students", map[string]any{
		"fullName": "Alice Student",
		"phone":    "123",
		"email":    "alice@example.com",
	})
	if st.FullName != "Alice Student" {
		t.Fatalf("student fullName = %q", st.FullName)
	}

	st = putJSON[backend.StudentDTO](t, env.Client, env.Server.URL, "/api/students/"+strconv.Itoa(st.ID), map[string]any{
		"fullName": "Alice Updated",
		"phone":    "555",
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
		"studentId":   st.ID,
		"courseId":    course.ID,
		"billingMode": "per_lesson",
		"discountPct": 0,
		"note":        "",
	})
	if enrollment.StudentID != st.ID {
		t.Fatalf("enrollment studentID = %d, want %d", enrollment.StudentID, st.ID)
	}

	items := getJSON[[]backend.EnrollmentDTO](t, env.Client, env.Server.URL, "/api/enrollments?studentId="+strconv.Itoa(st.ID))
	if len(items) != 1 {
		t.Fatalf("enrollment count = %d, want 1", len(items))
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
		"studentId":   st.ID,
		"courseId":    course.ID,
		"billingMode": "per_lesson",
		"discountPct": 0,
		"note":        "",
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

	issue := postJSON[backend.IssueResult](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/issue", map[string]any{})
	pdfPath := invsvc.PDFPathByNumberAndName(env.Runtime.Dirs.Invoices, 2026, 6, issue.Number, st.FullName)
	if err := os.MkdirAll(filepath.Dir(pdfPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n%test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status := getJSON[map[string]bool](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/pdf-status")
	if !status["ready"] {
		t.Fatal("expected pdf-status ready=true")
	}

	pdfMeta := postJSON[map[string]string](t, env.Client, env.Server.URL, "/api/invoices/"+strconv.Itoa(invoiceID)+"/pdf", map[string]any{})
	if pdfMeta["downloadUrl"] == "" {
		t.Fatal("missing pdf downloadUrl")
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
	resp, _ = rawRequest(t, env.Client, http.MethodDelete, env.Server.URL+"/api/students/"+strconv.Itoa(activeStudent.ID), nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("delete active student status = %d, want 409", resp.StatusCode)
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
		bytes.NewReader([]byte(`{"email":"admin@example.com","password":"wrong"}`)),
	)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d, want 401", resp.StatusCode)
	}

	login := postJSON[backend.SessionDTO](t, env.Client, env.Server.URL, "/api/auth/login", map[string]any{
		"email":    env.AdminEmail,
		"password": env.AdminPassword,
	})
	if !login.Authenticated || login.User == nil || login.User.Email != env.AdminEmail {
		t.Fatalf("unexpected login session: %+v", login)
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
		"email":    "staff@example.com",
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
		"email":    "staff@example.com",
		"password": "staff-pass-123",
	})
	if !login.Authenticated || login.User == nil || login.User.Role != "staff" {
		t.Fatalf("unexpected staff session: %+v", login)
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

	resp, _ = rawRequest(t, staffClient, http.MethodDelete, env.Server.URL+"/api/students/"+strconv.Itoa(staffStudent.ID), nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff delete student status = %d, want 403", resp.StatusCode)
	}

	updated := putJSON[backend.UserDTO](t, env.Client, env.Server.URL, "/api/users/"+strconv.Itoa(created.ID), map[string]any{
		"email":    "staff@example.com",
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
		bytes.NewReader([]byte(`{"email":"staff@example.com","password":"staff-pass-123"}`)),
	)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("inactive staff login status = %d, want 401", resp.StatusCode)
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
	AdminEmail    string
	AdminPassword string
}

func (e *testServerEnv) Close() {
	e.Server.Close()
}

func newTestServer(t *testing.T) *testServerEnv {
	t.Helper()
	env := newAnonymousTestServerWithDist(t, "")
	env.login(t)
	return env
}

func newAnonymousTestServer(t *testing.T) *testServerEnv {
	t.Helper()
	return newAnonymousTestServerWithDist(t, "")
}

func newTestServerWithDist(t *testing.T, distDir string) *testServerEnv {
	t.Helper()
	env := newAnonymousTestServerWithDist(t, distDir)
	env.login(t)
	return env
}

func newAnonymousTestServerWithDist(t *testing.T, distDir string) *testServerEnv {
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
	adminEmail := "admin@example.com"
	adminPassword := "test-password-123"
	rt, err := appruntime.Start(context.Background(), appruntime.Config{
		BaseDir:       filepath.Join(root, "base"),
		DataDir:       filepath.Join(root, "data"),
		BackupsDir:    filepath.Join(root, "backups"),
		InvoicesDir:   filepath.Join(root, "invoices"),
		ExportsDir:    filepath.Join(root, "exports"),
		FontsDir:      fontsDir,
		AdminEmail:    adminEmail,
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
		Server:        httptest.NewServer(NewHandler(backend.New(rt), HandlerOptions{DistDir: distDir})),
		Runtime:       rt,
		Client:        &http.Client{Jar: jar},
		AdminEmail:    adminEmail,
		AdminPassword: adminPassword,
	}
}

func (e *testServerEnv) login(t *testing.T) {
	t.Helper()
	_ = postJSON[backend.SessionDTO](t, e.Client, e.Server.URL, "/api/auth/login", map[string]any{
		"email":    e.AdminEmail,
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
