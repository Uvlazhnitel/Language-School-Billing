import { describe, expect, it } from "vitest";

import {
  validateEmail,
  validatePersonName,
  validatePersonalCode,
  validatePhone,
  validateStudentForm,
} from "./inputValidation";

describe("inputValidation", () => {
  it("accepts multilingual person names and rejects digits", () => {
    expect(validatePersonName("Adelia Dumani", "student", true)).toBeNull();
    expect(validatePersonName("Vsevolods Fiļimonovs", "teacher", true)).toBeNull();
    expect(validatePersonName("Ivan2", "student", true)).toBe("msg.invalidStudentName");
  });

  it("validates personal code format", () => {
    expect(validatePersonalCode("131008-22451")).toBeNull();
    expect(validatePersonalCode("13100822451")).toBe("msg.invalidPersonalCode");
    expect(validatePersonalCode("abc")).toBe("msg.invalidPersonalCode");
  });

  it("validates phone format", () => {
    expect(validatePhone("+371 22137936")).toBeNull();
    expect(validatePhone("22137")).toBeNull();
    expect(validatePhone("abc123")).toBe("msg.invalidPhone");
  });

  it("validates email format", () => {
    expect(validateEmail("hello@example.com")).toBeNull();
    expect(validateEmail("bad-email")).toBe("msg.invalidEmail");
  });

  it("validates student form in the expected order", () => {
    expect(
      validateStudentForm({
        fullName: "Test 123",
        personalCode: "131008-22451",
        phone: "+371 22137936",
        email: "ok@example.com",
        isMinor: false,
        payerName: "",
        payerRole: "",
      })
    ).toBe("msg.invalidStudentName");

    expect(
      validateStudentForm({
        fullName: "Valid Name",
        personalCode: "bad",
        phone: "+371 22137936",
        email: "ok@example.com",
        isMinor: false,
        payerName: "",
        payerRole: "",
      })
    ).toBe("msg.invalidPersonalCode");
  });
});
