import {
  AttendanceListPerLesson,
  AttendanceUpsert,
  EnrollmentDelete,
} from "../../wailsjs/go/main/App";
import { BillingMode, CourseType, InvoiceStatus } from "./constants";

export type Row = {
  enrollmentId: number;
  studentId: number;
  studentName: string;
  courseId: number;
  courseName: string;
  courseType: CourseType;
  billingMode: BillingMode;
  lessonPrice: number;
  hours: number;
  hasRecord: boolean;
  canDelete: boolean;
  attendanceLocked: boolean;
  invoiceStatus?: InvoiceStatus;
};

function normalizeCourseId(courseId?: number): number | undefined {
  return typeof courseId === "number" && courseId > 0 ? courseId : undefined;
}

export async function fetchRows(year: number, month: number, courseId?: number): Promise<Row[]> {
  const cid = normalizeCourseId(courseId);
  return (await AttendanceListPerLesson(year, month, cid)) as Row[];
}

export function saveHours(
  studentId: number,
  courseId: number,
  year: number,
  month: number,
  hours: number
): Promise<void> {
  return AttendanceUpsert(studentId, courseId, year, month, hours);
}

export function deleteEnrollment(enrollmentId: number) {
  return EnrollmentDelete(enrollmentId);
}
