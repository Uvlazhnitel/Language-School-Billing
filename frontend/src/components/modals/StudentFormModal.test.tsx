import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { StudentFormModal } from "./StudentFormModal";
import { createTranslator } from "../../lib/i18n";

describe("StudentFormModal", () => {
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
    courseId: 0,
    enrollmentMode: "per_lesson" as const,
    enrollmentChargeMaterials: true,
    enrollmentLessonPrice: "0",
    enrollmentSubscriptionPrice: "0",
    enrollmentNote: "",
    enrollmentSettingsOpen: false,
    formatEUR: (value: number) => `€${value.toFixed(2)}`,
    onNameChange: vi.fn(),
    onPersonalCodeChange: vi.fn(),
    onPhoneChange: vi.fn(),
    onEmailChange: vi.fn(),
    onNoteChange: vi.fn(),
    onIsMinorChange: vi.fn(),
    onPayerNameChange: vi.fn(),
    onPayerRoleChange: vi.fn(),
    onCourseIdChange: vi.fn(),
    onEnrollmentModeChange: vi.fn(),
    onEnrollmentChargeMaterialsChange: vi.fn(),
    onEnrollmentLessonPriceChange: vi.fn(),
    onEnrollmentSubscriptionPriceChange: vi.fn(),
    onEnrollmentNoteChange: vi.fn(),
    onEnrollmentSettingsOpenChange: vi.fn(),
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
        courseId={4}
        enrollmentLessonPrice="15"
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
        courseId={4}
        enrollmentLessonPrice="15"
        enrollmentSettingsOpen
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
        courseId={4}
        enrollmentLessonPrice="15"
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
});
