// Package app contains shared constants and utilities used across the application.
package app

// Course types define the type of course offering.
const (
	CourseTypeGroup      = "group"      // Group course: multiple students attend together
	CourseTypeIndividual = "individual" // Individual course: one-on-one instruction
)

// Billing modes define how students are charged for courses.
const (
	BillingModeSubscription = "subscription" // Monthly subscription billing
	BillingModePerLesson    = "per_lesson"   // Pay per lesson attended
)

// Invoice statuses represent the lifecycle state of an invoice.
const (
	InvoiceStatusDraft            = "draft"              // Draft: not yet issued, can be modified or deleted
	InvoiceStatusIssuedPendingPDF = "issued_pending_pdf" // Issued with a number, but PDF is not ready yet
	InvoiceStatusIssued           = "issued"             // Issued with an актуальный PDF
	InvoiceStatusPaidPendingPDF   = "paid_pending_pdf"   // Fully paid, but PDF is not ready yet
	InvoiceStatusPaid             = "paid"               // Fully paid with an актуальный PDF
	InvoiceStatusCanceled         = "canceled"           // Canceled: voided invoice, cannot be paid
)

func InvoiceStatusIsPendingPDF(status string) bool {
	return status == InvoiceStatusIssuedPendingPDF || status == InvoiceStatusPaidPendingPDF
}

func InvoiceStatusIsIssuedFamily(status string) bool {
	return status == InvoiceStatusIssued || status == InvoiceStatusIssuedPendingPDF
}

func InvoiceStatusIsPaidFamily(status string) bool {
	return status == InvoiceStatusPaid || status == InvoiceStatusPaidPendingPDF
}

func InvoiceStatusIsProtected(status string) bool {
	return status != "" && status != InvoiceStatusDraft
}

func InvoiceStatusAllowsPayments(status string) bool {
	return InvoiceStatusIsIssuedFamily(status) || InvoiceStatusIsPaidFamily(status)
}

// Payment methods define how payments are made.
const (
	PaymentMethodCash = "cash" // Cash payment
	PaymentMethodBank = "bank" // Bank transfer or other electronic payment
)

// Database constants
const (
	// SettingsSingletonID is the ID used for the singleton Settings entity.
	// There is always exactly one Settings record with this ID in the database.
	// This pattern ensures application-wide settings are stored in a single record.
	SettingsSingletonID = 1
)

// File system permissions
const (
	// DirPermission is the default permission for created directories.
	// 0o755 means: owner can read/write/execute, group and others can read/execute.
	// This is a standard permission for application data directories.
	DirPermission = 0o755
)
