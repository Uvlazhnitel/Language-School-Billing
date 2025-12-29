# Software Requirements Specification

**Language School Billing System**

---

## 1. Introduction

### 1.1 Purpose

This document provides a comprehensive specification of functional and non-functional requirements for the Language School Billing System.

### 1.2 Scope

The system is a single-user desktop application for managing language school operations, including:
- Student and course management
- Enrollment tracking
- Attendance monitoring
- Invoice generation and management
- Payment tracking

### 1.3 Definitions and Acronyms

- **Student**: Person enrolled in language courses
- **Course**: Teaching program (group or individual)
- **Enrollment**: Student-course assignment with billing configuration
- **Billing Mode**: Per-lesson or subscription pricing model
- **Draft Invoice**: Invoice awaiting issuance
- **Issued Invoice**: Finalized invoice with sequential number
- **Attendance Month**: Record of lessons attended in a specific month
- **Lock**: Prevent further modifications to attendance data

---

## 2. Functional Requirements

### 2.1 Student Management (FR-SM)

#### FR-SM-01: Create Student
**Description**: System shall allow creating new student records  
**Priority**: High  
**Input**: Full name (required), phone (optional), email (optional), note (optional)  
**Processing**: Validate non-empty name, sanitize inputs, save to database  
**Output**: New student record with generated ID  
**Success Criteria**: Student appears in student list

#### FR-SM-02: Update Student
**Description**: System shall allow updating existing student information  
**Priority**: High  
**Input**: Student ID, updated fields  
**Processing**: Validate inputs, update database record  
**Output**: Updated student record  
**Success Criteria**: Changes reflected in student list

#### FR-SM-03: Activate/Deactivate Student
**Description**: System shall allow toggling student active status  
**Priority**: Medium  
**Input**: Student ID, active boolean  
**Processing**: Update is_active field  
**Output**: Updated student status  
**Success Criteria**: Status change reflected in UI

#### FR-SM-04: Prevent Student Deletion with Dependencies
**Description**: System shall prevent deleting students with existing enrollments or invoices  
**Priority**: High  
**Input**: Student ID  
**Processing**: Check for related records  
**Output**: Error message if dependencies exist  
**Success Criteria**: Data integrity maintained

#### FR-SM-05: Validate Student Name
**Description**: System shall validate that student names are non-empty  
**Priority**: High  
**Input**: Full name string  
**Processing**: Check for empty or whitespace-only strings  
**Output**: Validation error if invalid  
**Success Criteria**: Cannot save student without valid name

#### FR-SM-06: Sanitize Text Inputs
**Description**: System shall sanitize all text inputs to prevent XSS  
**Priority**: Critical  
**Input**: Any text field  
**Processing**: HTML escape special characters  
**Output**: Sanitized string  
**Success Criteria**: No script execution from user inputs

---

### 2.2 Course Management (FR-CM)

#### FR-CM-01: Support Course Types
**Description**: System shall support "group" and "individual" course types  
**Priority**: High  
**Input**: Course type selection  
**Processing**: Validate enum value  
**Output**: Course with specified type  
**Success Criteria**: Both types available in UI

#### FR-CM-02: Configure Course Pricing
**Description**: System shall allow setting lesson and subscription prices  
**Priority**: High  
**Input**: Lesson price (float), subscription price (float)  
**Processing**: Validate non-negative numbers  
**Output**: Course with pricing configuration  
**Success Criteria**: Prices used in invoice calculations

#### FR-CM-03: Validate Prices
**Description**: System shall validate that prices are non-negative  
**Priority**: High  
**Input**: Price values  
**Processing**: Check >= 0  
**Output**: Validation error if negative  
**Success Criteria**: Cannot save negative prices

#### FR-CM-04: CRUD Operations for Courses
**Description**: System shall allow create, read, update, delete operations  
**Priority**: High  
**Input**: Course data  
**Processing**: Validate, persist to database  
**Output**: Course records  
**Success Criteria**: All operations work correctly

#### FR-CM-05: Prevent Course Deletion with Active Enrollments
**Description**: System shall prevent deleting courses with active enrollments  
**Priority**: High  
**Input**: Course ID  
**Processing**: Check for related enrollments  
**Output**: Error if dependencies exist  
**Success Criteria**: Data integrity maintained

---

### 2.3 Enrollment Management (FR-EM)

#### FR-EM-01: Enroll Students in Courses
**Description**: System shall allow creating enrollment records  
**Priority**: High  
**Input**: Student ID, course ID, billing mode, discount percentage  
**Processing**: Validate inputs, create enrollment  
**Output**: New enrollment record  
**Success Criteria**: Enrollment appears in list

#### FR-EM-02: Support Billing Modes
**Description**: System shall support "per_lesson" and "subscription" billing  
**Priority**: High  
**Input**: Billing mode selection  
**Processing**: Validate enum value  
**Output**: Enrollment with billing configuration  
**Success Criteria**: Correct billing applied in invoices

#### FR-EM-03: Set Custom Discounts
**Description**: System shall allow setting discount percentages per enrollment  
**Priority**: Medium  
**Input**: Discount percentage (0-100)  
**Processing**: Validate range, save to enrollment  
**Output**: Enrollment with discount  
**Success Criteria**: Discount applied in invoice calculation

#### FR-EM-04: Validate Discount Range
**Description**: System shall validate discount is between 0 and 100  
**Priority**: High  
**Input**: Discount value  
**Processing**: Check 0 <= value <= 100  
**Output**: Validation error if out of range  
**Success Criteria**: Cannot save invalid discounts

#### FR-EM-05: Add Notes to Enrollments
**Description**: System shall allow adding optional notes  
**Priority**: Low  
**Input**: Note text  
**Processing**: Sanitize and save  
**Output**: Enrollment with note  
**Success Criteria**: Notes display in enrollment details

#### FR-EM-06: Prevent Duplicate Enrollments
**Description**: System shall prevent duplicate student-course pairs  
**Priority**: Medium  
**Input**: Student ID, course ID  
**Processing**: Check for existing enrollment  
**Output**: Error if duplicate found  
**Success Criteria**: One enrollment per student-course pair

---

### 2.4 Attendance Tracking (FR-AT)

#### FR-AT-01: Track Monthly Attendance
**Description**: System shall track lesson counts per month per student-course  
**Priority**: High  
**Input**: Student ID, course ID, year, month, lessons count  
**Processing**: Create or update attendance record  
**Output**: Attendance record  
**Success Criteria**: Attendance data saved correctly

#### FR-AT-02: Edit Lesson Counts
**Description**: System shall allow editing lesson counts for any month  
**Priority**: High  
**Input**: Attendance record, new count  
**Processing**: Validate, update if not locked  
**Output**: Updated attendance  
**Success Criteria**: Changes reflected immediately

#### FR-AT-03: Bulk Attendance Update
**Description**: System shall provide "Add +1 to all" feature  
**Priority**: Medium  
**Input**: Selection of all/filtered students  
**Processing**: Increment lesson count for each  
**Output**: Updated attendance records  
**Success Criteria**: All counts incremented by 1

#### FR-AT-04: Lock Months
**Description**: System shall allow locking months to prevent changes  
**Priority**: Medium  
**Input**: Year, month  
**Processing**: Set locked flag on attendance records  
**Output**: Locked attendance records  
**Success Criteria**: Cannot edit locked months

#### FR-AT-05: Prevent Editing Locked Months
**Description**: System shall prevent editing attendance for locked months  
**Priority**: High  
**Input**: Edit attempt on locked record  
**Processing**: Check locked flag  
**Output**: Error message  
**Success Criteria**: Locked months remain unchanged

#### FR-AT-06: Auto-create Attendance Records
**Description**: System shall automatically create attendance records when needed  
**Priority**: Medium  
**Input**: Student enrollment exists  
**Processing**: Create attendance record with 0 lessons  
**Output**: New attendance record  
**Success Criteria**: Records available for editing

---

### 2.5 Invoice Generation (FR-IG)

#### FR-IG-01: Generate Invoice Drafts
**Description**: System shall generate drafts from attendance and subscriptions  
**Priority**: High  
**Input**: Year, month  
**Processing**: Query enrollments, calculate amounts, create invoice drafts  
**Output**: Draft invoices  
**Success Criteria**: Drafts match billing data

#### FR-IG-02: Sequential Invoice Numbering
**Description**: System shall assign numbers in format PREFIX-YYYYMM-SEQ  
**Priority**: High  
**Input**: Invoice to issue  
**Processing**: Get next sequence, format number, increment sequence  
**Output**: Invoice with number  
**Success Criteria**: Sequential numbers without gaps

#### FR-IG-03: Issue Individual Invoices
**Description**: System shall allow issuing individual draft invoices  
**Priority**: High  
**Input**: Draft invoice ID  
**Processing**: Assign number, change status to issued, generate PDF  
**Output**: Issued invoice with PDF  
**Success Criteria**: Invoice issued, PDF saved

#### FR-IG-04: Batch Issue Invoices
**Description**: System shall allow issuing all draft invoices at once  
**Priority**: Medium  
**Input**: List of draft invoices  
**Processing**: Issue each invoice sequentially  
**Output**: All invoices issued  
**Success Criteria**: Batch operation completes successfully

#### FR-IG-05: Generate PDF Invoices
**Description**: System shall generate PDF files for issued invoices  
**Priority**: High  
**Input**: Issued invoice data  
**Processing**: Format PDF with gofpdf, save to file system  
**Output**: PDF file  
**Success Criteria**: PDF readable, contains correct data

#### FR-IG-06: Organize PDF Storage
**Description**: System shall save PDFs to ~/LangSchool/Invoices/YYYY/MM/  
**Priority**: Medium  
**Input**: Invoice year, month, number  
**Processing**: Create directory structure, save file  
**Output**: PDF in organized location  
**Success Criteria**: Easy to find PDFs by date

#### FR-IG-07: Support Invoice Statuses
**Description**: System shall support draft, issued, paid, canceled statuses  
**Priority**: High  
**Input**: Status transition  
**Processing**: Validate allowed transitions, update status  
**Output**: Invoice with new status  
**Success Criteria**: Status workflow enforced

#### FR-IG-08: Calculate Invoice Totals
**Description**: System shall calculate totals from lines with discounts  
**Priority**: High  
**Input**: Invoice lines, discount percentages  
**Processing**: Sum (qty * unit_price * (1 - discount/100))  
**Output**: Total amount  
**Success Criteria**: Totals mathematically correct

#### FR-IG-09: Cancel Invoices
**Description**: System shall allow canceling invoices  
**Priority**: Medium  
**Input**: Invoice ID  
**Processing**: Change status to canceled  
**Output**: Canceled invoice  
**Success Criteria**: Canceled invoices excluded from balances

#### FR-IG-10: Prevent Modifying Issued Invoices
**Description**: System shall prevent modifying issued invoices  
**Priority**: High  
**Input**: Edit attempt on issued invoice  
**Processing**: Check status  
**Output**: Error message  
**Success Criteria**: Issued invoices immutable

---

### 2.6 Payment Tracking (FR-PT)

#### FR-PT-01: Record Payments
**Description**: System shall allow recording payment transactions  
**Priority**: High  
**Input**: Amount, method, date, student, optional invoice  
**Processing**: Validate, create payment record  
**Output**: Payment record  
**Success Criteria**: Payment saved correctly

#### FR-PT-02: Support Payment Methods
**Description**: System shall support "cash" and "bank" methods  
**Priority**: Medium  
**Input**: Method selection  
**Processing**: Validate enum value  
**Output**: Payment with method  
**Success Criteria**: Both methods available

#### FR-PT-03: Link Payments to Invoices
**Description**: System shall allow linking payments to specific invoices  
**Priority**: High  
**Input**: Payment data with invoice ID  
**Processing**: Create payment with invoice reference  
**Output**: Linked payment  
**Success Criteria**: Payment associated with invoice

#### FR-PT-04: Auto-mark Invoices Paid
**Description**: System shall mark invoices as paid when fully paid  
**Priority**: High  
**Input**: Payment linking to invoice  
**Processing**: Sum payments, compare to invoice total, update status  
**Output**: Invoice status changed to paid  
**Success Criteria**: Status automatically updated

#### FR-PT-05: Calculate Student Balances
**Description**: System shall calculate total invoiced minus total paid  
**Priority**: High  
**Input**: Student ID  
**Processing**: Sum invoice totals, sum payments, subtract  
**Output**: Balance amount  
**Success Criteria**: Balance mathematically correct

#### FR-PT-06: Generate Debtor List
**Description**: System shall provide list of students with negative balances  
**Priority**: Medium  
**Input**: None  
**Processing**: Calculate balances for all students, filter < 0  
**Output**: List of debtors with amounts  
**Success Criteria**: Accurate debtor identification

---

### 2.7 Settings Management (FR-SET)

#### FR-SET-01: Store Organization Details
**Description**: System shall store organization name and address  
**Priority**: Medium  
**Input**: Name, address strings  
**Processing**: Save to settings singleton  
**Output**: Updated settings  
**Success Criteria**: Details appear on invoices

#### FR-SET-02: Configure Invoice Prefix
**Description**: System shall allow customizing invoice prefix  
**Priority**: Medium  
**Input**: Prefix string  
**Processing**: Validate, save to settings  
**Output**: Updated prefix  
**Success Criteria**: New prefix used in invoice numbers

#### FR-SET-03: Maintain Sequential Numbering
**Description**: System shall track next sequence number  
**Priority**: Critical  
**Input**: None (automatic)  
**Processing**: Increment after each invoice issued  
**Output**: Next available sequence  
**Success Criteria**: No duplicate numbers

#### FR-SET-04: Configure Currency and Locale
**Description**: System shall support currency and locale settings  
**Priority**: Low  
**Input**: Currency code, locale code  
**Processing**: Save to settings  
**Output**: Updated settings  
**Success Criteria**: Settings available for future use

#### FR-SET-05: Auto-issue Feature
**Description**: System shall support auto-issue toggle  
**Priority**: Low  
**Input**: Boolean flag  
**Processing**: Save to settings  
**Output**: Auto-issue configuration  
**Success Criteria**: Feature flag persisted

---

## 3. Non-Functional Requirements

### 3.1 Performance (NFR-PERF)

#### NFR-PERF-01: Startup Time
**Requirement**: Application shall start within 5 seconds  
**Measurement**: Time from launch to UI ready  
**Priority**: Medium

#### NFR-PERF-02: UI Responsiveness
**Requirement**: UI operations shall complete within 1 second  
**Measurement**: Time from user action to UI update  
**Priority**: High

#### NFR-PERF-03: PDF Generation Speed
**Requirement**: PDF generation shall complete within 3 seconds  
**Measurement**: Time from issue command to PDF saved  
**Priority**: Medium

#### NFR-PERF-04: Scalability
**Requirement**: System shall handle at least 1000 students efficiently  
**Measurement**: Query times remain < 1 second  
**Priority**: Medium

---

### 3.2 Usability (NFR-USE)

#### NFR-USE-01: Intuitive Interface
**Requirement**: UI shall require minimal training  
**Measurement**: User can complete basic tasks without documentation  
**Priority**: High

#### NFR-USE-02: Clear Error Messages
**Requirement**: Error messages shall be descriptive and actionable  
**Measurement**: Users understand what went wrong and how to fix it  
**Priority**: High

#### NFR-USE-03: Consistent Terminology
**Requirement**: Same concepts shall use same terms throughout  
**Measurement**: No conflicting terminology in UI  
**Priority**: Medium

#### NFR-USE-04: Cyrillic Support
**Requirement**: PDFs shall render Cyrillic characters correctly  
**Measurement**: All Cyrillic text readable in PDFs  
**Priority**: High

---

### 3.3 Reliability (NFR-REL)

#### NFR-REL-01: Data Loss Prevention
**Requirement**: System shall prevent data loss through error handling  
**Measurement**: No data lost during normal operations  
**Priority**: Critical

#### NFR-REL-02: Database Integrity
**Requirement**: System shall maintain integrity through transactions  
**Measurement**: Database remains consistent after crashes  
**Priority**: Critical

#### NFR-REL-03: Input Validation
**Requirement**: System shall validate all user inputs  
**Measurement**: 100% of inputs validated  
**Priority**: High

#### NFR-REL-04: Backup-Friendly Storage
**Requirement**: Data shall be stored in easily backupable format  
**Measurement**: Single SQLite file can be copied  
**Priority**: Medium

---

### 3.4 Security (NFR-SEC)

#### NFR-SEC-01: XSS Prevention
**Requirement**: System shall sanitize text inputs  
**Measurement**: HTML characters escaped  
**Priority**: Critical

#### NFR-SEC-02: Input Validation
**Requirement**: System shall validate numeric inputs  
**Measurement**: Invalid numbers rejected  
**Priority**: High

#### NFR-SEC-03: Local Data Storage
**Requirement**: Data shall remain on local machine  
**Measurement**: No network transmission  
**Priority**: High

#### NFR-SEC-04: SQL Injection Prevention
**Requirement**: System shall use prepared statements  
**Measurement**: ent ORM generates parameterized queries  
**Priority**: Critical

---

### 3.5 Maintainability (NFR-MAINT)

#### NFR-MAINT-01: Code Standards
**Requirement**: Code shall follow language best practices  
**Measurement**: Passes linter checks  
**Priority**: Medium

#### NFR-MAINT-02: Separation of Concerns
**Requirement**: System shall separate presentation, logic, data layers  
**Measurement**: Clear package boundaries  
**Priority**: High

#### NFR-MAINT-03: Documentation
**Requirement**: Code and system shall be documented  
**Measurement**: Documentation exists and is current  
**Priority**: High

#### NFR-MAINT-04: Version Control
**Requirement**: Code shall be version controlled  
**Measurement**: Git repository with history  
**Priority**: High

---

### 3.6 Portability (NFR-PORT)

#### NFR-PORT-01: Cross-Platform Support
**Requirement**: System shall run on Windows, macOS, Linux  
**Measurement**: Builds and runs on all three platforms  
**Priority**: High

#### NFR-PORT-02: Cross-Platform Libraries
**Requirement**: System shall use cross-platform dependencies  
**Measurement**: No platform-specific code  
**Priority**: High

#### NFR-PORT-03: Path Conventions
**Requirement**: System shall follow platform conventions  
**Measurement**: Uses appropriate path separators and locations  
**Priority**: Medium

---

## 4. User Stories

### US-01: Add Student
**As** a school administrator  
**I want** to add new students  
**So that** I can track their enrollments and billing

**Acceptance Criteria**:
- Can enter full name, phone, email, notes
- Student appears in student list after creation
- Validation prevents empty names

### US-02: Create Course
**As** a school administrator  
**I want** to create courses with different pricing  
**So that** I can accommodate various teaching formats

**Acceptance Criteria**:
- Can select group or individual type
- Can set lesson and subscription prices
- Prices must be non-negative

### US-03: Enroll Student
**As** a school administrator  
**I want** to enroll students in courses  
**So that** I can configure their billing

**Acceptance Criteria**:
- Can select student and course
- Can choose per-lesson or subscription billing
- Can set custom discount percentage
- Cannot create duplicate enrollments

### US-04: Track Attendance
**As** a school administrator  
**I want** to track monthly attendance  
**So that** I can bill accurately for lessons

**Acceptance Criteria**:
- Can edit lesson counts for each month
- Can use "Add +1 to all" for quick updates
- Can lock months to prevent changes
- Locked months cannot be edited

### US-05: Generate Invoices
**As** a school administrator  
**I want** to generate invoices automatically  
**So that** I save time on billing

**Acceptance Criteria**:
- Drafts created from attendance and subscriptions
- Amounts calculated correctly with discounts
- Can review drafts before issuing

### US-06: Issue Invoices
**As** a school administrator  
**I want** to issue invoices with sequential numbers  
**So that** I maintain proper accounting

**Acceptance Criteria**:
- Numbers follow PREFIX-YYYYMM-SEQ format
- No gaps in sequence
- Cannot modify after issuing

### US-07: Export PDFs
**As** a school administrator  
**I want** to export invoices as PDFs  
**So that** I can send them to students

**Acceptance Criteria**:
- PDFs generated when issuing
- Cyrillic characters display correctly
- PDFs organized by date in file system

### US-08: Record Payments
**As** a school administrator  
**I want** to record payments  
**So that** I can track student balances

**Acceptance Criteria**:
- Can enter amount, method, date
- Can link to specific invoice
- Invoice status updates automatically when paid
- Balance calculations are correct

### US-09: Lock Months
**As** a school administrator  
**I want** to lock past months  
**So that** attendance cannot be accidentally changed

**Acceptance Criteria**:
- Can lock any month
- Locked months show locked indicator
- Cannot edit locked attendance

### US-10: Apply Discounts
**As** a school administrator  
**I want** to apply discounts to enrollments  
**So that** I can offer special pricing

**Acceptance Criteria**:
- Can set 0-100% discount
- Discount applied in invoice calculation
- Discount shows on invoice line items

---

## 5. Constraints

### C-01: Single-User Application
**Constraint**: Must be single-user only  
**Rationale**: Simplifies architecture, no concurrent access needed  
**Impact**: No multi-user features

### C-02: Local Storage Only
**Constraint**: All data stored locally  
**Rationale**: No cloud dependencies, privacy, offline operation  
**Impact**: No sync, no remote access

### C-03: Offline Operation
**Constraint**: Must work without internet  
**Rationale**: Not dependent on external services  
**Impact**: No online features

### C-04: SQLite Database
**Constraint**: Must use SQLite  
**Rationale**: Zero-configuration, single-file, portable  
**Impact**: No client-server database features

### C-05: Cyrillic PDF Support
**Constraint**: PDFs must support Cyrillic  
**Rationale**: Target users need local language support  
**Impact**: Requires specific font files

### C-06: Wails Framework
**Constraint**: Must use Wails v2  
**Rationale**: Desktop integration, web tech, type-safe bindings  
**Impact**: Limited to Wails capabilities

---

## 6. Traceability Matrix

| Requirement | User Story | Test Case | Status |
|------------|-----------|-----------|--------|
| FR-SM-01 | US-01 | Manual | ✅ Implemented |
| FR-SM-06 | - | Unit | ✅ Tested |
| FR-CM-01 | US-02 | Manual | ✅ Implemented |
| FR-CM-03 | US-02 | Unit | ✅ Tested |
| FR-EM-01 | US-03 | Manual | ✅ Implemented |
| FR-EM-04 | US-10 | Unit | ✅ Tested |
| FR-AT-01 | US-04 | Manual | ✅ Implemented |
| FR-AT-04 | US-09 | Manual | ✅ Implemented |
| FR-IG-01 | US-05 | Manual | ✅ Implemented |
| FR-IG-02 | US-06 | Manual | ✅ Implemented |
| FR-IG-05 | US-07 | Manual | ✅ Implemented |
| FR-PT-01 | US-08 | Manual | ✅ Implemented |
| FR-PT-04 | US-08 | Manual | ✅ Implemented |
| NFR-SEC-01 | - | Unit | ✅ Tested |
| NFR-SEC-04 | - | Arch | ✅ Implemented |

---

**Document Version**: 1.0  
**Last Updated**: December 2024  
**Status**: Complete
