import {
  AttendanceListPerLesson,
  AttendanceUpsert,
  AttendanceAddOne,
  DevSeed,
  DevReset,
  EnrollmentDelete,
} from "../../wailsjs/go/main/App";

export type Row = {
  enrollmentId: number;
  studentId: number;
  studentName: string;
  courseId: number;
  courseName: string;
  courseType: "group" | "individual";
  lessonPrice: number;
  count: number;
};

function normalizeCourseId(courseId?: number): number | undefined {
  return typeof courseId === "number" && courseId > 0 ? courseId : undefined;
}

export function devSeed() {
  return DevSeed();
}

export function devReset() {
  return DevReset();
}

export async function fetchRows(year: number, month: number, courseId?: number): Promise<Row[]> {
  const cid = normalizeCourseId(courseId);
  return (await AttendanceListPerLesson(year, month, cid)) as Row[];
}

export function saveCount(
  studentId: number,
  courseId: number,
  year: number,
  month: number,
  count: number
): Promise<void> {
  return AttendanceUpsert(studentId, courseId, year, month, count);
}

export function addOneMass(year: number, month: number, courseId?: number) {
  const cid = normalizeCourseId(courseId);
  return AttendanceAddOne(year, month, cid);
}

export function deleteEnrollment(enrollmentId: number) {
  return EnrollmentDelete(enrollmentId);
}
