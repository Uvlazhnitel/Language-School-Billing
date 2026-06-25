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

type ActionBlocker = {
  id: string;
  eyebrow: string;
  title: string;
  subtitle: string;
  amount?: string;
  emphasis?: "warning" | "danger";
  primaryLabel: string;
  onPrimaryAction: () => void;
};

type ClosingStep = {
  id: string;
  title: string;
  subtitle: string;
  amount: string;
  tone: "success" | "warning" | "info";
  statusLabel: string;
  primaryLabel: string;
  onPrimaryAction: () => void;
};

function formatFeedDate(value: string): string {
  return value.slice(0, 10);
}

function actionReasonLabels(item: DebtorActionQueueItem, index: number, t: TranslateFn): string[] {
  const labels: string[] = [];
  if (index === 0) {
    labels.push(t("msg.largestDebt"));
  }
  if (!item.hasRecentPayment) {
    labels.push(t("msg.noRecentPaymentLabel"));
  }
  if (labels.length === 0) {
    labels.push(t("msg.recentPaymentButDebt"));
  }
  return labels;
}

function monthClosingStageLabel(stage: MonthOverviewDTO["monthClosingStage"], t: TranslateFn): string {
  return t(`msg.monthClosingStage.${stage}`);
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

  const monthControlReady = overview.monthControlTotal === 0 || overview.monthControlMissing === 0;
  const stageLabel = monthClosingStageLabel(overview.monthClosingStage, t);
  const closingSummary = overview.monthReadyToClose
    ? t("msg.monthReadyToClose")
    : overview.monthControlMissing > 0
      ? t("msg.monthClosingRemainingData", { count: overview.monthControlMissing })
      : overview.draftInvoices > 0
        ? t("msg.monthClosingRemainingDrafts", { count: overview.draftInvoices })
        : overview.pendingPdfInvoices > 0
          ? t("msg.monthClosingRemainingPdfs", { count: overview.pendingPdfInvoices })
          : overview.notEmailedInvoices > 0
            ? t("msg.monthClosingRemainingEmails", { count: overview.notEmailedInvoices })
            : t("msg.ready");

  const closingSteps: ClosingStep[] = [
    {
      id: "data",
      title: t("msg.monthClosingStep.data"),
      subtitle: t("field.filledMonthControl", {
        filled: overview.monthControlFilled,
        total: overview.monthControlTotal,
      }),
      amount: `${overview.monthControlFilled}/${overview.monthControlTotal}`,
      tone: overview.monthControlMissing === 0 ? "success" : "warning",
      statusLabel:
        overview.monthControlMissing === 0 ? t("label.stepDone") : t("label.stepNeedsAction"),
      primaryLabel: t("button.openAttendance"),
      onPrimaryAction: onOpenAttendance,
    },
    {
      id: "invoices",
      title: t("msg.monthClosingStep.invoices"),
      subtitle:
        overview.draftInvoices > 0
          ? t("msg.draftInvoicesNeedReview", { count: overview.draftInvoices })
          : t("msg.ready"),
      amount: String(overview.draftInvoices),
      tone: overview.draftInvoices === 0 ? "success" : "warning",
      statusLabel:
        overview.draftInvoices === 0 ? t("label.stepDone") : t("label.stepNeedsAction"),
      primaryLabel: t("button.openInvoices"),
      onPrimaryAction: onOpenInvoices,
    },
    {
      id: "pdf",
      title: t("msg.monthClosingStep.pdf"),
      subtitle: t("msg.monthClosingInvoicesReady", {
        ready: overview.readyPdfInvoices,
        total: overview.monthInvoicesTotal,
      }),
      amount: String(overview.pendingPdfInvoices),
      tone: overview.pendingPdfInvoices === 0 ? "success" : "warning",
      statusLabel:
        overview.pendingPdfInvoices === 0 ? t("label.stepDone") : t("label.stepNeedsAction"),
      primaryLabel: t("button.openInvoices"),
      onPrimaryAction: onOpenInvoices,
    },
    {
      id: "email",
      title: t("msg.monthClosingStep.email"),
      subtitle: t("msg.monthClosingEmailsSent", {
        sent: overview.emailedInvoices,
        total: overview.readyPdfInvoices + overview.pendingPdfInvoices,
      }),
      amount: String(overview.notEmailedInvoices),
      tone: "info",
      statusLabel: t("label.optionalStep"),
      primaryLabel: t("button.openInvoices"),
      onPrimaryAction: onOpenInvoices,
    },
    {
      id: "debt",
      title: t("msg.monthClosingStep.debt"),
      subtitle:
        overview.historicalDebtTotal > 0
          ? t("msg.monthClosingHistoricalDebt", { count: overview.overdueInvoicesCount })
          : t("msg.noActionQueue"),
      amount: formatEUR(overview.historicalDebtTotal),
      tone: "info",
      statusLabel: t("label.followUp"),
      primaryLabel: t("button.openDebts"),
      onPrimaryAction: onOpenDebtors,
    },
  ];

  const blockers: ActionBlocker[] = [];
  if (overview.monthControlMissing > 0) {
    blockers.push({
      id: "attendance",
      eyebrow: t("label.workflowBlockers"),
      title: t("label.monthDataIncomplete"),
      subtitle: t("msg.monthControlBlocksIssuing", { count: overview.monthControlMissing }),
      amount: `${overview.monthControlFilled}/${overview.monthControlTotal}`,
      emphasis: "warning",
      primaryLabel: t("button.openAttendance"),
      onPrimaryAction: onOpenAttendance,
    });
  }
  if (overview.draftInvoices > 0) {
    blockers.push({
      id: "drafts",
      eyebrow: t("tabs.invoice"),
      title: t("label.needIssueInvoices"),
      subtitle: t("msg.draftInvoicesNeedReview", { count: overview.draftInvoices }),
      amount: String(overview.draftInvoices),
      emphasis: "warning",
      primaryLabel: t("button.openInvoices"),
      onPrimaryAction: onOpenInvoices,
    });
  }
  if (overview.historicalDebtTotal > 0) {
    blockers.push({
      id: "historical-debt",
      eyebrow: t("tabs.debtors"),
      title: t("label.olderDebt"),
      subtitle: t("msg.historicalDebtNeedsAttention", {
        count: overview.overdueInvoicesCount,
      }),
      amount: formatEUR(overview.historicalDebtTotal),
      emphasis: "danger",
      primaryLabel: t("button.openDebts"),
      onPrimaryAction: onOpenDebtors,
    });
  }

  return (
    <div className="dashboard">
      <section className="dashboardCard dashboardCard--closing">
        <div className="dashboardCardHeader">
          <div>
            <div className="dashboardCardEyebrow">{t("label.monthClosing")}</div>
            <h3>{monthLabel}</h3>
          </div>
          <span className={`statusPill ${overview.monthReadyToClose ? "success" : "warning"}`}>
            {stageLabel}
          </span>
        </div>
        <p className="dashboardCardLead">{closingSummary}</p>
        <p className="dashboardCardCaption">{t("msg.monthClosingIntro")}</p>
        <div className="dashboardClosingSummary">
          <div className="dashboardClosingSummaryCard">
            <span>{t("label.closingProgress")}</span>
            <strong>{overview.monthClosingProgressPct}%</strong>
            <small>{t("msg.monthClosingMetric", {
              done: overview.requiredStepsDone,
              total: overview.requiredStepsTotal,
            })}</small>
          </div>
          <div className="dashboardClosingSummaryCard">
            <span>{t("label.currentStage")}</span>
            <strong>{stageLabel}</strong>
            <small>{overview.monthReadyToClose ? t("msg.ready") : closingSummary}</small>
          </div>
        </div>
        <div className="dashboardProgress">
          <div
            className="dashboardProgressValue"
            style={{ width: `${overview.monthClosingProgressPct}%` }}
          />
        </div>
        <div className="dashboardClosingSteps">
          {closingSteps.map((step) => (
            <div key={step.id} className={`dashboardStep dashboardStep--${step.tone}`}>
              <div className="dashboardStepMain">
                <div className="dashboardStepHeader">
                  <strong>{step.title}</strong>
                  <span className={`statusPill ${step.tone === "success" ? "success" : step.tone === "warning" ? "warning" : ""}`}>
                    {step.statusLabel}
                  </span>
                </div>
                <span>{step.subtitle}</span>
              </div>
              <div className="dashboardStepMeta">
                <strong>{step.amount}</strong>
                <button className="secondaryActionButton" onClick={step.onPrimaryAction}>
                  {step.primaryLabel}
                </button>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="dashboardCard dashboardCard--overview">
        <div className="dashboardOverviewHeader">
          <div className="dashboardOverviewIntro">
            <div className="dashboardCardEyebrow">{t("label.monthlyOverview")}</div>
            <h2>{monthLabel}</h2>
            <p>{t("msg.dashboardIntro")}</p>
          </div>
          <div className="dashboardOverviewStats">
            <div className="dashboardOverviewStat">
              <span>{t("label.monthDataIncomplete")}</span>
              <strong>{overview.monthControlMissing}</strong>
            </div>
            <div className="dashboardOverviewStat">
              <span>{t("label.needIssueInvoices")}</span>
              <strong>{overview.draftInvoices}</strong>
            </div>
            <div className="dashboardOverviewStat">
              <span>{t("label.monthDebt")}</span>
              <strong>{formatEUR(overview.monthDebtTotal)}</strong>
            </div>
            <div className="dashboardOverviewStat">
              <span>{t("label.monthPayments")}</span>
              <strong>{formatEUR(overview.paymentsMonthTotal)}</strong>
            </div>
          </div>
        </div>
      </section>

      <section className="dashboardCard dashboardCard--queue">
        <div className="dashboardCardHeader">
          <div>
            <div className="dashboardCardEyebrow">{t("label.actionQueue")}</div>
            <h3>{t("label.needsAction")}</h3>
          </div>
          <span className={`statusPill ${overview.actionQueueCount > 0 ? "warning" : "success"}`}>
            {overview.actionQueueCount > 0
              ? t("msg.priorityCount", { count: overview.actionQueueCount })
              : t("msg.ready")}
          </span>
        </div>

        <div className="actionQueueList actionQueueList--dashboard">
          {blockers.map((blocker) => (
            <div
              key={blocker.id}
              className={`actionQueueItem actionQueueItem--${blocker.emphasis ?? "warning"}`}
            >
              <div className="actionQueueContent">
                <span className="dashboardCardEyebrow">{blocker.eyebrow}</span>
                <strong>{blocker.title}</strong>
                <span>{blocker.subtitle}</span>
              </div>
              <div className="actionQueueMeta">
                <strong>{blocker.amount ?? "—"}</strong>
                <div className="actionQueueActions">
                  <button
                    className="workspaceActionButton workspaceActionButtonPrimary actionQueuePrimary"
                    onClick={blocker.onPrimaryAction}
                  >
                    {blocker.primaryLabel}
                  </button>
                </div>
              </div>
            </div>
          ))}

          {actionQueue.slice(0, 3).map((item, index) => (
            <div key={item.studentId} className="actionQueueItem actionQueueItem--student">
              <div className="actionQueueItemTop">
                <div className="actionQueueContent">
                  <strong>{item.studentName}</strong>
                  <span>{item.subtitle}</span>
                </div>
                <div className="actionQueueMeta">
                  <strong>{formatEUR(item.debt)}</strong>
                </div>
              </div>
              <div className="actionQueueItemBottom">
                <div className="actionQueueReasonTags">
                  {actionReasonLabels(item, index, t).map((label) => (
                    <span key={label} className="statusPill warning">
                      {label}
                    </span>
                  ))}
                </div>
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

          {blockers.length === 0 && actionQueue.length === 0 && (
            <div className="empty">{t("msg.noActionQueue")}</div>
          )}
        </div>
      </section>

      <div className="dashboardSplitGrid">
        <section className="dashboardCard dashboardCard--money">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("label.monthReview")}</div>
              <h3>{t("label.moneyFlow")}</h3>
            </div>
            <span className={`statusPill ${overview.totalDebt > 0 ? "danger" : "success"}`}>
              {overview.totalDebt > 0 ? t("label.outstandingDebt") : t("msg.ready")}
            </span>
          </div>
          <div className="dashboardMetrics dashboardMetrics--expanded">
            <div>
              <span>{t("label.invoiced")}</span>
              <strong>{formatEUR(overview.totalIssued)}</strong>
            </div>
            <div>
              <span>{t("label.paid")}</span>
              <strong>{formatEUR(overview.totalPaid)}</strong>
            </div>
            <div>
              <span>{t("label.monthDebt")}</span>
              <strong>{formatEUR(overview.monthDebtTotal)}</strong>
            </div>
            <div>
              <span>{t("label.olderDebt")}</span>
              <strong>{formatEUR(overview.historicalDebtTotal)}</strong>
            </div>
            <div>
              <span>{t("label.cashCollected")}</span>
              <strong>{formatEUR(overview.paymentsMonthCashTotal)}</strong>
            </div>
            <div>
              <span>{t("label.bankCollected")}</span>
              <strong>{formatEUR(overview.paymentsMonthBankTotal)}</strong>
            </div>
            <div>
              <span>{t("label.creditOnAccount")}</span>
              <strong>{formatEUR(overview.unlinkedCreditTotal)}</strong>
            </div>
            <div>
              <span>{t("field.overdueInvoices")}</span>
              <strong>{overview.overdueInvoicesCount}</strong>
            </div>
          </div>
          <div className="dashboardActions">
            <button
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={onOpenDebtors}
            >
              {t("button.openDebts")}
            </button>
            <button className="secondaryActionButton" onClick={onOpenInvoices}>
              {t("button.openInvoices")}
            </button>
          </div>
        </section>

        <section className="dashboardCard dashboardCard--compact">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">{t("label.schoolSnapshot")}</div>
              <h3>{t("label.monthlyOverview")}</h3>
            </div>
          </div>
          <div className="dashboardMetrics dashboardMetrics--snapshot">
            <div>
              <span>{t("field.activeStudents")}</span>
              <strong>{overview.activeStudents}</strong>
            </div>
            <div>
              <span>{t("field.activeCourses")}</span>
              <strong>{overview.activeCourses}</strong>
            </div>
            <div>
              <span>{t("field.enrollments")}</span>
              <strong>{overview.enrollments}</strong>
            </div>
            <div>
              <span>{t("tabs.debtors")}</span>
              <strong>{overview.debtorsCount}</strong>
            </div>
          </div>
        </section>
      </div>

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
          <div className="activityFeed activityFeed--compact">
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
                  <div className="dashboardPaymentTagRow">
                    <span className="statusPill success">
                      {payment.invoiceId ? t("msg.linkedToInvoice") : t("msg.creditOnAccountTag")}
                    </span>
                  </div>
                </div>
                <div className="activityFeedMeta">
                  <strong>{formatEUR(payment.amount)}</strong>
                  {payment.invoiceId ? (
                    <span>{t("msg.invoiceRef", { id: payment.invoiceId })}</span>
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
  );
}
