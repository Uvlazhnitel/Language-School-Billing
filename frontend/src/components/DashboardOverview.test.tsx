import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { DashboardOverview } from "./DashboardOverview";
import { buildDashboardTasks } from "./dashboardTasks";
import type { MonthOverviewDTO, RecentPaymentDTO } from "../lib/dashboard";
import { createTranslator } from "../lib/i18n";

const t = createTranslator("en-US");

function createOverview(overrides: Partial<MonthOverviewDTO> = {}): MonthOverviewDTO {
  return {
    year: 2026,
    month: 6,
    activeStudents: 20,
    activeCourses: 6,
    enrollments: 24,
    perLessonEnrollments: 10,
    attendanceFilled: 8,
    attendanceMissing: 2,
    subscriptionCoursesTracked: 4,
    subscriptionFilled: 4,
    subscriptionMissing: 0,
    monthControlTotal: 14,
    monthControlFilled: 12,
    monthControlMissing: 2,
    draftInvoices: 3,
    issuedInvoices: 6,
    paidInvoices: 5,
    pendingPdfInvoices: 4,
    readyPdfInvoices: 7,
    monthInvoicesTotal: 14,
    emailedInvoices: 5,
    notEmailedInvoices: 6,
    overdueInvoicesCount: 2,
    requiredStepsTotal: 3,
    requiredStepsDone: 0,
    monthClosingProgressPct: 0,
    monthClosingStage: "collecting_data",
    monthReadyToClose: false,
    totalIssued: 800,
    totalPaid: 500,
    paymentsMonthTotal: 400,
    paymentsMonthCashTotal: 250,
    paymentsMonthBankTotal: 150,
    unlinkedCreditTotal: 50,
    monthDebtTotal: 120,
    historicalDebtTotal: 80,
    actionQueueCount: 5,
    debtorsCount: 2,
    totalDebt: 200,
    ...overrides,
  };
}

function createPayments(): RecentPaymentDTO[] {
  return ["First Student", "Second Student", "Third Student", "Fourth Student"].map(
    (studentName, index) => ({
      id: index + 1,
      studentId: index + 10,
      studentName,
      amount: 25 + index,
      method: index % 2 === 0 ? "cash" : "bank",
      paidAt: `2026-06-${String(index + 1).padStart(2, "0")}`,
      note: "",
    })
  );
}

function renderDashboard(overview: MonthOverviewDTO, recentPayments: RecentPaymentDTO[] = []) {
  return renderToStaticMarkup(
    <DashboardOverview
      overview={overview}
      loading={false}
      monthLabel="June 2026"
      t={t}
      formatEUR={(value) => `€${value.toFixed(2)}`}
      paymentMethodLabel={(method) => method}
      onOpenAttendance={vi.fn()}
      onOpenInvoices={vi.fn()}
      onOpenDebtors={vi.fn()}
      onOpenStudent={vi.fn()}
      recentPayments={recentPayments}
    />
  );
}

describe("DashboardOverview", () => {
  it("renders four compact KPIs and only three recent payments", () => {
    const markup = renderDashboard(createOverview(), createPayments());

    expect(markup).toContain("Month data incomplete");
    expect(markup).toContain("Invoices to issue");
    expect(markup).toContain("Payments this month");
    expect(markup).toContain("Total debt");
    expect(markup).toContain("€400.00");
    expect(markup).toContain("€200.00");
    expect(markup).toContain("First Student");
    expect(markup).toContain("Third Student");
    expect(markup).not.toContain("Fourth Student");
  });

  it("builds unresolved tasks in workflow order and preserves navigation actions", () => {
    const onOpenAttendance = vi.fn();
    const onOpenInvoices = vi.fn();
    const onOpenDebtors = vi.fn();
    const tasks = buildDashboardTasks({
      overview: createOverview(),
      t,
      formatEUR: (value) => `€${value.toFixed(2)}`,
      onOpenAttendance,
      onOpenInvoices,
      onOpenDebtors,
    });

    expect(tasks.map((task) => task.id)).toEqual(["attendance", "invoices", "pdf", "debt"]);
    tasks[0].onAction();
    tasks[1].onAction();
    tasks[2].onAction();
    tasks[3].onAction();
    expect(onOpenAttendance).toHaveBeenCalledOnce();
    expect(onOpenInvoices).toHaveBeenCalledTimes(2);
    expect(onOpenDebtors).toHaveBeenCalledOnce();
  });

  it("hides completed tasks and renders one ready state", () => {
    const markup = renderDashboard(
      createOverview({
        monthControlMissing: 0,
        monthControlFilled: 14,
        draftInvoices: 0,
        pendingPdfInvoices: 0,
        debtorsCount: 0,
        totalDebt: 0,
      })
    );

    expect(markup).toContain("All important tasks are complete");
    expect(markup).not.toContain("monthly entries still block issuing");
    expect(markup).not.toContain("draft invoices still need review");
    expect(markup).not.toContain("invoices still need a ready PDF");
    expect(markup).not.toContain("students have an outstanding balance");
  });

  it("does not render the removed expanded dashboard sections", () => {
    const markup = renderDashboard(createOverview());

    expect(markup).not.toContain("Closing progress");
    expect(markup).not.toContain("School snapshot");
    expect(markup).not.toContain("Money breakdown");
    expect(markup).not.toContain("Email sending");
  });
});
