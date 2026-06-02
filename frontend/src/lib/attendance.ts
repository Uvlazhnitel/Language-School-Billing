import { BillingMode, CourseType, InvoiceStatus } from "./constants";
import { getTransport, type Row } from "./api";
export type { Row } from "./api";

function normalizeCourseId(courseId?: number): number | undefined {
  return typeof courseId === "number" && courseId > 0 ? courseId : undefined;
}

export async function fetchRows(year: number, month: number, courseId?: number): Promise<Row[]> {
  const cid = normalizeCourseId(courseId);
  const transport = await getTransport();
  return transport.fetchAttendanceRows(year, month, cid);
}

export async function saveHours(
  studentId: number,
  courseId: number,
  year: number,
  month: number,
  hours: number
): Promise<void> {
  const transport = await getTransport();
  return transport.saveAttendanceHours(studentId, courseId, year, month, hours);
}

export async function deleteEnrollment(enrollmentId: number) {
  const transport = await getTransport();
  return transport.deleteEnrollment(enrollmentId);
}
