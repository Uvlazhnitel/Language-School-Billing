export type StudentFormDraft = {
  name: string;
  personalCode: string;
  phone: string;
  email: string;
  note: string;
  isMinor: boolean;
  payerName: string;
  payerRole: string;
};

export function buildNextStudentDraft(isMinor: boolean): StudentFormDraft {
  return {
    name: "",
    personalCode: "",
    phone: "",
    email: "",
    note: "",
    isMinor,
    payerName: "",
    payerRole: "",
  };
}
