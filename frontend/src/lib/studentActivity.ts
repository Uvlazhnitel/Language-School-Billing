import { InvoiceListItemView } from "./invoices";
import { DebtInvoiceDTO, DebtorDTO, PaymentDTO } from "./payments";
import { EnrollmentDTO } from "./enrollments";
import { TranslateFn } from "./i18n";

export type StudentActivityItem = {
  id: string;
  kind: "payment" | "debt" | "enrollment" | "invoice";
  date: string;
  title: string;
  subtitle: string;
  amount?: number;
  status?: string;
  actionTarget?: string;
};

export type StudentNextAction = {
  label: string;
  secondaryLabel: string;
  reason: string;
  action: "payment" | "reminder" | "enrollment" | "invoice";
  outstandingIssue: string;
};

export type DebtorActionQueueItem = {
  studentId: number;
  studentName: string;
  debt: number;
  priority: number;
  subtitle: string;
  hasRecentPayment: boolean;
};

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

function monthIsoDate(year: number, month: number): string {
  return `${year}-${pad(month)}-01`;
}

export function buildStudentNextAction(args: {
  debt: number;
  enrollments: EnrollmentDTO[];
  debts: DebtInvoiceDTO[];
  payments: PaymentDTO[];
  monthInvoices: InvoiceListItemView[];
  t: TranslateFn;
}): StudentNextAction {
  const { debt, enrollments, debts, payments, monthInvoices, t } = args;

  if (debt > 0 || debts.length > 0) {
    return {
      label: t("action.takePayment"),
      secondaryLabel: t("action.debtReminder"),
      reason: t("msg.outstandingDebtReason", {
        amount: debt > 0 ? `€${debt.toFixed(2)}` : t("msg.debtBalanceIssue"),
      }),
      action: "payment",
      outstandingIssue:
        debts.length > 0
          ? t("msg.debtInvoicesIssue", { count: debts.length })
          : t("msg.debtBalanceIssue"),
    };
  }

  if (enrollments.length === 0) {
    return {
      label: t("action.addEnrollment"),
      secondaryLabel: t("action.openStudent"),
      reason: t("msg.noEnrollmentsReason"),
      action: "enrollment",
      outstandingIssue: t("msg.noEnrollmentsIssue"),
    };
  }

  if (monthInvoices.length > 0) {
    const draftInvoice = monthInvoices.find((invoice) => invoice.status === "draft");
    if (draftInvoice) {
      return {
        label: t("action.openInvoices"),
        secondaryLabel: t("action.checkDraft"),
        reason: t("msg.draftInvoiceReason"),
        action: "invoice",
        outstandingIssue: t("msg.invoiceDraftIssue"),
      };
    }
  }

  if (payments.length === 0) {
    return {
      label: t("action.recordPayment"),
      secondaryLabel: t("action.openInvoices"),
      reason: t("msg.noPaymentsReason"),
      action: "payment",
      outstandingIssue: t("msg.noPaymentHistoryIssue"),
    };
  }

  return {
    label: t("action.openInvoices"),
    secondaryLabel: t("action.recordPayment"),
    reason: t("msg.workflowOk"),
    action: "invoice",
    outstandingIssue: t("msg.noProblem"),
  };
}

export function buildStudentActivity(args: {
  enrollments: EnrollmentDTO[];
  payments: PaymentDTO[];
  debts: DebtInvoiceDTO[];
  monthInvoices: InvoiceListItemView[];
  months: string[];
  paymentMethodLabel: (method: string) => string;
  billingModeLabel: (mode: string) => string;
  t: TranslateFn;
}): StudentActivityItem[] {
  const { enrollments, payments, debts, monthInvoices, months, paymentMethodLabel, billingModeLabel, t } =
    args;
  const items: StudentActivityItem[] = [];

  for (const payment of payments.slice(0, 6)) {
    items.push({
      id: `payment-${payment.id}`,
      kind: "payment",
      date: payment.paidAt,
      title: `${t("action.recordPayment")} ${payment.amount.toFixed(2)} EUR`,
      subtitle: payment.note || `${t("field.method")}: ${paymentMethodLabel(payment.method)}`,
      amount: payment.amount,
      actionTarget: "payment",
    });
  }

  for (const debt of debts.slice(0, 4)) {
    items.push({
      id: `debt-${debt.invoiceId}`,
      kind: "debt",
      date: monthIsoDate(debt.year, debt.month),
      title: `${t("label.openDebts")} ${months[debt.month - 1]} ${debt.year}`,
      subtitle: debt.number ? `${t("tabs.invoice")} ${debt.number}` : `${t("tabs.invoice")} —`,
      amount: debt.remaining,
      status: debt.status,
      actionTarget: "debt",
    });
  }

  for (const enrollment of enrollments.slice(0, 4)) {
    items.push({
      id: `enrollment-${enrollment.id}`,
      kind: "enrollment",
      date: `9999-12-${pad(Math.max(1, 31 - enrollment.id))}`,
      title: `${t("action.addEnrollment")}: ${enrollment.courseName}`,
      subtitle: `${enrollment.teacherName || t("field.teacher")} · ${billingModeLabel(enrollment.billingMode)}`,
      actionTarget: "enrollment",
    });
  }

  for (const invoice of monthInvoices.slice(0, 3)) {
    items.push({
      id: `invoice-${invoice.id}`,
      kind: "invoice",
      date: monthIsoDate(invoice.year, invoice.month),
      title: `${t("tabs.invoice")} ${months[invoice.month - 1]} ${invoice.year}`,
      subtitle: invoice.number ? `${t("field.number")} ${invoice.number}` : t("status.draft"),
      amount: invoice.total,
      status: invoice.status,
      actionTarget: "invoice",
    });
  }

  return items.sort((a, b) => String(b.date).localeCompare(String(a.date))).slice(0, 12);
}

export function buildDebtorActionQueue(
  debtors: DebtorDTO[],
  recentPayments: PaymentDTO[] | Array<{ studentId: number }>,
  t: TranslateFn
): DebtorActionQueueItem[] {
  const recentStudentIds = new Set(recentPayments.map((payment) => payment.studentId));

  return debtors
    .map((debtor) => {
      const hasRecentPayment = recentStudentIds.has(debtor.studentId);
      const priority = Math.round(debtor.debt * 100) + (hasRecentPayment ? -250 : 250);
      return {
        studentId: debtor.studentId,
        studentName: debtor.studentName,
        debt: debtor.debt,
        priority,
        subtitle: hasRecentPayment
          ? t("msg.recentPaymentButDebt")
          : t("msg.followUpDebt"),
        hasRecentPayment,
      };
    })
    .sort((a, b) => b.priority - a.priority)
    .slice(0, 5);
}
