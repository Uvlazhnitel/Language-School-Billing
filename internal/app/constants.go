package app

// Course types
const (
	CourseTypeGroup      = "group"
	CourseTypeIndividual = "individual"
)

// Billing modes
const (
	BillingModeSubscription = "subscription"
	BillingModePerLesson    = "per_lesson"
)

// Invoice statuses
const (
	InvoiceStatusDraft    = "draft"
	InvoiceStatusIssued   = "issued"
	InvoiceStatusPaid     = "paid"
	InvoiceStatusCanceled = "canceled"
)

// Payment methods
const (
	PaymentMethodCash = "cash"
	PaymentMethodBank = "bank"
)

// Database constants
const (
	// SettingsSingletonID is the ID used for the singleton Settings entity.
	// There is always exactly one Settings record with this ID in the database.
	SettingsSingletonID = 1
)

// File system permissions
const (
	// DirPermission is the default permission for created directories.
	// 0o755 means: owner can read/write/execute, group and others can read/execute.
	DirPermission = 0o755
)
