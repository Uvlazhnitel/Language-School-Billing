# System Architecture and Detailed Design

**Language School Billing System**

---

## 1. Architecture Overview

### 1.1 Architectural Style

The system employs a **Layered Architecture** pattern with clear separation of concerns across four primary layers:

1. **Presentation Layer** (Frontend - React/TypeScript)
2. **Application Layer** (Backend - Go controllers)
3. **Business Logic Layer** (Services)
4. **Data Access Layer** (ent ORM + SQLite)

### 1.2 Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                  PRESENTATION LAYER                         │
│                   (React + TypeScript)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ Students │  │  Courses │  │Attendance│  │ Invoices │  │
│  │   Tab    │  │    Tab   │  │   Tab    │  │   Tab    │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│        ▲              ▲              ▲              ▲       │
│        └──────────────┴──────────────┴──────────────┘       │
│                           │                                  │
│                  frontend/src/lib/* (API Wrappers)          │
└───────────────────────────┼─────────────────────────────────┘
                            │ Wails Bindings (Type-Safe)
┌───────────────────────────▼─────────────────────────────────┐
│               APPLICATION LAYER (Go)                        │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  app.go - Application Controller                    │   │
│  │  - Lifecycle management (startup/shutdown)          │   │
│  │  - Service initialization                           │   │
│  │  - Directory setup                                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  crud.go - CRUD Operations                          │   │
│  │  - Student, Course, Enrollment CRUD                 │   │
│  │  - Input validation                                 │   │
│  │  - DTO conversion                                   │   │
│  └─────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│            BUSINESS LOGIC LAYER                             │
│              (internal/app/*)                               │
│                                                             │
│  ┌──────────────────────┐  ┌───────────────────────────┐   │
│  │ Attendance Service   │  │   Invoice Service         │   │
│  │ - Monthly tracking   │  │   - Draft generation      │   │
│  │ - Lock/unlock        │  │   - Sequential numbering  │   │
│  │ - Bulk updates       │  │   - PDF coordination      │   │
│  └──────────────────────┘  └───────────────────────────┘   │
│                                                             │
│  ┌──────────────────────┐  ┌───────────────────────────┐   │
│  │  Payment Service     │  │   PDF Generation          │   │
│  │  - Payment recording │  │   - gofpdf integration    │   │
│  │  - Balance calc      │  │   - Cyrillic fonts        │   │
│  │  - Debtor tracking   │  │   - File organization     │   │
│  └──────────────────────┘  └───────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Supporting Services                                │   │
│  │  - Validation (sanitization, checks)                │   │
│  │  - Paths (directory management)                     │   │
│  │  - Constants (shared enums)                         │   │
│  └─────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│              DATA ACCESS LAYER                              │
│                 (ent ORM)                                   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  ent/schema/* - Entity Definitions                  │   │
│  │  - Student, Course, Enrollment                      │   │
│  │  - Invoice, InvoiceLine, Payment                    │   │
│  │  - AttendanceMonth, Settings, PriceOverride         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  ent/* - Generated Code                             │   │
│  │  - Type-safe queries                                │   │
│  │  - Automatic migrations                             │   │
│  │  - Relationship handling                            │   │
│  └─────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────┐
│                  DATA STORAGE LAYER                         │
│                      (SQLite)                               │
│                                                             │
│              ~/LangSchool/Data/app.sqlite                   │
│                                                             │
│  - Zero-configuration database                              │
│  - ACID transactions                                        │
│  - Single-file portability                                  │
│  - Cross-platform compatibility                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Component Design

### 2.1 Presentation Layer Components

#### 2.1.1 Frontend Structure

**Location**: `frontend/src/`

**Main Components**:

- **App.tsx**: Root component
  - Tab-based navigation
  - State management with React hooks
  - Error boundary handling

- **lib/students.ts**: Student API wrapper
  - `listStudents()`
  - `createStudent(input)`
  - `updateStudent(id, input)`
  - `deleteStudent(id)`

- **lib/courses.ts**: Course API wrapper
  - Similar CRUD operations for courses

- **lib/enrollments.ts**: Enrollment API wrapper
  - Enrollment management operations

- **lib/attendance.ts**: Attendance API wrapper
  - `getAttendance(year, month)`
  - `updateAttendance(id, lessonsCount)`
  - `lockMonth(year, month)`

- **lib/invoices.ts**: Invoice API wrapper
  - `generateDrafts(year, month)`
  - `issueInvoice(id)`
  - `cancelInvoice(id)`

- **lib/payments.ts**: Payment API wrapper
  - `createPayment(input)`
  - `getBalance(studentId)`
  - `getDebtors()`

- **lib/constants.ts**: Type-safe constants
  - Enums mirroring backend constants
  - Type definitions for DTOs

#### 2.1.2 Wails Bindings

**Location**: `frontend/src/wailsjs/`

**Auto-generated** by Wails CLI:
- Type-safe TypeScript interfaces
- Direct Go method invocation
- Automatic serialization/deserialization

### 2.2 Application Layer Components

#### 2.2.1 Main Application (app.go)

**Responsibilities**:
- Application lifecycle management
- Service initialization and coordination
- Directory structure setup
- Database connection management

**Key Methods**:
```go
func (a *App) startup(ctx context.Context)
func (a *App) shutdown(ctx context.Context)
func (a *App) domReady(ctx context.Context)
```

**Services Managed**:
- Attendance Service
- Invoice Service
- Payment Service
- Database client (ent)

#### 2.2.2 CRUD Controller (crud.go)

**Responsibilities**:
- High-level CRUD operations
- Input validation coordination
- DTO conversion
- Error handling and messaging

**Key Method Patterns**:
```go
func (a *App) ListStudents() ([]StudentDTO, error)
func (a *App) CreateStudent(input StudentInput) (StudentDTO, error)
func (a *App) UpdateStudent(id int, input StudentInput) (StudentDTO, error)
func (a *App) DeleteStudent(id int) error
```

**Validation Helpers**:
- `sanitizeInput(s string) string`
- `validateNonEmpty(fieldName, value string) error`
- `validatePrices(lesson, subscription float64) error`
- `validateDiscountPct(discount float64) error`

**DTO Converters**:
- `studentToDTO(student *ent.Student) StudentDTO`
- `courseToDTO(course *ent.Course) CourseDTO`
- `enrollmentToDTO(enr *ent.Enrollment) EnrollmentDTO`

### 2.3 Business Logic Layer

#### 2.3.1 Attendance Service

**Location**: `internal/app/attendance/service.go`

**Responsibilities**:
- Monthly attendance tracking
- Lock/unlock month functionality
- Bulk attendance updates

**Key Methods**:
```go
func (s *Service) GetAttendanceForMonth(ctx context.Context, year, month int) ([]AttendanceDTO, error)
func (s *Service) UpdateLessonsCount(ctx context.Context, id, count int) error
func (s *Service) LockMonth(ctx context.Context, year, month int) error
func (s *Service) UnlockMonth(ctx context.Context, year, month int) error
```

**Business Rules**:
- Cannot edit locked months
- Attendance auto-created for active enrollments
- Lessons count must be >= 0

#### 2.3.2 Invoice Service

**Location**: `internal/app/invoice/service.go`

**Responsibilities**:
- Invoice draft generation
- Sequential numbering
- Invoice issuing
- PDF generation coordination
- Status management

**Key Methods**:
```go
func (s *Service) GenerateDrafts(ctx context.Context, year, month int) ([]InvoiceDTO, error)
func (s *Service) Issue(ctx context.Context, invoiceID int) (InvoiceDTO, error)
func (s *Service) Cancel(ctx context.Context, invoiceID int) error
```

**Draft Generation Algorithm**:
1. Query active students
2. For each student, query enrollments
3. For per-lesson enrollments:
   - Get attendance for month
   - Calculate: lessons * lesson_price * (1 - discount/100)
4. For subscription enrollments:
   - Calculate: subscription_price * (1 - discount/100)
5. Create invoice with calculated lines
6. Sum lines to get total

**Issuing Algorithm**:
1. Validate invoice status is draft
2. Get settings (prefix, next sequence)
3. Format number: PREFIX-YYYYMM-SEQ
4. Update invoice: status=issued, number=formatted, issued_at=now
5. Increment settings.next_seq
6. Generate PDF
7. Return updated invoice

#### 2.3.3 Payment Service

**Location**: `internal/app/payment/service.go`

**Responsibilities**:
- Payment recording
- Balance calculation
- Automatic invoice status updates
- Debtor list generation

**Key Methods**:
```go
func (s *Service) Create(ctx context.Context, input PaymentInput) (PaymentDTO, error)
func (s *Service) GetBalance(ctx context.Context, studentID int) (float64, error)
func (s *Service) GetDebtors(ctx context.Context) ([]DebtorDTO, error)
```

**Balance Calculation**:
```
balance = sum(invoices.total_amount WHERE status IN (issued, paid)) 
        - sum(payments.amount)
```

**Auto-status Update**:
- When payment linked to invoice
- Sum all payments for invoice
- If sum >= invoice.total_amount: status = paid

#### 2.3.4 PDF Generation

**Location**: `internal/pdf/invoice_pdf.go`

**Responsibilities**:
- PDF document creation
- Font loading (DejaVu Sans for Cyrillic)
- Invoice formatting
- File system organization

**PDF Structure**:
1. Header with organization details
2. Invoice metadata (number, date, student)
3. Table with invoice lines
4. Total amount
5. Footer (optional)

**Font Handling**:
- Load DejaVuSans.ttf from ~/LangSchool/Fonts/
- Fallback to default if not found (no Cyrillic)
- Bold variant for headers

**File Path**: `~/LangSchool/Invoices/YYYY/MM/NUMBER.pdf`

### 2.4 Data Access Layer

#### 2.4.1 ent ORM Integration

**Schema Definition**: `ent/schema/*.go`

**Code Generation**:
```bash
go generate ./ent
```

Generates:
- Query builders
- Mutation builders
- Type-safe predicates
- Automatic migrations
- Edge loading

**Query Examples**:
```go
// Find student by ID with enrollments
student, err := client.Student.
    Query().
    Where(student.IDEQ(id)).
    WithEnrollments().
    Only(ctx)

// Query invoices for month
invoices, err := client.Invoice.
    Query().
    Where(
        invoice.YearEQ(year),
        invoice.MonthEQ(month),
        invoice.StatusEQ("draft"),
    ).
    WithStudent().
    WithLines().
    All(ctx)
```

#### 2.4.2 Database Schema

See THESIS.md Section 3.3 for complete entity definitions.

**Key Relationships**:

```
Student 1───M Enrollment M───1 Course
   │            │
   │            └── M InvoiceLine
   │
   ├─── M Invoice 1───M InvoiceLine
   │       │
   │       └─── M Payment
   │
   └─── M Payment
```

**Indexes**:
- Student.id (primary)
- Course.id (primary)
- Invoice.student_id (foreign key index)
- AttendanceMonth (student_id, course_id, year, month) unique

---

## 3. Design Patterns

### 3.1 Service Layer Pattern

**Purpose**: Encapsulate business logic

**Implementation**:
- Each service struct holds database client
- Services coordinate multiple entity operations
- Services enforce business rules

**Benefits**:
- Clear separation from CRUD layer
- Reusable business logic
- Easier testing

### 3.2 Data Transfer Object (DTO) Pattern

**Purpose**: Decouple entities from API responses

**Implementation**:
- Separate structs for API communication
- Converter functions: entity → DTO
- JSON tags for serialization

**Benefits**:
- Control over API surface
- Simplify frontend data structures
- Version API independently from database

### 3.3 Repository Pattern

**Purpose**: Abstract data access

**Implementation**:
- ent ORM provides repository abstraction
- Generated query builders
- Type-safe predicates

**Benefits**:
- Database-agnostic queries
- No SQL injection risk
- Compile-time query validation

### 3.4 Singleton Pattern

**Purpose**: Ensure single settings instance

**Implementation**:
- Settings table with singleton_id = 1
- Unique constraint on singleton_id
- Always query WHERE singleton_id = 1

**Benefits**:
- Single source of configuration
- No duplicate settings
- Predictable behavior

### 3.5 Factory Pattern

**Purpose**: Service initialization

**Implementation**:
- NewApp() creates application
- app.startup() initializes services
- Each service has New(client) constructor

**Benefits**:
- Centralized initialization
- Dependency injection
- Easy to test

---

## 4. Data Flow

### 4.1 Create Student Flow

```
User fills form → React onChange updates state
                ↓
User clicks Save → API wrapper calls CreateStudent()
                ↓
Wails binding serializes and calls Go method
                ↓
crud.go: App.CreateStudent(input)
  - Validates input (non-empty name)
  - Sanitizes text fields
  - Creates ent mutation
  - Saves to database
  - Converts entity to DTO
  - Returns DTO
                ↓
Wails binding serializes response
                ↓
React receives DTO → Updates UI state → Renders updated list
```

### 4.2 Generate and Issue Invoice Flow

```
User selects month → Clicks "Generate Drafts"
                ↓
invoices.ts: generateDrafts(year, month)
                ↓
app.go: App.GenerateInvoiceDrafts(year, month)
                ↓
invoice.Service.GenerateDrafts(ctx, year, month)
  - Query active students
  - For each student:
    - Query enrollments
    - Get attendance (per-lesson)
    - Calculate amounts with discounts
    - Create invoice entity
    - Create invoice line entities
  - Return draft DTOs
                ↓
React receives drafts → Displays in UI
                ↓
User clicks "Issue" on draft
                ↓
invoices.ts: issueInvoice(id)
                ↓
invoice.Service.Issue(ctx, id)
  - Load invoice (validate draft status)
  - Load settings
  - Generate number: PREFIX-YYYYMM-SEQ
  - Update invoice: status=issued, number, issued_at
  - Increment settings.next_seq
  - Generate PDF:
    - Create PDF with gofpdf
    - Add organization header
    - Add invoice metadata
    - Add line items table
    - Save to ~/LangSchool/Invoices/YYYY/MM/NUMBER.pdf
  - Return updated DTO
                ↓
React receives updated invoice → Shows success message
```

### 4.3 Record Payment Flow

```
User fills payment form → Clicks "Save"
                ↓
payments.ts: createPayment(input)
                ↓
payment.Service.Create(ctx, input)
  - Validate amount > 0
  - Create payment entity
  - If invoice_id provided:
    - Load invoice
    - Query all payments for invoice
    - Calculate total paid
    - If total >= invoice.total_amount:
      - Update invoice.status = "paid"
  - Return payment DTO
                ↓
React receives payment → Updates UI → Shows balance
```

---

## 5. Security Architecture

### 5.1 Input Validation Layer

**Location**: `internal/validation/validate.go`

**Functions**:
- `SanitizeInput(s string) string`: HTML escape
- `ValidateNonEmpty(field, value string) error`: Required field check
- `ValidatePrices(lesson, sub float64) error`: Non-negative check
- `ValidateDiscountPct(pct float64) error`: Range check (0-100)

**Applied At**:
- All CRUD operations in crud.go
- Before database persistence
- After user input, before processing

### 5.2 XSS Prevention

**Mechanism**: HTML escaping via `html.EscapeString()`

**Coverage**:
- All text inputs (names, notes, descriptions)
- Applied automatically in SanitizeInput()
- 100% test coverage

### 5.3 SQL Injection Prevention

**Mechanism**: ent ORM generates parameterized queries

**Example**:
```go
// User input
studentName := "John'; DROP TABLE students; --"

// ent generates safe query
student, err := client.Student.
    Query().
    Where(student.FullNameEQ(studentName)). // Parameterized
    Only(ctx)

// SQL executed: SELECT * FROM students WHERE full_name = ?
// Parameter: ["John'; DROP TABLE students; --"]
```

### 5.4 Access Control

**Single-user Application**:
- No authentication required
- File system permissions protect database
- No network exposure

**Business Rule Enforcement**:
- Cannot delete students with enrollments
- Cannot modify issued invoices
- Cannot edit locked months

---

## 6. Performance Considerations

### 6.1 Database Optimization

**Indexes**:
- Primary keys on all tables
- Foreign key indexes automatically created by ent
- Composite unique index on (student_id, course_id, year, month) for attendance

**Query Optimization**:
- Use WithX() edge loading to avoid N+1 queries
- Limit results with pagination where applicable
- Use transactions for multi-step operations

### 6.2 Caching Strategy

**Current**: No caching (not needed for single-user app)

**Future**: Consider caching for:
- Settings (rarely change)
- Student/course lists (cache invalidation on CRUD)

### 6.3 PDF Generation

**Performance**:
- Asynchronous generation (doesn't block UI)
- File system I/O optimized with buffering
- Fonts loaded once per generation

**Future Optimization**:
- Pre-load fonts at startup
- Batch PDF generation
- Progress indicators

---

## 7. Scalability

### 7.1 Current Limits

**Designed For**:
- ~1,000 students
- ~100 courses
- ~10,000 enrollments
- ~100,000 invoices

**Tested With**:
- ~100 students (manual testing)
- All operations < 1 second response time

### 7.2 Scaling Considerations

**If Needed**:
- Add pagination to list views
- Implement lazy loading
- Add database indexes for common queries
- Consider SQLite performance tuning

**SQLite Limits** (sufficient for use case):
- Database size: 281 TB max
- Rows per table: 2^64
- Queries/second: Thousands

---

## 8. Error Handling

### 8.1 Error Propagation

```
Database Error
    ↓
Service Layer (log, wrap with context)
    ↓
Application Layer (convert to user message)
    ↓
Wails Binding (serialize error)
    ↓
Frontend (display alert/message)
```

### 8.2 Error Types

**Validation Errors**:
- User-facing messages
- Indicate how to fix
- Example: "Student name cannot be empty"

**Business Rule Errors**:
- Explain constraint violation
- Example: "Cannot delete student with active enrollments"

**System Errors**:
- Generic user message
- Detailed log for debugging
- Example: "Database error. Please try again."

---

## 9. Deployment Architecture

### 9.1 Development Environment

```
Developer Machine
    ↓
    ├─ Go toolchain (1.24+)
    ├─ Node.js (18+)
    ├─ Wails CLI
    └─ Code editor

Development Mode:
    ├─ `wails dev`
    ├─ Hot reload (frontend + backend)
    └─ DevTools enabled
```

### 9.2 Production Build

```
Build Machine
    ↓
    ├─ Generate ent code
    ├─ Build frontend (Vite)
    ├─ Compile Go + embed assets
    └─ Package executable

Executable:
    ├─ Platform-specific binary
    ├─ Embedded frontend assets
    └─ No external dependencies
```

### 9.3 Runtime Environment

```
User Machine
    ↓
Application Directories:
    ~/LangSchool/
        ├─ Data/
        │   └─ app.sqlite (created on first run)
        ├─ Fonts/
        │   ├─ DejaVuSans.ttf (user provides)
        │   └─ DejaVuSans-Bold.ttf (user provides)
        ├─ Invoices/
        │   └─ YYYY/MM/*.pdf (created by app)
        ├─ Backups/ (reserved)
        └─ Exports/ (reserved)
```

---

## 10. Technology Stack Details

### 10.1 Backend Stack

- **Go 1.24.0**: Core language
- **Wails v2.10.2**: Desktop framework
- **ent v0.14.5**: ORM
- **go-sqlite3**: SQLite driver
- **gofpdf**: PDF generation

### 10.2 Frontend Stack

- **React 18.3**: UI library
- **TypeScript 5.7**: Type safety
- **Vite 6.0**: Build tool
- **ESLint**: Linting
- **Prettier**: Formatting

### 10.3 Development Tools

- **golangci-lint**: Go linting
- **Git**: Version control
- **npm**: Frontend package management
- **go mod**: Go dependency management

---

**Document Version**: 1.0  
**Last Updated**: December 2024  
**Status**: Complete
