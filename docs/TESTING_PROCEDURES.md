# Comprehensive Testing Procedures

**Language School Billing System**

---

## 1. Testing Overview

### 1.1 Testing Strategy

The testing strategy employs multiple testing levels:

1. **Unit Testing**: Individual functions (validation, helpers)
2. **Manual Testing**: End-to-end user workflows
3. **Integration Testing**: Service-level operations
4. **Regression Testing**: Verify no existing features broken

### 1.2 Test Environment

**Requirements**:
- Go 1.22+ installed
- Repository cloned
- Dependencies installed (`go mod download`)
- ent code generated (`go generate ./ent`)

---

## 2. Unit Testing

### 2.1 Test Location

All unit tests are in: `internal/validation/validate_test.go`

### 2.2 Running Unit Tests

#### Basic Test Run
```bash
cd /path/to/Language-School-Billing
go test ./...
```

**Expected Output**:
```
?       langschool       [no test files]
ok      langschool/internal/validation  0.002s
```

#### Verbose Output
```bash
go test -v ./...
```

**Expected Output**:
```
=== RUN   TestSanitizeInput
=== RUN   TestSanitizeInput/normal_text
=== RUN   TestSanitizeInput/text_with_spaces
=== RUN   TestSanitizeInput/HTML_tags
=== RUN   TestSanitizeInput/special_characters
=== RUN   TestSanitizeInput/quotes
--- PASS: TestSanitizeInput (0.00s)
...
PASS
ok      langschool/internal/validation  0.002s
```

#### Coverage Report
```bash
go test -cover ./...
```

**Expected Output**:
```
ok      langschool/internal/validation  0.002s  coverage: 100.0% of statements
```

#### Detailed Coverage Report
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

This opens an HTML report showing line-by-line coverage.

#### Race Condition Detection
```bash
go test -race ./...
```

Checks for data races in concurrent code.

#### Run Specific Test
```bash
go test -v ./internal/validation/... -run TestSanitizeInput
```

#### Run Specific Test Case
```bash
go test -v ./internal/validation/... -run TestSanitizeInput/HTML_tags
```

### 2.3 Test Cases

#### TestSanitizeInput (5 cases)

**TC-01: Normal Text**
- Input: `"John Doe"`
- Expected: `"John Doe"`
- Purpose: Verify normal text unchanged

**TC-02: Text with Spaces**
- Input: `"  John Doe  "`
- Expected: `"John Doe"`
- Purpose: Verify trimming

**TC-03: HTML Tags**
- Input: `"<script>alert('xss')</script>"`
- Expected: `"&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"`
- Purpose: Verify XSS prevention

**TC-04: Special Characters**
- Input: `"Test & <test>"`
- Expected: `"Test &amp; &lt;test&gt;"`
- Purpose: Verify HTML escaping

**TC-05: Quotes**
- Input: `"He said \"Hello\""`
- Expected: `"He said &#34;Hello&#34;"`
- Purpose: Verify quote escaping

#### TestValidateNonEmpty (4 cases)

**TC-06: Valid Value**
- Input: `"John Doe"`
- Expected: No error
- Purpose: Verify valid input accepted

**TC-07: Empty String**
- Input: `""`
- Expected: Error
- Purpose: Verify empty rejected

**TC-08: Only Spaces**
- Input: `"   "`
- Expected: Error
- Purpose: Verify whitespace-only rejected

**TC-09: With Spaces**
- Input: `"  valid  "`
- Expected: No error (trimmed first)
- Purpose: Verify trimming before validation

#### TestValidatePrices (4 cases)

**TC-10: Valid Positive Prices**
- Input: lesson=10.0, subscription=50.0
- Expected: No error
- Purpose: Verify positive prices accepted

**TC-11: Zero Prices**
- Input: lesson=0.0, subscription=0.0
- Expected: No error
- Purpose: Verify zero is valid

**TC-12: Negative Lesson Price**
- Input: lesson=-10.0, subscription=50.0
- Expected: Error
- Purpose: Verify negative rejected

**TC-13: Negative Subscription Price**
- Input: lesson=10.0, subscription=-50.0
- Expected: Error
- Purpose: Verify negative rejected

#### TestValidateDiscountPct (5 cases)

**TC-14: Valid Discount**
- Input: 25.0
- Expected: No error
- Purpose: Verify valid percentage accepted

**TC-15: Zero Discount**
- Input: 0.0
- Expected: No error
- Purpose: Verify zero is valid

**TC-16: 100% Discount**
- Input: 100.0
- Expected: No error
- Purpose: Verify 100% is valid

**TC-17: Negative Discount**
- Input: -10.0
- Expected: Error
- Purpose: Verify negative rejected

**TC-18: Over 100% Discount**
- Input: 110.0
- Expected: Error
- Purpose: Verify over 100% rejected

---

## 3. Manual Testing

### 3.1 Test Environment Setup

1. Build and run application:
```bash
wails dev
```

2. Application opens with empty database

3. Prepare test data (optional):
   - Students: 2-3 test students
   - Courses: 2 test courses (one group, one individual)
   - Enrollments: Link students to courses

### 3.2 Student Management Tests

#### MT-01: Create Student

**Steps**:
1. Click "Students" tab
2. Click "Add Student" button
3. Enter full name: "Test Student 1"
4. Enter phone: "+371 12345678"
5. Enter email: "test@example.com"
6. Enter note: "Test student"
7. Click "Save"

**Expected**:
- Student appears in list
- All fields populated correctly
- Active status = true

#### MT-02: Update Student

**Steps**:
1. Click "Edit" on existing student
2. Change name to "Updated Name"
3. Click "Save"

**Expected**:
- Student name updated in list
- Other fields unchanged

#### MT-03: Validate Empty Name

**Steps**:
1. Click "Add Student"
2. Leave name empty
3. Click "Save"

**Expected**:
- Error message: "Full name cannot be empty"
- Student not created

#### MT-04: Deactivate Student

**Steps**:
1. Click "Deactivate" on student
2. Confirm action

**Expected**:
- Student marked as inactive
- Visual indicator (grayed out or marked)

#### MT-05: Delete Student

**Steps**:
1. Create new student with no enrollments
2. Click "Delete"
3. Confirm action

**Expected**:
- Student removed from list

#### MT-06: Prevent Delete with Enrollments

**Steps**:
1. Create student with enrollment
2. Try to delete student

**Expected**:
- Error message preventing deletion

### 3.3 Course Management Tests

#### MT-07: Create Group Course

**Steps**:
1. Click "Courses" tab
2. Click "Add Course"
3. Enter name: "English Group A1"
4. Select type: "Group"
5. Enter lesson price: 5.00
6. Enter subscription price: 40.00
7. Click "Save"

**Expected**:
- Course appears in list with correct type

#### MT-08: Create Individual Course

**Steps**:
1. Create course with type "Individual"
2. Set lesson price: 15.00
3. Set subscription price: 120.00

**Expected**:
- Course created with individual type

#### MT-09: Validate Negative Prices

**Steps**:
1. Try to create course with lesson price: -5.00

**Expected**:
- Error message: "Prices cannot be negative"

### 3.4 Enrollment Management Tests

#### MT-10: Create Per-Lesson Enrollment

**Steps**:
1. Click "Enrollments" tab
2. Click "Add Enrollment"
3. Select student
4. Select course
5. Select billing mode: "Per Lesson"
6. Set discount: 10%
7. Click "Save"

**Expected**:
- Enrollment created
- Billing mode shown correctly

#### MT-11: Create Subscription Enrollment

**Steps**:
1. Create enrollment with billing mode: "Subscription"

**Expected**:
- Subscription billing mode applied

#### MT-12: Validate Discount Range

**Steps**:
1. Try to create enrollment with discount: 150%

**Expected**:
- Error message: "Discount must be between 0 and 100"

### 3.5 Attendance Tracking Tests

#### MT-13: Add Attendance

**Steps**:
1. Click "Attendance" tab
2. Select current month
3. Find student-course pair
4. Click edit, enter lessons: 8
5. Save

**Expected**:
- Attendance saved
- Shows 8 lessons

#### MT-14: Bulk Update

**Steps**:
1. Select multiple students
2. Click "Add +1 to All"

**Expected**:
- All selected attendance incremented by 1

#### MT-15: Lock Month

**Steps**:
1. Click "Lock Month" for current month
2. Try to edit attendance

**Expected**:
- Month marked as locked
- Cannot edit attendance
- Error message when trying

#### MT-16: Unlock Month

**Steps**:
1. Click "Unlock Month"
2. Try to edit attendance

**Expected**:
- Can edit attendance again

### 3.6 Invoice Generation Tests

#### MT-17: Generate Drafts

**Pre-requisites**:
- Students with enrollments exist
- Attendance data entered for month

**Steps**:
1. Click "Invoices" tab
2. Select year and month
3. Click "Generate Drafts"

**Expected**:
- Draft invoices created for students with:
  - Per-lesson enrollments: lessons * price
  - Subscription enrollments: subscription price
- Discounts applied correctly
- Totals calculated correctly

#### MT-18: Verify Draft Calculation

**Example**:
- Student has enrollment: per-lesson, 10% discount
- Course lesson price: 5.00
- Attendance: 8 lessons
- Expected total: 8 * 5.00 * 0.90 = 36.00

**Steps**:
1. Check draft invoice total

**Expected**:
- Total matches calculation

#### MT-19: Issue Single Invoice

**Steps**:
1. Select a draft invoice
2. Click "Issue"

**Expected**:
- Invoice status changes to "Issued"
- Invoice number assigned: LS-202412-001
- PDF generated
- Cannot edit anymore

#### MT-20: Verify PDF

**Steps**:
1. Navigate to `~/LangSchool/Invoices/2024/12/`
2. Open PDF file

**Expected**:
- PDF contains:
  - Organization details (if set)
  - Invoice number
  - Date
  - Student name
  - Line items with descriptions
  - Quantities and prices
  - Total amount
- Cyrillic characters (if any) display correctly

#### MT-21: Sequential Numbering

**Steps**:
1. Issue invoice (gets LS-202412-001)
2. Issue another invoice

**Expected**:
- Second invoice gets LS-202412-002
- No gaps in sequence

#### MT-22: Batch Issue

**Steps**:
1. Generate multiple drafts
2. Click "Issue All"

**Expected**:
- All drafts issued
- Sequential numbers assigned
- All PDFs generated

#### MT-23: Cancel Invoice

**Steps**:
1. Select issued invoice
2. Click "Cancel"
3. Confirm

**Expected**:
- Invoice status changes to "Canceled"
- Excluded from balance calculations

### 3.7 Payment Tracking Tests

#### MT-24: Record Cash Payment

**Steps**:
1. Click "Payments" tab
2. Click "Add Payment"
3. Select student
4. Enter amount: 36.00
5. Select method: "Cash"
6. Select date
7. Link to invoice (optional)
8. Click "Save"

**Expected**:
- Payment recorded
- Shows in payment list

#### MT-25: Auto-mark Invoice Paid

**Steps**:
1. Have issued invoice for 36.00
2. Record payment for 36.00 linked to invoice

**Expected**:
- Invoice status automatically changes to "Paid"

#### MT-26: Partial Payment

**Steps**:
1. Invoice total: 50.00
2. Record payment: 30.00 linked to invoice

**Expected**:
- Invoice remains "Issued" (not fully paid)
- Balance shows 20.00 owed

#### MT-27: Check Balance

**Steps**:
1. View student details or balance section
2. Check balance amount

**Expected**:
- Balance = Total invoiced - Total paid
- Correct calculation

#### MT-28: Debtor List

**Steps**:
1. Have students with unpaid invoices
2. View "Debtors" section

**Expected**:
- List shows students with negative balance
- Amounts are correct

### 3.8 Settings Tests

#### MT-29: Configure Organization

**Steps**:
1. Open settings
2. Enter organization name: "My Language School"
3. Enter address: "123 Main St, City"
4. Save

**Expected**:
- Settings saved
- Appears on next generated invoices

#### MT-30: Change Invoice Prefix

**Steps**:
1. Change prefix from "LS" to "MLS"
2. Save
3. Issue new invoice

**Expected**:
- New invoice number: MLS-202412-001

---

## 4. Integration Testing

### 4.1 End-to-End Workflow

#### IT-01: Complete Billing Cycle

**Steps**:
1. Create student "John Doe"
2. Create course "English A1" (group, 5€/lesson)
3. Enroll John in English A1 (per-lesson, 10% discount)
4. Add attendance: 8 lessons for December
5. Generate draft invoices for December
6. Verify John's invoice:
   - Expected: 8 * 5 * 0.90 = 36.00€
7. Issue John's invoice
8. Verify PDF created
9. Record payment: 36.00€ cash
10. Verify invoice marked paid
11. Verify balance is 0

**Expected**:
- Complete cycle works without errors
- All calculations correct
- Data consistent across all views

#### IT-02: Multiple Students Workflow

**Steps**:
1. Create 3 students
2. Create 2 courses
3. Enroll students in various courses
4. Add attendance for all
5. Generate drafts
6. Issue all invoices
7. Record partial payments for some
8. Check debtor list

**Expected**:
- All operations complete successfully
- Debtors correctly identified
- Balances accurate

---

## 5. Regression Testing

### 5.1 After Code Changes

**Checklist**:
- [ ] Run all unit tests: `go test ./...`
- [ ] Test student CRUD operations
- [ ] Test invoice generation
- [ ] Test PDF generation with Cyrillic
- [ ] Test payment recording
- [ ] Test balance calculations
- [ ] Verify no errors in console

### 5.2 Critical Paths

**Must Test After Any Change**:
1. Invoice generation and issuing
2. Sequential numbering
3. PDF generation
4. Payment recording and status updates
5. Balance calculations

---

## 6. Performance Testing

### 6.1 Startup Time

**Test**:
```bash
time wails dev
```

**Expected**: Application ready in < 5 seconds

### 6.2 Large Dataset

**Test**:
1. Create 100 students
2. Create 10 courses
3. Create 500 enrollments
4. Add attendance for all
5. Generate invoices

**Expected**:
- All operations complete in < 1 second
- No UI lag

### 6.3 PDF Generation

**Test**:
1. Issue 100 invoices

**Expected**:
- Each PDF generates in < 3 seconds
- No memory issues

---

## 7. Security Testing

### 7.1 XSS Prevention

**Test**:
1. Try to create student with name: `<script>alert('xss')</script>`

**Expected**:
- Name saved as escaped: `&lt;script&gt;...`
- No script execution

### 7.2 SQL Injection

**Test**:
1. Try to create student with name: `John'; DROP TABLE students; --`

**Expected**:
- Name saved as-is (ent uses parameterized queries)
- No SQL injection occurs
- No error

### 7.3 Invalid Input

**Test**:
- Negative prices
- Discount > 100%
- Empty required fields

**Expected**:
- Validation errors displayed
- No invalid data saved

---

## 8. Test Reporting

### 8.1 Test Execution Report Template

```
Test Execution Report
Date: YYYY-MM-DD
Tester: [Name]
Version: [Version]

Unit Tests:
- Total: 19
- Passed: __
- Failed: __
- Coverage: __%

Manual Tests:
- Total: __
- Passed: __
- Failed: __

Critical Issues:
1. [Description]
2. [Description]

Non-Critical Issues:
1. [Description]

Notes:
[Additional observations]
```

### 8.2 Bug Report Template

```
Bug Report
ID: BUG-XXX
Date: YYYY-MM-DD
Reported by: [Name]

Title: [Short description]

Severity: Critical / High / Medium / Low

Steps to Reproduce:
1. [Step]
2. [Step]
3. [Step]

Expected Behavior:
[What should happen]

Actual Behavior:
[What actually happens]

Environment:
- OS: [Windows/macOS/Linux]
- Version: [App version]

Screenshots/Logs:
[Attach if available]
```

---

## 9. Continuous Testing

### 9.1 Pre-Commit Checklist

Before committing code:
- [ ] Run `go test ./...`
- [ ] Run `go fmt ./...`
- [ ] Run `golangci-lint run`
- [ ] Test affected functionality manually
- [ ] Update tests if behavior changed

### 9.2 Pre-Release Checklist

Before releasing:
- [ ] All unit tests pass
- [ ] Complete end-to-end workflow tested
- [ ] PDF generation tested (with Cyrillic)
- [ ] Cross-platform testing (if possible)
- [ ] Performance acceptable
- [ ] No critical bugs
- [ ] Documentation updated

---

## 10. Test Data

### 10.1 Sample Test Data

**Students**:
- John Doe, +371 12345678, john@example.com
- Jane Smith, +371 87654321, jane@example.com
- Петр Иванов, +371 11111111, petr@example.com (Cyrillic)

**Courses**:
- English A1 (Group), 5€/lesson, 40€/month
- German B1 (Individual), 15€/lesson, 120€/month
- French A2 (Group), 6€/lesson, 45€/month

**Enrollments**:
- John → English A1, per-lesson, 10% discount
- Jane → English A1, subscription, 0% discount
- Петр → German B1, per-lesson, 15% discount

### 10.2 Test Scenarios

**Scenario 1: Regular Monthly Billing**
- 3 students, 2 courses, mixed billing modes
- Full attendance for month
- Generate, issue, collect payments

**Scenario 2: Partial Attendance**
- Some students missed lessons
- Varied attendance counts
- Verify correct billing amounts

**Scenario 3: Multiple Courses per Student**
- Student enrolled in 2+ courses
- Different billing modes per enrollment
- Invoice combines all enrollments

---

**Document Version**: 1.0  
**Last Updated**: December 2024  
**Status**: Complete
