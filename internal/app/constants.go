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
	InvoiceStatusDraft    = "draft"    // Draft: not yet issued, can be modified or deleted
	InvoiceStatusIssued   = "issued"   // Issued: has been assigned a number and sent to student
	InvoiceStatusPaid     = "paid"      // Paid: fully paid by the student
	InvoiceStatusCanceled = "canceled" // Canceled: voided invoice, cannot be paid
)

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
