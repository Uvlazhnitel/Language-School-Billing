# Language School Billing System - Documentation Guide

## Table of Contents
1. [Project Overview](#project-overview)
2. [Architecture Map](#architecture-map)
3. [Technology Stack](#technology-stack)
4. [Project Structure](#project-structure)
5. [Key Components](#key-components)
6. [Recent Improvements](#recent-improvements)
7. [Development Setup](#development-setup)
8. [Testing Strategy](#testing-strategy)
9. [Security Features](#security-features)
10. [API Documentation](#api-documentation)
11. [Database Schema](#database-schema)
12. [Build and Deployment](#build-and-deployment)

---

## Project Overview

**Language School Billing System** is a desktop application for managing language school operations, including:
- Student enrollment and management
- Course scheduling and tracking
- Attendance recording
- Invoice generation and payment tracking
- PDF report generation

**Technology**: Desktop application built with Go backend and React/TypeScript frontend using Wails framework.

**Target Platform**: macOS, Windows, Linux

---

## Architecture Map

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Desktop Application                      │
│                        (Wails v2)                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐         ┌─────────────────────────┐  │
│  │   Frontend (UI)  │  ←───→  │   Backend (Business)    │  │
│  │                  │         │                         │  │
│  │  React 18.3      │  Wails  │   Go 1.21+             │  │
│  │  TypeScript 5.7  │  Bridge │   Ent ORM              │  │
│  │  Vite 6.0        │         │   SQLite Database      │  │
│  └──────────────────┘         └─────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
    User Interactions              File System Operations
    (UI Events)                    (PDFs, Database, Fonts)
```

### Layer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Presentation Layer (Frontend)                               │
│ - React Components (5 main tabs)                           │
│ - State Management                                          │
│ - UI/UX Logic                                              │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Application Layer (Go Backend)                              │
│ - CRUD Operations (crud.go)                                │
│ - Business Services (invoice, payment, attendance)         │
│ - Validation & Sanitization                                │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Domain Layer                                                │
│ - Ent Entities (Student, Course, Enrollment, Invoice)      │
│ - Business Logic                                           │
│ - Domain Models                                            │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Infrastructure Layer                                        │
│ - Database (SQLite via mattn/go-sqlite3)                   │
│ - File System (PDF generation, file operations)            │
│ - Logging (structured logging with log/slog)               │
└─────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Wails v2.10.2
- **ORM**: Ent (Facebook's entity framework)
- **Database**: SQLite 3
  - Driver: `mattn/go-sqlite3` (CGo-based, main runtime)
  - WASM Driver: `ncruces/go-sqlite3` (Pure Go, for WASM builds)
- **PDF Generation**: `jung-kurt/gofpdf`
- **Logging**: `log/slog` (Go 1.21+ structured logging)

### Frontend
- **Framework**: React 18.3.1
- **Language**: TypeScript 5.7.2
- **Build Tool**: Vite 6.0.1
- **State Management**: React Hooks (useState, useEffect)
- **UI Library**: Custom components

### Development Tools
- **Linting**: 
  - Go: golangci-lint 1.63.4
  - TypeScript: ESLint 9.15.0
- **Formatting**:
  - Go: gofmt (built-in)
  - TypeScript: Prettier 3.3.3
- **Testing**: Go test framework
- **CI/CD**: GitHub Actions

---

## Project Structure

### Root Directory Layout

```
Language-School-Billing/
├── .github/
│   └── workflows/
│       └── ci.yml                    # CI/CD pipeline (NEW)
├── frontend/                          # React TypeScript frontend
│   ├── src/
│   │   ├── App.tsx                   # Main application (1,217 lines - refactoring guide available)
│   │   ├── components/               # Component directory (NEW)
│   │   │   └── StudentsTab.example.tsx  # Example component template (NEW)
│   │   ├── hooks/                    # Custom hooks directory (NEW)
│   │   └── types/                    # TypeScript types directory (NEW)
│   ├── package.json                  # Updated dependencies (NEW)
│   ├── eslint.config.js              # ESLint configuration (NEW)
│   └── .prettierrc                   # Prettier configuration (NEW)
├── internal/                         # Go backend packages
│   ├── app/                          # Business logic layer
│   │   ├── constants.go              # Shared constants (UPDATED)
│   │   ├── attendance/               # Attendance service
│   │   ├── invoice/                  # Invoice service (UPDATED)
│   │   │   └── service.go           # N+1 query fixed, transaction pattern fixed
│   │   ├── payment/                  # Payment service (UPDATED)
│   │   │   └── service.go           # Uses shared Round2 utility
│   │   └── utils/                    # Shared utilities (NEW)
│   │       └── math.go              # Math utilities (Round2 function)
│   ├── infra/                        # Infrastructure layer
│   │   └── db.go                    # Database connection (mattn/go-sqlite3)
│   ├── logger/                       # Logging package (NEW)
│   │   └── logger.go                # Structured logging with slog
│   ├── paths/                        # Path management
│   ├── pdf/                          # PDF generation
│   │   └── invoice_pdf.go           # PDF generation (UPDATED - uses constants)
│   └── validation/                   # Input validation (NEW)
│       ├── validate.go              # Validation functions
│       └── validate_test.go         # Unit tests (19 test cases)
├── ent/                              # Ent ORM generated code
│   ├── schema/                       # Entity schemas
│   └── [generated files]
├── app.go                            # Main application (UPDATED - security fixes, constants)
├── crud.go                           # CRUD operations (UPDATED - sanitization, Godoc)
├── main.go                           # Application entry point
├── wails.json                        # Wails configuration (UPDATED - removed personal info)
├── .golangci.yml                     # Go linting configuration (NEW)
├── go.mod                            # Go dependencies
├── go.sum                            # Go dependency checksums
│
├── PROJECT_ANALYSIS.md               # Comprehensive code analysis (NEW)
├── P0_FIXES_SUMMARY.md              # P0 fixes documentation (NEW)
├── TESTING.md                        # Testing guide (NEW)
├── FRONTEND_REFACTORING.md          # Frontend refactoring guide (NEW)
└── DOCUMENTATION_GUIDE.md           # This file (NEW)
```

### Module Breakdown with Line Counts

#### Backend (Go)
```
app.go                     ~600 lines  (Main app, CRUD bindings, DevSeed)
crud.go                    ~550 lines  (Student/Course/Enrollment CRUD)
main.go                    ~50 lines   (Entry point)

internal/app/invoice/      ~680 lines  (Invoice generation & management)
internal/app/payment/      ~320 lines  (Payment processing)
internal/app/attendance/   ~280 lines  (Attendance tracking)
internal/infra/db.go       ~80 lines   (Database setup)
internal/pdf/              ~450 lines  (PDF generation)
internal/logger/           ~60 lines   (Structured logging - NEW)
internal/validation/       ~150 lines  (Validation & tests - NEW)
internal/app/utils/        ~30 lines   (Shared utilities - NEW)
internal/paths/            ~120 lines  (Path management)

ent/ (generated)           ~15,000 lines (ORM code - auto-generated)
```

#### Frontend (TypeScript/React)
```
frontend/src/App.tsx       ~1,217 lines (Main UI - refactoring guide available)
frontend/src/wailsjs/      ~500 lines   (Wails bindings - auto-generated)
frontend/src/components/   ~180 lines   (Example component - NEW)
```

#### Configuration & Documentation
```
.github/workflows/ci.yml   ~230 lines   (CI/CD pipeline - NEW)
.golangci.yml              ~80 lines    (Go linting config - NEW)
eslint.config.js           ~35 lines    (ESLint config - NEW)
.prettierrc                ~10 lines    (Prettier config - NEW)

PROJECT_ANALYSIS.md        ~920 lines   (Analysis document - NEW)
TESTING.md                 ~240 lines   (Testing guide - NEW)
FRONTEND_REFACTORING.md    ~300 lines   (Refactoring guide - NEW)
P0_FIXES_SUMMARY.md        ~300 lines   (Fixes summary - NEW)
DOCUMENTATION_GUIDE.md     ~1,200 lines (This file - NEW)
```

**Total Codebase**: ~21,500 lines (excluding generated code)
**Active Development**: ~6,500 lines of business logic

---

## Key Components

### 1. Main Application (app.go)

**Purpose**: Main application structure, Wails bindings, and initialization

**Key Responsibilities**:
- Application lifecycle management
- Database initialization
- CRUD operation bindings for frontend
- Developer seed data generation
- File operations (Open, ShowInFolder)
- Path management (directories for invoices, payments, settings)

**Recent Changes**:
- ✅ Added path traversal protection to `OpenFile()`
- ✅ Fixed nil pointer risks in `DevSeed()` with proper error handling
- ✅ Added import for `internal/app` package for constants
- ✅ Uses `app.SettingsSingletonID` constant

**Security Features**:
- Path validation prevents unauthorized file access
- Restricts file operations to `~/LangSchool/` directory

### 2. CRUD Operations (crud.go)

**Purpose**: Database CRUD operations for core entities

**Entities Managed**:
- Students (Create, Read, Update, Delete, SetActive)
- Courses (Create, List, Get, Update, Delete)
- Enrollments (Create, List, Get, Update, Delete)
- Settings (Get, Update)
- Price Overrides (Create, Delete)
- Demo Data (DevSeed)

**Recent Changes**:
- ✅ Removed duplicate constants, now imports from `internal/app`
- ✅ Added input sanitization with `sanitizeInput()` helper
- ✅ All user inputs (names, emails, notes) are HTML-escaped
- ✅ Added comprehensive Godoc comments

**Security Features**:
- XSS protection through HTML escaping
- Input validation for all fields
- Trimming whitespace from inputs

### 3. Invoice Service (internal/app/invoice/)

**Purpose**: Invoice generation, management, and PDF creation

**Key Features**:
- Draft invoice generation from enrollments and attendance
- Invoice issuing with sequential numbering
- PDF generation for issued invoices
- Payment tracking
- Bulk operations (generate drafts, issue multiple)

**Recent Changes**:
- ✅ Fixed incorrect transaction rollback pattern with `committed` flag
- ✅ Fixed N+1 query problem in invoice listing (batch query with GROUP BY)
- ✅ Added import for `internal/app/utils` for `Round2()` function
- ✅ Uses shared `utils.Round2()` for monetary calculations
- ✅ Added error context wrapping
- ✅ Added comprehensive Godoc comments for key methods

**Performance Improvements**:
- Before: N+1 queries (1 for invoices + N for line counts)
- After: 2 queries total (1 for invoices + 1 batch for all counts)

### 4. Payment Service (internal/app/payment/)

**Purpose**: Payment processing and PDF receipt generation

**Key Features**:
- Payment recording with automatic allocation
- PDF receipt generation
- Payment listing and filtering
- Multi-invoice payment support

**Recent Changes**:
- ✅ Uses shared `utils.Round2()` for monetary calculations
- ✅ Removed duplicate round2() function

### 5. Validation Package (internal/validation/) **NEW**

**Purpose**: Centralized input validation and sanitization

**Functions**:
- `SanitizeInput()` - HTML escaping for XSS protection
- `ValidateNonEmpty()` - Required field validation
- `ValidatePriceNonNegative()` - Price validation
- `ValidateDiscountPct()` - Discount percentage validation (0-100)

**Test Coverage**:
- 19 comprehensive test cases
- 100% pass rate
- Tests cover sanitization, validation, edge cases

### 6. Logger Package (internal/logger/) **NEW**

**Purpose**: Structured logging throughout the application

**Features**:
- JSON-formatted log output
- Log levels: Info, Error, Warn, Debug
- Field-based logging for structured data
- Uses Go 1.21+ `log/slog` package

**Usage Example**:
```go
import "langschool/internal/logger"

logger.Info("Invoice created", "id", invoiceID, "amount", total)
logger.Error("Database error", "error", err, "operation", "create")
```

### 7. Utils Package (internal/app/utils/) **NEW**

**Purpose**: Shared utility functions

**Functions**:
- `Round2(f float64) float64` - Round to 2 decimal places for monetary calculations

**Benefits**:
- Eliminates code duplication
- Single source of truth for rounding logic
- Consistent monetary calculations across services

---

## Recent Improvements

### P0 - Critical Fixes (5 issues)

#### 1. Unit Tests Added ✅
- **Package**: `internal/validation`
- **Test Cases**: 19 comprehensive tests
- **Coverage**: Sanitization, validation, edge cases
- **Pass Rate**: 100%
- **File**: `internal/validation/validate_test.go`

#### 2. CI/CD Pipeline ✅
- **File**: `.github/workflows/ci.yml`
- **Jobs**: 4 parallel jobs
  - Backend tests with coverage and race detection
  - Frontend TypeScript checking and build
  - Security scanning (Gosec + npm audit)
  - Linting (golangci-lint)
- **Triggers**: Push to main/develop, PRs, all copilot branches
- **Features**: 
  - Test coverage reporting
  - Artifact uploads
  - SARIF security reports for GitHub Security tab
  - Caching for faster builds

#### 3. Duplicate Constants Eliminated ✅
- **Before**: Constants duplicated in `crud.go` and `internal/app/constants.go`
- **After**: Single source of truth in `internal/app/constants.go`
- **Files Modified**: 
  - `crud.go` - now imports from `internal/app`
  - `internal/app/constants.go` - centralized constants
- **Constants**: CourseType, BillingMode, InvoiceStatus, etc.

#### 4. XSS Protection Implemented ✅
- **Function**: `sanitizeInput()` in `crud.go`
- **Protection**: HTML escapes all user inputs
- **Applied To**: 
  - Student names, emails, phone numbers, notes
  - Course names, descriptions
  - Enrollment notes
- **Method**: `html.EscapeString()` for proper HTML entity encoding

#### 5. Path Traversal Vulnerability Fixed ✅
- **Function**: `OpenFile()` in `app.go`
- **Protection**: Validates files are within `~/LangSchool/` directory
- **Before**: No validation - could open arbitrary files
- **After**: Checks file path prefix, returns error for unauthorized access
- **Error Message**: "access denied: file must be within {allowedBase} directory"

### P1 - High Priority Fixes (6 issues)

#### 6. Frontend Linting Configuration ✅
- **ESLint**: 9.15.0 with TypeScript support
  - Config: `frontend/eslint.config.js`
  - Plugins: @typescript-eslint, react-hooks
  - Script: `npm run lint`
- **Prettier**: 3.3.3 for code formatting
  - Config: `frontend/.prettierrc`
  - Script: `npm run format`
  - Settings: 2-space indent, single quotes, trailing commas

#### 7. Structured Logging ✅
- **Package**: `internal/logger`
- **Implementation**: Go 1.21+ `log/slog` with JSON output
- **Functions**: Info, Error, Warn, Debug with field support
- **Purpose**: Replace `fmt.Printf` calls with structured logging
- **Benefits**: 
  - Structured log data for parsing
  - Log levels for filtering
  - JSON output for log aggregation tools

#### 8. Transaction Rollback Pattern Fixed ✅
- **File**: `internal/app/invoice/service.go`
- **Function**: `issueOne()`
- **Before**: `defer func() { _ = tx.Rollback() }()` - always executes
- **After**: Added `committed` flag to prevent rollback after successful commit
- **Pattern**:
  ```go
  committed := false
  defer func() {
      if !committed {
          _ = tx.Rollback()
      }
  }()
  // ... operations ...
  if err := tx.Commit(); err != nil {
      return "", err
  }
  committed = true
  ```

#### 9. Frontend Dependencies Updated ✅
- **TypeScript**: 4.6.4 → 5.7.2 (major upgrade)
- **Vite**: 3.0.7 → 6.0.1 (major upgrade)
- **React**: 18.2.0 → 18.3.1
- **Type Definitions**: All @types/* packages updated
- **New Tools**: ESLint 9.15.0, Prettier 3.3.3
- **File**: `frontend/package.json`

#### 10. Nil Pointer Risks Fixed ✅
- **Function**: `DevSeed()` in `app.go`
- **Issues Fixed**:
  - Added error handling for all database operations
  - Added nil checks before using entity IDs
  - Returns descriptive errors instead of silently failing
- **Example**:
  ```go
  if stud1 == nil {
      return fmt.Errorf("failed to create student 1")
  }
  ```

#### 11. Frontend Refactoring Guide ✅
- **Document**: `FRONTEND_REFACTORING.md` (7,000+ words)
- **Content**: 
  - Step-by-step 3-phase refactoring plan
  - Component structure proposal
  - Example implementations
  - Testing strategy
- **Example**: `frontend/src/components/StudentsTab.example.tsx`
- **Directories Created**: `components/`, `hooks/`, `types/`
- **Estimated Effort**: 7-10 hours for full implementation

### P2 - Medium Priority Fixes (9 issues)

#### 12. Code Duplication Eliminated ✅
- **Function**: `round2()` extracted to `internal/app/utils/math.go`
- **Name**: `Round2()`
- **Updated Files**:
  - `internal/app/invoice/service.go` - uses `utils.Round2()`
  - `internal/app/payment/service.go` - uses `utils.Round2()`
- **Removed**: Duplicate implementations
- **Documentation**: Comprehensive Godoc with usage examples

#### 13. Magic Numbers Documented ✅
- **File**: `internal/app/constants.go`
- **Constants Added**:
  - `SettingsSingletonID = 1` - Explains singleton pattern
  - `DirPermission = 0o755` - Explains Unix permissions (rwxr-xr-x)
- **Updated Files**:
  - `app.go` - uses `app.SettingsSingletonID`
  - `internal/pdf/invoice_pdf.go` - uses `app.DirPermission`

#### 14. Unused Files Removed ✅
- **Directory**: `fonts/` removed
- **Files Deleted**: 
  - `fonts/DejaVuSans.ttf` (2 files)
  - `fonts/DejaVuSans-Bold.ttf`
- **Reason**: Fonts loaded from `~/LangSchool/Fonts/` at runtime

#### 15. N+1 Query Problem Fixed ✅
- **File**: `internal/app/invoice/service.go`
- **Function**: Invoice listing
- **Before**: 1 query for invoices + N queries for line counts
- **After**: 2 queries total
  - 1 query for invoices
  - 1 batched query for all counts using GROUP BY
- **Performance**: O(N) reduced to O(1) with map lookup
- **Code**:
  ```go
  // Batch query with GROUP BY
  s.db.InvoiceLine.Query().
      Where(invoiceline.InvoiceIDIn(invoiceIDs...)).
      GroupBy(invoiceline.FieldInvoiceID).
      Aggregate(ent.Count()).
      Scan(ctx, &counts)
  ```

#### 16. Code Formatting Configuration ✅
- **Completed in P1**: ESLint + Prettier
- **Go Formatting**: Built-in `gofmt`
- **Scripts**: `npm run lint`, `npm run format`

#### 17. Error Context Added ✅
- **Pattern**: Error wrapping with `fmt.Errorf("...: %w", err)`
- **Function**: `getSettings()` in `app.go`
- **Benefit**: Better error tracing and debugging
- **Example**:
  ```go
  if err != nil {
      return nil, fmt.Errorf("failed to query settings: %w", err)
  }
  ```

#### 18. Personal Information Removed ✅
- **File**: `wails.json`
- **Changed**: Author name and email to generic values
- **Before**: Personal email address
- **After**: `info@example.com`

#### 19. Documentation Added ✅
- **Files**:
  - `crud.go` - Godoc for all CRUD functions
  - `internal/app/invoice/service.go` - Godoc for service methods
- **Functions Documented**:
  - Student operations: List, Get, Create, Update, SetActive, Delete
  - Course operations: Create
  - Enrollment operations: Create
  - Invoice operations: GenerateDrafts
- **Style**: Parameter descriptions, return values, behavior notes

#### 20. Dead Dependencies (Adjusted) ⚠️
- **Initial Assessment**: Both SQLite drivers appeared redundant
- **Correction**: Both drivers are needed:
  - `mattn/go-sqlite3` - CGo-based, main runtime driver
  - `ncruces/go-sqlite3` - Pure Go, for WASM/JS builds (with build tags)
- **Status**: Intentional dual-driver configuration maintained

### Build & Runtime Fixes

#### Fix 1: Missing Utils Import ✅
- **Commit**: `995c9b9`
- **File**: `internal/app/invoice/service.go`
- **Issue**: undefined: utils
- **Fix**: Added `"langschool/internal/app/utils"` import

#### Fix 2: Missing App Import ✅
- **Commit**: `7b66cba`
- **File**: `app.go`
- **Issue**: undefined: app (for `app.SettingsSingletonID`)
- **Fix**: Added `"langschool/internal/app"` import

#### Fix 3: SQLite Driver Issue ✅
- **Commit**: `6bd5c3d`
- **File**: `internal/infra/db.go`
- **Issue**: `sql: unknown driver "sqlite3"`
- **Fix**: Reverted to `mattn/go-sqlite3` driver
- **Reason**: CGo driver needed for runtime, ncruces for WASM builds

---

## Development Setup

### Prerequisites
- Go 1.21 or higher
- Node.js 18 or higher
- npm or yarn
- Wails CLI v2.10.2
- Git

### Installation Steps

1. **Clone Repository**:
   ```bash
   git clone https://github.com/Uvlazhnitel/Language-School-Billing.git
   cd Language-School-Billing
   ```

2. **Install Wails CLI**:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

3. **Install Go Dependencies**:
   ```bash
   go mod download
   ```

4. **Install Frontend Dependencies**:
   ```bash
   cd frontend
   npm install
   cd ..
   ```

5. **Run Development Server**:
   ```bash
   wails dev
   ```

6. **Build for Production**:
   ```bash
   wails build
   ```

### Development Workflow

1. **Run Linters**:
   ```bash
   # Go linting
   golangci-lint run
   
   # Frontend linting
   cd frontend
   npm run lint
   npm run format
   ```

2. **Run Tests**:
   ```bash
   # All tests
   go test ./...
   
   # Verbose output
   go test -v ./...
   
   # With coverage
   go test -cover ./...
   
   # HTML coverage report
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

3. **Run with Race Detection**:
   ```bash
   go test -race ./...
   ```

---

## Testing Strategy

### Unit Tests

**Location**: `internal/validation/validate_test.go`

**Test Cases**: 19 tests covering:
- Input sanitization (XSS protection)
- Empty field validation
- Price validation (non-negative)
- Discount percentage validation (0-100 range)
- Edge cases and boundary conditions

**Running Tests**:
```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/validation/...

# Verbose output
go test -v ./internal/validation/...

# Coverage report
go test -cover ./internal/validation/...
```

### Integration Tests

**Status**: Not yet implemented (recommended for future)

**Recommended Areas**:
- Database operations (CRUD)
- Invoice generation workflow
- Payment processing
- PDF generation

### Manual Testing

**See**: `TESTING.md` for comprehensive manual testing guide

**Key Areas to Test**:
- Student enrollment flow
- Attendance recording
- Invoice generation and issuing
- Payment processing
- PDF report generation
- File operations

---

## Security Features

### 1. XSS Protection ✅

**Implementation**: `sanitizeInput()` function in `crud.go`

**Method**: HTML entity escaping using `html.EscapeString()`

**Protected Fields**:
- Student: fullName, email, phone, note
- Course: name, description
- Enrollment: note

**Example**:
```go
func sanitizeInput(input string) string {
    trimmed := strings.TrimSpace(input)
    return html.EscapeString(trimmed)
}
```

### 2. Path Traversal Protection ✅

**Implementation**: `OpenFile()` function in `app.go`

**Validation**: Checks file path is within allowed directory

**Protected Directories**: `~/LangSchool/` and subdirectories

**Example**:
```go
allowedBase := filepath.Clean(a.dirs.Base)
cleanPath := filepath.Clean(path)

if !strings.HasPrefix(cleanPath, allowedBase) {
    return fmt.Errorf("access denied: file must be within %s directory", allowedBase)
}
```

### 3. Input Validation ✅

**Package**: `internal/validation`

**Validations**:
- Required fields (non-empty)
- Price validation (non-negative)
- Discount percentage (0-100 range)
- Type safety through Go's type system

### 4. Automated Security Scanning ✅

**Tool**: Gosec (Go Security Checker)

**Integration**: GitHub Actions CI/CD pipeline

**Runs**: On every push and pull request

**Reports**: SARIF format uploaded to GitHub Security tab

### 5. Dependency Security ✅

**Tool**: npm audit (for frontend dependencies)

**Integration**: GitHub Actions CI/CD pipeline

**Runs**: On every push and pull request

**Action**: Fails build on high/critical vulnerabilities

---

## API Documentation

### Backend API (Go → Frontend Bridge)

Wails automatically exposes Go methods to JavaScript. All public methods on the `App` struct are available to the frontend.

### Student Operations

#### StudentList()
```go
func (a *App) StudentList() ([]*StudentDTO, error)
```
Returns all students ordered by full name.

#### StudentGet(id int)
```go
func (a *App) StudentGet(id int) (*StudentDTO, error)
```
Gets a single student by ID with active enrollments.

#### StudentCreate(fullName, phone, email, note string)
```go
func (a *App) StudentCreate(fullName, phone, email, note string) (*StudentDTO, error)
```
Creates a new student with sanitized inputs.

#### StudentUpdate(id int, fullName, phone, email, note string)
```go
func (a *App) StudentUpdate(id int, fullName, phone, email, note string) (*StudentDTO, error)
```
Updates existing student with sanitized inputs.

#### StudentSetActive(id int, active bool)
```go
func (a *App) StudentSetActive(id int, active bool) error
```
Sets student active/inactive status.

#### StudentDelete(id int)
```go
func (a *App) StudentDelete(id int) error
```
Deletes a student (if no enrollments exist).

### Course Operations

#### CourseList()
```go
func (a *App) CourseList() ([]*CourseDTO, error)
```
Lists all courses ordered by name.

#### CourseCreate(name, desc string, type_, billingMode string, lessonPrice, subPrice float64)
```go
func (a *App) CourseCreate(name, desc, type_, billingMode string, 
                           lessonPrice, subPrice float64) (*CourseDTO, error)
```
Creates a new course with sanitized inputs.

### Enrollment Operations

#### EnrollmentCreate(studentID, courseID int, note string, discountPct float64)
```go
func (a *App) EnrollmentCreate(studentID, courseID int, note string, 
                               discountPct float64) (*EnrollmentDTO, error)
```
Creates new enrollment with sanitized note.

### Invoice Operations

#### InvoiceGenerateDrafts(year, month int)
```go
func (a *App) InvoiceGenerateDrafts(year, month int) (count int, err error)
```
Generates draft invoices for all active enrollments in specified period.

#### InvoiceIssue(id int)
```go
func (a *App) InvoiceIssue(id int) (number, pdfPath string, err error)
```
Issues a draft invoice, assigns number, generates PDF.

#### InvoiceList()
```go
func (a *App) InvoiceList() ([]*InvoiceListDTO, error)
```
Lists all invoices with line counts (optimized, no N+1 queries).

### Payment Operations

#### PaymentCreate(studentID int, amount float64, note string, invoiceIDs []int)
```go
func (a *App) PaymentCreate(studentID int, amount float64, note string, 
                            invoiceIDs []int) (receiptNumber, pdfPath string, err error)
```
Creates payment and generates receipt PDF.

### Settings Operations

#### SettingsGet()
```go
func (a *App) SettingsGet() (*SettingsDTO, error)
```
Gets application settings (singleton).

#### SettingsUpdate(...)
```go
func (a *App) SettingsUpdate(invoicePrefix, schoolName, schoolAddress, schoolPhone, 
                             schoolEmail, schoolWebsite string) (*SettingsDTO, error)
```
Updates application settings.

### File Operations

#### OpenFile(path string)
```go
func (a *App) OpenFile(path string) error
```
Opens file in system default application (with path traversal protection).

#### ShowInFolder(path string)
```go
func (a *App) ShowInFolder(path string) error
```
Shows file in system file explorer.

---

## Database Schema

### Entity Overview

```
┌──────────────┐         ┌──────────────┐
│   Student    │         │    Course    │
├──────────────┤         ├──────────────┤
│ ID           │         │ ID           │
│ FullName     │         │ Name         │
│ Phone        │         │ Description  │
│ Email        │         │ Type         │
│ Note         │         │ BillingMode  │
│ IsActive     │         │ LessonPrice  │
│ CreatedAt    │         │ SubPrice     │
└──────┬───────┘         └──────┬───────┘
       │                        │
       │    ┌──────────────┐    │
       └────│  Enrollment  │────┘
            ├──────────────┤
            │ ID           │
            │ StudentID    │◄─────┐
            │ CourseID     │      │
            │ DiscountPct  │      │
            │ Note         │      │
            │ IsActive     │      │
            │ StartDate    │      │
            └──────┬───────┘      │
                   │              │
                   │              │
         ┌─────────▼─────────┐    │
         │    Invoice        │    │
         ├───────────────────┤    │
         │ ID                │    │
         │ StudentID         │────┘
         │ Year              │
         │ Month             │
         │ Status            │
         │ Number            │
         │ IssuedAt          │
         │ PDFPath           │
         └─────────┬─────────┘
                   │
         ┌─────────▼─────────┐
         │  InvoiceLine      │
         ├───────────────────┤
         │ ID                │
         │ InvoiceID         │
         │ EnrollmentID      │
         │ Description       │
         │ Quantity          │
         │ UnitPrice         │
         │ Amount            │
         └───────────────────┘

         ┌───────────────────┐
         │    Payment        │
         ├───────────────────┤
         │ ID                │
         │ StudentID         │
         │ Amount            │
         │ Date              │
         │ Note              │
         │ ReceiptNumber     │
         │ PDFPath           │
         └─────────┬─────────┘
                   │
         ┌─────────▼─────────┐
         │ PaymentAllocation │
         ├───────────────────┤
         │ ID                │
         │ PaymentID         │
         │ InvoiceID         │
         │ Amount            │
         └───────────────────┘

┌──────────────────┐
│    Settings      │
├──────────────────┤
│ ID (Singleton=1) │
│ InvoicePrefix    │
│ SchoolName       │
│ SchoolAddress    │
│ SchoolPhone      │
│ SchoolEmail      │
│ SchoolWebsite    │
└──────────────────┘

┌───────────────────┐
│ AttendanceMonth   │
├───────────────────┤
│ ID                │
│ EnrollmentID      │
│ Year              │
│ Month             │
│ DaysPresent       │
└───────────────────┘

┌───────────────────┐
│  PriceOverride    │
├───────────────────┤
│ ID                │
│ EnrollmentID      │
│ Year              │
│ Month             │
│ LessonPrice       │
│ SubPrice          │
└───────────────────┘
```

### Key Relationships

- **Student** → **Enrollment** (one-to-many)
- **Course** → **Enrollment** (one-to-many)
- **Student** → **Invoice** (one-to-many)
- **Enrollment** → **InvoiceLine** (one-to-many)
- **Invoice** → **InvoiceLine** (one-to-many)
- **Student** → **Payment** (one-to-many)
- **Payment** → **PaymentAllocation** (one-to-many)
- **Invoice** → **PaymentAllocation** (one-to-many)
- **Enrollment** → **AttendanceMonth** (one-to-many)
- **Enrollment** → **PriceOverride** (one-to-many)

### Database Files

**Location**: `~/LangSchool/data/app.sqlite`

**Driver**: `mattn/go-sqlite3` (CGo-based)

**Connection String**: 
```
file:path?_fk=1&_busy_timeout=5000&cache=shared&mode=rwc
```

**Features Enabled**:
- Foreign key constraints (`_fk=1`)
- Busy timeout (5000ms)
- Shared cache
- Read-write-create mode

---

## Build and Deployment

### Development Build

```bash
wails dev
```

**Features**:
- Hot reload for frontend changes
- Live Go code reload
- Developer tools enabled
- Debug mode

### Production Build

```bash
wails build
```

**Output**:
- macOS: `build/bin/Language-School-Billing.app`
- Windows: `build/bin/Language-School-Billing.exe`
- Linux: `build/bin/Language-School-Billing`

**Build Options**:
```bash
# Clean build
wails build -clean

# Skip frontend build
wails build -skipbindings

# Custom output directory
wails build -o custom/path/
```

### Platform-Specific Notes

#### macOS
- Requires Xcode Command Line Tools
- Builds as `.app` bundle
- Codesigning required for distribution
- **Note**: Development builds use private APIs (not AppStore compatible)

#### Windows
- Requires GCC (via MinGW-w64 or TDM-GCC)
- Builds as `.exe`
- Can create installer with NSIS

#### Linux
- Requires GCC and webkit2gtk
- Builds as executable
- Desktop file can be created for menu integration

### Cross-Compilation

**Not Recommended**: Wails apps with CGo dependencies (like SQLite) should be built on the target platform.

**Alternative**: Use CI/CD with runners for each platform.

---

## CI/CD Pipeline

### GitHub Actions Workflow

**File**: `.github/workflows/ci.yml`

**Triggers**:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`
- Push to any `copilot/*` branch

### Pipeline Jobs

#### 1. Backend Job
```yaml
runs-on: ubuntu-latest
steps:
  - Go setup (1.21)
  - Cache Go modules
  - Install dependencies
  - Run tests with coverage
  - Run tests with race detection
  - Run go vet
  - Upload coverage artifact
```

#### 2. Frontend Job
```yaml
runs-on: ubuntu-latest
steps:
  - Node setup (18)
  - Cache npm packages
  - Install dependencies
  - TypeScript type checking
  - Build frontend
  - Upload build artifact
```

#### 3. Security Job
```yaml
runs-on: ubuntu-latest
steps:
  - Run Gosec security scanner
  - Upload SARIF report
  - Run npm audit
  - Fail on high/critical vulnerabilities
```

#### 4. Lint Job
```yaml
runs-on: ubuntu-latest
steps:
  - Install golangci-lint
  - Cache lint results
  - Run golangci-lint with config
  - Report issues
```

### Status Checks

All jobs must pass before PR can be merged:
- ✅ Backend tests and coverage
- ✅ Frontend build and type checking
- ✅ Security scans (no high/critical issues)
- ✅ Linting passes

---

## Additional Documentation Files

### PROJECT_ANALYSIS.md
- Complete code analysis (920 lines)
- 20 prioritized issues (P0/P1/P2)
- Exact file locations and code examples
- 3-phase refactoring plan

### TESTING.md
- Manual testing guide (240 lines)
- Test commands and examples
- Coverage report generation
- Race detection
- Troubleshooting tips

### FRONTEND_REFACTORING.md
- Comprehensive refactoring guide (300 lines)
- 3-phase approach
- Component structure proposal
- Example implementations
- Migration checklist
- Estimated effort: 7-10 hours

### P0_FIXES_SUMMARY.md
- Detailed P0 fixes documentation (300 lines)
- Before/after code examples
- Security improvements
- Test results

---

## Development Best Practices

### Code Style

#### Go
- Use `gofmt` for formatting
- Follow Go standard conventions
- Use Godoc comments for exported functions
- Keep functions focused and small
- Use descriptive variable names

#### TypeScript
- Use ESLint and Prettier
- 2-space indentation
- Single quotes for strings
- Trailing commas
- Explicit types for function parameters and returns

### Security

1. **Always sanitize user inputs** using `sanitizeInput()`
2. **Validate file paths** before file operations
3. **Use error wrapping** for better context
4. **Run security scans** before deploying
5. **Keep dependencies updated** regularly

### Testing

1. **Write tests for new features**
2. **Aim for high coverage** (60%+ minimum)
3. **Test edge cases** and boundary conditions
4. **Use table-driven tests** for multiple scenarios
5. **Run race detector** for concurrent code

### Git Workflow

1. **Branch naming**: 
   - Features: `feature/description`
   - Fixes: `fix/description`
   - Copilot: `copilot/description`

2. **Commit messages**:
   - Clear, descriptive messages
   - Reference issue numbers when applicable
   - Use conventional commit format

3. **Pull Requests**:
   - All CI checks must pass
   - Code review required
   - Update documentation as needed

---

## Troubleshooting

### Common Issues

#### 1. Build Errors

**Issue**: `undefined: utils`
**Solution**: Check imports, add `"langschool/internal/app/utils"`

**Issue**: `sql: unknown driver "sqlite3"`
**Solution**: Ensure `mattn/go-sqlite3` is imported in `internal/infra/db.go`

#### 2. Runtime Errors

**Issue**: Database locked
**Solution**: Check busy timeout in connection string, ensure proper transaction handling

**Issue**: Path not found
**Solution**: Verify `~/LangSchool/` directory exists, check path permissions

#### 3. Frontend Issues

**Issue**: TypeScript errors after dependency update
**Solution**: Run `npm install`, check type definitions, update `tsconfig.json` if needed

**Issue**: Vite build fails
**Solution**: Clear `node_modules` and reinstall, check Vite compatibility

### Debug Mode

Enable verbose logging:
```go
import "langschool/internal/logger"

logger.SetLevel(logger.DebugLevel)
logger.Debug("Detailed message", "key", value)
```

### Performance Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof
go tool pprof mem.prof
```

---

## Future Improvements

### Recommended Next Steps

1. **Integration Tests**
   - Add database integration tests
   - Test complete workflows (enrollment → invoice → payment)
   - Mock file system operations

2. **Frontend Refactoring**
   - Implement component splitting per `FRONTEND_REFACTORING.md`
   - Extract custom hooks
   - Add React Testing Library tests

3. **Performance Optimization**
   - Add database indexes for frequently queried fields
   - Implement pagination for large lists
   - Cache frequently accessed data

4. **Feature Enhancements**
   - Email invoice/receipt sending
   - Recurring billing automation
   - Reporting dashboard
   - Multi-language support

5. **DevOps**
   - Automated release builds for all platforms
   - Version numbering automation
   - Changelog generation
   - Docker support for development environment

---

## Contact & Support

**Repository**: https://github.com/Uvlazhnitel/Language-School-Billing

**Issues**: Report bugs or request features via GitHub Issues

**Documentation**: All documentation files in repository root

**License**: [Check repository for license information]

---

## Appendix

### Glossary

- **Wails**: Go + Web frontend framework for desktop apps
- **Ent**: Facebook's entity framework for Go (ORM)
- **XSS**: Cross-Site Scripting vulnerability
- **N+1 Query**: Performance anti-pattern with excessive database queries
- **CI/CD**: Continuous Integration/Continuous Deployment
- **SARIF**: Static Analysis Results Interchange Format
- **DTO**: Data Transfer Object

### Useful Commands

```bash
# Development
wails dev                    # Run dev server
wails build                  # Production build
wails doctor                 # Check setup

# Testing
go test ./...               # All tests
go test -v -cover ./...     # Verbose with coverage
go test -race ./...         # Race detection

# Linting
golangci-lint run           # Go linting
cd frontend && npm run lint # Frontend linting
cd frontend && npm run format # Frontend formatting

# Database
sqlite3 ~/LangSchool/data/app.sqlite  # Open DB
.schema                     # Show schema
.tables                     # List tables

# Git
git log --oneline --graph   # View commit history
git diff --stat             # View changes summary
```

### References

- [Wails Documentation](https://wails.io/)
- [Ent Documentation](https://entgo.io/)
- [Go Documentation](https://go.dev/doc/)
- [React Documentation](https://react.dev/)
- [TypeScript Documentation](https://www.typescriptlang.org/)
- [Vite Documentation](https://vitejs.dev/)

---

**Document Version**: 1.0  
**Last Updated**: December 25, 2025  
**Author**: Copilot (based on comprehensive repository analysis and improvements)
