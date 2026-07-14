import { describe, expect, it } from "vitest";

import { findAttendanceFocusEnrollmentId } from "./attendanceFocus";
import type { Row } from "./attendance";

const rows: Row[] = [
  {
    enrollmentId: 10,
    enrollmentVersion: 1,
    studentId: 2,
    studentName: "Anna",
    courseId: 3,
    courseName: "Group",
    courseType: "group",
    billingMode: "per_lesson",
    lessonPrice: 15,
    subscriptionLessonPrice: 0,
    hours: 0,
    hasRecord: false,
    canDelete: true,
    attendanceLocked: false,
  },
];

describe("attendance focus", () => {
  it("finds the enrollment only when both student and course match", () => {
    expect(findAttendanceFocusEnrollmentId(rows, { studentId: 2, courseId: 3 })).toBe(10);
    expect(findAttendanceFocusEnrollmentId(rows, { studentId: 2, courseId: 4 })).toBeNull();
  });

  it("does not change attendance data while resolving focus", () => {
    findAttendanceFocusEnrollmentId(rows, { studentId: 2, courseId: 3 });
    expect(rows[0]).toMatchObject({ hours: 0, hasRecord: false });
  });
});
