import { InvoiceListItemView } from "./invoices";
import { DebtInvoiceDTO, DebtorDTO, PaymentDTO } from "./payments";
import { EnrollmentDTO } from "./enrollments";

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
}): StudentNextAction {
  const { debt, enrollments, debts, payments, monthInvoices } = args;

  if (debt > 0 || debts.length > 0) {
    return {
      label: "Принять оплату",
      secondaryLabel: "Напомнить о долге",
      reason: `Есть открытый долг ${debt > 0 ? `на €${debt.toFixed(2)}` : "по счетам ученика"}.`,
      action: "payment",
      outstandingIssue:
        debts.length > 0 ? `${debts.length} открытых счетов с остатком` : "Есть долг по балансу",
    };
  }

  if (enrollments.length === 0) {
    return {
      label: "Добавить зачисление",
      secondaryLabel: "Открыть ученика",
      reason: "У ученика пока нет активных зачислений.",
      action: "enrollment",
      outstandingIssue: "Нет зачислений",
    };
  }

  if (monthInvoices.length > 0) {
    const draftInvoice = monthInvoices.find((invoice) => invoice.status === "draft");
    if (draftInvoice) {
      return {
        label: "Открыть счета",
        secondaryLabel: "Проверить черновик",
        reason: "За текущий месяц уже есть черновик счёта.",
        action: "invoice",
        outstandingIssue: "Есть черновик счета за текущий месяц",
      };
    }
  }

  if (payments.length === 0) {
    return {
      label: "Записать оплату",
      secondaryLabel: "Открыть счета",
      reason: "По ученику ещё нет зарегистрированных оплат.",
      action: "payment",
      outstandingIssue: "Нет истории оплат",
    };
  }

  return {
    label: "Открыть счета",
    secondaryLabel: "Записать оплату",
    reason: "У ученика нет долгов, можно продолжать регулярный billing workflow.",
    action: "invoice",
    outstandingIssue: "Явных проблем нет",
  };
}

export function buildStudentActivity(args: {
  enrollments: EnrollmentDTO[];
  payments: PaymentDTO[];
  debts: DebtInvoiceDTO[];
  monthInvoices: InvoiceListItemView[];
  months: string[];
}): StudentActivityItem[] {
  const { enrollments, payments, debts, monthInvoices, months } = args;
  const items: StudentActivityItem[] = [];

  for (const payment of payments.slice(0, 6)) {
    items.push({
      id: `payment-${payment.id}`,
      kind: "payment",
      date: payment.paidAt,
      title: `Оплата ${payment.amount.toFixed(2)} EUR`,
      subtitle: payment.note || `Способ: ${payment.method}`,
      amount: payment.amount,
      actionTarget: "payment",
    });
  }

  for (const debt of debts.slice(0, 4)) {
    items.push({
      id: `debt-${debt.invoiceId}`,
      kind: "debt",
      date: monthIsoDate(debt.year, debt.month),
      title: `Открытый долг за ${months[debt.month - 1]} ${debt.year}`,
      subtitle: debt.number ? `Счёт ${debt.number}` : "Счёт без номера",
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
      title: `Активное зачисление: ${enrollment.courseName}`,
      subtitle: `${enrollment.teacherName || "Без учителя"} · ${enrollment.billingMode}`,
      actionTarget: "enrollment",
    });
  }

  for (const invoice of monthInvoices.slice(0, 3)) {
    items.push({
      id: `invoice-${invoice.id}`,
      kind: "invoice",
      date: monthIsoDate(invoice.year, invoice.month),
      title: `Счёт за ${months[invoice.month - 1]} ${invoice.year}`,
      subtitle: invoice.number ? `Номер ${invoice.number}` : "Черновик без номера",
      amount: invoice.total,
      status: invoice.status,
      actionTarget: "invoice",
    });
  }

  return items.sort((a, b) => String(b.date).localeCompare(String(a.date))).slice(0, 12);
}

export function buildDebtorActionQueue(
  debtors: DebtorDTO[],
  recentPayments: PaymentDTO[] | Array<{ studentId: number }>
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
          ? "Есть недавняя оплата, но долг ещё открыт"
          : "Нужен follow-up по открытому долгу",
        hasRecentPayment,
      };
    })
    .sort((a, b) => b.priority - a.priority)
    .slice(0, 5);
}
