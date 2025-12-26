# Language School Billing - Comprehensive Code Analysis & Refactoring Plan

**Date:** 2025-12-25  
**Reviewer:** Senior Developer (Code Review)  
**Repository:** Uvlazhnitel/Language-School-Billing

---

## Executive Summary

This document provides a comprehensive analysis of the Language School Billing application, identifying code quality issues, security concerns, and optimization opportunities. The codebase is a single-user desktop application built with Go (backend), Wails v2 (framework), and React/TypeScript (frontend).

**Overall Assessment:** The codebase is **moderately well-structured** with good separation of concerns. However, there are several areas requiring improvement:
- **Missing test coverage** (no tests found)
- **No CI/CD pipeline** (no GitHub Actions or other CI)
- **No linting configuration** (no golangci-lint, eslint configs)
- **Large monolithic frontend component** (App.tsx: 1217 lines)
- **Duplicate constants** across backend files
- **Missing error handling** in some critical paths
- **No input sanitization** for security

**Lines of Code:**
- Backend (Go): ~2,447 lines (excluding ent-generated code)
- Frontend (TypeScript/React): ~1,217 lines (App.tsx alone)

---

## 1. Project Structure Map

### Backend Architecture (Go)
```
├── main.go                           # Entry point (41 lines)
├── app.go                            # Application controller (523 lines)
├── crud.go                           # CRUD operations (547 lines)
├── sqlite_driver_import.go           # SQLite driver import (5 lines)
├── internal/
│   ├── app/
│   │   ├── constants.go              # Shared constants (28 lines)
│   │   ├── attendance/service.go     # Attendance logic (157 lines)
│   │   ├── invoice/service.go        # Invoice generation (537 lines)
│   │   └── payment/service.go        # Payment tracking (360 lines)
│   ├── infra/
│   │   └── db.go                     # Database initialization
│   ├── paths/
│   │   └── paths.go                  # App directory management
│   └── pdf/
│       └── invoice_pdf.go            # PDF generation (197 lines)
└── ent/                              # ORM schemas & generated code
    └── schema/
        ├── student.go
        ├── course.go
        ├── enrollment.go
        ├── invoice.go
        ├── invoiceline.go
        ├── payment.go
        ├── attendancemonth.go
        ├── priceoverride.go
        └── settings.go
```

### Frontend Architecture (TypeScript/React)
```
frontend/
├── src/
│   ├── main.tsx                      # React entry point
│   ├── App.tsx                       # **MONOLITHIC: 1,217 lines**
│   ├── App.css                       # Styles
│   └── lib/                          # API wrappers
│       ├── constants.ts              # Type-safe enums
│       ├── students.ts
│       ├── courses.ts
│       ├── enrollments.ts
│       ├── attendance.ts
│       ├── invoices.ts
│       ├── payments.ts
│       └── utils.ts
└── wailsjs/                          # Auto-generated bindings
```

---

## 2. Key Issues Found

### 2.1 Code Duplication

#### **Issue 1: Duplicate Constants (P0)**
**Location:** `crud.go` (lines 19-27) and `internal/app/constants.go` (lines 4-27)

**Problem:** Constants are defined in **two** places, creating risk of inconsistency.

**Example:**
```go
// crud.go
const (
    CourseTypeGroup      = "group"
    CourseTypeIndividual = "individual"
    BillingModeSubscription = "subscription"
    BillingModePerLesson    = "per_lesson"
)

// internal/app/constants.go
const (
    CourseTypeGroup      = "group"
    CourseTypeIndividual = "individual"
    BillingModeSubscription = "subscription"
    BillingModePerLesson    = "per_lesson"
    InvoiceStatusDraft    = "draft"
    InvoiceStatusIssued   = "issued"
    InvoiceStatusPaid     = "paid"
    InvoiceStatusCanceled = "canceled"
    PaymentMethodCash = "cash"
    PaymentMethodBank = "bank"
)
```

**Impact:** High - If one is updated without the other, bugs will occur.

**Fix:** Import constants from `internal/app` package in `crud.go`.

---

#### **Issue 2: Duplicate round2() Function (P1)**
**Location:** 
- `internal/app/invoice/service.go` (line 80)
- `internal/app/payment/service.go` (line 60)

**Problem:** Same helper function defined twice.

**Example:**
```go
// Both files have:
func round2(v float64) float64 { return math.Round(v*100) / 100 }
```

**Fix:** Move to shared utility package (e.g., `internal/app/utils.go`).

---

#### **Issue 3: Duplicate toDTO Conversion Pattern (P2)**
**Location:** `crud.go` (lines 104-144)

**Problem:** Manual DTO conversions without consistency checks.

**Example:**
```go
func toStudentDTO(s *ent.Student) StudentDTO { ... }
func toCourseDTO(c *ent.Course) CourseDTO { ... }
func toEnrollmentDTO(e *ent.Enrollment) EnrollmentDTO { ... }
```

**Impact:** Medium - Verbose, error-prone, but functional.

**Fix:** Consider code generation or struct tags for automatic mapping.

---

### 2.2 Missing Test Coverage

#### **Issue 4: No Unit Tests (P0)**
**Impact:** **CRITICAL** - No automated validation of business logic.

**Files Checked:**
```bash
$ find . -name "*_test.go"
# Returns: (empty)
```

**Missing Tests For:**
1. **CRUD operations** (`crud.go`) - Student/Course/Enrollment validation
2. **Invoice generation logic** (`internal/app/invoice/service.go`)
3. **Payment calculations** (`internal/app/payment/service.go`)
4. **Attendance tracking** (`internal/app/attendance/service.go`)
5. **PDF generation** (`internal/pdf/invoice_pdf.go`)

**Risk:** Regressions are undetectable until production use.

---

### 2.3 Missing CI/CD & Linting

#### **Issue 5: No CI Pipeline (P0)**
**Impact:** **CRITICAL** - No automated checks before merge.

**Missing:**
- No `.github/workflows/` directory
- No automated build verification
- No automated test execution
- No dependency vulnerability scanning

**Fix:** Add GitHub Actions workflow for:
1. Go build & test
2. Frontend build & typecheck
3. golangci-lint
4. npm audit

---

#### **Issue 6: No Linting Configuration (P1)**
**Impact:** High - Code style inconsistencies.

**Missing:**
- No `.golangci.yml` (Go linting)
- No `.eslintrc.json` (TypeScript linting)
- No `prettier` config (code formatting)

**Current State:** Manual code review only.

---

### 2.4 Security Issues

#### **Issue 7: No Input Sanitization (P0)**
**Location:** Multiple CRUD methods in `crud.go`

**Problem:** User inputs not sanitized for XSS/injection.

**Example:**
```go
// crud.go:214
func (a *App) StudentUpdate(id int, fullName, phone, email, note string) (*StudentDTO, error) {
    fullName = strings.TrimSpace(fullName)  // Only trimmed, not sanitized
    // ...
}
```

**Risk:** 
- HTML injection in `note` fields → XSS in generated PDFs
- SQL injection (mitigated by ent ORM, but still a concern)

**Fix:** Add HTML escaping for all user inputs, especially before PDF generation.

---

#### **Issue 8: Potential Path Traversal (P1)**
**Location:** `app.go:410-427` (OpenFile method)

**Problem:** User-controlled file path without validation.

**Example:**
```go
func (a *App) OpenFile(path string) error {
    if abs, err := filepath.Abs(path); err == nil {
        path = abs
    }
    // No validation that path is within expected directories
    // ...
}
```

**Risk:** Attacker could open arbitrary system files.

**Fix:** Validate that path is within `~/LangSchool/` directory tree.

---

#### **Issue 9: Hardcoded Sensitive Defaults (P2)**
**Location:** `app.go:79` and `wails.json:10`

**Problem:** Email address hardcoded in source.

**Example:**
```json
// wails.json
{
  "author": {
    "name": "Ilya",
    "email": "ilya.yunkins@gmail.com"  // ← Personal email in public repo
  }
}
```

**Risk:** Privacy exposure, spam.

**Fix:** Remove or redact personal information.

---

### 2.5 Code Quality Issues

#### **Issue 10: Monolithic Frontend Component (P1)**
**Location:** `frontend/src/App.tsx` (1,217 lines)

**Problem:** Entire UI in one file - difficult to maintain.

**Structure:**
```tsx
export default function App() {
  // 100+ lines of state declarations
  // 200+ lines of Student tab logic
  // 200+ lines of Courses tab logic
  // 200+ lines of Enrollments tab logic
  // 200+ lines of Attendance tab logic
  // 300+ lines of Invoice tab logic
  // Total: 1,217 lines
}
```

**Impact:** High - Hard to navigate, test, and modify.

**Fix:** Split into separate components:
- `StudentsTab.tsx`
- `CoursesTab.tsx`
- `EnrollmentsTab.tsx`
- `AttendanceTab.tsx`
- `InvoicesTab.tsx`

---

#### **Issue 11: Missing Error Context (P1)**
**Location:** Multiple service methods

**Problem:** Errors lack context for debugging.

**Example:**
```go
// internal/app/invoice/service.go:116
if err != nil {
    if !ent.IsNotFound(err) {
        fmt.Printf("AttendanceMonth query error: %v\n", err)  // ← Printf to stdout
    }
}
```

**Issues:**
- `fmt.Printf` instead of proper logging
- No structured logging (e.g., with fields)
- Errors not wrapped with context

**Fix:** Use proper logger (e.g., `log/slog`) with structured fields.

---

#### **Issue 12: Magic Numbers & Strings (P2)**
**Location:** Multiple files

**Examples:**
```go
// app.go:68 - Magic number
Where(settings.SingletonIDEQ(1))  // Why 1?

// internal/pdf/invoice_pdf.go:105 - Magic file permissions
os.MkdirAll(dir, 0o755)  // Not documented

// crud.go:95 - Magic range
if discountPct < 0 || discountPct > 100 {  // Should be constant
```

**Fix:** Define named constants with documentation.

---

### 2.6 Unused/Dead Code

#### **Issue 13: Unused Files/Directories (P2)**
**Location:** Root directory

**Found:**
- `Users/` directory (empty or leftover?)
- `fonts/` directory (appears unused, fonts stored in `~/LangSchool/Fonts/`)

**Verification:**
```bash
$ find . -name "Users" -type d
# Found: ./Users (but empty)
```

**Fix:** Remove if truly unused.

---

#### **Issue 14: Unused Function (P2)**
**Location:** `app.go:217-221`

**Code:**
```go
func fileExists(path string) bool {
    _, err := os.Stat(path)
    return err == nil
}
```

**Analysis:** Only used in `dirHasFonts()`, could be inlined.

**Impact:** Low, but clutters code.

---

### 2.7 Potential Bugs

#### **Issue 15: Race Condition Risk (P1)**
**Location:** `internal/app/invoice/service.go:388-438` (issueOne method)

**Problem:** Transaction rollback deferred without checking commit status.

**Example:**
```go
func (s *Service) issueOne(ctx context.Context, id int) (string, error) {
    tx, err := s.db.Tx(ctx)
    if err != nil {
        return "", err
    }
    defer func() { _ = tx.Rollback() }()  // ← Always called, even after commit

    // ... business logic ...

    if err := tx.Commit(); err != nil {
        return "", err
    }
    return number, nil
    // Rollback is called here (no-op after commit, but incorrect pattern)
}
```

**Issue:** While `Rollback()` after `Commit()` is a no-op in most drivers, this pattern is confusing and error-prone.

**Fix:** Use proper defer pattern:
```go
defer func() {
    if err != nil {
        tx.Rollback()
    }
}()
```

---

#### **Issue 16: Potential Division by Zero (P2)**
**Location:** `internal/app/payment/service.go:67`

**Code:**
```go
func eps() float64 { return 0.009 }
```

**Comment says:** "epsilon value used for floating-point comparisons"

**Issue:** While unlikely, using `eps()` in division without validation could cause issues.

**Current Usage:** Only in comparisons (`paid+eps() >= total`), so **safe**.

**Status:** False alarm, but good to document usage.

---

#### **Issue 17: Missing Null Checks (P1)**
**Location:** `app.go:143-159` (DevSeed method)

**Problem:** Query errors are ignored, proceeding with nil pointers.

**Example:**
```go
sAnna, err := db.Student.Query().Where(student.FullNameEQ("Anna K.")).Only(ctx)
if err != nil {
    sAnna, _ = db.Student.Create().SetFullName("Anna K.").Save(ctx)
    // ^ If Create fails, sAnna could be nil
} else {
    _, _ = sAnna.Update().SetIsActive(true).Save(ctx)
    // ^ If sAnna is nil, this panics
}
```

**Fix:** Check for nil before use, or return error.

---

### 2.8 Dependencies Issues

#### **Issue 18: Outdated Dependencies (P1)**
**Location:** `frontend/package.json`

**Current:**
```json
{
  "dependencies": {
    "react": "^18.2.0",        // Latest: 18.3.1
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "typescript": "^4.6.4",    // Latest: 5.7.x (major version behind)
    "vite": "^3.0.7"           // Latest: 6.x (major version behind)
  }
}
```

**Risk:** Security vulnerabilities, missing features.

**Fix:** Update to latest stable versions (with testing).

---

#### **Issue 19: Dead Dependencies (P2)**
**Location:** `go.mod`

**Analysis:**
```
github.com/mattn/go-sqlite3 v1.14.17       // ← Used or not?
github.com/ncruces/go-sqlite3 v0.29.1      // ← Duplicate driver?
```

**Both are SQLite drivers. Only one should be needed.**

**Verification:** `sqlite_driver_import.go` imports `github.com/ncruces/go-sqlite3`.

**Fix:** Remove `github.com/mattn/go-sqlite3` if not used.

---

### 2.9 Performance Concerns

#### **Issue 20: N+1 Query Problem (P2)**
**Location:** `internal/app/invoice/service.go:522-534`

**Problem:** Fetching invoice lines count in a loop.

**Example:**
```go
for _, iv := range invs {
    cnt, _ := s.db.InvoiceLine.Query().Where(invoiceline.InvoiceIDEQ(iv.ID)).Count(ctx)
    // ← N additional queries for N invoices
    out = append(out, ListItem{...})
}
```

**Impact:** Slow for large datasets (e.g., 100 invoices = 100 extra queries).

**Fix:** Use a single query with `GROUP BY` or load lines with `WithInvoiceLines()`.

---

---

## 3. Prioritized Improvement Recommendations

### Legend
- **P0**: Critical - Must fix before production
- **P1**: High - Should fix soon
- **P2**: Medium - Nice to have

---

### P0 - Critical Issues

1. **Add Unit Tests**
   - Files: All service files
   - Reason: No validation of business logic
   - Effort: High (2-3 days)
   - Impact: Critical

2. **Add CI/CD Pipeline**
   - File: `.github/workflows/ci.yml` (new)
   - Tasks: Build, test, lint, security scan
   - Effort: Medium (1 day)
   - Impact: High

3. **Consolidate Constants**
   - Files: `crud.go` → `internal/app/constants.go`
   - Reason: Prevent inconsistency
   - Effort: Low (1 hour)
   - Impact: High

4. **Add Input Sanitization**
   - Files: `crud.go`, `internal/pdf/invoice_pdf.go`
   - Tasks: HTML escape, path validation
   - Effort: Medium (4 hours)
   - Impact: High (security)

5. **Add Path Traversal Protection**
   - File: `app.go:410-427`
   - Task: Validate file paths
   - Effort: Low (1 hour)
   - Impact: High (security)

---

### P1 - High Priority

6. **Split Monolithic Frontend**
   - File: `frontend/src/App.tsx`
   - Task: Extract 5 tab components
   - Effort: High (1 day)
   - Impact: High (maintainability)

7. **Add Linting Configuration**
   - Files: `.golangci.yml`, `.eslintrc.json` (new)
   - Effort: Medium (2 hours)
   - Impact: Medium

8. **Add Structured Logging**
   - Files: All service files
   - Task: Replace `fmt.Printf` with `log/slog`
   - Effort: Medium (4 hours)
   - Impact: Medium

9. **Fix Transaction Pattern**
   - File: `internal/app/invoice/service.go:388-438`
   - Task: Proper defer/rollback
   - Effort: Low (30 min)
   - Impact: Medium (correctness)

10. **Update Dependencies**
    - File: `frontend/package.json`
    - Task: Update TypeScript, Vite, React
    - Effort: Medium (2 hours + testing)
    - Impact: Medium (security)

11. **Fix DevSeed Nil Checks**
    - File: `app.go:143-159`
    - Task: Add proper error handling
    - Effort: Low (30 min)
    - Impact: Medium (stability)

12. **Remove Dead Dependency**
    - File: `go.mod`
    - Task: Remove unused `mattn/go-sqlite3`
    - Effort: Low (15 min)
    - Impact: Low

---

### P2 - Medium Priority

13. **Extract round2() to Shared Util**
    - Files: `invoice/service.go`, `payment/service.go`
    - Task: Create `internal/app/utils/math.go`
    - Effort: Low (30 min)
    - Impact: Low (DRY)

14. **Document Magic Numbers**
    - Files: Multiple
    - Task: Add constants with comments
    - Effort: Low (1 hour)
    - Impact: Low (readability)

15. **Remove Unused Files/Directories**
    - Locations: `Users/`, `fonts/`
    - Effort: Low (5 min)
    - Impact: Low (cleanup)

16. **Fix N+1 Query**
    - File: `internal/app/invoice/service.go:522-534`
    - Task: Use batch loading
    - Effort: Medium (1 hour)
    - Impact: Medium (performance)

17. **Add Prettier Configuration**
    - File: `.prettierrc` (new)
    - Task: Enforce consistent formatting
    - Effort: Low (30 min)
    - Impact: Low

18. **Add Error Wrapping**
    - Files: All service files
    - Task: Use `fmt.Errorf("...: %w", err)`
    - Effort: Medium (2 hours)
    - Impact: Medium (debugging)

19. **Remove Personal Info**
    - File: `wails.json`
    - Task: Redact email
    - Effort: Low (2 min)
    - Impact: Low (privacy)

20. **Add Godoc Comments**
    - Files: All exported functions
    - Task: Add standard Go documentation
    - Effort: Medium (3 hours)
    - Impact: Low (documentation)

---

## 4. Three-Phase Refactoring Plan

### **PR #1: Quick Fixes (Safety & Security)**
**Goal:** Address critical security issues and low-hanging fruit  
**Estimated Time:** 1-2 days  
**Files Changed:** 5-8 files

**Tasks:**
1. ✅ Consolidate constants (`crud.go` → `internal/app/constants.go`)
2. ✅ Add input sanitization for XSS (HTML escape in `crud.go`)
3. ✅ Add path traversal protection (`app.go:OpenFile`)
4. ✅ Fix transaction defer pattern (`invoice/service.go`)
5. ✅ Fix DevSeed nil checks (`app.go`)
6. ✅ Remove personal email (`wails.json`)
7. ✅ Remove unused `mattn/go-sqlite3` dependency
8. ✅ Remove unused files/directories (`Users/`, empty `fonts/`)

**Benefits:**
- Closes security vulnerabilities
- Reduces technical debt
- No breaking changes

**Testing:**
- Manual smoke tests (no automated tests yet)
- Build verification

---

### **PR #2: Structural Improvements**
**Goal:** Improve code organization and maintainability  
**Estimated Time:** 2-3 days  
**Files Changed:** 10-15 files

**Tasks:**
1. ✅ Split `App.tsx` into 5 tab components
   - `StudentsTab.tsx`
   - `CoursesTab.tsx`
   - `EnrollmentsTab.tsx`
   - `AttendanceTab.tsx`
   - `InvoicesTab.tsx`
2. ✅ Extract `round2()` to `internal/app/utils/math.go`
3. ✅ Add structured logging (`log/slog`) to replace `fmt.Printf`
4. ✅ Fix N+1 query in invoice listing
5. ✅ Add error wrapping with context
6. ✅ Document magic numbers with constants
7. ✅ Update frontend dependencies (TypeScript, Vite)

**Benefits:**
- Better code organization
- Easier to navigate and modify
- Improved performance
- Better debugging

**Testing:**
- Manual testing of all features
- Build verification

---

### **PR #3: Quality Infrastructure (Tests & CI)**
**Goal:** Add automated validation and quality gates  
**Estimated Time:** 3-4 days  
**Files Changed:** 20+ files (mostly new)

**Tasks:**
1. ✅ Add unit tests for:
   - Student/Course/Enrollment CRUD
   - Invoice generation logic
   - Payment calculations
   - Attendance tracking
2. ✅ Add CI/CD pipeline (`.github/workflows/ci.yml`)
   - Go build & test
   - Frontend build & typecheck
   - golangci-lint
   - npm audit
3. ✅ Add linting configuration
   - `.golangci.yml` (Go)
   - `.eslintrc.json` (TypeScript)
   - `.prettierrc` (formatting)
4. ✅ Add Godoc comments for all exported functions
5. ✅ Add code coverage reporting
6. ✅ Add pre-commit hooks (optional)

**Benefits:**
- Automated validation
- Prevents regressions
- Enforces code standards
- Better documentation

**Testing:**
- All new tests must pass
- CI pipeline must pass
- Code coverage > 60% (initial target)

---

## 5. Metrics & Goals

### Current State
- **Test Coverage:** 0%
- **Lines of Code:** ~3,664 (backend + frontend)
- **Cyclomatic Complexity:** High (App.tsx)
- **Dependencies:** Some outdated
- **CI/CD:** None
- **Linting:** None
- **Security Issues:** 3 critical

### Target State (After Refactoring)
- **Test Coverage:** 60%+
- **Cyclomatic Complexity:** Reduced by 50%
- **Dependencies:** All up-to-date
- **CI/CD:** Automated checks on all PRs
- **Linting:** Enforced on all commits
- **Security Issues:** 0 critical

---

## 6. Risk Assessment

### High Risk Items
1. **No tests** - Regressions undetectable
2. **No CI** - Manual validation only
3. **Security issues** - XSS, path traversal

### Medium Risk Items
1. **Outdated dependencies** - Known vulnerabilities
2. **Monolithic frontend** - Hard to maintain
3. **Transaction pattern** - Potential data corruption

### Low Risk Items
1. **Code duplication** - Annoying but not breaking
2. **Missing docs** - Slows onboarding
3. **Performance (N+1)** - Only affects large datasets

---

## 7. Conclusion

The Language School Billing application has a **solid foundation** but requires improvements in:
1. **Testing** - Critical gap
2. **Security** - Several issues to address
3. **Code organization** - Frontend needs splitting
4. **CI/CD** - No automation

The proposed 3-phase refactoring plan balances:
- **Quick wins** (PR #1) for immediate security improvements
- **Structural improvements** (PR #2) for long-term maintainability
- **Quality infrastructure** (PR #3) for sustainable development

**Estimated Total Time:** 6-9 days  
**Risk Level:** Low (incremental changes)  
**Business Value:** High (reduced bugs, faster development)

---

## Appendix A: File-Specific Recommendations

### Backend Files

#### `crud.go`
- Line 19-27: ❌ Remove duplicate constants → use `internal/app/constants.go`
- Line 195, 214: ⚠️ Add HTML sanitization for user inputs
- Line 62-99: ℹ️ Consider using validator library

#### `app.go`
- Line 68: ℹ️ Document why `SingletonIDEQ(1)` is hardcoded
- Line 143-159: ⚠️ Fix nil pointer risk in DevSeed
- Line 410-427: ❌ Add path validation in OpenFile

#### `internal/app/invoice/service.go`
- Line 80: ℹ️ Move `round2()` to utils
- Line 388-438: ⚠️ Fix transaction defer pattern
- Line 522-534: ⚠️ Fix N+1 query with batch loading

#### `internal/app/payment/service.go`
- Line 60: ℹ️ Move `round2()` to utils
- Line 67: ℹ️ Document `eps()` usage more clearly

#### `internal/pdf/invoice_pdf.go`
- Line 105: ℹ️ Document `0o755` permission choice
- Line 174-178: ⚠️ Sanitize `l.Description` before PDF output

### Frontend Files

#### `frontend/src/App.tsx`
- Entire file: ❌ Split into 5 separate components (1,217 lines → ~200 each)

#### `frontend/package.json`
- Line 19: ⚠️ Update TypeScript 4.6.4 → 5.7.x
- Line 20: ⚠️ Update Vite 3.0.7 → 6.x

---

## Appendix B: Security Checklist

- [ ] Input validation on all user inputs
- [ ] HTML escaping before PDF generation
- [ ] Path traversal protection in file operations
- [ ] SQL injection protection (✅ ent ORM handles this)
- [ ] XSS protection in frontend
- [ ] Dependency vulnerability scanning
- [ ] Secrets management (no hardcoded secrets found ✅)
- [ ] Rate limiting (not applicable for desktop app ✅)
- [ ] Authentication (not required for single-user app ✅)

---

## Appendix C: Testing Strategy

### Unit Tests (60% coverage target)
1. **CRUD Operations** (`crud_test.go`)
   - Student create/update/delete
   - Course create/update/delete
   - Enrollment create/update/delete
   - Validation error cases

2. **Invoice Service** (`invoice/service_test.go`)
   - Draft generation
   - Issue numbering
   - Price calculation
   - Status transitions

3. **Payment Service** (`payment/service_test.go`)
   - Payment creation
   - Balance calculation
   - Debtor list
   - Invoice status updates

4. **Attendance Service** (`attendance/service_test.go`)
   - Upsert operations
   - Lock/unlock
   - Bulk operations

### Integration Tests (20% coverage)
1. End-to-end invoice workflow
2. Payment → invoice status update
3. Attendance → invoice line generation

### Manual Testing Checklist
1. Create student → enroll → add attendance → generate invoice → issue → pay
2. PDF generation with Cyrillic characters
3. Backup/restore operations
4. Demo data seeding

---

**END OF ANALYSIS**
