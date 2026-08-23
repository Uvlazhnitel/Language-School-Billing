import { describe, expect, it } from "vitest";

import type { Row } from "./attendance";
import {
  defaultAttendanceQuickFilters,
  filterAndSortAttendanceRows,
  hasActiveAttendanceQuickFilters,
  toggleAttendanceQuickFilter,
} from "./attendanceList";
import type { StudentDTO } from "./students";

const students: StudentDTO[] = [
  {
    id: 1,
    version: 1,
    fullName: "Anna Missing",
    createdAt: "2026-07-01T10:00:00Z",
    personalCode: "",
    phone: "111",
    email: "",
    note: "",
    isMinor: false,
    payerName: "",
    payerRole: "",
    isActive: true,
    balance: 0,
    debt: 0,
  },
  {
    id: 2,
    version: 1,
    fullName: "Bella Zero",
    createdAt: "2026-07-01T10:00:00Z",
    personalCode: "",
    phone: "222",
    email: "",
    note: "",
    isMinor: false,
    payerName: "",
    payerRole: "",
    isActive: true,
    balance: 0,
    debt: 0,
  },
  {
    id: 3,
    version: 1,
    fullName: "Cara Locked",
    createdAt: "2026-07-01T10:00:00Z",
    personalCode: "",
    phone: "333",
    email: "",
    note: "",
    isMinor: false,
    payerName: "",
    payerRole: "",
    isActive: true,
    balance: 0,
    debt: 0,
  },
  {
    id: 4,
    version: 1,
    fullName: "Dana Normal",
    createdAt: "2026-07-01T10:00:00Z",
    personalCode: "",
    phone: "444",
    email: "",
    note: "",
    isMinor: false,
    payerName: "",
    payerRole: "",
    isActive: true,
    balance: 0,
    debt: 0,
  },
  {
    id: 5,
    version: 1,
    fullName: "Eva Subscription",
    createdAt: "2026-07-01T10:00:00Z",
    personalCode: "",
    phone: "555",
    email: "",
    note: "",
    isMinor: false,
    payerName: "",
    payerRole: "",
    isActive: true,
    balance: 0,
    debt: 0,
  },
];

const studentIndex = new Map<number, StudentDTO>(students.map((student) => [student.id, student]));

const rows: Row[] = [
  {
    enrollmentId: 1,
    enrollmentVersion: 1,
    studentId: 1,
    studentName: "Anna Missing",
    courseId: 10,
    courseName: "English",
    courseType: "group",
    teacherName: "English Teacher",
    billingMode: "per_lesson",
    lessonPrice: 20,
    subscriptionLessonPrice: 0,
    hours: 0,
    hasRecord: false,
    canDelete: true,
    attendanceLocked: false,
  },
  {
    enrollmentId: 2,
    enrollmentVersion: 1,
    studentId: 2,
    studentName: "Bella Zero",
    courseId: 20,
    courseName: "German",
    courseType: "group",
    teacherName: "German Teacher",
    billingMode: "per_lesson",
    lessonPrice: 20,
    subscriptionLessonPrice: 0,
    hours: 0,
    hasRecord: true,
    canDelete: true,
    attendanceLocked: false,
  },
  {
    enrollmentId: 3,
    enrollmentVersion: 1,
    studentId: 3,
    studentName: "Cara Locked",
    courseId: 30,
    courseName: "French",
    courseType: "group",
    teacherName: "French Teacher",
    billingMode: "per_lesson",
    lessonPrice: 20,
    subscriptionLessonPrice: 0,
    hours: 1,
    hasRecord: true,
    canDelete: true,
    attendanceLocked: true,
  },
  {
    enrollmentId: 4,
    enrollmentVersion: 1,
    studentId: 4,
    studentName: "Dana Normal",
    courseId: 40,
    courseName: "Spanish",
    courseType: "group",
    teacherName: "Spanish Teacher",
    billingMode: "per_lesson",
    lessonPrice: 20,
    subscriptionLessonPrice: 0,
    hours: 2,
    hasRecord: true,
    canDelete: true,
    attendanceLocked: false,
  },
  {
    enrollmentId: 5,
    enrollmentVersion: 1,
    studentId: 5,
    studentName: "Eva Subscription",
    courseId: 50,
    courseName: "Latvian",
    courseType: "group",
    teacherName: "Latvian Teacher",
    billingMode: "subscription",
    lessonPrice: 20,
    subscriptionLessonPrice: 18,
    hours: 1,
    hasRecord: true,
    canDelete: true,
    attendanceLocked: false,
  },
];

describe("attendanceList", () => {
  it("detects and toggles active quick filters", () => {
    expect(hasActiveAttendanceQuickFilters(defaultAttendanceQuickFilters)).toBe(false);
    const toggled = toggleAttendanceQuickFilter("locked", defaultAttendanceQuickFilters);
    expect(toggled.locked).toBe(true);
    expect(hasActiveAttendanceQuickFilters(toggled)).toBe(true);
  });

  it("sorts rows by attention priority before alphabetical order", () => {
    const result = filterAndSortAttendanceRows(
      rows,
      "",
      "all",
      defaultAttendanceQuickFilters,
      studentIndex
    );

    expect(result.map((row) => row.studentName)).toEqual([
      "Anna Missing",
      "Bella Zero",
      "Cara Locked",
      "Dana Normal",
      "Eva Subscription",
    ]);
  });

  it("filters locked and subscription rows through quick filters", () => {
    expect(
      filterAndSortAttendanceRows(
        rows,
        "",
        "all",
        { ...defaultAttendanceQuickFilters, locked: true },
        studentIndex
      ).map((row) => row.studentName)
    ).toEqual(["Cara Locked"]);

    expect(
      filterAndSortAttendanceRows(
        rows,
        "",
        "all",
        { ...defaultAttendanceQuickFilters, subscription: true },
        studentIndex
      ).map((row) => row.studentName)
    ).toEqual(["Eva Subscription"]);
  });

  it("combines search, status filter, and quick filters as an intersection", () => {
    const result = filterAndSortAttendanceRows(
      rows,
      "bella",
      "filled",
      { ...defaultAttendanceQuickFilters, zero: true },
      studentIndex
    );

    expect(result.map((row) => row.studentName)).toEqual(["Bella Zero"]);
  });

  it("still searches by phone number", () => {
    const result = filterAndSortAttendanceRows(
      rows,
      "333",
      "all",
      defaultAttendanceQuickFilters,
      studentIndex
    );

    expect(result.map((row) => row.studentName)).toEqual(["Cara Locked"]);
  });
});
