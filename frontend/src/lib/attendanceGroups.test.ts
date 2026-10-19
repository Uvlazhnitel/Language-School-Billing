import { describe, expect, it } from "vitest";
import { attendanceRow as row } from "./attendanceFixture";
import {
  emptySheetFilters,
  filterSheet,
  groupSheet,
  initialSheetGroup,
  monthSummary,
  rowTotal,
} from "./attendanceGroups";
describe("attendance groups", () => {
  const rows = [
    row({ studentName: "Zoe" }),
    row({ enrollmentId: 2, studentName: "Anna", hasRecord: true, hours: 1.25, lessonPrice: 10 }),
    row({
      enrollmentId: 3,
      billingMode: "subscription",
      hasRecord: true,
      hours: 0.25,
      subscriptionLessonPrice: 15,
    }),
    row({ enrollmentId: 4, teacherId: 2, teacherName: "Teacher", attendanceLocked: true }),
    row({ enrollmentId: 5, teacherId: undefined, teacherName: "", hasRecord: true, hours: 0 }),
  ];
  it("groups by teacher ID then course and billing mode, sorts names stably and keeps every row", () => {
    const groups = groupSheet(rows);
    expect(groups).toHaveLength(3);
    const teacher = groups.find((g) => g.key === "teacher:1")!;
    expect(teacher.courses).toHaveLength(2);
    expect(teacher.courses[0].rows.map((r) => r.studentName)).toEqual(["Anna", "Zoe"]);
    expect(groups.flatMap((t) => t.courses.flatMap((c) => c.rows))).toHaveLength(5);
    expect(initialSheetGroup(groups)).toEqual({ teacher: "teacher:1", course: "10:per_lesson" });
    expect(initialSheetGroup(groupSheet(rows.map((r) => ({ ...r, hasRecord: true }))))).toEqual({
      teacher: "unassigned",
      course: "10:per_lesson",
    });
  });
  it("keeps full-month summary disjoint and applies every state filter before totals", () => {
    expect(monthSummary(rows)).toEqual({ missing: 1, filled: 3, locked: 1 });
    expect(
      ["all", "missing", "filled", "zero", "locked"].map(
        (status) =>
          filterSheet(
            rows,
            { ...emptySheetFilters, status: status as typeof emptySheetFilters.status },
            new Map()
          ).length
      )
    ).toEqual([5, 1, 3, 1, 1]);
    expect(rows.reduce((sum, r) => sum + rowTotal(r), 0)).toBe(16.25);
  });
  it("combines teacher, course, billing and search by name, phone, course or teacher", () => {
    const phones = new Map([[1, "123456"]]);
    for (const query of ["anna", "123456", "english", "teacher"])
      expect(filterSheet(rows, { ...emptySheetFilters, query }, phones).length).toBeGreaterThan(0);
    expect(
      filterSheet(
        rows,
        {
          ...emptySheetFilters,
          teacher: "teacher:1",
          course: "10",
          billing: "subscription",
          status: "filled",
        },
        phones
      ).map((r) => r.enrollmentId)
    ).toEqual([3]);
    expect(filterSheet(rows, { ...emptySheetFilters, query: "not found" }, phones)).toEqual([]);
  });
  it("prefers an editable completed group over locked groups when nothing remains to fill", () => {
    const groups = groupSheet([
      row({ teacherId: undefined, attendanceLocked: true }),
      row({ teacherId: 1, hasRecord: true, hours: 2 }),
    ]);
    expect(initialSheetGroup(groups)).toEqual({ teacher: "teacher:1", course: "10:per_lesson" });
  });
});
