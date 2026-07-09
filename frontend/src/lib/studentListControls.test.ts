import { describe, expect, it } from "vitest";

import type { StudentDTO } from "./api";
import { applyStudentListControls } from "./studentListControls";

const students: StudentDTO[] = [
  {
    id: 1,
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
    debt: 12,
  },
  {
    id: 2,
    version: 1,
    fullName: "Berta Student",
    createdAt: "2026-07-03T10:00:00Z",
    personalCode: "",
    phone: "",
    email: "",
    note: "",
    isMinor: true,
    payerName: "",
    payerRole: "",
    isActive: false,
    balance: 20,
    debt: 0,
  },
  {
    id: 3,
    version: 1,
    fullName: "Clara Student",
    createdAt: "2026-07-02T10:00:00Z",
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
  },
];

describe("applyStudentListControls", () => {
  it("defaults to active students sorted by name", () => {
    const result = applyStudentListControls(students, {
      statusFilter: "active",
      debtFilter: "all",
      balanceFilter: "all",
      ageFilter: "all",
      sortOption: "name_asc",
    });

    expect(result.map((item) => item.id)).toEqual([1, 3]);
  });

  it("filters inactive students only", () => {
    const result = applyStudentListControls(students, {
      statusFilter: "inactive",
      debtFilter: "all",
      balanceFilter: "all",
      ageFilter: "all",
      sortOption: "name_asc",
    });

    expect(result.map((item) => item.id)).toEqual([2]);
  });

  it("filters debt-only and credit-only views", () => {
    const debtOnly = applyStudentListControls(students, {
      statusFilter: "all",
      debtFilter: "debt_only",
      balanceFilter: "all",
      ageFilter: "all",
      sortOption: "name_asc",
    });
    const creditOnly = applyStudentListControls(students, {
      statusFilter: "all",
      debtFilter: "all",
      balanceFilter: "credit_only",
      ageFilter: "all",
      sortOption: "name_asc",
    });

    expect(debtOnly.map((item) => item.id)).toEqual([1]);
    expect(creditOnly.map((item) => item.id)).toEqual([2]);
  });

  it("filters minors only", () => {
    const result = applyStudentListControls(students, {
      statusFilter: "all",
      debtFilter: "all",
      balanceFilter: "all",
      ageFilter: "minors_only",
      sortOption: "name_asc",
    });

    expect(result.map((item) => item.id)).toEqual([2]);
  });

  it("sorts by createdAt descending", () => {
    const result = applyStudentListControls(students, {
      statusFilter: "all",
      debtFilter: "all",
      balanceFilter: "all",
      ageFilter: "all",
      sortOption: "created_desc",
    });

    expect(result.map((item) => item.id)).toEqual([2, 3, 1]);
  });

  it("sorts by debt and balance descending", () => {
    const byDebt = applyStudentListControls(students, {
      statusFilter: "all",
      debtFilter: "all",
      balanceFilter: "all",
      ageFilter: "all",
      sortOption: "debt_desc",
    });
    const byBalance = applyStudentListControls(students, {
      statusFilter: "all",
      debtFilter: "all",
      balanceFilter: "all",
      ageFilter: "all",
      sortOption: "balance_desc",
    });

    expect(byDebt.map((item) => item.id)).toEqual([1, 2, 3]);
    expect(byBalance.map((item) => item.id)).toEqual([2, 1, 3]);
  });
});
