import { describe, expect, it } from "vitest";

import { buildNextStudentDraft } from "./studentFormState";

describe("studentFormState", () => {
  it("clears all fields and preserves adult mode when preparing the next draft", () => {
    expect(buildNextStudentDraft(false)).toEqual({
      name: "",
      personalCode: "",
      phone: "",
      email: "",
      note: "",
      isMinor: false,
      payerName: "",
      payerRole: "",
    });
  });

  it("clears payer fields and preserves minor mode when preparing the next draft", () => {
    expect(buildNextStudentDraft(true)).toEqual({
      name: "",
      personalCode: "",
      phone: "",
      email: "",
      note: "",
      isMinor: true,
      payerName: "",
      payerRole: "",
    });
  });
});
