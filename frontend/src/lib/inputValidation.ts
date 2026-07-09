const personNamePattern = /^[\p{L} .'\u2019-]+$/u;
const phonePattern = /^[+\d][\d\s().-]*$/;
const personalCodePattern = /^\d{6}-\d{5}$/;
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function normalizePersonName(value: string): string {
  return value.trim().replace(/\s+/g, " ");
}

function digitCount(value: string): number {
  return (value.match(/\d/g) ?? []).length;
}

export function validatePersonName(
  value: string,
  field: "student" | "payer" | "teacher",
  required: boolean
): string | null {
  const normalized = normalizePersonName(value);
  if (!normalized) {
    if (!required) return null;
    if (field === "student") return "msg.studentNameRequired";
    if (field === "payer") return "msg.studentPayerRequired";
    return "msg.teacherNameRequired";
  }
  if (!personNamePattern.test(normalized)) {
    if (field === "student") return "msg.invalidStudentName";
    if (field === "payer") return "msg.invalidPayerName";
    return "msg.invalidTeacherName";
  }
  return null;
}

export function validatePersonalCode(value: string): string | null {
  const normalized = value.trim();
  if (!normalized) return null;
  if (!personalCodePattern.test(normalized)) return "msg.invalidPersonalCode";
  return null;
}

export function validatePhone(value: string): string | null {
  const normalized = value.trim();
  if (!normalized) return null;
  if (!phonePattern.test(normalized) || digitCount(normalized) < 5) {
    return "msg.invalidPhone";
  }
  return null;
}

export function validateEmail(value: string): string | null {
  const normalized = value.trim();
  if (!normalized) return null;
  if (!emailPattern.test(normalized)) return "msg.invalidEmail";
  return null;
}

export function validateStudentForm(fields: {
  fullName: string;
  personalCode: string;
  phone: string;
  email: string;
  isMinor: boolean;
  payerName: string;
  payerRole: string;
}): string | null {
  return (
    validatePersonName(fields.fullName, "student", true) ??
    validatePersonalCode(fields.personalCode) ??
    validatePhone(fields.phone) ??
    validateEmail(fields.email) ??
    validatePersonName(fields.payerName, "payer", fields.isMinor) ??
    (fields.isMinor && !fields.payerRole ? "msg.studentPayerRoleRequired" : null)
  );
}
