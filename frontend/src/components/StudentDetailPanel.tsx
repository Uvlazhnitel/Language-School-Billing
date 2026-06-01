import { BalanceDTO, DebtInvoiceDTO, PaymentDTO } from "../lib/payments";
import { EnrollmentDTO } from "../lib/enrollments";
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
  payerRoleLabel: (relation: string) => string;
  billingModeLabel: (mode: string) => string;
  paymentMethodLabel: (method: string) => string;
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  months: string[];
  deletingPaymentId: number | null;
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
  payerRoleLabel,
  billingModeLabel,
  paymentMethodLabel,
  invoiceStatusLabel,
  formatEUR,
  months,
  deletingPaymentId,
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
    return <div className="empty">Выберите ученика слева, чтобы открыть рабочую карточку.</div>;
  }

  if (loading) {
    return <div className="empty">Загрузка карточки ученика…</div>;
  }

  const latestPayment = payments[0]?.paidAt?.slice(0, 10) ?? "ещё не было";
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
          <div className="dashboardCardEyebrow">Карточка ученика</div>
          <h2>{student.fullName}</h2>
          <p>{student.isMinor ? "Ребёнок / несовершеннолетний" : "Взрослый ученик"}</p>
          {nextAction && (
            <div className="crmActionStrip">
              <div className="crmActionCopy">
                <span className="crmActionLabel">Следующее действие</span>
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
                {nextAction.secondaryLabel === "Напомнить о долге" && debts.length > 0 && (
                  <button className="workspaceActionButton" onClick={onCopyDebtRu}>
                    {nextAction.secondaryLabel}
                  </button>
                )}
                {nextAction.secondaryLabel === "Проверить черновик" && (
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
            Записать оплату
          </button>
          <button className="workspaceActionButton" onClick={onEditStudent}>
            Редактировать
          </button>
          <button className="workspaceActionButton" onClick={onManageEnrollments}>
            Зачисления
          </button>
        </div>
      </div>

      <div className="studentSummaryGrid">
        <div className="summaryMetricCard">
          <span>Выставлено</span>
          <strong>{balance ? formatEUR(balance.totalInvoiced) : "—"}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>Оплачено</span>
          <strong>{balance ? formatEUR(balance.totalPaid) : "—"}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>Долг</span>
          <strong className={balance && balance.debt > 0 ? "metricDanger" : "metricSuccess"}>
            {balance ? (balance.debt > 0 ? formatEUR(balance.debt) : "Нет долга") : "—"}
          </strong>
        </div>
        <div className="summaryMetricCard">
          <span>Последняя оплата</span>
          <strong>{latestPayment}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>Активных зачислений</span>
          <strong>{activeEnrollmentsCount}</strong>
        </div>
        <div className="summaryMetricCard">
          <span>Счета месяца</span>
          <strong>{monthInvoices.length}</strong>
        </div>
      </div>

      <div className="studentDetailGrid">
        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>CRM timeline</h3>
          </div>
          {activity.length === 0 ? (
            <div className="empty">История пока пустая.</div>
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
            <h3>Контакты и статус</h3>
          </div>
          <div className="detailKeyValue">
            <span>Телефон</span>
            <strong>{student.phone || "—"}</strong>
          </div>
          <div className="detailKeyValue">
            <span>E-mail</span>
            <strong>{student.email || "—"}</strong>
          </div>
          <div className="detailKeyValue">
            <span>Персональный код</span>
            <strong>{student.personalCode || "—"}</strong>
          </div>
          <div className="detailKeyValue">
            <span>Статус</span>
            <strong>{student.isActive ? "Активен" : "Неактивен"}</strong>
          </div>
          {student.note && (
            <div className="detailNote">
              <span>Заметка</span>
              <p>{student.note}</p>
            </div>
          )}
        </section>

        {student.isMinor && (
          <section className="detailCard">
            <div className="detailCardHeader">
              <h3>Плательщик</h3>
            </div>
            <div className="detailKeyValue">
              <span>Имя</span>
              <strong>{student.payerName || "—"}</strong>
            </div>
            <div className="detailKeyValue">
              <span>Кем приходится</span>
              <strong>{student.payerRole ? payerRoleLabel(student.payerRole) : "—"}</strong>
            </div>
            <div className="detailKeyValue">
              <span>Контакты</span>
              <strong>{[student.phone, student.email].filter(Boolean).join(" · ") || "—"}</strong>
            </div>
          </section>
        )}

        <section className="detailCard detailCard--wide">
          <div className="detailCardHeader">
            <h3>Курсы и зачисления</h3>
          </div>
          {enrollments.length === 0 ? (
            <div className="empty">Зачислений нет.</div>
          ) : (
            <div className="tableWrap">
              <table>
                <thead>
                  <tr>
                    <th>Курс</th>
                    <th>Учитель</th>
                    <th>Оплата</th>
                    <th style={{ textAlign: "right" }}>Скидка</th>
                    <th>Заметка</th>
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
            <h3>Открытые долги</h3>
            {debts.length > 0 && (
              <div className="inlineActions">
                <button className="secondaryActionButton" onClick={onCopyDebtRu}>
                  Напомнить RU
                </button>
                <button className="secondaryActionButton" onClick={onCopyDebtLv}>
                  Напомнить LV
                </button>
              </div>
            )}
          </div>
          {debts.length === 0 ? (
            <div className="empty">Всё оплачено, открытых долгов нет.</div>
          ) : (
            <div className="tableWrap">
              <table>
                <thead>
                  <tr>
                    <th>Месяц</th>
                    <th>Счёт</th>
                    <th style={{ textAlign: "right" }}>Сумма</th>
                    <th style={{ textAlign: "right" }}>Оплачено</th>
                    <th style={{ textAlign: "right" }}>Осталось</th>
                    <th>Статус</th>
                  </tr>
                </thead>
                <tbody>
                  {debts.map((debt) => (
                    <tr key={debt.invoiceId}>
                      <td>
                        {months[debt.month - 1]} {debt.year}
                      </td>
                      <td>{debt.number ?? "Без номера"}</td>
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
            <h3>Последние оплаты</h3>
          </div>
          {payments.length === 0 ? (
            <div className="empty">Оплат пока нет.</div>
          ) : (
            <div className="tableWrap">
              <table>
                <thead>
                  <tr>
                    <th>Дата</th>
                    <th style={{ textAlign: "right" }}>Сумма</th>
                    <th>Способ</th>
                    <th>Заметка</th>
                    <th>Действия</th>
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
                        <button
                          onClick={() => onDeletePayment(payment)}
                          disabled={deletingPaymentId === payment.id}
                        >
                          {deletingPaymentId === payment.id ? "Удаление..." : "Удалить"}
                        </button>
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
