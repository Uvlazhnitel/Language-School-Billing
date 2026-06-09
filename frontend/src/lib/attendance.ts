import { getTransport, type CourseMonthSubscriptionDTO, type Row } from "./api";
export type { CourseMonthSubscriptionDTO, Row } from "./api";

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

export async function listCourseMonthSubscriptions(
  year: number,
  month: number,
  courseId?: number
): Promise<CourseMonthSubscriptionDTO[]> {
  const cid = normalizeCourseId(courseId);
  const transport = await getTransport();
  return transport.listCourseMonthSubscriptions(year, month, cid);
}

export async function saveCourseMonthSubscriptionLessons(
  courseId: number,
  year: number,
  month: number,
  lessonsHeld: number
): Promise<CourseMonthSubscriptionDTO> {
  const transport = await getTransport();
  return transport.saveCourseMonthSubscriptionLessons(courseId, year, month, lessonsHeld);
}

export async function deleteEnrollment(enrollmentId: number, version: number) {
  const transport = await getTransport();
  return transport.deleteEnrollment(enrollmentId, version);
}
