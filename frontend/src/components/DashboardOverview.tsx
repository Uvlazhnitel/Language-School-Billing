import { MonthOverviewDTO, RecentPaymentDTO } from "../lib/dashboard";
import { TranslateFn } from "../lib/i18n";
import { DebtorActionQueueItem } from "../lib/studentActivity";

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
  onOpenStudents: () => void;
  onOpenStudent: (studentId: number) => void;
  onOpenPaymentQueueStudent: (studentId: number) => void;
  onCopyDebtQueueRu: (studentId: number) => void;
  onCopyDebtQueueLv: (studentId: number) => void;
  recentPayments: RecentPaymentDTO[];
  actionQueue: DebtorActionQueueItem[];
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
  onOpenStudents,
  onOpenStudent,
  onOpenPaymentQueueStudent,
  onCopyDebtQueueRu,
  onCopyDebtQueueLv,
  recentPayments,
  actionQueue,
}: DashboardOverviewProps) {
  if (loading) {
    return <div className="empty">{t("msg.dashboardLoading")}</div>;
  }

  if (!overview) {
    return <div className="empty">{t("msg.dashboardError")}</div>;
  }

  const attendanceReady = overview.perLessonEnrollments > 0 && overview.attendanceMissing === 0;
  const topDebtor = actionQueue[0] ?? null;
  const noRecentPaymentCount = actionQueue.filter((item) => !item.hasRecentPayment).length;

  return (
    <div className="dashboard">
      <div className="dashboardHero">
        <div>
          <div className="dashboardHeroEyebrow">{t("label.monthlyOverview")}</div>
          <h2>{monthLabel}</h2>
          <p>{t("msg.dashboardIntro")}</p>
        </div>
        <div className="dashboardHeroStats">
          <div className="dashboardHeroStat">
            <span>{t("field.activeStudents")}</span>
            <strong>{overview.activeStudents}</strong>
          </div>
          <div className="dashboardHeroStat">
            <span>{t("field.activeCourses")}</span>
            <strong>{overview.activeCourses}</strong>
          </div>
          <div className="dashboardHeroStat">
            <span>{t("field.enrollments")}</span>
            <strong>{overview.enrollments}</strong>
          </div>
        </div>
      </div>

      <div className="dashboardGrid">
        <section className="dashboardCard">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("tabs.invoice")}</div>
              <h3>{t("label.needIssueInvoices")}</h3>
            </div>
            <span className={`statusPill ${overview.draftInvoices > 0 ? "warning" : "success"}`}>
              {overview.draftInvoices > 0
                ? t("msg.overviewDrafts", { count: overview.draftInvoices })
                : t("msg.ready")}
            </span>
          </div>
          <p className="dashboardCardLead">
            {t("label.invoiced")} {formatEUR(overview.totalIssued)}, {t("label.paid").toLowerCase()}{" "}
            {formatEUR(overview.totalPaid)}.
          </p>
          <div className="dashboardMetrics">
            <div>
              <span>{t("field.issuedInvoices")}</span>
              <strong>{overview.issuedInvoices}</strong>
            </div>
            <div>
              <span>{t("field.paidInvoices")}</span>
              <strong>{overview.paidInvoices}</strong>
            </div>
          </div>
          <div className="dashboardActions">
            <button
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={onOpenInvoices}
            >
              {t("button.openInvoices")}
            </button>
          </div>
        </section>

        <section className="dashboardCard">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("label.monthControl")}</div>
              <h3>{t("label.attendanceIncomplete")}</h3>
            </div>
            <span className={`statusPill ${attendanceReady ? "success" : "warning"}`}>
              {attendanceReady
                ? t("msg.canIssue")
                : t("msg.leftCount", { count: overview.attendanceMissing })}
            </span>
          </div>
          <p className="dashboardCardLead">{t("field.filledAttendance", { filled: overview.attendanceFilled, total: overview.perLessonEnrollments })}</p>
          <div className="dashboardProgress">
            <div
              className="dashboardProgressValue"
              style={{
                width:
                  overview.perLessonEnrollments > 0
                    ? `${(overview.attendanceFilled / overview.perLessonEnrollments) * 100}%`
                    : "0%",
              }}
            />
          </div>
          <div className="dashboardActions">
            <button
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={onOpenAttendance}
            >
              {t("button.openAttendance")}
            </button>
          </div>
        </section>

        <section className="dashboardCard dashboardCard--compact">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("tabs.debtors")}</div>
              <h3>{t("label.outstandingDebt")}</h3>
            </div>
            <span className={`statusPill ${overview.totalDebt > 0 ? "danger" : "success"}`}>
              {t("msg.studentsCount", { count: overview.debtorsCount })}
            </span>
          </div>
          <p className="dashboardCardLead">{formatEUR(overview.totalDebt)}</p>
          <div className="dashboardMetrics">
            <div>
              <span>{t("field.needsAttention")}</span>
              <strong>{overview.debtorsCount}</strong>
            </div>
            <div>
              <span>{t("field.noRecentPayment")}</span>
              <strong>{noRecentPaymentCount}</strong>
            </div>
            <div>
              <span>{t("field.biggestDebt")}</span>
              <strong>{topDebtor ? formatEUR(topDebtor.debt) : "—"}</strong>
            </div>
            <div>
              <span>{t("field.latestRisk")}</span>
              <strong>{topDebtor ? topDebtor.studentName : "—"}</strong>
            </div>
          </div>
          <div className="dashboardActions">
            <button
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={onOpenDebtors}
            >
              {t("button.openDebts")}
            </button>
          </div>
        </section>

        <section className="dashboardCard dashboardCard--queue">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("label.actionQueue")}</div>
              <h3>{t("label.needsAction")}</h3>
            </div>
            <span className={`statusPill ${actionQueue.length > 0 ? "warning" : "success"}`}>
              {actionQueue.length > 0
                ? t("msg.priorityCount", { count: actionQueue.length })
                : t("msg.ready")}
            </span>
          </div>
          {actionQueue.length === 0 ? (
            <div className="empty">{t("msg.noActionQueue")}</div>
          ) : (
            <div className="actionQueueList">
              {actionQueue.slice(0, 3).map((item) => (
                <div key={item.studentId} className="actionQueueItem">
                  <div className="actionQueueContent">
                    <strong>{item.studentName}</strong>
                    <span>{item.subtitle}</span>
                  </div>
                  <div className="actionQueueMeta">
                    <strong>{formatEUR(item.debt)}</strong>
                    <div className="actionQueueActions">
                      <button
                        className="workspaceActionButton workspaceActionButtonPrimary actionQueuePrimary"
                        onClick={() => onOpenPaymentQueueStudent(item.studentId)}
                      >
                        {t("button.payment")}
                      </button>
                      <button
                        className="secondaryActionButton"
                        onClick={() => onOpenStudent(item.studentId)}
                      >
                        {t("button.card")}
                      </button>
                      <div className="actionQueueSecondaryGroup">
                        <button
                          className="secondaryActionButton secondaryActionButton--mini"
                          onClick={() => onCopyDebtQueueRu(item.studentId)}
                        >
                          RU
                        </button>
                        <button
                          className="secondaryActionButton secondaryActionButton--mini"
                          onClick={() => onCopyDebtQueueLv(item.studentId)}
                        >
                          LV
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="dashboardCard dashboardCard--feed dashboardCard--activity">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("label.monthReview")}</div>
              <h3>{t("label.recentPayments")}</h3>
            </div>
            <button className="secondaryActionButton" onClick={onOpenStudents}>
              {t("button.showStudents")}
            </button>
          </div>

          {recentPayments.length === 0 ? (
            <div className="empty">{t("msg.noRecentPayments")}</div>
          ) : (
            <div className="activityFeed">
              {recentPayments.map((payment) => (
                <button
                  key={payment.id}
                  type="button"
                  className="activityFeedItem"
                  onClick={() => onOpenStudent(payment.studentId)}
                >
                  <div>
                    <strong>{payment.studentName}</strong>
                    <span>
                      {paymentMethodLabel(payment.method)} · {formatFeedDate(payment.paidAt)}
                    </span>
                  </div>
                  <div className="activityFeedMeta">
                    <strong>{formatEUR(payment.amount)}</strong>
                    {payment.invoiceId ? (
                      <span>Счёт #{payment.invoiceId}</span>
                    ) : (
                      <span>{t("msg.noInvoice")}</span>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
