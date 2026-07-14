import type { EnrollmentDTO } from "./enrollments";

export function enrollmentPrice(enrollment: EnrollmentDTO): number {
  return enrollment.billingMode === "per_lesson"
    ? enrollment.lessonPriceOverride
    : enrollment.subscriptionLessonPrice;
}

export function parseEnrollmentPrice(value: string): number | null {
  const normalized = value.trim().replace(",", ".");
  if (normalized === "") return 0;
  if (!/^\d+(\.\d{0,2})?$/.test(normalized)) return null;

  const price = Number(normalized);
  return Number.isFinite(price) && price >= 0 ? price : null;
}

export function enrollmentPriceUpdateValues(enrollment: EnrollmentDTO, price: number) {
  return {
    billingMode: enrollment.billingMode,
    chargeMaterials: enrollment.chargeMaterials,
    lessonPriceOverride: enrollment.billingMode === "per_lesson" ? price : 0,
    subscriptionLessonPrice: enrollment.billingMode === "subscription" ? price : 0,
    note: enrollment.note,
  };
}
