import type { Row } from "./attendance";
export type SheetFilters = {
  query: string;
  status: "all" | "missing" | "filled" | "zero" | "locked";
  teacher: string;
  course: string;
  billing: string;
};
export const emptySheetFilters: SheetFilters = {
  query: "",
  status: "all",
  teacher: "",
  course: "",
  billing: "",
};
export type CourseGroup = { key: string; rows: Row[]; missing: number };
export type TeacherGroup = { key: string; name: string; courses: CourseGroup[]; missing: number };
export const teacherKey = (row: Row) =>
  row.teacherId != null ? `teacher:${row.teacherId}` : "unassigned";
export const courseKey = (row: Row) => `${row.courseId}:${row.billingMode}`;
export const rowPrice = (row: Row) =>
  row.billingMode === "subscription" ? row.subscriptionLessonPrice : row.lessonPrice;
export const rowTotal = (row: Row) =>
  row.hasRecord ? Math.round(row.hours * Math.round(rowPrice(row) * 100)) / 100 : 0;
export function monthSummary(rows: Row[]) {
  return {
    missing: rows.filter((r) => !r.attendanceLocked && !r.hasRecord).length,
    filled: rows.filter((r) => !r.attendanceLocked && r.hasRecord).length,
    locked: rows.filter((r) => r.attendanceLocked).length,
  };
}
export function filterSheet(rows: Row[], filters: SheetFilters, phones: Map<number, string>) {
  const query = filters.query.trim().toLocaleLowerCase();
  return rows.filter((row) => {
    if (filters.teacher && teacherKey(row) !== filters.teacher) return false;
    if (filters.course && String(row.courseId) !== filters.course) return false;
    if (filters.billing && row.billingMode !== filters.billing) return false;
    if (filters.status === "locked" && !row.attendanceLocked) return false;
    if (filters.status !== "all" && filters.status !== "locked" && row.attendanceLocked)
      return false;
    if (filters.status === "missing" && row.hasRecord) return false;
    if (filters.status === "filled" && !row.hasRecord) return false;
    if (filters.status === "zero" && (!row.hasRecord || row.hours !== 0)) return false;
    return (
      !query ||
      [row.studentName, row.courseName, row.teacherName, phones.get(row.studentId)].some((value) =>
        value?.toLocaleLowerCase().includes(query)
      )
    );
  });
}
export function groupSheet(rows: Row[]): TeacherGroup[] {
  const groups = new Map<string, TeacherGroup>();
  for (const row of rows) {
    const key = teacherKey(row);
    let teacher = groups.get(key);
    if (!teacher) {
      teacher = {
        key,
        name: row.teacherId != null ? row.teacherName : "",
        courses: [],
        missing: 0,
      };
      groups.set(key, teacher);
    }
    let course = teacher.courses.find((c) => c.key === courseKey(row));
    if (!course) {
      course = { key: courseKey(row), rows: [], missing: 0 };
      teacher.courses.push(course);
    }
    course.rows.push(row);
    if (!row.hasRecord && !row.attendanceLocked) {
      course.missing++;
      teacher.missing++;
    }
  }
  const result = [...groups.values()].sort(
    (a, b) => a.name.localeCompare(b.name) || a.key.localeCompare(b.key)
  );
  result.forEach((teacher) => {
    teacher.courses.sort(
      (a, b) =>
        a.rows[0].courseName.localeCompare(b.rows[0].courseName) || a.key.localeCompare(b.key)
    );
    teacher.courses.forEach((course) =>
      course.rows.sort(
        (a, b) => a.studentName.localeCompare(b.studentName) || a.enrollmentId - b.enrollmentId
      )
    );
  });
  return result;
}
export function initialSheetGroup(groups: TeacherGroup[]) {
  const teacher =
    groups.find((group) => group.missing > 0) ??
    groups.find((group) =>
      group.courses.some((course) => course.rows.some((row) => !row.attendanceLocked))
    ) ??
    groups[0];
  const course =
    teacher?.courses.find((group) => group.missing > 0) ??
    teacher?.courses.find((group) => group.rows.some((row) => !row.attendanceLocked)) ??
    teacher?.courses[0];
  return { teacher: teacher?.key ?? "", course: course?.key ?? "" };
}
