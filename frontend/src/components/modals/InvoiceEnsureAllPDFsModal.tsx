import type { EnsureAllPDFsResult } from "../../lib/api";
import type { TranslateFn } from "../../lib/i18n";

type InvoiceEnsureAllPDFsModalProps = {
  result: EnsureAllPDFsResult;
  months: string[];
  invoiceStatusLabel: (status: string) => string;
  onClose: () => void;
  t: TranslateFn;
};

export function InvoiceEnsureAllPDFsModal({
  result,
  months,
  invoiceStatusLabel,
  onClose,
  t,
}: InvoiceEnsureAllPDFsModalProps) {
  return (
    <div className="modal" onClick={onClose}>
      <div className="modalBody modalBodyWide" onClick={(event) => event.stopPropagation()}>
        <h3>{t("modal.ensureAllPdfsTitle")}</h3>
        <div className="invSummary" style={{ marginBottom: "1rem" }}>
          <div className="invSummaryRow">
            <span>{t("field.period")}:</span>
            <span>
              {months[result.month - 1]} {result.year}
            </span>
          </div>
          <div className="invSummaryRow">
            <span>{t("field.summary")}:</span>
            <span>
              {t("msg.ensureAllPdfsSummary", {
                processed: result.processed,
                generated: result.generatedCount,
                ready: result.alreadyReadyCount,
                failed: result.failedCount,
              })}
            </span>
          </div>
        </div>

        {result.items.length === 0 ? (
          <div>{t("msg.ensureAllPdfsEmpty")}</div>
        ) : (
          <div style={{ overflowX: "auto" }}>
            <table>
              <thead>
                <tr>
                  <th>{t("field.number")}</th>
                  <th>{t("field.student")}</th>
                  <th>{t("field.status")}</th>
                  <th>{t("field.result")}</th>
                  <th>{t("field.note")}</th>
                </tr>
              </thead>
              <tbody>
                {result.items.map((item) => (
                  <tr key={item.invoiceId}>
                    <td>{item.number}</td>
                    <td>{item.studentName}</td>
                    <td>{invoiceStatusLabel(item.status)}</td>
                    <td>{t(`result.${item.result}`)}</td>
                    <td>{item.message ?? ""}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="modalActions">
          <button onClick={onClose}>{t("button.close")}</button>
        </div>
      </div>
    </div>
  );
}
