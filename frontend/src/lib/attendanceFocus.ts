import type { Row } from "./attendance";

export type AttendanceFocusTarget = { studentId: number; courseId: number };

export function findAttendanceFocusEnrollmentId(
  rows: Row[],
  target: AttendanceFocusTarget | null | undefined
): number | null {
  if (!target) return null;
  return (
    rows.find(
      (row) => row.studentId === target.studentId && row.courseId === target.courseId
    )?.enrollmentId ?? null
  );
}
