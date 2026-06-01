import { MonthOverviewDTO, RecentPaymentDTO } from "../lib/dashboard";
import { DebtorActionQueueItem } from "../lib/studentActivity";

type DashboardOverviewProps = {
  overview: MonthOverviewDTO | null;
  loading: boolean;
  monthLabel: string;
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
    return <div className="empty">Загружаем обзор месяца…</div>;
  }

  if (!overview) {
    return <div className="empty">Не удалось загрузить обзор месяца.</div>;
  }

  const attendanceReady = overview.perLessonEnrollments > 0 && overview.attendanceMissing === 0;
  const topDebtor = actionQueue[0] ?? null;
  const noRecentPaymentCount = actionQueue.filter((item) => !item.hasRecentPayment).length;

  return (
    <div className="dashboard">
      <div className="dashboardHero">
        <div>
          <div className="dashboardHeroEyebrow">Месячный обзор</div>
          <h2>{monthLabel}</h2>
          <p>
            Сначала видим приоритеты месяца, потом идём в нужный рабочий сценарий без лишних
            переходов.
          </p>
        </div>
        <div className="dashboardHeroStats">
          <div className="dashboardHeroStat">
            <span>Активных учеников</span>
            <strong>{overview.activeStudents}</strong>
          </div>
          <div className="dashboardHeroStat">
            <span>Активных курсов</span>
            <strong>{overview.activeCourses}</strong>
          </div>
          <div className="dashboardHeroStat">
            <span>Зачислений</span>
            <strong>{overview.enrollments}</strong>
          </div>
        </div>
      </div>

      <div className="dashboardGrid">
        <section className="dashboardCard">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">Счета</div>
              <h3>Нужно выставить счета</h3>
            </div>
            <span className={`statusPill ${overview.draftInvoices > 0 ? "warning" : "success"}`}>
              {overview.draftInvoices > 0 ? `${overview.draftInvoices} черновиков` : "Готово"}
            </span>
          </div>
          <p className="dashboardCardLead">
            Выставлено на {formatEUR(overview.totalIssued)}, оплачено{" "}
            {formatEUR(overview.totalPaid)}.
          </p>
          <div className="dashboardMetrics">
            <div>
              <span>Выставлены</span>
              <strong>{overview.issuedInvoices}</strong>
            </div>
            <div>
              <span>Оплачены</span>
              <strong>{overview.paidInvoices}</strong>
            </div>
          </div>
          <div className="dashboardActions">
            <button
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={onOpenInvoices}
            >
              Открыть счета
            </button>
          </div>
        </section>

        <section className="dashboardCard">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">Контроль месяца</div>
              <h3>Не заполнена посещаемость</h3>
            </div>
            <span className={`statusPill ${attendanceReady ? "success" : "warning"}`}>
              {attendanceReady ? "Можно выставлять" : `${overview.attendanceMissing} осталось`}
            </span>
          </div>
          <p className="dashboardCardLead">
            Заполнено {overview.attendanceFilled} из {overview.perLessonEnrollments} строк по оплате
            за занятия.
          </p>
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
              К посещаемости
            </button>
          </div>
        </section>

        <section className="dashboardCard dashboardCard--compact">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">Долги</div>
              <h3>Открытый долг</h3>
            </div>
            <span className={`statusPill ${overview.totalDebt > 0 ? "danger" : "success"}`}>
              {overview.debtorsCount} учеников
            </span>
          </div>
          <p className="dashboardCardLead">{formatEUR(overview.totalDebt)}</p>
          <div className="dashboardMetrics">
            <div>
              <span>Требуют внимания</span>
              <strong>{overview.debtorsCount}</strong>
            </div>
            <div>
              <span>Без недавней оплаты</span>
              <strong>{noRecentPaymentCount}</strong>
            </div>
            <div>
              <span>Крупнейший долг</span>
              <strong>{topDebtor ? formatEUR(topDebtor.debt) : "—"}</strong>
            </div>
            <div>
              <span>Лидер риска</span>
              <strong>{topDebtor ? topDebtor.studentName : "—"}</strong>
            </div>
          </div>
          <div className="dashboardActions">
            <button
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={onOpenDebtors}
            >
              Открыть долги
            </button>
          </div>
        </section>

        <section className="dashboardCard dashboardCard--queue">
          <div className="dashboardCardHeader">
            <div>
              <div className="dashboardCardEyebrow">Action queue</div>
              <h3>Требуют действия сейчас</h3>
            </div>
            <span className={`statusPill ${actionQueue.length > 0 ? "warning" : "success"}`}>
              {actionQueue.length > 0 ? `${actionQueue.length} в приоритете` : "Очередь пуста"}
            </span>
          </div>
          {actionQueue.length === 0 ? (
            <div className="empty">Сейчас нет срочных follow-up задач по долгам.</div>
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
                        Оплата
                      </button>
                      <button
                        className="secondaryActionButton"
                        onClick={() => onOpenStudent(item.studentId)}
                      >
                        Карточка
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
              <div className="dashboardCardEyebrow">Активность</div>
              <h3>Последние оплаты</h3>
            </div>
            <button className="secondaryActionButton" onClick={onOpenStudents}>
              База учеников
            </button>
          </div>

          {recentPayments.length === 0 ? (
            <div className="empty">Оплат пока нет.</div>
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
                      <span>Без счёта</span>
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
