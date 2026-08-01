import { BillingModeSubscription } from "./constants";
import type { Row } from "./attendance";
import type { StudentDTO } from "./students";

export type AttendanceFilter = "all" | "missing" | "filled" | "zero";

export type AttendanceQuickFilterKey = "missing" | "zero" | "locked" | "subscription";

export type AttendanceQuickFilters = Record<AttendanceQuickFilterKey, boolean>;

export const defaultAttendanceQuickFilters: AttendanceQuickFilters = {
  missing: false,
  zero: false,
  locked: false,
  subscription: false,
};

export function hasActiveAttendanceQuickFilters(filters: AttendanceQuickFilters): boolean {
  return Object.values(filters).some(Boolean);
}

export function toggleAttendanceQuickFilter(
  key: AttendanceQuickFilterKey,
  filters: AttendanceQuickFilters
): AttendanceQuickFilters {
  return {
    ...filters,
    [key]: !filters[key],
  };
}

function matchesQuickFilters(row: Row, filters: AttendanceQuickFilters): boolean {
  if (filters.missing && row.hasRecord) return false;
  if (filters.zero && (!row.hasRecord || row.hours !== 0)) return false;
  if (filters.locked && !row.attendanceLocked) return false;
  if (filters.subscription && row.billingMode !== BillingModeSubscription) return false;
  return true;
}

function priorityBucket(row: Row): number {
  if (!row.hasRecord) return 0;
  if (row.hours === 0) return 1;
  if (row.attendanceLocked) return 2;
  return 3;
}

export function filterAndSortAttendanceRows(
  rows: Row[],
  query: string,
  filter: AttendanceFilter,
  quickFilters: AttendanceQuickFilters,
  studentIndex: Map<number, StudentDTO>
): Row[] {
  const q = query.trim().toLowerCase();
  let filtered = rows;

  if (q) {
    filtered = filtered.filter((row) => {
      const student = studentIndex.get(row.studentId);
      const studentName = (row.studentName ?? "").toLowerCase();
      const courseName = (row.courseName ?? "").toLowerCase();
      const phone = (student?.phone ?? "").toLowerCase();
      return studentName.includes(q) || courseName.includes(q) || phone.includes(q);
    });
  }

  if (filter === "missing") {
    filtered = filtered.filter((row) => !row.hasRecord);
  } else if (filter === "filled") {
    filtered = filtered.filter((row) => row.hasRecord);
  } else if (filter === "zero") {
    filtered = filtered.filter((row) => row.hasRecord && row.hours === 0);
  }

  filtered = filtered.filter((row) => matchesQuickFilters(row, quickFilters));

  return [...filtered].sort((a, b) => {
    const priorityCompare = priorityBucket(a) - priorityBucket(b);
    if (priorityCompare !== 0) return priorityCompare;

    const aSubscription = a.billingMode === BillingModeSubscription;
    const bSubscription = b.billingMode === BillingModeSubscription;
    if (aSubscription !== bSubscription) {
      return aSubscription ? 1 : -1;
    }

    const studentCompare = a.studentName.localeCompare(b.studentName);
    if (studentCompare !== 0) return studentCompare;

    const courseCompare = a.courseName.localeCompare(b.courseName);
    if (courseCompare !== 0) return courseCompare;

    return a.enrollmentId - b.enrollmentId;
  });
}
