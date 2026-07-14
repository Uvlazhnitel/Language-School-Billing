import { MonthOverviewDTO, RecentPaymentDTO } from "../lib/dashboard";
import { TranslateFn } from "../lib/i18n";
import { buildDashboardTasks } from "./dashboardTasks";

type DashboardOverviewProps = {
  overview: MonthOverviewDTO | null;
  loading: boolean;
  monthLabel: string;
  t: TranslateFn;
  formatEUR: (value: number) => string;
  paymentMethodLabel: (method: string) => string;
  onOpenAttendance: () => void;
  onOpenInvoices: () => void;
  onOpenDebtors: () => void;
  onOpenStudent: (studentId: number) => void;
  recentPayments: RecentPaymentDTO[];
};

function formatFeedDate(value: string): string {
  return value.slice(0, 10);
}

export function DashboardOverview({
  overview,
  loading,
  monthLabel,
  t,
  formatEUR,
  paymentMethodLabel,
  onOpenAttendance,
  onOpenInvoices,
  onOpenDebtors,
  onOpenStudent,
  recentPayments,
}: DashboardOverviewProps) {
  if (loading) {
    return <div className="empty">{t("msg.dashboardLoading")}</div>;
  }

  if (!overview) {
    return <div className="empty">{t("msg.dashboardError")}</div>;
  }

  const tasks = buildDashboardTasks({
    overview,
    t,
    formatEUR,
    onOpenAttendance,
    onOpenInvoices,
    onOpenDebtors,
  });
  const recentPaymentItems = recentPayments.slice(0, 3);

  return (
    <div className="dashboard dashboard--compact">
      <section className="dashboardKpiGrid" aria-label={t("label.monthlyOverview")}>
        <button type="button" className="dashboardKpi dashboardKpi--action" onClick={onOpenAttendance}>
          <span>{t("label.monthDataIncomplete")}</span>
          <strong>{overview.monthControlMissing}</strong>
        </button>
        <button type="button" className="dashboardKpi dashboardKpi--action" onClick={onOpenInvoices}>
          <span>{t("label.needIssueInvoices")}</span>
          <strong>{overview.draftInvoices}</strong>
        </button>
        <div className="dashboardKpi dashboardKpi--positive">
          <span>{t("label.monthPayments")}</span>
          <strong>{formatEUR(overview.paymentsMonthTotal)}</strong>
        </div>
        <button
          type="button"
          className={`dashboardKpi dashboardKpi--action ${overview.totalDebt > 0 ? "dashboardKpi--danger" : "dashboardKpi--positive"}`}
          onClick={onOpenDebtors}
        >
          <span>{t("label.dashboardTotalDebt")}</span>
          <strong>{formatEUR(overview.totalDebt)}</strong>
        </button>
      </section>

      <div className="dashboardOperationalGrid">
        <section className="dashboardCompactPanel dashboardTaskPanel">
          <div className="dashboardCompactHeader">
            <div>
              <div className="dashboardCardEyebrow">{monthLabel}</div>
              <h3>{t("label.needsAction")}</h3>
            </div>
            <span className={`statusPill ${tasks.length > 0 ? "warning" : "success"}`}>
              {tasks.length > 0
                ? t("msg.priorityCount", { count: tasks.length })
                : t("msg.ready")}
            </span>
          </div>

          {tasks.length === 0 ? (
            <div className="dashboardAllDone">
              <strong>{t("msg.dashboardAllDone")}</strong>
              <span>{t("msg.dashboardAllDoneHint")}</span>
            </div>
          ) : (
            <div className="dashboardTaskList">
              {tasks.map((task) => (
                <div key={task.id} className={`dashboardTask dashboardTask--${task.tone}`}>
                  <div className="dashboardTaskContent">
                    <strong>{task.title}</strong>
                    <span>{task.subtitle}</span>
                  </div>
                  <strong className="dashboardTaskValue">{task.value}</strong>
                  <button type="button" className="secondaryActionButton" onClick={task.onAction}>
                    {task.actionLabel}
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="dashboardCompactPanel dashboardPaymentsPanel">
          <div className="dashboardCompactHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("label.monthReview")}</div>
              <h3>{t("label.recentPayments")}</h3>
            </div>
          </div>

          {recentPaymentItems.length === 0 ? (
            <div className="empty">{t("msg.noRecentPayments")}</div>
          ) : (
            <div className="dashboardPaymentList">
              {recentPaymentItems.map((payment) => (
                <button
                  key={payment.id}
                  type="button"
                  className="dashboardPaymentItem"
                  onClick={() => onOpenStudent(payment.studentId)}
                >
                  <div>
                    <strong>{payment.studentName}</strong>
                    <span>
                      {paymentMethodLabel(payment.method)} · {formatFeedDate(payment.paidAt)}
                    </span>
                  </div>
                  <strong>{formatEUR(payment.amount)}</strong>
                </button>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
