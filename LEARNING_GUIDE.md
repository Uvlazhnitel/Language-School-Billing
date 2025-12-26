# Learning Guide: Language School Billing System

## Overview
This guide provides a structured path to understand the entire Language School Billing project, from architecture to implementation details. Follow this step-by-step to become fully proficient in explaining and presenting the system.

---

## Phase 1: High-Level Understanding (Day 1 - 2 hours)

### 1.1 What Problem Does This Solve?
**Read**: `DOCUMENTATION_GUIDE.md` - Section 1 (Project Overview)

**Key Questions to Answer:**
- What is the business purpose of this application?
- Who are the target users? (Language school administrators)
- What are the main features? (Student management, course management, billing, invoicing)

**Action Items:**
- [ ] Write a 2-minute elevator pitch about the project
- [ ] List 5 core features and explain each in one sentence

### 1.2 System Architecture
**Read**: `DOCUMENTATION_GUIDE.md` - Section 2 (Architecture Map)

**Understand:**
- **Backend**: Go + Wails framework (desktop application)
- **Frontend**: React + TypeScript + Vite
- **Database**: SQLite with Ent ORM
- **PDF Generation**: gofpdf library

**Draw on Paper:**
```
┌─────────────┐
│   React UI  │ (Frontend - TypeScript)
└──────┬──────┘
       │ Wails Bridge (Go bindings)
┌──────▼──────┐
│  Go Backend │ (Business Logic)
└──────┬──────┘
       │
┌──────▼──────┐
│  Ent ORM    │
└──────┬──────┘
       │
┌──────▼──────┐
│   SQLite    │ (Database)
└─────────────┘
```

**Action Items:**
- [ ] Explain how data flows from UI to database
- [ ] Understand why Wails was chosen (desktop app with web technologies)

---

## Phase 2: Database & Data Model (Day 1-2 - 3 hours)

### 2.1 Database Schema
**Read**: `DOCUMENTATION_GUIDE.md` - Section 11 (Database Schema)

**Study These Tables:**
1. **Student** - Who are the students?
2. **Course** - What courses are offered?
3. **Enrollment** - Which students are in which courses?
4. **Invoice** - How is billing tracked?
5. **InvoiceLine** - What are the line items?
6. **Payment** - How are payments recorded?
7. **Setting** - What are configurable settings?

**Action Items:**
- [ ] Draw the Entity Relationship Diagram on paper
- [ ] Trace a complete workflow: "Student enrolls → Invoice created → Payment received"
- [ ] Answer: "How does the system calculate what a student owes?"

### 2.2 Ent ORM Basics
**Location**: `ent/schema/` directory

**Key Concepts:**
- Each schema file defines a database table
- Ent auto-generates Go code for database operations
- Relationships: `Edge()` defines connections between tables

**Hands-On Exercise:**
```go
// Open and read: ent/schema/student.go
// Understand:
// - Fields: FullName, Phone, Email, Note, IsActive
// - Edges: Enrollments (one student → many enrollments)
```

**Action Items:**
- [ ] Explain what `Edge()` does in schema files
- [ ] Find where foreign keys are defined (e.g., Student → Enrollment)

---

## Phase 3: Backend Code Structure (Day 2-3 - 4 hours)

### 3.1 Main Application Entry
**File**: `main.go` (or root-level Go files)

**Understand:**
- How Wails initializes
- How the App struct is created
- How the backend connects to frontend

### 3.2 Core CRUD Operations
**File**: `crud.go` (lines ~400)

**Key Functions to Study:**
1. **StudentList** - How to fetch all students
2. **StudentCreate** - How to add a new student
3. **StudentUpdate** - How to modify student data
4. **CourseCreate** - How courses are created
5. **EnrollmentCreate** - How students are enrolled in courses

**Tracing Exercise:**
Pick `StudentCreate` and trace:
1. Frontend calls the function
2. Backend receives parameters
3. Input is sanitized (XSS protection!)
4. Ent ORM creates database record
5. Result is returned to frontend

**Action Items:**
- [ ] Explain the sanitization pattern: `sanitizeInput()` function
- [ ] Find where constants are defined (`internal/app/constants.go`)
- [ ] Answer: "Why do we need to sanitize user input?"

### 3.3 Business Logic Services
**Location**: `internal/app/` directory

**Key Services:**
- **invoice/service.go** - Invoice generation and management
- **payment/service.go** - Payment processing
- **utils/math.go** - Shared utilities (like Round2)

**Deep Dive: Invoice Service**
**File**: `internal/app/invoice/service.go`

**Key Methods:**
1. `GenerateDrafts()` - Creates invoices for students
2. `issueOne()` - Marks invoice as issued
3. `listInvoices()` - Retrieves invoices (with N+1 query optimization!)

**Critical Concept: Billing Modes**
```go
// Two billing modes:
BillingModeSubscription  // Monthly recurring
BillingModePay          // Pay per class

// Two course types:
CourseTypeGroup         // Group classes
CourseTypeIndividual    // 1-on-1 lessons
```

**Action Items:**
- [ ] Explain how `GenerateDrafts()` calculates invoice amounts
- [ ] Understand the transaction pattern fix (committed flag)
- [ ] Answer: "What's the difference between subscription and pay mode?"

---

## Phase 4: Frontend Understanding (Day 3-4 - 4 hours)

### 4.1 React Application Structure
**File**: `frontend/src/App.tsx` (1,217 lines - monolithic)

**Main Sections (as tabs):**
1. **Students** - Student management
2. **Courses** - Course management
3. **Enrollments** - Link students to courses
4. **Invoices** - Billing management
5. **Payments** - Payment tracking
6. **Reports** - Financial reports
7. **Settings** - System configuration

**Study Pattern: Student Tab**
Look at how the Students tab works:
- State management with `useState`
- Data fetching with backend calls
- Form handling for create/edit
- Table display with filtering

### 4.2 Wails Bridge
**File**: `frontend/wailsjs/go/main/App.js`

**Key Concept:**
```javascript
// Frontend calls Go functions like this:
import { StudentList, StudentCreate } from './wailsjs/go/main/App'

// Usage:
const students = await StudentList()
const newStudent = await StudentCreate("John Doe", "+1234", "john@example.com", "Note")
```

**Action Items:**
- [ ] Find 3 examples of frontend calling backend
- [ ] Understand async/await pattern
- [ ] Explain: "How does TypeScript frontend call Go backend?"

### 4.3 UI Components & State
**Study the pattern:**
```typescript
// Typical component structure:
const [students, setStudents] = useState([])  // State
const [loading, setLoading] = useState(false) // Loading state
const [error, setError] = useState("")        // Error handling

// Fetch data
useEffect(() => {
  loadStudents()
}, [])

// CRUD operations
const loadStudents = async () => { /* ... */ }
const createStudent = async () => { /* ... */ }
```

**Action Items:**
- [ ] Explain React hooks: useState, useEffect
- [ ] Find where error messages are displayed
- [ ] Answer: "How does the UI update when data changes?"

---

## Phase 5: Key Features Deep Dive (Day 4-5 - 4 hours)

### 5.1 Student Management Workflow
**Trace This Scenario:**
1. Admin clicks "Add Student"
2. Fills form (name, phone, email, note)
3. Clicks "Save"
4. What happens?

**Answer:**
```
Frontend (App.tsx) 
  → calls StudentCreate()
Backend (crud.go)
  → sanitizes input (XSS protection)
  → calls Ent ORM
Database (SQLite)
  → INSERT INTO student...
  → Returns new student with ID
Backend
  → converts to StudentDTO
  → returns to frontend
Frontend
  → updates student list
  → shows success message
```

### 5.2 Enrollment & Billing Workflow
**Trace This Scenario:**
1. Student "Anna" enrolls in "English A1" course
2. Course costs 100 EUR/month (subscription mode)
3. Invoice is generated
4. Payment is recorded

**Study These Files:**
- `crud.go` - `EnrollmentCreate()`
- `internal/app/invoice/service.go` - `GenerateDrafts()`
- `internal/app/payment/service.go` - Payment recording

**Action Items:**
- [ ] Draw a flowchart of this process
- [ ] Find where discounts are calculated
- [ ] Answer: "How does the system know what to charge each student?"

### 5.3 PDF Invoice Generation
**File**: `internal/pdf/invoice_pdf.go`

**Key Concept:**
```go
// Creates a PDF invoice with:
// - School logo/header
// - Invoice number and date
// - Student information
// - Line items (courses, prices)
// - Total amount
// - Payment instructions
```

**Action Items:**
- [ ] Find where fonts are loaded
- [ ] Understand how tables are rendered
- [ ] Locate where invoice number is generated

---

## Phase 6: Security & Quality (Day 5 - 2 hours)

### 6.1 Security Features Implemented

**1. XSS Protection**
**File**: `crud.go` - `sanitizeInput()` function
```go
// All user input is HTML-escaped:
// "<script>alert('xss')</script>" 
// becomes: "&lt;script&gt;alert('xss')&lt;/script&gt;"
```

**2. Path Traversal Protection**
**File**: `app.go` - `OpenFile()` method
```go
// Validates file paths before opening
// Prevents: OpenFile("../../etc/passwd")
// Only allows: files in ~/LangSchool/ directory
```

**3. Input Validation**
**File**: `internal/validation/validate.go`
- Validates email formats
- Validates price ranges
- Validates discount percentages

### 6.2 Testing
**File**: `internal/validation/validate_test.go`

**Study the test structure:**
```go
func TestSanitizeInput(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"XSS script", "<script>alert('xss')</script>", "&lt;script&gt;..."},
        // ... more test cases
    }
    // Run tests
}
```

**Action Items:**
- [ ] Run tests: `go test ./internal/validation/...`
- [ ] Understand why security is important
- [ ] Answer: "What attacks does this system prevent?"

### 6.3 CI/CD Pipeline
**File**: `.github/workflows/ci.yml`

**Understand the 4 jobs:**
1. **Backend Tests** - Runs all Go tests
2. **Frontend Build** - Checks TypeScript compilation
3. **Security Scan** - Runs Gosec security scanner
4. **Linting** - Checks code quality

**Action Items:**
- [ ] Explain what CI/CD means
- [ ] Describe what happens on every git push
- [ ] Answer: "Why is automated testing important?"

---

## Phase 7: Recent Improvements (Day 5-6 - 2 hours)

### 7.1 P0 Critical Fixes
**Read**: `P0_FIXES_SUMMARY.md`

**What was fixed:**
1. Added unit tests (0% → test coverage)
2. Added CI/CD pipeline
3. Eliminated duplicate constants
4. Added XSS protection
5. Fixed path traversal vulnerability

### 7.2 P1 High-Priority Fixes
**What was fixed:**
1. Added linting (ESLint, Prettier)
2. Fixed transaction pattern (proper rollback)
3. Updated dependencies (TypeScript 5.7, Vite 6.0)
4. Fixed nil pointer risks
5. Created frontend refactoring guide

### 7.3 P2 Medium-Priority Fixes
**What was fixed:**
1. Extracted shared utilities (Round2 function)
2. Documented magic numbers with constants
3. Fixed N+1 query problem (invoice listing optimization)
4. Added error context wrapping
5. Removed personal information

**Action Items:**
- [ ] Explain what "N+1 query problem" means
- [ ] Understand why code deduplication matters
- [ ] Answer: "What makes code 'maintainable'?"

---

## Phase 8: Hands-On Practice (Day 6-7 - 4 hours)

### 8.1 Run the Application
```bash
# Step 1: Start development mode
wails dev

# Step 2: Application opens in desktop window
# Step 3: Explore the UI
```

### 8.2 Make a Small Change
**Exercise 1: Add a field to Student**
1. Edit `ent/schema/student.go` - add a new field
2. Run `go generate ./ent`
3. Update `crud.go` to handle new field
4. Update frontend to display it
5. Test your changes

**Exercise 2: Modify Invoice Template**
1. Open `internal/pdf/invoice_pdf.go`
2. Change header text or add a footer
3. Generate an invoice and view PDF
4. Iterate until satisfied

### 8.3 Debug a Problem
**Exercise: Intentionally break something**
1. Comment out the `sanitizeInput()` call in `StudentCreate`
2. Try to create a student with `<script>` in the name
3. Understand why sanitization is needed
4. Fix it back

**Action Items:**
- [ ] Successfully run the application
- [ ] Add a student, course, and enrollment
- [ ] Generate an invoice
- [ ] Record a payment
- [ ] Generate a report

---

## Phase 9: Presentation Preparation (Day 7 - 3 hours)

### 9.1 Create Your Presentation Structure

**Slide 1: Introduction**
- Project name and purpose
- Your role in development
- Technologies used

**Slide 2: Business Problem**
- Language schools need billing management
- Manual processes are error-prone
- Need automated invoice generation

**Slide 3: Solution Overview**
- Desktop application (works offline)
- Modern web technologies (React + Go)
- Cross-platform (Windows, Mac, Linux)

**Slide 4: Architecture**
- Show the architecture diagram
- Explain each layer
- Highlight why this stack was chosen

**Slide 5: Key Features**
- Student management
- Course scheduling
- Automatic invoicing
- Payment tracking
- PDF generation
- Financial reports

**Slide 6: Database Design**
- Show ERD diagram
- Explain key relationships
- Highlight billing flexibility (subscription vs. pay-per-class)

**Slide 7: Security Features**
- XSS protection (input sanitization)
- Path traversal prevention
- Input validation
- Automated security scanning (Gosec)

**Slide 8: Code Quality**
- Unit tests (19 test cases)
- CI/CD pipeline (4 automated jobs)
- Linting and formatting
- Type safety (TypeScript + Go)

**Slide 9: Recent Improvements**
- Fixed 20 prioritized issues (P0/P1/P2)
- Eliminated 214 lines of dead code
- Added comprehensive documentation
- Performance optimizations (N+1 query fix)

**Slide 10: Demo**
- Live demonstration of key workflows
- Show student enrollment → invoice → payment

**Slide 11: Technical Challenges**
- Challenge 1: Desktop app with web tech → Solution: Wails
- Challenge 2: Offline operation → Solution: SQLite
- Challenge 3: PDF generation → Solution: gofpdf

**Slide 12: Future Improvements**
- Multi-currency support
- Email notifications
- Online payment integration
- Mobile companion app

### 9.2 Demo Script
**Prepare a 5-minute live demo:**

```
1. Open application (0:30)
   "Let me show you the Language School Billing System..."

2. Add a student (0:45)
   "First, we can add a new student with their contact information..."
   [Show input sanitization by trying to enter <script>]

3. Create a course (0:30)
   "Next, let's create a new English course..."

4. Enroll student (0:30)
   "Now we enroll our student in the course..."

5. Generate invoice (1:00)
   "The system automatically generates invoices based on enrollment..."
   [Show invoice details, PDF preview]

6. Record payment (0:30)
   "When the student pays, we record it here..."

7. View report (0:45)
   "Finally, we can see financial reports for our school..."

8. Show code (0:30)
   "Behind the scenes, here's how the invoice generation works..."
   [Show one simple code snippet]
```

---

## Phase 10: Q&A Preparation (Day 7 - 2 hours)

### Common Questions & Answers

**Q: Why did you choose Wails instead of Electron?**
A: Wails is lighter weight, has better performance, and uses native Go for the backend instead of Node.js. This means smaller binaries and faster execution.

**Q: How does the billing system handle different currencies?**
A: Currently, it uses a single currency (EUR) configured in settings. Multi-currency support would be a future enhancement.

**Q: What happens if a student is enrolled in multiple courses?**
A: The system creates separate line items on the invoice for each enrollment. Prices are calculated based on each course's billing mode (subscription or pay-per-class).

**Q: Can you explain the N+1 query problem you fixed?**
A: Originally, the system would query the database N+1 times (once for invoices, then once for each invoice's line count). Now it uses a batch query with GROUP BY, reducing it to just 2 queries total. This dramatically improves performance when there are many invoices.

**Q: How do you ensure data integrity?**
A: Multiple layers: database constraints (foreign keys), ORM validation (Ent), input validation (backend), and type safety (TypeScript + Go). Plus automated tests and CI/CD checks.

**Q: What's your testing strategy?**
A: Unit tests for business logic (validation, calculations), integration tests would test full workflows, and CI/CD runs all tests automatically on every push.

**Q: How do you handle concurrent users?**
A: This is a desktop app, so typically one user per installation. SQLite handles locking for database access.

**Q: Can this scale to a large language school?**
A: For a single school with hundreds of students, yes. For thousands of students or multiple schools, you'd want to upgrade to PostgreSQL or MySQL and add a client-server architecture.

**Q: How did you learn all this?**
A: "I studied the codebase systematically - starting with architecture, then data models, then tracing key workflows. I also ran the app and made small modifications to understand how each piece works together."

**Q: What was the hardest part?**
A: "Understanding the billing logic with different modes (subscription vs. pay-per-class) and course types (group vs. individual), and how they interact to calculate invoices."

**Q: What are you most proud of?**
A: "The security improvements - adding XSS protection and path traversal prevention. Also the performance optimization that fixed the N+1 query problem."

---

## Quick Reference Cheat Sheet

### Architecture
```
Frontend (React/TypeScript) 
  ↔ Wails Bridge 
  ↔ Backend (Go) 
  ↔ Ent ORM 
  ↔ SQLite
```

### Key Files
- `main.go` - Entry point
- `crud.go` - CRUD operations
- `app.go` - Main App struct
- `internal/app/invoice/service.go` - Invoice logic
- `frontend/src/App.tsx` - UI
- `ent/schema/` - Database schema

### Key Concepts
- **Wails**: Desktop app framework using web tech
- **Ent**: Type-safe ORM for Go
- **SQLite**: Embedded database (no server needed)
- **Billing Mode**: Subscription (monthly) or Pay (per class)
- **Course Type**: Group or Individual
- **XSS Protection**: HTML escape user input
- **N+1 Query**: Performance anti-pattern we fixed

### Commands
```bash
wails dev          # Run in development mode
go test ./...      # Run all tests
go generate ./ent  # Regenerate ORM code
npm run lint       # Lint frontend
npm run format     # Format frontend code
```

### Statistics
- **Languages**: Go (backend), TypeScript (frontend)
- **Total Lines**: ~21,500
- **Test Cases**: 19 (all passing)
- **Security Fixes**: 2 critical (XSS, path traversal)
- **Performance Fixes**: 1 major (N+1 query)
- **Dead Code Removed**: 214 lines

---

## Final Checklist

Before your presentation, ensure you can:
- [ ] Explain the system in 2 minutes
- [ ] Draw the architecture from memory
- [ ] Trace the invoice generation workflow
- [ ] Run the application and complete a demo
- [ ] Answer "why this technology?"
- [ ] Explain one security feature in detail
- [ ] Describe the testing strategy
- [ ] Discuss future improvements
- [ ] Confidently handle Q&A

---

## Tips for Success

1. **Practice Out Loud**: Present to yourself, a friend, or record yourself
2. **Time Yourself**: Ensure you fit within time limits
3. **Prepare Backup**: Have screenshots in case live demo fails
4. **Know Your Audience**: Adjust technical depth based on who's listening
5. **Be Honest**: If asked something you don't know, say "That's a great question. Let me show you where that code is and we can explore it together."
6. **Show Enthusiasm**: Talk about what you learned and found interesting
7. **Code on Screen**: Have 2-3 simple code snippets ready to show
8. **Own It**: You studied this system thoroughly - be confident!

---

## Estimated Timeline

- **Day 1**: Phases 1-2 (5 hours) - Overview & Data Model
- **Day 2-3**: Phase 3 (4 hours) - Backend Code
- **Day 3-4**: Phase 4 (4 hours) - Frontend Code
- **Day 4-5**: Phase 5 (4 hours) - Features Deep Dive
- **Day 5-6**: Phases 6-7 (4 hours) - Security & Improvements
- **Day 6-7**: Phases 8-9 (7 hours) - Hands-On & Presentation Prep
- **Day 7**: Phase 10 (2 hours) - Q&A Preparation

**Total**: ~30 hours over 7 days (or less if you accelerate)

---

## Your Learning Journey Begins Now!

Start with Phase 1 and work through systematically. Take notes, draw diagrams, and most importantly - **run the code and experiment**. The best way to learn is by doing.

Good luck with your presentation! 🚀
