import type { CourseDTO } from "./courses";
import { BillingModePerLesson, BillingModeSubscription } from "./constants";
import type { EnrollmentCreateInput, EnrollmentDTO } from "./api";

export type StudentOnboardingEnrollmentRow = {
  id: number;
  courseId: number;
  billingMode: EnrollmentDTO["billingMode"];
  chargeMaterials: boolean;
  lessonPrice: string;
  subscriptionPrice: string;
  note: string;
  settingsOpen: boolean;
};

export type StudentOnboardingEnrollmentRowPatch = Partial<
  Omit<StudentOnboardingEnrollmentRow, "id" | "courseId" | "billingMode">
>;

let nextRowId = 1;

export function createEmptyOnboardingEnrollmentRow(): StudentOnboardingEnrollmentRow {
  return {
    id: nextRowId++,
    courseId: 0,
    billingMode: BillingModePerLesson,
    chargeMaterials: true,
    lessonPrice: "0",
    subscriptionPrice: "0",
    note: "",
    settingsOpen: false,
  };
}

export function selectOnboardingCourse(
  row: StudentOnboardingEnrollmentRow,
  courseId: number,
  courses: CourseDTO[]
): StudentOnboardingEnrollmentRow {
  const course = courses.find((item) => item.id === courseId);
  return {
    ...row,
    courseId,
    billingMode: BillingModePerLesson,
    chargeMaterials: true,
    lessonPrice: String(course?.lessonPrice ?? 0),
    subscriptionPrice: String(course?.subscriptionPrice ?? 0),
    note: "",
    settingsOpen: false,
  };
}

export function changeOnboardingBillingMode(
  row: StudentOnboardingEnrollmentRow,
  billingMode: EnrollmentDTO["billingMode"],
  courses: CourseDTO[]
): StudentOnboardingEnrollmentRow {
  const course = courses.find((item) => item.id === row.courseId);
  const next = { ...row, billingMode };
  if (billingMode === BillingModePerLesson && Number(row.lessonPrice) === 0) {
    next.lessonPrice = String(course?.lessonPrice ?? 0);
  }
  if (billingMode === BillingModeSubscription && Number(row.subscriptionPrice) === 0) {
    next.subscriptionPrice = String(course?.subscriptionPrice ?? 0);
  }
  return next;
}

export function buildOnboardingEnrollmentInputs(rows: StudentOnboardingEnrollmentRow[]): {
  inputs: EnrollmentCreateInput[];
  errorKey?: "msg.lessonPriceOverrideRange" | "msg.subscriptionLessonPriceRange" | "msg.duplicateOnboardingCourse";
} {
  const inputs: EnrollmentCreateInput[] = [];
  const courseIds = new Set<number>();
  for (const row of rows) {
    if (row.courseId <= 0) continue;
    if (courseIds.has(row.courseId)) {
      return { inputs: [], errorKey: "msg.duplicateOnboardingCourse" };
    }
    courseIds.add(row.courseId);

    const lessonPrice = row.lessonPrice.trim() === "" ? 0 : Number(row.lessonPrice);
    const subscriptionPrice =
      row.subscriptionPrice.trim() === "" ? 0 : Number(row.subscriptionPrice);
    if (!Number.isFinite(lessonPrice) || lessonPrice < 0) {
      return { inputs: [], errorKey: "msg.lessonPriceOverrideRange" };
    }
    if (!Number.isFinite(subscriptionPrice) || subscriptionPrice < 0) {
      return { inputs: [], errorKey: "msg.subscriptionLessonPriceRange" };
    }
    inputs.push({
      courseId: row.courseId,
      billingMode: row.billingMode,
      chargeMaterials: row.chargeMaterials,
      lessonPriceOverride: row.billingMode === BillingModePerLesson ? lessonPrice : 0,
      subscriptionLessonPrice:
        row.billingMode === BillingModeSubscription ? subscriptionPrice : 0,
      note: row.note,
    });
  }
  return { inputs };
}
