# Application Flow Guide

**Purpose**: Understand exactly how all components in the Language School Billing system connect and work together.

**Read this to**: Trace requests through the system, understand data flow, and see how frontend, backend, and database interact.

---

## Table of Contents

1. [Quick Overview (5 min)](#1-quick-overview)
2. [Request Lifecycle](#2-request-lifecycle)
3. [Key Workflows (Step-by-Step)](#3-key-workflows)
4. [Component Connection Map](#4-component-connection-map)
5. [How Wails Binding Works](#5-how-wails-binding-works)
6. [How Database Layer Works](#6-how-database-layer-works)
7. [Common Scenarios Traced](#7-common-scenarios-traced)
8. [Troubleshooting: Tracing Requests](#8-troubleshooting-tracing-requests)

---

## 1. Quick Overview

### The 4 Layers

```
┌─────────────────────────────────────────┐
│  Layer 1: React Frontend (TypeScript)   │
│  - User Interface                       │
│  - State Management                     │
│  - User Actions                         │
└─────────────────────────────────────────┘
                  ↓ ↑
         Wails Bridge (automatic)
                  ↓ ↑
┌─────────────────────────────────────────┐
│  Layer 2: Go Backend (main package)     │
│  - CRUD operations (crud.go)            │
│  - Application logic (app.go)           │
│  - Input validation & sanitization      │
└─────────────────────────────────────────┘
                  ↓ ↑
┌─────────────────────────────────────────┐
│  Layer 3: Business Services              │
│  - Invoice service                      │
│  - Payment service                      │
│  - PDF generation                       │
└─────────────────────────────────────────┘
                  ↓ ↑
┌─────────────────────────────────────────┐
│  Layer 4: Data Layer (Ent ORM)          │
│  - Database queries                     │
│  - SQLite storage                       │
│  - Data validation                      │
└─────────────────────────────────────────┘
```

### How They Connect

1. **Frontend → Backend**: Wails automatically generates TypeScript bindings for Go functions
2. **Backend → Services**: Direct Go function calls
3. **Services → Database**: Ent ORM provides type-safe database access
4. **Response flows back**: Each layer returns data to the previous layer

---

## 2. Request Lifecycle

### Example: User Creates a Student

```
┌─────────────────────────────────────────────────────────┐
│ 1. USER ACTION                                           │
│    User fills form and clicks "Add Student" button      │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 2. REACT COMPONENT (frontend/src/App.tsx)               │
│    - validateStudent() checks inputs                    │
│    - StudentCreate() called                             │
│    - Shows loading state                                │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 3. WAILS BRIDGE (auto-generated)                        │
│    - Converts TypeScript call to Go call                │
│    - Handles IPC communication                          │
│    - Manages type conversion                            │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 4. GO BACKEND (crud.go: StudentCreate)                  │
│    a) Sanitize all inputs (HTML escape)                 │
│    b) Validate non-empty fields                         │
│    c) Call Ent ORM to create student                    │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 5. ENT ORM (ent/student/create.go - generated)          │
│    - Build SQL INSERT statement                         │
│    - Execute query with parameters                      │
│    - Return created student entity                      │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 6. SQLITE DATABASE (~/LangSchool/data/app.sqlite)       │
│    - Insert record into students table                  │
│    - Auto-generate ID                                   │
│    - Return new record                                  │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 7. RESPONSE FLOWS BACK                                  │
│    Database → Ent → Backend → Wails → Frontend          │
│    - StudentDTO returned with all fields                │
│    - ID is now populated                                │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│ 8. FRONTEND UPDATES                                     │
│    - Add student to state array                         │
│    - UI re-renders with new student                     │
│    - Show success notification                          │
│    - Clear form                                         │
└─────────────────────────────────────────────────────────┘
```

**Time**: Entire flow takes ~10-50ms

---

## 3. Key Workflows

### 3.1. Student Creation (Detailed)

**File Journey:**

1. **User Input** → `frontend/src/App.tsx` (lines 200-250)
   - Form fields: fullName, phone, email, note
   - Validation: `validateStudent()` checks non-empty name
   
2. **Frontend Call** → `StudentCreate(fullName, phone, email, note)`
   - TypeScript function from `frontend/wailsjs/go/main/App.js`
   - Auto-generated by Wails

3. **Backend Receives** → `crud.go` (line 50)
   ```go
   func (a *App) StudentCreate(fullName, phone, email, note string) (*StudentDTO, error)
   ```

4. **Sanitization** → `crud.go` (line 52-55)
   ```go
   fullName = sanitizeInput(fullName)
   phone = sanitizeInput(phone)
   email = sanitizeInput(email)
   note = sanitizeInput(note)
   ```
   - Trims whitespace
   - HTML-escapes special characters
   - Prevents XSS attacks

5. **Database Create** → `crud.go` (line 62-67)
   ```go
   s, err := a.db.Ent.Student.Create().
       SetFullName(fullName).
       SetPhone(phone).
       SetEmail(email).
       SetNote(note).
       SetIsActive(true).
       Save(a.ctx)
   ```

6. **Convert to DTO** → `crud.go` (line 72)
   ```go
   return StudentToDTO(s), nil
   ```
   - Converts Ent entity to simple struct
   - DTO = Data Transfer Object

7. **Response** → Frontend receives `StudentDTO` with ID

### 3.2. Course Enrollment (Detailed)

**Complete Flow:**

```
User selects student + course → Frontend validation
           ↓
EnrollmentCreate(studentID, courseID, startDate)
           ↓
Backend checks: Student exists? Course exists?
           ↓
Calculate billing based on course.billingMode
           ↓
If subscription → Set fixedAmount
If pay-per-class → Amount calculated per attendance
           ↓
Create enrollment record in database
           ↓
Return EnrollmentDTO to frontend
           ↓
UI updates enrollment list
```

**Code Trace:**

1. **Frontend**: `App.tsx` (lines 450-500)
   ```typescript
   const handleEnrollmentSave = async () => {
     const dto = await EnrollmentCreate(
       selectedStudent,
       selectedCourse,
       startDate
     );
     setEnrollments([...enrollments, dto]);
   }
   ```

2. **Backend**: `crud.go` (lines 200-250)
   ```go
   func (a *App) EnrollmentCreate(studentID, courseID int, startDate string) (*EnrollmentDTO, error) {
       // Validate student exists
       student, err := a.db.Ent.Student.Get(a.ctx, studentID)
       
       // Validate course exists
       course, err := a.db.Ent.Course.Get(a.ctx, courseID)
       
       // Create enrollment
       e, err := a.db.Ent.Enrollment.Create().
           SetStudent(student).
           SetCourse(course).
           SetStartDate(parsedDate).
           SetIsActive(true).
           Save(a.ctx)
       
       return EnrollmentToDTO(e), nil
   }
   ```

### 3.3. Invoice Generation (Complex Flow)

**This is where billing logic happens:**

```
User clicks "Generate Invoices" button
           ↓
InvoiceService.GenerateDrafts(month, year) called
           ↓
FOR EACH active enrollment:
    ↓
    Load course billing settings
    ↓
    IF billingMode == "subscription":
        amount = course.price (fixed monthly)
    ↓
    ELSE IF billingMode == "by-class":
        count attendances for this month
        amount = attendances × course.pricePerClass
    ↓
    Apply discount if enrollment.discountPct > 0:
        amount = amount × (1 - discountPct/100)
    ↓
    Round to 2 decimals
    ↓
    Create invoice draft (not yet issued)
    ↓
END FOR
           ↓
Return count of invoices created
```

**Code Files:**

1. **Frontend Trigger**: `App.tsx`
   ```typescript
   const handleGenerateInvoices = async () => {
     const count = await GenerateDrafts(selectedMonth, selectedYear);
     alert(`Generated ${count} invoice drafts`);
     await loadInvoices();
   }
   ```

2. **Service Logic**: `internal/app/invoice/service.go` (lines 100-200)
   ```go
   func (s *Service) GenerateDrafts(ctx context.Context, month int, year int) (int, error) {
       // Get all active enrollments
       enrollments, err := s.db.Enrollment.Query().
           Where(enrollment.IsActiveEQ(true)).
           WithStudent().
           WithCourse().
           All(ctx)
       
       for _, enr := range enrollments {
           // Calculate amount based on billing mode
           var amt float64
           
           if enr.Edges.Course.BillingMode == "subscription" {
               amt = enr.Edges.Course.Price
           } else {
               // Count attendances for the month
               cnt, _ := s.db.Attendance.Query().
                   Where(
                       attendance.EnrollmentIDEQ(enr.ID),
                       attendance.DateGTE(monthStart),
                       attendance.DateLTE(monthEnd),
                   ).
                   Count(ctx)
               amt = float64(cnt) * enr.Edges.Course.PricePerClass
           }
           
           // Apply discount
           if enr.DiscountPct > 0 {
               amt = amt * (1 - enr.DiscountPct/100)
           }
           
           // Round to 2 decimals
           amt = utils.Round2(amt)
           
           // Create invoice
           s.db.Invoice.Create().
               SetEnrollment(enr).
               SetStudent(enr.Edges.Student).
               SetPeriodMonth(month).
               SetPeriodYear(year).
               SetAmount(amt).
               SetStatus("draft").
               Save(ctx)
       }
   }
   ```

### 3.4. PDF Export Flow

**Complete PDF Generation:**

```
User clicks "Export PDF" on invoice
           ↓
Frontend calls ExportInvoicePDF(invoiceID)
           ↓
Backend loads invoice with all relations:
    - Invoice data
    - Student info
    - Enrollment details
    - Course info
    - All invoice line items
           ↓
PDF Service creates document:
    - Load fonts from ~/LangSchool/Fonts/
    - Add header with school info
    - Add student name and invoice number
    - Add table with line items
    - Add totals and discounts
    - Add footer
           ↓
Save PDF to ~/LangSchool/Invoices/
           ↓
Return file path to frontend
           ↓
Frontend opens PDF with system viewer
```

**Code Trace:**

1. **Frontend**: `App.tsx`
   ```typescript
   const handleExportPDF = async (invoiceId: number) => {
     const path = await ExportInvoicePDF(invoiceId);
     console.log(`PDF saved to: ${path}`);
     // System will auto-open PDF
   }
   ```

2. **Backend**: `app.go` (lines 300-350)
   ```go
   func (a *App) ExportInvoicePDF(invoiceID int) (string, error) {
       // Load invoice with all relations
       inv, err := a.db.Ent.Invoice.Query().
           Where(invoice.IDEQ(invoiceID)).
           WithStudent().
           WithEnrollment(func(q *ent.EnrollmentQuery) {
               q.WithCourse()
           }).
           WithLines().
           Only(a.ctx)
       
       // Generate PDF
       pdfPath, err := a.pdfSvc.GenerateInvoice(inv)
       
       return pdfPath, nil
   }
   ```

3. **PDF Service**: `internal/pdf/invoice_pdf.go` (lines 50-200)
   ```go
   func (s *Service) GenerateInvoice(inv *ent.Invoice) (string, error) {
       pdf := gofpdf.New("P", "mm", "A4", "")
       pdf.AddPage()
       
       // Add fonts
       pdf.AddUTF8Font("DejaVu", "", fontPath)
       
       // Header
       pdf.SetFont("DejaVu", "", 16)
       pdf.Cell(0, 10, "Language School Invoice")
       
       // Student info
       pdf.SetFont("DejaVu", "", 12)
       pdf.Cell(0, 8, fmt.Sprintf("Student: %s", inv.Edges.Student.FullName))
       
       // Invoice number
       pdf.Cell(0, 8, fmt.Sprintf("Invoice #: %s", inv.Number))
       
       // Line items table
       for _, line := range inv.Edges.Lines {
           pdf.Cell(40, 8, line.Description)
           pdf.Cell(20, 8, fmt.Sprintf("%.2f", line.Amount))
       }
       
       // Total
       pdf.Cell(0, 10, fmt.Sprintf("Total: %.2f EUR", inv.Amount))
       
       // Save to file
       filename := fmt.Sprintf("invoice_%s.pdf", inv.Number)
       filepath := path.Join(s.dirs.Invoices, filename)
       pdf.OutputFileAndClose(filepath)
       
       return filepath, nil
   }
   ```

---

## 4. Component Connection Map

### 4.1. Frontend Components

**Main Component**: `frontend/src/App.tsx` (1,217 lines - monolithic)

**Sections:**
1. **State Management** (lines 1-100)
   - useState hooks for all data
   - Students, courses, enrollments, invoices, etc.

2. **Data Loading** (lines 100-200)
   - useEffect hooks
   - Load data on startup
   - Refresh functions

3. **CRUD Handlers** (lines 200-600)
   - Student operations
   - Course operations
   - Enrollment operations

4. **Business Logic** (lines 600-900)
   - Invoice generation
   - Payment recording
   - PDF exports

5. **UI Rendering** (lines 900-1217)
   - Tabs for each section
   - Forms and tables
   - Buttons and dialogs

### 4.2. Backend Functions

**Main Files:**

1. **crud.go** (500 lines)
   - StudentCreate, StudentUpdate, StudentDelete
   - CourseCreate, CourseUpdate, CourseDelete
   - EnrollmentCreate, EnrollmentUpdate
   - All basic CRUD operations

2. **app.go** (400 lines)
   - App struct initialization
   - Settings management
   - File operations (OpenFile)
   - Wails lifecycle hooks

3. **internal/app/invoice/service.go** (600 lines)
   - GenerateDrafts
   - IssueInvoice
   - ListInvoices
   - Complex billing logic

4. **internal/app/payment/service.go** (400 lines)
   - RecordPayment
   - Payment allocation
   - Balance calculations

### 4.3. Database Schema

**Tables:**

```
students
  ├── id (PRIMARY KEY)
  ├── full_name
  ├── phone
  ├── email
  ├── note
  └── is_active

courses
  ├── id (PRIMARY KEY)
  ├── name
  ├── billing_mode (subscription | by-class)
  ├── price (for subscription)
  ├── price_per_class (for by-class)
  └── is_active

enrollments
  ├── id (PRIMARY KEY)
  ├── student_id (FOREIGN KEY → students.id)
  ├── course_id (FOREIGN KEY → courses.id)
  ├── start_date
  ├── end_date
  ├── discount_pct
  └── is_active

invoices
  ├── id (PRIMARY KEY)
  ├── student_id (FOREIGN KEY → students.id)
  ├── enrollment_id (FOREIGN KEY → enrollments.id)
  ├── number (e.g., "INV-2024-001")
  ├── period_month
  ├── period_year
  ├── amount
  ├── status (draft | issued | paid)
  └── issue_date

invoice_lines
  ├── id (PRIMARY KEY)
  ├── invoice_id (FOREIGN KEY → invoices.id)
  ├── description
  ├── quantity
  ├── unit_price
  └── amount

payments
  ├── id (PRIMARY KEY)
  ├── student_id (FOREIGN KEY → students.id)
  ├── amount
  ├── payment_date
  └── note

payment_allocations
  ├── id (PRIMARY KEY)
  ├── payment_id (FOREIGN KEY → payments.id)
  ├── invoice_id (FOREIGN KEY → invoices.id)
  └── amount

attendances
  ├── id (PRIMARY KEY)
  ├── enrollment_id (FOREIGN KEY → enrollments.id)
  ├── date
  └── attended (boolean)

settings (singleton)
  ├── id (always 1)
  ├── org_name
  ├── org_address
  └── next_invoice_number
```

---

## 5. How Wails Binding Works

### 5.1. The Magic

Wails automatically creates a bridge between React and Go:

**Step 1**: You write a Go function in `app.go` or `crud.go`:
```go
func (a *App) StudentCreate(fullName, phone, email, note string) (*StudentDTO, error) {
    // Implementation
}
```

**Step 2**: Wails generates TypeScript bindings in `frontend/wailsjs/go/main/App.js`:
```typescript
export function StudentCreate(fullName: string, phone: string, email: string, note: string): Promise<main.StudentDTO> {
    return window['go']['main']['App']['StudentCreate'](fullName, phone, email, note);
}
```

**Step 3**: Frontend imports and calls it like a normal function:
```typescript
import { StudentCreate } from '../wailsjs/go/main/App';

const dto = await StudentCreate(name, phone, email, note);
```

### 5.2. Under the Hood

```
Frontend: StudentCreate("John", "123", "john@ex.com", "note")
    ↓
window.go.main.App.StudentCreate (Wails runtime)
    ↓
IPC Message sent to Go process
    ↓
Wails router receives message
    ↓
Finds App.StudentCreate method
    ↓
Calls Go function with parameters
    ↓
Go executes function
    ↓
Returns result (or error)
    ↓
Wails serializes to JSON
    ↓
IPC Message sent back to frontend
    ↓
Promise resolves with result
    ↓
Frontend continues execution
```

**Key Points:**
- Communication is async (uses Promises)
- Errors are automatically handled
- Types are preserved (TypeScript types match Go types)
- No manual API endpoints needed

---

## 6. How Database Layer Works

### 6.1. Ent ORM

**What is Ent?**
- Code generator for type-safe database access
- You define schema, Ent generates Go code
- Prevents SQL injection automatically
- Provides query builder

**Schema Definition**: `ent/schema/student.go`
```go
type Student struct {
    ent.Schema
}

func (Student) Fields() []ent.Field {
    return []ent.Field{
        field.String("full_name"),
        field.String("phone"),
        field.String("email"),
        field.String("note"),
        field.Bool("is_active").Default(true),
    }
}
```

**Generated Code**: `ent/student.go`, `ent/student/student.go`, `ent/student_create.go`, etc.

**Usage in Code**:
```go
// Create
student := db.Student.Create().
    SetFullName("John").
    SetPhone("123").
    Save(ctx)

// Query
students := db.Student.Query().
    Where(student.IsActiveEQ(true)).
    All(ctx)

// Update
student.Update().
    SetPhone("456").
    Save(ctx)

// Delete
db.Student.DeleteOneID(id).Exec(ctx)
```

### 6.2. Query Examples

**Simple Query:**
```go
// Get student by ID
student, err := db.Student.Get(ctx, 1)
```

**With Filtering:**
```go
// Get active students
students, err := db.Student.Query().
    Where(student.IsActiveEQ(true)).
    All(ctx)
```

**With Relations (Eager Loading):**
```go
// Get enrollment with student and course
enrollment, err := db.Enrollment.Query().
    Where(enrollment.IDEQ(id)).
    WithStudent().
    WithCourse().
    Only(ctx)

// Now you can access:
enrollment.Edges.Student.FullName
enrollment.Edges.Course.Name
```

**Aggregation:**
```go
// Count active enrollments
count, err := db.Enrollment.Query().
    Where(enrollment.IsActiveEQ(true)).
    Count(ctx)
```

---

## 7. Common Scenarios Traced

### Scenario 1: Creating a New Student

**Files Involved:**
1. `frontend/src/App.tsx` (line 200)
2. `frontend/wailsjs/go/main/App.js` (generated)
3. `crud.go` (line 50)
4. `ent/student_create.go` (generated)
5. `~/LangSchool/data/app.sqlite` (database file)

**Data Flow:**
```
Input: { fullName: "Anna K.", phone: "+123", email: "anna@ex.com", note: "VIP" }
   ↓ (sanitize)
Sanitized: { fullName: "Anna K.", phone: "+123", email: "anna@ex.com", note: "VIP" }
   ↓ (database)
INSERT INTO students (full_name, phone, email, note, is_active) VALUES (?, ?, ?, ?, ?)
   ↓ (response)
Output: { id: 5, fullName: "Anna K.", phone: "+123", email: "anna@ex.com", note: "VIP", isActive: true }
```

### Scenario 2: Generating Monthly Invoices

**Files Involved:**
1. `frontend/src/App.tsx` (line 600)
2. `internal/app/invoice/service.go` (line 100)
3. `ent/attendance_query.go` (generated)
4. `ent/invoice_create.go` (generated)

**Logic Flow:**
```
Input: { month: 12, year: 2024 }
   ↓
Query: SELECT * FROM enrollments WHERE is_active = true
Result: 10 active enrollments
   ↓
For each enrollment:
   IF course.billing_mode = "subscription":
      amount = 50.00 (fixed)
   ELSE:
      Query: SELECT COUNT(*) FROM attendances WHERE enrollment_id = ? AND date >= ? AND date <= ?
      Result: 8 classes attended
      amount = 8 × 10.00 = 80.00
   
   IF enrollment.discount_pct > 0:
      amount = 80.00 × (1 - 10/100) = 72.00
   
   CREATE invoice with amount = 72.00
   ↓
Output: { count: 10 } (10 invoices created)
```

### Scenario 3: Recording a Payment

**Files Involved:**
1. `frontend/src/App.tsx` (line 700)
2. `internal/app/payment/service.go` (line 50)
3. `ent/payment_create.go` (generated)
4. `ent/payment_allocation_create.go` (generated)

**Allocation Logic:**
```
Input: { studentId: 1, amount: 150.00, date: "2024-12-26" }
   ↓
Query: SELECT * FROM invoices WHERE student_id = 1 AND status = 'issued' ORDER BY issue_date
Result: 3 unpaid invoices [100.00, 75.00, 50.00]
   ↓
Allocate payment:
   Invoice 1: 100.00 (fully paid) → remaining = 50.00
   Invoice 2: 50.00 (partially paid) → remaining = 0.00
   Invoice 3: 0.00 (not paid)
   ↓
Create payment record: 150.00
Create allocation 1: invoice_id=1, amount=100.00
Create allocation 2: invoice_id=2, amount=50.00
   ↓
Update invoice 1: status = "paid"
Update invoice 2: status = "issued" (still partially unpaid)
   ↓
Output: { paymentId: 42, allocated: 150.00 }
```

---

## 8. Troubleshooting: Tracing Requests

### How to Debug a Feature

**Problem**: "Student creation doesn't work"

**Trace Path:**

1. **Check Frontend Console** (browser DevTools)
   - Look for JavaScript errors
   - Check if function is called
   - Verify parameters

2. **Check Backend Logs** (terminal running `wails dev`)
   - Look for Go errors
   - Check if function received request
   - See SQL errors

3. **Add Temporary Logging**

   **Frontend** (`App.tsx`):
   ```typescript
   const handleStudentSave = async () => {
       console.log("Creating student:", fullName, phone, email);
       try {
           const dto = await StudentCreate(fullName, phone, email, note);
           console.log("Student created:", dto);
       } catch (err) {
           console.error("Error creating student:", err);
       }
   }
   ```

   **Backend** (`crud.go`):
   ```go
   func (a *App) StudentCreate(fullName, phone, email, note string) (*StudentDTO, error) {
       log.Printf("StudentCreate called: name=%s, phone=%s", fullName, phone)
       
       s, err := a.db.Ent.Student.Create()...
       if err != nil {
           log.Printf("StudentCreate error: %v", err)
           return nil, err
       }
       
       log.Printf("StudentCreate success: id=%d", s.ID)
       return StudentToDTO(s), nil
   }
   ```

4. **Check Database** (SQLite)
   ```bash
   sqlite3 ~/LangSchool/data/app.sqlite
   SELECT * FROM students;
   ```

### Common Error Patterns

**Error**: "undefined: utils"
- **Cause**: Missing import statement
- **Fix**: Add `import "langschool/internal/app/utils"` to top of file

**Error**: "sql: unknown driver sqlite3"
- **Cause**: SQLite driver not imported
- **Fix**: Check `internal/infra/db.go` has `_ "github.com/mattn/go-sqlite3"`

**Error**: "student not found"
- **Cause**: Querying with wrong ID or student was deleted
- **Fix**: Verify ID exists in database

**Error**: "XHR failed" or "Promise rejected"
- **Cause**: Backend function panic or unhandled error
- **Fix**: Check backend logs for panic stack trace

---

## Quick Reference: File → Feature Map

| Feature | Frontend File | Backend File | Service File | Database |
|---------|--------------|--------------|--------------|----------|
| Student CRUD | App.tsx (200-300) | crud.go (50-150) | - | ent/student |
| Course CRUD | App.tsx (300-400) | crud.go (150-250) | - | ent/course |
| Enrollment | App.tsx (400-500) | crud.go (250-350) | - | ent/enrollment |
| Invoice Generation | App.tsx (600-650) | app.go (200-250) | invoice/service.go | ent/invoice |
| Payment Recording | App.tsx (700-750) | app.go (250-300) | payment/service.go | ent/payment |
| PDF Export | App.tsx (800-850) | app.go (300-350) | pdf/invoice_pdf.go | - |
| Settings | App.tsx (900-950) | crud.go (350-400) | - | ent/settings |

---

## Summary

**Key Takeaways:**

1. **Wails bridges React and Go automatically** - no manual API needed
2. **Ent ORM generates type-safe database code** - prevents SQL errors
3. **Data flows in 4 layers**: Frontend → Backend → Services → Database
4. **Each request is traced** through files in predictable order
5. **Business logic** lives in service files (invoice, payment)
6. **CRUD operations** are in crud.go (simple operations)
7. **Complex workflows** use multiple layers working together

**Next Steps:**

1. Read LEARNING_GUIDE.md to understand concepts deeper
2. Run the application and trace a request through DevTools
3. Add console.log statements to see data flow
4. Modify a small feature to practice
5. Use this guide as reference when stuck

**Remember:** Every feature follows the same pattern:
```
UI Action → Frontend Function → Wails Bridge → Backend Function → Service Logic → Database → Response Back
```

Once you understand this pattern, you can trace any feature in the application!
