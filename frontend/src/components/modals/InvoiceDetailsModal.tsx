import type { InvoiceDTO } from "../../lib/invoices";
import type { InvoiceSummaryDTO } from "../../lib/payments";
import type { TranslateFn } from "../../lib/i18n";

type InvoiceDetailsModalProps = {
  invoice: InvoiceDTO;
  summary: InvoiceSummaryDTO | null;
  months: string[];
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  formatHoursValue: (value: number) => string;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onIssue: (invoiceId: number) => void | Promise<void>;
  onDownloadPdf: (invoiceId: number) => void | Promise<void>;
  onSendEmail: (invoiceId: number) => void | Promise<void>;
  onAddPayment: () => void;
  onReopenToDraft: (invoiceId: number) => void | Promise<void>;
  onClose: () => void;
  canSendEmail: boolean;
  t: TranslateFn;
};

export function InvoiceDetailsModal({
  invoice,
  summary,
  months,
  invoiceStatusLabel,
  formatEUR,
  formatHoursValue,
  onOpenStudent,
  onIssue,
  onDownloadPdf,
  onSendEmail,
  onAddPayment,
  onReopenToDraft,
  onClose,
  canSendEmail,
  t,
}: InvoiceDetailsModalProps) {
  return (
    <div className="modal" onClick={onClose}>
      <div className="modalBody modalBodyWide" onClick={(e) => e.stopPropagation()}>
        <div style={{ marginBottom: "1rem" }}>
          <h3>
            {t("modal.invoiceTitle")} {invoice.number ? `#${invoice.number}` : ""} —{" "}
            <button className="linkButton" onClick={() => void onOpenStudent(invoice.studentId)}>
              {invoice.studentName}
            </button>{" "}
            — {months[invoice.month - 1]} {invoice.year}
          </h3>
        </div>

        {summary && invoice.status !== "draft" && (
          <div className="invSummary">
            <div className="invSummaryRow">
              <span>{t("field.recipient")}:</span>
              <span>{invoice.recipientName || invoice.studentName}</span>
            </div>
            {invoice.studentPersonalCode && (
              <div className="invSummaryRow">
                <span>
                  {invoice.isMinor
                    ? `${t("field.personalCode")} child:`
                    : `${t("field.personalCode")}:`}
                </span>
                <span>{invoice.studentPersonalCode}</span>
              </div>
            )}
            {invoice.isMinor && (
              <div className="invSummaryRow">
                <span>{t("field.forChild")}:</span>
                <span>{invoice.childName}</span>
              </div>
            )}
            <div className="invSummaryRow">
              <span>{t("field.amount")}:</span>
              <span className="money">{formatEUR(summary.total)}</span>
            </div>
            <div className="invSummaryRow">
              <span>{t("label.paid")}:</span>
              <span className="money good">{formatEUR(summary.paid)}</span>
            </div>
            <div className="invSummaryRow">
              <span>{t("field.remaining")}:</span>
              <span className={`money ${summary.remaining > 0 ? "bad" : "good"}`}>
                {formatEUR(summary.remaining)}
              </span>
            </div>
            <div className="invSummaryRow">
              <span>{t("field.status")}:</span>
              <span className="money">{invoiceStatusLabel(summary.status)}</span>
            </div>
          </div>
        )}

        <div style={{ overflowX: "auto" }}>
          <table>
            <thead>
              <tr>
                <th>{t("field.description")}</th>
                <th style={{ textAlign: "right" }}>{t("field.quantity")}</th>
                <th style={{ textAlign: "right" }}>{t("field.lessonPrice")} (EUR)</th>
                <th style={{ textAlign: "right" }}>{t("field.amount")} (EUR)</th>
              </tr>
            </thead>
            <tbody>
              {invoice.lines.map((line, index) => (
                <tr key={index}>
                  <td>{line.description}</td>
                  <td style={{ textAlign: "right" }}>{formatHoursValue(line.qty)}</td>
                  <td style={{ textAlign: "right" }}>{formatEUR(line.unitPrice)}</td>
                  <td style={{ textAlign: "right" }}>{formatEUR(line.amount)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td colSpan={3} style={{ textAlign: "right" }}>
                  {t("field.totalEur")}:
                </td>
                <td style={{ textAlign: "right" }}>{formatEUR(invoice.total)}</td>
              </tr>
            </tfoot>
          </table>
        </div>

        <div className="modalActions">
          {invoice.status === "draft" && (
            <button onClick={() => void onIssue(invoice.id)}>{t("button.issue")}</button>
          )}
          {invoice.status !== "draft" && (
            <button onClick={() => void onDownloadPdf(invoice.id)}>{t("button.downloadPdf")}</button>
          )}
          {invoice.status !== "draft" && canSendEmail && (
            <button onClick={() => void onSendEmail(invoice.id)}>{t("button.sendEmail")}</button>
          )}
          {invoice.status !== "draft" && <button onClick={onAddPayment}>{t("button.recordPayment")}</button>}
          {invoice.status === "issued" && (
            <button onClick={() => void onReopenToDraft(invoice.id)}>{t("button.reopenDraft")}</button>
          )}
          <button onClick={onClose}>{t("button.close")}</button>
        </div>
      </div>
    </div>
  );
}
