// Constants for course types
export const CourseTypeGroup = "group" as const;
export const CourseTypeIndividual = "individual" as const;
export type CourseType = typeof CourseTypeGroup | typeof CourseTypeIndividual;

// Constants for billing modes
export const BillingModeSubscription = "subscription" as const;
export const BillingModePerLesson = "per_lesson" as const;
export type BillingMode = typeof BillingModeSubscription | typeof BillingModePerLesson;

// Constants for invoice statuses
export const InvoiceStatusDraft = "draft" as const;
export const InvoiceStatusIssued = "issued" as const;
export const InvoiceStatusPaid = "paid" as const;
export const InvoiceStatusCanceled = "canceled" as const;
export type InvoiceStatus =
  | typeof InvoiceStatusDraft
  | typeof InvoiceStatusIssued
  | typeof InvoiceStatusPaid
  | typeof InvoiceStatusCanceled;

// Constants for payment methods
export const PaymentMethodCash = "cash" as const;
export const PaymentMethodBank = "bank" as const;
export type PaymentMethod = typeof PaymentMethodCash | typeof PaymentMethodBank;
