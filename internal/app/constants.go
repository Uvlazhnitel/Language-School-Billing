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
