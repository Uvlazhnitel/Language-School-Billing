import {
  getTransport,
  type EnrollmentBulkCreateResult,
  type EnrollmentCreateInput,
  type EnrollmentDTO,
} from "./api";
export type { EnrollmentBulkCreateResult, EnrollmentDTO } from "./api";

function toNullableId(id?: number): number | null {
  return typeof id === "number" && id > 0 ? id : null;
}

export async function listEnrollments(
  studentId?: number,
  courseId?: number
): Promise<EnrollmentDTO[]> {
  const transport = await getTransport();
  return transport.listEnrollments(toNullableId(studentId) ?? undefined, toNullableId(courseId) ?? undefined);
}

export async function createEnrollment(
  studentId: number,
  courseId: number,
  billingMode: EnrollmentDTO["billingMode"],
  chargeMaterials: boolean,
  lessonPriceOverride: number,
  subscriptionLessonPrice: number,
  note: string
): Promise<EnrollmentDTO> {
  const transport = await getTransport();
  return transport.createEnrollment(
    studentId,
    courseId,
    billingMode,
    chargeMaterials,
    lessonPriceOverride,
    subscriptionLessonPrice,
    note
  );
}

export async function createEnrollmentsBulk(
  studentId: number,
  enrollments: EnrollmentCreateInput[]
): Promise<EnrollmentBulkCreateResult> {
  const transport = await getTransport();
  return transport.createEnrollmentsBulk(studentId, enrollments);
}

export async function updateEnrollment(
  enrollmentId: number,
  version: number,
  billingMode: EnrollmentDTO["billingMode"],
  chargeMaterials: boolean,
  lessonPriceOverride: number,
  subscriptionLessonPrice: number,
  note: string
): Promise<EnrollmentDTO> {
  const transport = await getTransport();
  return transport.updateEnrollment(
    enrollmentId,
    version,
    billingMode,
    chargeMaterials,
    lessonPriceOverride,
    subscriptionLessonPrice,
    note
  );
}

export async function deleteEnrollment(enrollmentId: number, version: number) {
  const transport = await getTransport();
  return transport.deleteEnrollment(enrollmentId, version);
}
