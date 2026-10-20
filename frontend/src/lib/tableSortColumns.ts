import type { DebtorDTO } from "./payments";
import type { InvoiceListItemView } from "./invoices";
import type { CourseDTO } from "./courses";
import type { EnrollmentDTO } from "./enrollments";
import type { TranslateFn } from "./i18n";
import { priceInCents, type SortColumn } from "./tableSort";

export function debtorSortColumns(t: TranslateFn): SortColumn<DebtorDTO>[] {
  return [
    { key: "student", label: t("field.student"), value: (row) => row.studentName },
    { key: "debt", label: t("field.debtEur"), value: (row) => priceInCents(row.debt) },
    { key: "total", label: t("field.totalEur"), value: (row) => priceInCents(row.totalInvoiced) },
    { key: "paid", label: t("field.paidEur"), value: (row) => priceInCents(row.totalPaid) },
  ];
}

export function invoiceSortColumns(
  t: TranslateFn,
  invoiceStatusLabel: (status: string) => string
): SortColumn<InvoiceListItemView>[] {
  return [
    { key: "student", label: t("field.student"), value: (row) => row.studentName },
    { key: "period", label: t("field.period"), value: (row) => row.year * 12 + row.month },
    { key: "total", label: `${t("field.amount")} (EUR)`, value: (row) => priceInCents(row.total) },
    { key: "status", label: t("field.status"), value: (row) => invoiceStatusLabel(row.status) },
    { key: "number", label: t("field.number"), value: (row) => row.number },
  ];
}

export function courseSortColumns(
  t: TranslateFn,
  courseTypeLabel: (type: string) => string
): SortColumn<CourseDTO>[] {
  return [
    { key: "teacher", label: t("field.teacher"), value: (row) => row.teacherName },
    { key: "type", label: t("field.type"), value: (row) => courseTypeLabel(row.type) },
    {
      key: "rate",
      label: `${t("field.lessonPrice")} (EUR)`,
      value: (row) => priceInCents(row.lessonPrice),
    },
    {
      key: "subscription",
      label: `${t("field.subscriptionPrice")} (EUR)`,
      value: (row) => priceInCents(row.subscriptionPrice),
    },
  ];
}

export function enrollmentSortColumns(
  t: TranslateFn,
  billingModeLabel: (mode: string) => string
): SortColumn<EnrollmentDTO>[] {
  return [
    { key: "student", label: t("field.student"), value: (row) => row.studentName },
    {
      key: "billing",
      label: t("field.billing"),
      value: (row) => billingModeLabel(row.billingMode),
    },
    {
      key: "rate",
      label: t("field.lessonPriceOverride"),
      value: (row) =>
        priceInCents(
          row.billingMode === "per_lesson" ? row.lessonPriceOverride : row.subscriptionLessonPrice
        ),
    },
    { key: "materials", label: t("field.materials"), value: (row) => row.chargeMaterials },
    { key: "note", label: t("field.note"), value: (row) => row.note },
  ];
}
