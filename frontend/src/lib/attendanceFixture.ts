import type { Row } from "./attendance";
export function attendanceRow(patch: Partial<Row> = {}): Row {
  return {
    enrollmentId: 1,
    enrollmentVersion: 1,
    studentId: 1,
    studentName: "Anna",
    courseId: 10,
    courseName: "English",
    courseType: "group",
    teacherId: 1,
    teacherName: "Teacher",
    billingMode: "per_lesson",
    lessonPrice: 25,
    subscriptionLessonPrice: 15,
    hours: 0,
    hasRecord: false,
    canDelete: true,
    attendanceLocked: false,
    ...patch,
  };
}
