import { BalanceDTO, DebtInvoiceDTO, PaymentDTO } from "../lib/payments";
import { EnrollmentDTO } from "../lib/enrollments";
import { TranslateFn } from "../lib/i18n";
import { StudentDTO } from "../lib/students";
import { InvoiceListItemView } from "../lib/invoices";
import { StudentActivityItem, StudentNextAction } from "../lib/studentActivity";

type StudentDetailPanelProps = {
  student: StudentDTO | null;
  loading: boolean;
  enrollments: EnrollmentDTO[];
  balance: BalanceDTO | null;
  debts: DebtInvoiceDTO[];
  payments: PaymentDTO[];
  monthInvoices: InvoiceListItemView[];
  nextAction: StudentNextAction | null;
  activity: StudentActivityItem[];
  t: TranslateFn;
  payerRoleLabel: (relation: string) => string;
  billingModeLabel: (mode: string) => string;
  paymentMethodLabel: (method: string) => string;
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  months: string[];
  deletingPaymentId: number | null;
  canDeletePayment: boolean;
  onEditStudent: () => void;
  onAddPayment: () => void;
  onCopyDebtRu: () => void;
  onCopyDebtLv: () => void;
  onDeletePayment: (payment: PaymentDTO) => void;
  onManageEnrollments: () => void;
  onOpenInvoices: () => void;
  footer?: React.ReactNode;
};

export function StudentDetailPanel({
  student,
  loading,
  enrollments,
  balance,
  debts,
  payments,
  monthInvoices,
  nextAction,
  activity,
  t,
  payerRoleLabel,
  billingModeLabel,
  paymentMethodLabel,
  invoiceStatusLabel,
  formatEUR,
  months,
  deletingPaymentId,
  canDeletePayment,
  onEditStudent,
  onAddPayment,
  onCopyDebtRu,
  onCopyDebtLv,
  onDeletePayment,
  onManageEnrollments,
  onOpenInvoices,
  footer,
}: StudentDetailPanelProps) {
  if (!student) {
    return <div className="empty">{t("msg.noStudentSelected")}</div>;
  }

  if (loading) {
    return <div className="empty">{t("msg.studentCardLoading")}</div>;
  }

  const latestPayment = payments[0]?.paidAt?.slice(0, 10) ?? t("msg.noPayments");
  const activeEnrollmentsCount = enrollments.length;

  const handlePrimaryAction = () => {
    switch (nextAction?.action) {
      case "reminder":
        onCopyDebtRu();
        break;
      case "enrollment":
        onManageEnrollments();
        break;
      case "invoice":
        onOpenInvoices();
        break;
      case "payment":
      default:
        onAddPayment();
        break;
    }
  };

  return (
    <div className="studentDetailPanel">
      <div className="studentDetailHero">
        <div>
          <div className="dashboardCardEyebrow">{t("student.card")}</div>
          <h2>{student.fullName}</h2>
          <p>{student.isMinor ? t("student.minor") : t("student.adult")}</p>
          {nextAction && (
            <div className="crmActionStrip">
              <div className="crmActionCopy">
                <span className="crmActionLabel">{t("label.nextAction")}</span>
                <strong>{nextAction.reason}</strong>
                <p>{nextAction.outstandingIssue}</p>
              </div>
              <div className="crmActionButtons">
                <button
                  className="workspaceActionButton workspaceActionButtonPrimary"
                  onClick={handlePrimaryAction}
                >
                  {nextAction.label}
                </button>
                {nextAction.secondaryLabel === t("action.debtReminder") && debts.length > 0 && (
                  <button className="workspaceActionButton" onClick={onCopyDebtRu}>
                    {nextAction.secondaryLabel}
                  </button>
                )}
                {nextAction.secondaryLabel === t("action.checkDraft") && (
                  <button className="workspaceActionButton" onClick={onOpenInvoices}>
                    {nextAction.secondaryLabel}
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
        <div className="studentDetailActions">
          <button
            className="workspaceActionButton workspaceActionButtonPrimary"
            onClick={onAddPayment}
          >
            {t("button.recordPayment")}
          </button>
          <button className="workspaceActionButton" onClick={onEditStudent}>
            {t("button.edit")}
          </button>
          <button className="workspaceActionButton" onClick={onManageEnrollments}>
            {t("button.manageEnrollments")}
          </button>
        </div>
      </div>

      <div className="studentSummaryGrid">
        <div className="summaryMetricCard">
          <span>{t("label.invoiced")}</span>
          <strong>{balance ? formatEUR(balance.totalInvoiced) : "—"}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>{t("label.paid")}</span>
          <strong>{balance ? formatEUR(balance.totalPaid) : "—"}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>{t("label.debt")}</span>
          <strong className={balance && balance.debt > 0 ? "metricDanger" : "metricSuccess"}>
            {balance ? (balance.debt > 0 ? formatEUR(balance.debt) : t("msg.noOpenDebts")) : "—"}
          </strong>
        </div>
        <div className="summaryMetricCard">
          <span>{t("label.lastPayment")}</span>
          <strong>{latestPayment}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>{t("label.activeEnrollments")}</span>
          <strong>{activeEnrollmentsCount}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>{t("label.monthInvoices")}</span>
          <strong>{monthInvoices.length}</strong>
        </div>
      </div>

      <div className="studentDetailGrid">
        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>{t("label.crmTimeline")}</h3>
          </div>
          {activity.length === 0 ? (
            <div className="empty">{t("msg.studentHistoryEmpty")}</div>
          ) : (
            <div className="crmTimeline">
              {activity.map((item) => (
                <div key={item.id} className={`crmTimelineItem crmTimelineItem--${item.kind}`}>
                  <div className="crmTimelineDate">{item.date.slice(0, 10)}</div>
                  <div className="crmTimelineBody">
                    <div className="crmTimelineTitleRow">
                      <strong>{item.title}</strong>
                      {typeof item.amount === "number" && (
                        <span
                          className={
                            item.kind === "debt"
                              ? "crmTimelineAmount metricDanger"
                              : "crmTimelineAmount"
                          }
                        >
                          {formatEUR(item.amount)}
                        </span>
                      )}
                    </div>
                    <p>{item.subtitle}</p>
                    {item.status && (
                      <span className={`statusPill statusPill--${item.status}`}>
                        {invoiceStatusLabel(item.status)}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="detailCard">
          <div className="detailCardHeader">
            <h3>{t("label.contactsAndStatus")}</h3>
          </div>
          <div className="detailKeyValue">
            <span>{t("field.phone")}</span>
            <strong>{student.phone || "—"}</strong>
          </div>
          <div className="detailKeyValue">
            <span>{t("field.email")}</span>
            <strong>{student.email || "—"}</strong>
          </div>
          <div className="detailKeyValue">
            <span>{t("field.personalCode")}</span>
            <strong>{student.personalCode || "—"}</strong>
          </div>
          <div className="detailKeyValue">
            <span>{t("field.status")}</span>
            <strong>{student.isActive ? t("status.active") : t("status.inactive")}</strong>
          </div>
          {student.note && (
            <div className="detailNote">
              <span>{t("field.note")}</span>
              <p>{student.note}</p>
            </div>
          )}
        </section>

        {student.isMinor && (
          <section className="detailCard">
            <div className="detailCardHeader">
              <h3>{t("label.payer")}</h3>
            </div>
            <div className="detailKeyValue">
              <span>{t("field.name")}</span>
              <strong>{student.payerName || "—"}</strong>
            </div>
            <div className="detailKeyValue">
              <span>{t("field.payerRole")}</span>
              <strong>{student.payerRole ? payerRoleLabel(student.payerRole) : "—"}</strong>
            </div>
            <div className="detailKeyValue">
              <span>{t("field.contacts")}</span>
              <strong>{[student.phone, student.email].filter(Boolean).join(" · ") || "—"}</strong>
            </div>
          </section>
        )}

        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>{t("label.coursesAndEnrollments")}</h3>
          </div>
          {enrollments.length === 0 ? (
            <div className="empty">{t("msg.noEnrollments")}</div>
          ) : (
            <div className="tableWrap">
              <table>
                <thead>
                  <tr>
                    <th>{t("field.course")}</th>
                    <th>{t("field.teacher")}</th>
                    <th>{t("field.billing")}</th>
                    <th style={{ textAlign: "right" }}>{t("field.discount")}</th>
                    <th>{t("field.note")}</th>
                  </tr>
                </thead>
                <tbody>
                  {enrollments.map((enrollment) => (
                    <tr key={enrollment.id}>
                      <td>{enrollment.courseName}</td>
                      <td>{enrollment.teacherName || "—"}</td>
                      <td>{billingModeLabel(enrollment.billingMode)}</td>
                      <td style={{ textAlign: "right" }}>{enrollment.discountPct.toFixed(1)}%</td>
                      <td>{enrollment.note || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>{t("label.openDebts")}</h3>
            {debts.length > 0 && (
              <div className="inlineActions">
                <button className="secondaryActionButton" onClick={onCopyDebtRu}>
                  {t("button.copyRu")}
                </button>
                <button className="secondaryActionButton" onClick={onCopyDebtLv}>
                  {t("button.copyLv")}
                </button>
              </div>
            )}
          </div>
          {debts.length === 0 ? (
            <div className="empty">{t("msg.noOpenDebts")}</div>
          ) : (
            <div className="tableWrap">
              <table>
                <thead>
                  <tr>
                    <th>{t("field.month")}</th>
                    <th>{t("field.number")}</th>
                    <th style={{ textAlign: "right" }}>{t("field.amount")}</th>
                    <th style={{ textAlign: "right" }}>{t("label.paid")}</th>
                    <th style={{ textAlign: "right" }}>{t("field.remaining")}</th>
                    <th>{t("field.status")}</th>
                  </tr>
                </thead>
                <tbody>
                  {debts.map((debt) => (
                    <tr key={debt.invoiceId}>
                      <td>
                        {months[debt.month - 1]} {debt.year}
                      </td>
                      <td>{debt.number ?? "—"}</td>
                      <td style={{ textAlign: "right" }}>{formatEUR(debt.total)}</td>
                      <td style={{ textAlign: "right" }}>{formatEUR(debt.paid)}</td>
                      <td style={{ textAlign: "right" }}>
                        <strong className="metricDanger">{formatEUR(debt.remaining)}</strong>
                      </td>
                      <td>{invoiceStatusLabel(debt.status)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>{t("label.recentPayments")}</h3>
          </div>
          {payments.length === 0 ? (
            <div className="empty">{t("msg.noPayments")}</div>
          ) : (
            <div className="tableWrap">
              <table>
                <thead>
                  <tr>
                    <th>{t("field.date")}</th>
                    <th style={{ textAlign: "right" }}>{t("field.amount")}</th>
                    <th>{t("field.method")}</th>
                    <th>{t("field.note")}</th>
                    <th>{t("field.actions")}</th>
                  </tr>
                </thead>
                <tbody>
                  {payments.slice(0, 10).map((payment) => (
                    <tr key={payment.id}>
                      <td>{payment.paidAt.slice(0, 10)}</td>
                      <td style={{ textAlign: "right" }}>{formatEUR(payment.amount)}</td>
                      <td>{paymentMethodLabel(payment.method)}</td>
                      <td>{payment.note || "—"}</td>
                      <td>
                        {canDeletePayment && (
                          <button
                            onClick={() => onDeletePayment(payment)}
                            disabled={deletingPaymentId === payment.id}
                          >
                            {deletingPaymentId === payment.id ? `${t("button.delete")}...` : t("button.delete")}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      {footer}
    </div>
  );
}
