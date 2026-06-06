import type { DebtorDTO, DebtInvoiceDTO } from "../../lib/payments";
import type { TranslateFn } from "../../lib/i18n";

type DebtDetailsModalProps = {
  debtor: DebtorDTO;
  details: DebtInvoiceDTO[];
  loading: boolean;
  months: string[];
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onRecordPayment: () => void;
  onCopyRu: () => void | Promise<void>;
  onCopyLv: () => void | Promise<void>;
  onClose: () => void;
  t: TranslateFn;
};

export function DebtDetailsModal({
  debtor,
  details,
  loading,
  months,
  invoiceStatusLabel,
  formatEUR,
  onOpenStudent,
  onRecordPayment,
  onCopyRu,
  onCopyLv,
  onClose,
  t,
}: DebtDetailsModalProps) {
  return (
    <div className="modal" onClick={onClose}>
      <div className="modalBody" onClick={(e) => e.stopPropagation()}>
        <h3>{t("modal.debtBreakdown")}</h3>

        <div className="invSummary">
          <div className="invSummaryRow">
            <span>{t("field.student")}</span>
            <button className="linkButton" onClick={() => void onOpenStudent(debtor.studentId)}>
              {debtor.studentName}
            </button>
          </div>
          <div className="invSummaryRow">
            <span>{t("field.debtEur")}</span>
            <strong className="money bad">{formatEUR(debtor.debt)}</strong>
          </div>
        </div>

        {loading ? (
          <div>{t("label.loading")}</div>
        ) : details.length === 0 ? (
          <div className="empty">{t("msg.noOpenDebts")}</div>
        ) : (
          <div style={{ overflowX: "auto" }}>
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
                {details.map((item) => (
                  <tr key={item.invoiceId}>
                    <td>
                      {months[item.month - 1]} {item.year}
                    </td>
                    <td>{item.number ?? t("msg.noInvoiceNumber")}</td>
                    <td style={{ textAlign: "right" }}>{formatEUR(item.total)}</td>
                    <td style={{ textAlign: "right" }}>{formatEUR(item.paid)}</td>
                    <td style={{ textAlign: "right" }}>
                      <strong className="money bad">{formatEUR(item.remaining)}</strong>
                    </td>
                    <td>{invoiceStatusLabel(item.status)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="modalActions">
          {!loading && details.length > 0 && (
            <>
              <button onClick={onRecordPayment}>{t("button.recordPayment")}</button>
              <button onClick={() => void onCopyRu()}>{t("button.copyRu")}</button>
              <button onClick={() => void onCopyLv()}>{t("button.copyLv")}</button>
            </>
          )}
          <button onClick={onClose}>{t("button.close")}</button>
        </div>
      </div>
    </div>
  );
}
