import { MonthOverviewDTO } from "../lib/dashboard";
import { TranslateFn } from "../lib/i18n";

export type DashboardTask = {
  id: "attendance" | "invoices" | "pdf" | "debt";
  title: string;
  subtitle: string;
  value: string;
  tone: "warning" | "danger";
  actionLabel: string;
  onAction: () => void;
};

type BuildDashboardTasksInput = {
  overview: MonthOverviewDTO;
  t: TranslateFn;
  formatEUR: (value: number) => string;
  onOpenAttendance: () => void;
  onOpenInvoices: () => void;
  onOpenDebtors: () => void;
};

export function buildDashboardTasks({
  overview,
  t,
  formatEUR,
  onOpenAttendance,
  onOpenInvoices,
  onOpenDebtors,
}: BuildDashboardTasksInput): DashboardTask[] {
  const tasks: DashboardTask[] = [];

  if (overview.monthControlMissing > 0) {
    tasks.push({
      id: "attendance",
      title: t("label.monthDataIncomplete"),
      subtitle: t("msg.monthControlBlocksIssuing", { count: overview.monthControlMissing }),
      value: `${overview.monthControlFilled}/${overview.monthControlTotal}`,
      tone: "warning",
      actionLabel: t("button.openAttendance"),
      onAction: onOpenAttendance,
    });
  }

  if (overview.draftInvoices > 0) {
    tasks.push({
      id: "invoices",
      title: t("label.needIssueInvoices"),
      subtitle: t("msg.draftInvoicesNeedReview", { count: overview.draftInvoices }),
      value: String(overview.draftInvoices),
      tone: "warning",
      actionLabel: t("button.openInvoices"),
      onAction: onOpenInvoices,
    });
  }

  if (overview.pendingPdfInvoices > 0) {
    tasks.push({
      id: "pdf",
      title: t("label.dashboardPendingPdfs"),
      subtitle: t("msg.monthClosingRemainingPdfs", { count: overview.pendingPdfInvoices }),
      value: String(overview.pendingPdfInvoices),
      tone: "warning",
      actionLabel: t("button.openInvoices"),
      onAction: onOpenInvoices,
    });
  }

  if (overview.totalDebt > 0) {
    tasks.push({
      id: "debt",
      title: t("label.outstandingDebt"),
      subtitle: t("msg.dashboardDebtAction", { count: overview.debtorsCount }),
      value: formatEUR(overview.totalDebt),
      tone: "danger",
      actionLabel: t("button.openDebts"),
      onAction: onOpenDebtors,
    });
  }

  return tasks;
}
