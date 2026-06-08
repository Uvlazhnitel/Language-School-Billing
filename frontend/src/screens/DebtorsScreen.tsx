import type { DebtorDTO } from "../lib/payments";
import type { DebtorActionQueueItem } from "../lib/studentActivity";
import type { TranslateFn } from "../lib/i18n";

type DebtorsScreenProps = {
  loading: boolean;
  debtors: DebtorDTO[];
  actionQueue: DebtorActionQueueItem[];
  formatEUR: (value: number) => string;
  onRefresh: () => void;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onOpenStudentWorkspace: (studentId: number) => void | Promise<void>;
  onOpenPaymentForStudent: (studentId: number) => void;
  onOpenPaymentForDebtor: (debtor: DebtorDTO) => void;
  onOpenDebtDetails: (debtor: DebtorDTO) => void;
  onCopyDebtForStudentRu: (studentId: number) => void | Promise<void>;
  onCopyDebtForStudentLv: (studentId: number) => void | Promise<void>;
  onCopyDebtForDebtorRu: (debtor: DebtorDTO) => void | Promise<void>;
  onCopyDebtForDebtorLv: (debtor: DebtorDTO) => void | Promise<void>;
  t: TranslateFn;
};

export function DebtorsScreen({
  loading,
  debtors,
  actionQueue,
  formatEUR,
  onRefresh,
  onOpenStudent,
  onOpenStudentWorkspace,
  onOpenPaymentForStudent,
  onOpenPaymentForDebtor,
  onOpenDebtDetails,
  onCopyDebtForStudentRu,
  onCopyDebtForStudentLv,
  onCopyDebtForDebtorRu,
  onCopyDebtForDebtorLv,
  t,
}: DebtorsScreenProps) {
  return (
    <>
      <div className="sectionBanner">
        <div>
          <div className="dashboardCardEyebrow">{t("msg.collection")}</div>
          <strong>{t("label.needsAction")}</strong>
        </div>
        <div className="sectionBannerActions">
          <button className="workspaceActionButton" onClick={onRefresh}>
            {t("button.refresh")}
          </button>
        </div>
      </div>

      {actionQueue.length > 0 && (
        <div className="detailCard detailCard--wide actionQueuePanel">
          <div className="detailCardHeader">
            <h3>{t("label.needsAction")}</h3>
            <span className="statusPill warning">{t("msg.queueCount", { count: actionQueue.length })}</span>
          </div>
          <div className="actionQueueList">
            {actionQueue.map((item) => (
              <div key={item.studentId} className="actionQueueItem">
                <div>
                  <strong>{item.studentName}</strong>
                  <span>{item.subtitle}</span>
                </div>
                <div className="actionQueueMeta">
                  <strong>{formatEUR(item.debt)}</strong>
                  <div className="actionQueueActions">
                    <button
                      className="workspaceActionButton workspaceActionButtonPrimary"
                      onClick={() => onOpenPaymentForStudent(item.studentId)}
                    >
                      {t("button.takePayment")}
                    </button>
                    <button
                      className="secondaryActionButton"
                      onClick={() => void onOpenStudentWorkspace(item.studentId)}
                    >
                      {t("button.card")}
                    </button>
                    <div className="actionQueueSecondaryGroup">
                      <button
                        className="secondaryActionButton secondaryActionButton--mini"
                        onClick={() => void onCopyDebtForStudentRu(item.studentId)}
                      >
                        RU
                      </button>
                      <button
                        className="secondaryActionButton secondaryActionButton--mini"
                        onClick={() => void onCopyDebtForStudentLv(item.studentId)}
                      >
                        LV
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : debtors.length === 0 ? (
        <div className="empty">{t("msg.noDebtors")}</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>{t("field.student")}</th>
              <th style={{ textAlign: "right" }}>{t("field.debtEur")}</th>
              <th style={{ textAlign: "right" }}>{t("field.totalEur")}</th>
              <th style={{ textAlign: "right" }}>{t("field.paidEur")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {debtors.map((debtor) => (
              <tr key={debtor.studentId}>
                <td>
                  <button className="linkButton" onClick={() => void onOpenStudent(debtor.studentId)}>
                    {debtor.studentName}
                  </button>
                </td>
                <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                  {formatEUR(debtor.debt)}
                </td>
                <td style={{ textAlign: "right" }}>{formatEUR(debtor.totalInvoiced)}</td>
                <td style={{ textAlign: "right" }}>{formatEUR(debtor.totalPaid)}</td>
                <td>
                  <div className="invoiceRowActions">
                    <button
                      className="workspaceActionButton workspaceActionButtonPrimary invoicePrimaryAction"
                      onClick={() => onOpenPaymentForDebtor(debtor)}
                    >
                      {t("button.takePayment")}
                    </button>
                    <button
                      className="workspaceActionButton"
                      onClick={() => onOpenDebtDetails(debtor)}
                    >
                      {t("modal.debtBreakdown")}
                    </button>
                    <button
                      className="secondaryActionButton"
                      onClick={() => void onCopyDebtForDebtorRu(debtor)}
                    >
                      {t("button.copyRu")}
                    </button>
                    <button
                      className="secondaryActionButton"
                      onClick={() => void onCopyDebtForDebtorLv(debtor)}
                    >
                      {t("button.copyLv")}
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <td style={{ fontWeight: "bold" }}>{t("field.debtEur")}:</td>
              <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                {formatEUR(debtors.reduce((sum, debtor) => sum + debtor.debt, 0))}
              </td>
              <td colSpan={3}></td>
            </tr>
          </tfoot>
        </table>
      )}
    </>
  );
}
