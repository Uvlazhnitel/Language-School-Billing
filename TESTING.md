# Manual Testing Guide

This guide explains how to manually run tests in the Language School Billing application.

## Prerequisites

- Go 1.22+ installed
- Git repository cloned
- Terminal/command line access

## Running Tests

### 1. Run All Tests

To run all tests in the project:

```bash
cd /path/to/Language-School-Billing
go test ./...
```

Expected output:
```
?       langschool       [no test files]
ok      langschool/internal/validation  0.002s
```

### 2. Run Tests with Verbose Output

To see detailed test results including which test cases pass:

```bash
go test -v ./...
```

Expected output:
```
=== RUN   TestSanitizeInput
=== RUN   TestSanitizeInput/normal_text
=== RUN   TestSanitizeInput/text_with_spaces
=== RUN   TestSanitizeInput/HTML_tags
=== RUN   TestSanitizeInput/special_characters
=== RUN   TestSanitizeInput/quotes
--- PASS: TestSanitizeInput (0.00s)
    --- PASS: TestSanitizeInput/normal_text (0.00s)
    --- PASS: TestSanitizeInput/text_with_spaces (0.00s)
    --- PASS: TestSanitizeInput/HTML_tags (0.00s)
    --- PASS: TestSanitizeInput/special_characters (0.00s)
    --- PASS: TestSanitizeInput/quotes (0.00s)
=== RUN   TestValidateNonEmpty
=== RUN   TestValidateNonEmpty/valid_value
=== RUN   TestValidateNonEmpty/empty_string
=== RUN   TestValidateNonEmpty/only_spaces
=== RUN   TestValidateNonEmpty/with_spaces
--- PASS: TestValidateNonEmpty (0.00s)
=== RUN   TestValidatePrices
--- PASS: TestValidatePrices (0.00s)
=== RUN   TestValidateDiscountPct
--- PASS: TestValidateDiscountPct (0.00s)
PASS
ok      langschool/internal/validation  0.002s
```

### 3. Run Specific Package Tests

To run tests only in the validation package:

```bash
go test ./internal/validation/...
```

Or with verbose output:

```bash
go test -v ./internal/validation/...
```

### 4. Run Tests with Coverage

To see test coverage:

```bash
go test -cover ./...
```

Expected output:
```
ok      langschool/internal/validation  0.002s  coverage: 100.0% of statements
```

To get detailed coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

This will open an HTML report in your browser showing which lines are covered by tests.

### 5. Run Tests with Race Detection

To check for race conditions:

```bash
go test -race ./...
```

This is useful for detecting concurrency issues.

### 6. Run a Specific Test

To run only one test function:

```bash
go test -v ./internal/validation/... -run TestSanitizeInput
```

To run a specific test case within a test:

```bash
go test -v ./internal/validation/... -run TestSanitizeInput/HTML_tags
```

## Available Test Suites

### Validation Package Tests

Location: `internal/validation/validate_test.go`

**Test Functions:**
1. `TestSanitizeInput` - Tests HTML escaping and XSS protection (5 test cases)
2. `TestValidateNonEmpty` - Tests empty value validation (4 test cases)
3. `TestValidatePrices` - Tests price validation (4 test cases)
4. `TestValidateDiscountPct` - Tests discount percentage validation (5 test cases)

**Total:** 19 test cases

## Common Test Commands

```bash
# Quick test run
go test ./...

# Verbose with all details
go test -v ./...

# Coverage report
go test -cover ./...

# Race detection (recommended before commits)
go test -race ./...

# Full coverage HTML report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Run only fast tests (skip slow integration tests if any)
go test -short ./...
```

## Continuous Integration

Tests are automatically run by GitHub Actions CI/CD pipeline on:
- Every push to `main`, `develop`, or `copilot/*` branches
- Every pull request to `main` or `develop`

See `.github/workflows/ci.yml` for the full CI configuration.

## Test Structure

Each test in this project follows the table-driven testing pattern:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        expected  string
        wantError bool
    }{
        {"test case 1", "input1", "output1", false},
        {"test case 2", "input2", "output2", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FunctionUnderTest(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Troubleshooting

### "no test files" message

This is normal for packages that don't have tests yet. The main application package and several internal packages don't have tests yet, which is why you see:
```
?       langschool       [no test files]
```

### Tests fail with "pattern all:frontend/dist: no matching files found"

This is a known issue with the embedded frontend assets. Tests in the main package are skipped for now. The validation package tests work fine.

### "cannot find package"

Make sure you're in the project root directory and run:
```bash
go mod download
```

## Next Steps

To add more tests:
1. Create a `*_test.go` file in the package you want to test
2. Write test functions starting with `Test`
3. Run `go test ./...` to verify they work
4. The CI pipeline will automatically run them on push

## Example: Running Tests Step by Step

```bash
# 1. Navigate to project
cd /home/user/Language-School-Billing

# 2. Run all tests with verbose output
go test -v ./...

# 3. Check coverage
go test -cover ./...

# 4. View detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 5. Run with race detection (recommended)
go test -race ./...
```

All tests should pass with 100% pass rate!
