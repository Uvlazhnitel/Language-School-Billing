import { describe, expect, it } from "vitest";

import type { EnrollmentDTO } from "./enrollments";
import {
  UNASSIGNED_TEACHER_KEY,
  groupEnrollmentsByTeacher,
  normalizeEnrollmentSelectionPreferences,
  parseEnrollmentSelectionPreferences,
  resolveEnrollmentSelection,
} from "./enrollmentTeacherGroups";

function enrollment(
  id: number,
  studentName: string,
  courseId: number,
  courseName: string,
  teacherId?: number,
  teacherName = ""
): EnrollmentDTO {
  return {
    id,
    version: 1,
    studentId: id,
    studentName,
    courseId,
    courseName,
    courseType: "group",
    teacherId,
    teacherName,
    billingMode: "per_lesson",
    chargeMaterials: true,
    lessonPriceOverride: 15,
    subscriptionLessonPrice: 0,
    note: "",
    createdAt: "2026-07-01T10:00:00Z",
  };
}

describe("enrollment teacher groups", () => {
  it("groups by teacher id and sorts teachers, courses, and students alphabetically", () => {
    const groups = groupEnrollmentsByTeacher(
      [
        enrollment(1, "Zane", 20, "Zulu", 2, "Bob"),
        enrollment(2, "Anna", 10, "Alpha", 1, "Alice"),
        enrollment(3, "Bella", 10, "Alpha", 1, "Alice"),
        enrollment(4, "Cara", 11, "Beta", 1, "Alice"),
        enrollment(5, "No Teacher", 30, "Open course"),
      ],
      "Without teacher"
    );

    expect(groups.map((group) => group.teacherName)).toEqual(["Alice", "Bob", "Without teacher"]);
    expect(groups[0].courses.map((course) => course.courseName)).toEqual(["Alpha", "Beta"]);
    expect(groups[0].courses[0].enrollments.map((item) => item.studentName)).toEqual([
      "Anna",
      "Bella",
    ]);
    expect(groups[0].studentCount).toBe(3);
    expect(groups[2].key).toBe(UNASSIGNED_TEACHER_KEY);
  });

  it("restores valid preferences and falls back from removed teachers and courses", () => {
    const groups = groupEnrollmentsByTeacher(
      [
        enrollment(1, "Anna", 10, "Alpha", 1, "Alice"),
        enrollment(2, "Bella", 11, "Beta", 1, "Alice"),
        enrollment(3, "Cara", 20, "Gamma", 2, "Bob"),
      ],
      "Without teacher"
    );
    const restored = resolveEnrollmentSelection(groups, {
      activeTeacherKey: "teacher:2",
      courseByTeacher: { "teacher:2": 20 },
    });
    const normalized = normalizeEnrollmentSelectionPreferences(groups, {
      activeTeacherKey: "teacher:missing",
      courseByTeacher: { "teacher:1": 999 },
    });

    expect(restored).toEqual({ teacherKey: "teacher:2", courseId: 20 });
    expect(normalized.activeTeacherKey).toBe("teacher:1");
    expect(normalized.courseByTeacher["teacher:1"]).toBe(10);
  });

  it("ignores malformed local storage values", () => {
    expect(parseEnrollmentSelectionPreferences("not-json")).toEqual({ courseByTeacher: {} });
    expect(
      parseEnrollmentSelectionPreferences(
        JSON.stringify({
          activeTeacherKey: "teacher:1",
          courseByTeacher: { "teacher:1": 10, bad: -2, string: "20" },
        })
      )
    ).toEqual({
      activeTeacherKey: "teacher:1",
      courseByTeacher: { "teacher:1": 10 },
    });
  });
});
