import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { createTranslator } from "../../lib/i18n";
import { EnrollmentFormModal } from "./EnrollmentFormModal";

describe("EnrollmentFormModal", () => {
  const student = {
    id: 7,
    version: 1,
    fullName: "Anna Student",
    createdAt: "2026-07-01T10:00:00Z",
    personalCode: "",
    phone: "",
    email: "",
    note: "",
    isMinor: false,
    payerName: "",
    payerRole: "",
    isActive: true,
    balance: 0,
    debt: 0,
  };

  const baseProps = {
    editing: null,
    studentSearch: student.fullName,
    studentId: student.id,
    studentPickerOpen: false,
    filteredStudents: [student],
    selectedStudent: student,
    courseId: 4,
    mode: "per_lesson" as const,
    chargeMaterials: true,
    lessonPriceOverride: "15",
    subscriptionLessonPrice: "0",
    note: "",
    allCourses: [
      {
        id: 4,
        version: 1,
        name: "Evening Group",
        teacherName: "",
        type: "group" as const,
        lessonPrice: 15,
        subscriptionPrice: 0,
      },
    ],
    studentComboRef: null,
    onStudentSearchChange: vi.fn(),
    onStudentIdChange: vi.fn(),
    onStudentPickerOpenChange: vi.fn(),
    onCourseIdChange: vi.fn(),
    onModeChange: vi.fn(),
    onChargeMaterialsChange: vi.fn(),
    onLessonPriceOverrideChange: vi.fn(),
    onSubscriptionLessonPriceChange: vi.fn(),
    onNoteChange: vi.fn(),
    onSave: vi.fn(),
    onCancel: vi.fn(),
    t: createTranslator("en-US"),
  };

  it("locks a preselected student and uses the attendance save label", () => {
    const markup = renderToStaticMarkup(
      <EnrollmentFormModal
        {...baseProps}
        studentLocked
        saveLabel="Save and go to attendance"
      />
    );

    expect(markup).toContain('disabled=""');
    expect(markup).toContain('value="Anna Student"');
    expect(markup).toContain("Save and go to attendance");
    expect(markup).not.toContain("Search by name, phone or email");
  });
});
