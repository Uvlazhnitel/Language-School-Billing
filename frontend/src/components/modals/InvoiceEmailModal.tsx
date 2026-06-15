import type { TranslateFn } from "../../lib/i18n";

type InvoiceEmailModalProps = {
  isOpen: boolean;
  to: string;
  subject: string;
  body: string;
  attachmentFilename: string;
  sending: boolean;
  onToChange: (value: string) => void;
  onSubjectChange: (value: string) => void;
  onBodyChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
  t: TranslateFn;
};

export function InvoiceEmailModal({
  isOpen,
  to,
  subject,
  body,
  attachmentFilename,
  sending,
  onToChange,
  onSubjectChange,
  onBodyChange,
  onCancel,
  onSubmit,
  t,
}: InvoiceEmailModalProps) {
  if (!isOpen) return null;

  const canSubmit = to.trim() !== "" && subject.trim() !== "" && body.trim() !== "" && !sending;

  return (
    <div className="modal" onClick={onCancel}>
      <div className="modalBody" onClick={(e) => e.stopPropagation()}>
        <h3>{t("modal.invoiceEmailTitle")}</h3>
        <div className="formRow">
          <label>{t("field.to")}</label>
          <input value={to} onChange={(e) => onToChange(e.target.value)} autoFocus />
        </div>
        <div className="formRow">
          <label>{t("field.subject")}</label>
          <input value={subject} onChange={(e) => onSubjectChange(e.target.value)} />
        </div>
        <div className="formRow">
          <label>{t("field.attachment")}</label>
          <input value={attachmentFilename} disabled />
        </div>
        <div className="formRow formRowTopAligned">
          <label>{t("field.message")}</label>
          <textarea
            className="modalTextarea"
            value={body}
            onChange={(e) => onBodyChange(e.target.value)}
            rows={8}
          />
        </div>
        {to.trim() === "" && <p className="formHint formHintError">{t("msg.invoiceEmailRecipientRequired")}</p>}
        <div className="modalActions">
          <button onClick={onCancel} disabled={sending}>
            {t("button.cancel")}
          </button>
          <button className="btnPrimary" onClick={onSubmit} disabled={!canSubmit}>
            {sending ? t("label.loading") : t("button.sendEmail")}
          </button>
        </div>
      </div>
    </div>
  );
}
