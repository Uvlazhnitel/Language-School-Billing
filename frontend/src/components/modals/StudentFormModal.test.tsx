import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { StudentFormModal } from "./StudentFormModal";
import { createTranslator } from "../../lib/i18n";
import type { StudentOnboardingEnrollmentRow } from "../../lib/studentOnboarding";

describe("StudentFormModal", () => {
  const emptyEnrollmentRow: StudentOnboardingEnrollmentRow = {
    id: 1,
    courseId: 0,
    billingMode: "per_lesson",
    chargeMaterials: true,
    lessonPrice: "0",
    subscriptionPrice: "0",
    note: "",
    settingsOpen: false,
  };

  const baseProps = {
    editing: false,
    name: "",
    personalCode: "",
    phone: "",
    email: "",
    note: "",
    isMinor: false,
    payerName: "",
    payerRole: "",
    payerRoleOptions: [] as const,
    payerRoleLabel: (role: string) => role,
    allCourses: [],
    enrollmentRows: [emptyEnrollmentRow],
    formatEUR: (value: number) => `€${value.toFixed(2)}`,
    onNameChange: vi.fn(),
    onPersonalCodeChange: vi.fn(),
    onPhoneChange: vi.fn(),
    onEmailChange: vi.fn(),
    onNoteChange: vi.fn(),
    onIsMinorChange: vi.fn(),
    onPayerNameChange: vi.fn(),
    onPayerRoleChange: vi.fn(),
    onAddEnrollmentRow: vi.fn(),
    onRemoveEnrollmentRow: vi.fn(),
    onEnrollmentCourseChange: vi.fn(),
    onEnrollmentModeChange: vi.fn(),
    onEnrollmentRowChange: vi.fn(),
    onSave: vi.fn(),
    onSaveAndAddAnother: vi.fn(),
    onCancel: vi.fn(),
    onOpenExistingStudent: vi.fn(),
    onEnrollExistingStudent: vi.fn(),
    onCreateAnyway: vi.fn(),
    t: createTranslator("en-US"),
  };

  it("renders save-and-add-another action only while creating a student", () => {
    const createMarkup = renderToStaticMarkup(<StudentFormModal {...baseProps} />);
    const editMarkup = renderToStaticMarkup(<StudentFormModal {...baseProps} editing />);

    expect(createMarkup).toContain("Save");
    expect(createMarkup).toContain("Save and add another");
    expect(createMarkup).toContain("Cancel");
    expect(editMarkup).not.toContain("Save and add another");
  });

  it("shows optional course onboarding only while creating", () => {
    const createMarkup = renderToStaticMarkup(<StudentFormModal {...baseProps} />);
    const editMarkup = renderToStaticMarkup(<StudentFormModal {...baseProps} editing />);

    expect(createMarkup).toContain("Course or group");
    expect(createMarkup).toContain("No course yet");
    expect(editMarkup).not.toContain("Course or group");
  });

  it("shows course summary and attendance action when a course is selected", () => {
    const markup = renderToStaticMarkup(
      <StudentFormModal
        {...baseProps}
        allCourses={[
          {
            id: 4,
            version: 1,
            name: "Evening Group",
            teacherName: "Teacher",
            type: "group",
            lessonPrice: 15,
            subscriptionPrice: 0,
          },
        ]}
        enrollmentRows={[{ ...emptyEnrollmentRow, courseId: 4, lessonPrice: "15" }]}
      />
    );

    expect(markup).toContain("Evening Group");
    expect(markup).toContain("€15.00");
    expect(markup).toContain("Change terms");
    expect(markup).toContain("Create and go to attendance");
  });

  it("renders advanced enrollment terms only when expanded", () => {
    const markup = renderToStaticMarkup(
      <StudentFormModal
        {...baseProps}
        allCourses={[
          {
            id: 4,
            version: 1,
            name: "Evening Group",
            teacherName: "",
            type: "group",
            lessonPrice: 15,
            subscriptionPrice: 0,
          },
        ]}
        enrollmentRows={[
          { ...emptyEnrollmentRow, courseId: 4, lessonPrice: "15", settingsOpen: true },
        ]}
      />
    );

    expect(markup).toContain("Enrollment note");
    expect(markup).toContain("Hide terms");
  });

  it("renders exact duplicate warning without create-anyway action", () => {
    const markup = renderToStaticMarkup(
      <StudentFormModal
        {...baseProps}
        duplicateCheckResult={{
          exactMatch: {
            id: 1,
            version: 1,
            fullName: "Anna Student",
            createdAt: "2026-07-01T10:00:00Z",
            personalCode: "020202-23456",
            phone: "123",
            email: "anna@example.com",
            note: "",
            isMinor: false,
            payerName: "",
            payerRole: "",
            isActive: true,
            balance: 0,
            debt: 0,
          },
          possibleMatches: [],
        }}
      />
    );

    expect(markup).toContain("This student already exists");
    expect(markup).toContain("Open student");
    expect(markup).not.toContain("Create anyway");
  });

  it("renders possible duplicate warning with create-anyway action", () => {
    const markup = renderToStaticMarkup(
      <StudentFormModal
        {...baseProps}
        duplicateCheckResult={{
          possibleMatches: [
            {
              id: 2,
              version: 3,
              fullName: "Berta Student",
              createdAt: "2026-07-02T10:00:00Z",
              personalCode: "",
              phone: "222",
              email: "",
              note: "",
              isMinor: false,
              payerName: "",
              payerRole: "",
              isActive: false,
              balance: 0,
              debt: 0,
            },
          ],
        }}
      />
    );

    expect(markup).toContain("Possible duplicate students found");
    expect(markup).toContain("Berta Student");
    expect(markup).toContain("Inactive");
    expect(markup).toContain("Create anyway");
  });

  it("offers enrolling an existing duplicate when a course is selected", () => {
    const markup = renderToStaticMarkup(
      <StudentFormModal
        {...baseProps}
        allCourses={[
          {
            id: 4,
            version: 1,
            name: "Evening Group",
            teacherName: "",
            type: "group",
            lessonPrice: 15,
            subscriptionPrice: 0,
          },
        ]}
        enrollmentRows={[{ ...emptyEnrollmentRow, courseId: 4, lessonPrice: "15" }]}
        duplicateCheckResult={{
          possibleMatches: [
            {
              id: 2,
              version: 3,
              fullName: "Berta Student",
              createdAt: "2026-07-02T10:00:00Z",
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
          ],
        }}
      />
    );

    expect(markup).toContain("Enroll existing student");
    expect(markup).not.toContain(">Open student<");
  });

  it("renders multiple independent course rows with add and remove actions", () => {
    const markup = renderToStaticMarkup(
      <StudentFormModal
        {...baseProps}
        allCourses={[
          {
            id: 4,
            version: 1,
            name: "Evening Group",
            teacherName: "",
            type: "group",
            lessonPrice: 15,
            subscriptionPrice: 0,
          },
          {
            id: 5,
            version: 1,
            name: "Private Lesson",
            teacherName: "",
            type: "individual",
            lessonPrice: 25,
            subscriptionPrice: 0,
          },
          {
            id: 6,
            version: 1,
            name: "Conversation Club",
            teacherName: "",
            type: "group",
            lessonPrice: 15,
            subscriptionPrice: 0,
          },
        ]}
        enrollmentRows={[
          { ...emptyEnrollmentRow, courseId: 4, lessonPrice: "15" },
          {
            ...emptyEnrollmentRow,
            id: 2,
            courseId: 5,
            lessonPrice: "25",
            chargeMaterials: false,
          },
        ]}
      />
    );

    expect(markup).toContain("Course 1");
    expect(markup).toContain("Course 2");
    expect(markup).toContain("Remove course");
    expect(markup).toContain("Add another course");
    expect(markup).toContain('option value="4" disabled=""');
    expect(markup).toContain('option value="5" disabled=""');
  });
});
