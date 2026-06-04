import { getTransport, type EnrollmentDTO } from "./api";
export type { EnrollmentDTO } from "./api";

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
  discountPct: number,
  note: string
): Promise<EnrollmentDTO> {
  const transport = await getTransport();
  return transport.createEnrollment(studentId, courseId, billingMode, discountPct, note);
}

export async function updateEnrollment(
  enrollmentId: number,
  billingMode: EnrollmentDTO["billingMode"],
  discountPct: number,
  note: string
): Promise<EnrollmentDTO> {
  const transport = await getTransport();
  return transport.updateEnrollment(enrollmentId, billingMode, discountPct, note);
}
