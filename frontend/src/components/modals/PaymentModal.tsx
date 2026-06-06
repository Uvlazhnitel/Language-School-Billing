import type { TranslateFn } from "../../lib/i18n";

type PaymentModalProps = {
  studentId: number;
  studentName: string;
  invoiceId: number | null;
  amount: string;
  method: "cash" | "bank";
  note: string;
  onAmountChange: (value: string) => void;
  onMethodChange: (value: "cash" | "bank") => void;
  onNoteChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
  t: TranslateFn;
};

export function PaymentModal({
  studentId,
  studentName,
  invoiceId,
  amount,
  method,
  note,
  onAmountChange,
  onMethodChange,
  onNoteChange,
  onCancel,
  onSubmit,
  t,
}: PaymentModalProps) {
  if (studentId <= 0) return null;

  return (
    <div className="modal" onClick={onCancel}>
      <div className="modalBody" onClick={(e) => e.stopPropagation()}>
        <h3>{t("modal.paymentTitle")}</h3>
        <div className="formRow">
          <label>{t("tabs.students")}</label>
          <input value={studentName} disabled />
        </div>
        {invoiceId && (
          <div className="formRow">
            <label>{t("field.course")}</label>
            <input value={`Счёт #${invoiceId}`} disabled />
          </div>
        )}
        <div className="formRow">
          <label>{t("field.amount")} (EUR):</label>
          <input
            type="number"
            step="0.01"
            value={amount}
            onChange={(e) => onAmountChange(e.target.value)}
            autoFocus
          />
        </div>
        <div className="formRow">
          <label>{t("field.method")}:</label>
          <select value={method} onChange={(e) => onMethodChange(e.target.value as "cash" | "bank")}>
            <option value="cash">{t("payment.cash")}</option>
            <option value="bank">{t("payment.bank")}</option>
          </select>
        </div>
        <div className="formRow">
          <label>{t("field.note")}:</label>
          <input
            type="text"
            value={note}
            onChange={(e) => onNoteChange(e.target.value)}
            placeholder={t("field.note")}
          />
        </div>
        <div className="modalActions">
          <button onClick={onCancel}>{t("button.cancel")}</button>
          <button onClick={onSubmit}>{t("button.recordPayment")}</button>
        </div>
      </div>
    </div>
  );
}
