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
    onNameChange: vi.fn(),
    onPersonalCodeChange: vi.fn(),
    onPhoneChange: vi.fn(),
    onEmailChange: vi.fn(),
    onNoteChange: vi.fn(),
    onIsMinorChange: vi.fn(),
    onPayerNameChange: vi.fn(),
    onPayerRoleChange: vi.fn(),
    onSave: vi.fn(),
    onCancel: vi.fn(),
    onOpenExistingStudent: vi.fn(),
    onCreateAnyway: vi.fn(),
    t: createTranslator("en-US"),
  };

  it("renders exact duplicate warning without create-anyway action", () => {
    const markup = renderToStaticMarkup(
      <StudentFormModal
        {...baseProps}
        duplicateCheckResult={{
          exactMatch: {
            id: 1,
            version: 1,
            fullName: "Anna Student",
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
});
