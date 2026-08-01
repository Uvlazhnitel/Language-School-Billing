import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { AttendanceScreen } from "./AttendanceScreen";
import { createTranslator } from "../lib/i18n";
import { defaultAttendanceQuickFilters } from "../lib/attendanceList";
import type { Row } from "../lib/attendance";

const rows: Row[] = [
  {
    enrollmentId: 1,
    enrollmentVersion: 1,
    studentId: 1,
    studentName: "Anna Student",
    courseId: 10,
    courseName: "English",
    courseType: "group",
    billingMode: "per_lesson",
    lessonPrice: 25,
    subscriptionLessonPrice: 0,
    hours: 0,
    hasRecord: false,
    canDelete: true,
    attendanceLocked: false,
  },
];

function buildProps() {
  return {
    attendanceSummary: { missing: 1, total: 1, filled: 0, zero: 0 },
    courseFilter: undefined,
    allCourses: [],
    query: "",
    filter: "all" as const,
    quickFilters: defaultAttendanceQuickFilters,
    hasActiveQuickFilters: false,
    rows,
    filteredRows: rows,
    loading: false,
    attendanceSavingRows: {},
    attendancePendingSelectRef: { current: null },
    year: 2026,
    month: 7,
    perLessonTotal: 0,
    courseTypeLabel: vi.fn((value: string) => value),
    formatEUR: vi.fn((value: number) => `${value.toFixed(2)} EUR`),
    normalizeHoursDraftInput: vi.fn((value: string) => value),
    getAttendanceStepBase: vi.fn((row: Row) => row.hours),
    getAttendanceInputValue: vi.fn((row: Row) => String(row.hours)),
    setAttendanceDraft: vi.fn(),
    clearAttendanceDraft: vi.fn(),
    commitAttendanceDraft: vi.fn(),
    onChangeHours: vi.fn(),
    onRefresh: vi.fn(),
    onOpenInvoices: vi.fn(),
    onOpenEnrollments: vi.fn(),
    onCourseFilterChange: vi.fn(),
    onQueryChange: vi.fn(),
    onFilterChange: vi.fn(),
    onToggleQuickFilter: vi.fn(),
    onResetFilters: vi.fn(),
    onOpenStudent: vi.fn(),
    onDeleteEnrollmentFromSheet: vi.fn(),
    focusTarget: null,
    onFocusTargetHandled: vi.fn(),
    t: createTranslator("en-US"),
  };
}

describe("AttendanceScreen", () => {
  it("renders attendance quick filter chips", () => {
    const markup = renderToStaticMarkup(<AttendanceScreen {...buildProps()} />);

    expect(markup).toContain("Missing");
    expect(markup).toContain("Zero hours");
    expect(markup).toContain("Locked");
    expect(markup).toContain("Subscriptions");
  });

  it("marks an active quick chip and shows clear filters", () => {
    const markup = renderToStaticMarkup(
      <AttendanceScreen
        {...buildProps()}
        quickFilters={{ ...defaultAttendanceQuickFilters, zero: true }}
        hasActiveQuickFilters
      />
    );

    expect(markup).toContain('quickFilterChip active');
    expect(markup).toContain("Clear filters");
  });
});
