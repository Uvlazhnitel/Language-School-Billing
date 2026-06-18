import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { DashboardOverview } from "./DashboardOverview";
import { createTranslator } from "../lib/i18n";

describe("DashboardOverview", () => {
  it("renders month closing workflow with progress and steps", () => {
    const markup = renderToStaticMarkup(
      <DashboardOverview
        overview={{
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
        }}
        loading={false}
        monthLabel="June 2026"
        t={createTranslator("en-US")}
        formatEUR={(value) => `€${value.toFixed(2)}`}
        paymentMethodLabel={(method) => method}
        onOpenAttendance={vi.fn()}
        onOpenInvoices={vi.fn()}
        onOpenDebtors={vi.fn()}
        onOpenStudents={vi.fn()}
        onOpenStudent={vi.fn()}
        onOpenPaymentQueueStudent={vi.fn()}
        onCopyDebtQueueRu={vi.fn()}
        onCopyDebtQueueLv={vi.fn()}
        recentPayments={[]}
        actionQueue={[]}
      />
    );

    expect(markup).toContain("Month closing");
    expect(markup).toContain("Closing progress");
    expect(markup).toContain("Collecting month data");
    expect(markup).toContain("Month data");
    expect(markup).toContain("Invoices");
    expect(markup).toContain("PDF");
    expect(markup).toContain("Email sending");
    expect(markup).toContain("Debt follow-up");
    expect(markup).toContain("0%");
    expect(markup).toContain("12/14");
    expect(markup).toContain("3 draft invoices still need review before issue.");
    expect(markup).toContain("Sent 5 of 11");
  });
});
