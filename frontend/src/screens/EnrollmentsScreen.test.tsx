import { createRef } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { createTranslator } from "../lib/i18n";
import type { CourseDTO } from "../lib/courses";
import type { EnrollmentDTO } from "../lib/enrollments";
import { EnrollmentsScreen } from "./EnrollmentsScreen";

const courses: CourseDTO[] = [
  {
    id: 1,
    version: 1,
    name: "Alpha Course",
    teacherName: "Alice Teacher",
    type: "group",
    lessonPrice: 15,
    subscriptionPrice: 60,
  },
  {
    id: 2,
    version: 1,
    name: "Beta Course",
    teacherName: "Bob Teacher",
    type: "individual",
    lessonPrice: 25,
    subscriptionPrice: 100,
  },
];

const enrollments: EnrollmentDTO[] = [
  {
    id: 3,
    version: 1,
    studentId: 3,
    studentName: "Beta Student",
    courseId: 2,
    courseName: "Beta Course",
    courseType: "individual",
    teacherName: "Bob Teacher",
    billingMode: "subscription",
    chargeMaterials: false,
    lessonPriceOverride: 0,
    subscriptionLessonPrice: 100,
    note: "Remote lessons",
    createdAt: "2026-07-01T10:00:00Z",
  },
  {
    id: 1,
    version: 1,
    studentId: 1,
    studentName: "Alpha Student One",
    courseId: 1,
    courseName: "Alpha Course",
    courseType: "group",
    teacherName: "Alice Teacher",
    billingMode: "per_lesson",
    chargeMaterials: true,
    lessonPriceOverride: 15,
    subscriptionLessonPrice: 0,
    note: "Needs printed materials",
    createdAt: "2026-07-01T10:00:00Z",
  },
  {
    id: 2,
    version: 1,
    studentId: 2,
    studentName: "Alpha Student Two",
    courseId: 1,
    courseName: "Alpha Course",
    courseType: "group",
    teacherName: "Alice Teacher",
    billingMode: "per_lesson",
    chargeMaterials: false,
    lessonPriceOverride: 15,
    subscriptionLessonPrice: 0,
    note: "",
    createdAt: "2026-07-01T10:00:00Z",
  },
];

function buildProps() {
  return {
    loading: false,
    enrollments,
    studentFilter: undefined,
    courseFilter: undefined,
    allStudents: [],
    allCourses: courses,
    enrollmentCourseOptions: courses,
    billingModeLabel: (mode: string) => mode,
    courseTypeLabel: (type: string) => type,
    onStudentFilterChange: vi.fn(),
    onCourseFilterChange: vi.fn(),
    onAddEnrollment: vi.fn(),
    onOpenStudents: vi.fn(),
    onOpenCourses: vi.fn(),
    onOpenStudent: vi.fn(),
    onEditEnrollment: vi.fn(),
    enrollmentModalOpen: false,
    editingEnrollment: null,
    studentSearch: "",
    studentId: undefined,
    studentPickerOpen: false,
    filteredStudents: [],
    selectedStudent: null,
    enrollmentStudentLocked: false,
    enrollmentSaveLabel: undefined,
    enrollmentCourseId: 0,
    enrollmentMode: "per_lesson" as const,
    enrollmentChargeMaterials: true,
    enrollmentLessonPriceOverride: "0",
    enrollmentSubscriptionLessonPrice: "0",
    enrollmentNote: "",
    studentComboRef: createRef<HTMLDivElement>(),
    onStudentSearchChange: vi.fn(),
    onStudentIdChange: vi.fn(),
    onStudentPickerOpenChange: vi.fn(),
    onEnrollmentCourseIdChange: vi.fn(),
    onEnrollmentModeChange: vi.fn(),
    onEnrollmentChargeMaterialsChange: vi.fn(),
    onEnrollmentLessonPriceOverrideChange: vi.fn(),
    onEnrollmentSubscriptionLessonPriceChange: vi.fn(),
    onEnrollmentNoteChange: vi.fn(),
    onSaveEnrollment: vi.fn(),
    onCloseEnrollmentModal: vi.fn(),
    t: createTranslator("en-US"),
  };
}

describe("EnrollmentsScreen", () => {
  it("groups enrollments by course and opens only the first group by default", () => {
    const markup = renderToStaticMarkup(<EnrollmentsScreen {...buildProps()} />);

    expect(markup).toContain("Alpha Course");
    expect(markup).toContain("Search student in enrollments...");
    expect(markup).toContain("group · Alice Teacher");
    expect(markup).toContain("2 students");
    expect(markup).toContain("Alpha Student One");
    expect(markup).toContain("Needs printed materials");
    expect(markup).not.toContain("Beta Student");
    expect(markup).toContain("Expand all");
    expect(markup).toContain("Collapse all");
  });

  it("opens the visible course immediately when a course filter is active", () => {
    const markup = renderToStaticMarkup(
      <EnrollmentsScreen {...buildProps()} enrollments={[enrollments[0]]} courseFilter={2} />
    );

    expect(markup).toContain("Beta Student");
    expect(markup).toContain("subscription");
    expect(markup).toContain("€100.00");
    expect(markup).toContain("Remote lessons");
  });
});
