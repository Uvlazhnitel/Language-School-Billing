import {
  EnrollmentList,
  EnrollmentCreate,
  EnrollmentUpdate,
} from "../../wailsjs/go/main/App";

export type EnrollmentDTO = {
  id: number;
  studentId: number;
  studentName: string;
  courseId: number;
  courseName: string;
  billingMode: "subscription" | "per_lesson";
  discountPct: number;
  note: string;
};

function toNullableId(id?: number): number | null {
  return typeof id === "number" && id > 0 ? id : null;
}

export async function listEnrollments(
  studentId?: number,
  courseId?: number
): Promise<EnrollmentDTO[]> {
  return (await EnrollmentList(toNullableId(studentId), toNullableId(courseId))) as EnrollmentDTO[];
}

export function createEnrollment(
  studentId: number,
  courseId: number,
  billingMode: EnrollmentDTO["billingMode"],
  discountPct: number,
  note: string
): Promise<EnrollmentDTO> {
  return EnrollmentCreate(studentId, courseId, billingMode, discountPct, note) as Promise<EnrollmentDTO>;
}

export function updateEnrollment(
  enrollmentId: number,
  billingMode: EnrollmentDTO["billingMode"],
  discountPct: number,
  note: string
): Promise<EnrollmentDTO> {
  return EnrollmentUpdate(enrollmentId, billingMode, discountPct, note) as Promise<EnrollmentDTO>;
}
