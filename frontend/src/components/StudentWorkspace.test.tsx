import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { StudentWorkspace } from "./StudentWorkspace";
import { createTranslator } from "../lib/i18n";
import type { EnrollmentDTO } from "../lib/enrollments";
import type { StudentDTO } from "../lib/students";

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
    debt: 0,
  },
];

const enrollments: EnrollmentDTO[] = [
  {
    id: 1,
    version: 1,
    studentId: 1,
    studentName: "Anna Student",
    courseId: 1,
    courseName: "English",
    courseType: "group",
    teacherName: "Teacher One",
    billingMode: "per_lesson",
    chargeMaterials: true,
    lessonPriceOverride: 25,
    subscriptionLessonPrice: 0,
    note: "",
    createdAt: "2026-07-01T10:00:00Z",
  },
  {
    id: 2,
    version: 1,
    studentId: 1,
    studentName: "Anna Student",
    courseId: 2,
    courseName: "Latvian",
    courseType: "individual",
    teacherName: "Teacher Two",
    billingMode: "per_lesson",
    chargeMaterials: false,
    lessonPriceOverride: 25,
    subscriptionLessonPrice: 0,
    note: "",
    createdAt: "2026-07-01T10:00:00Z",
  },
];

function buildProps() {
  return {
    students,
    loading: false,
    query: "",
    statusFilter: "active" as const,
    debtFilter: "all" as const,
    balanceFilter: "all" as const,
    ageFilter: "all" as const,
    sortOption: "created_desc" as const,
    hasActiveStudentFilters: false,
    selectedStudent: students[0],
    detailLoading: false,
    detailEnrollments: [],
    detailBalance: null,
    detailDebts: [],
    detailPayments: [],
    detailMonthInvoices: [],
    detailNextAction: null,
    detailActivity: [],
    t: createTranslator("en-US"),
    deletingPaymentId: null,
    onQueryChange: vi.fn(),
    onStatusFilterChange: vi.fn(),
    onDebtFilterChange: vi.fn(),
    onBalanceFilterChange: vi.fn(),
    onAgeFilterChange: vi.fn(),
    onSortOptionChange: vi.fn(),
    onResetStudentFilters: vi.fn(),
    onAddStudent: vi.fn(),
    onSelectStudent: vi.fn(),
    onEditStudent: vi.fn(),
    onToggleActive: vi.fn(),
    onDeleteStudent: vi.fn(),
    onAddPayment: vi.fn(),
    onCopyDebtRu: vi.fn(),
    onCopyDebtLv: vi.fn(),
    onDeletePayment: vi.fn(),
    canDeleteStudent: true,
    canDeletePayment: true,
    onManageEnrollments: vi.fn(),
    onOpenInvoices: vi.fn(),
    payerRoleLabel: vi.fn((value: string) => value),
    billingModeLabel: vi.fn((value: string) => value),
    paymentMethodLabel: vi.fn((value: string) => value),
    invoiceStatusLabel: vi.fn((value: string) => value),
    formatEUR: vi.fn((value: number) => `${value.toFixed(2)} EUR`),
    months: [],
  };
}

describe("StudentWorkspace", () => {
  it("shows quick chips and keeps advanced filters collapsed by default", () => {
    const markup = renderToStaticMarkup(<StudentWorkspace {...buildProps()} />);

    expect(markup).toContain("Add student");
    expect(markup).toContain("Active");
    expect(markup).toContain("Has debt");
    expect(markup).toContain("Recently added");
    expect(markup).toContain("Filters");
    expect(markup).not.toContain("Active only");
    expect(markup).not.toContain("Debt only");
  });

  it("renders advanced filters expanded when a non-chip filter is active", () => {
    const markup = renderToStaticMarkup(
      <StudentWorkspace {...buildProps()} balanceFilter="credit_only" hasActiveStudentFilters />
    );

    expect(markup).toContain("Credit only");
    expect(markup).toContain("Zero or debt");
    expect(markup).toContain("Clear filters");
  });

  it("shows whether materials are charged for each enrollment", () => {
    const markup = renderToStaticMarkup(
      <StudentWorkspace {...buildProps()} detailEnrollments={enrollments} />
    );

    expect(markup).toContain("<th>Materials</th>");
    expect(markup).toContain("<td>Yes</td>");
    expect(markup).toContain("<td>No</td>");
  });
});
