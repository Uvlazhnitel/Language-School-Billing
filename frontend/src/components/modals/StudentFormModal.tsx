import type { TranslateFn } from "../../lib/i18n";
import type { StudentDuplicateCheckResult } from "../../lib/students";

type StudentFormModalProps = {
  editing: boolean;
  name: string;
  personalCode: string;
  phone: string;
  email: string;
  note: string;
  isMinor: boolean;
  payerName: string;
  payerRole: string;
  payerRoleOptions: readonly string[];
  payerRoleLabel: (role: string) => string;
  onNameChange: (value: string) => void;
  onPersonalCodeChange: (value: string) => void;
  onPhoneChange: (value: string) => void;
  onEmailChange: (value: string) => void;
  onNoteChange: (value: string) => void;
  onIsMinorChange: (value: boolean) => void;
  onPayerNameChange: (value: string) => void;
  onPayerRoleChange: (value: string) => void;
  onSave: () => void;
  onCancel: () => void;
  duplicateCheckResult?: StudentDuplicateCheckResult | null;
  onOpenExistingStudent: (studentId: number) => void;
  onCreateAnyway: () => void;
  t: TranslateFn;
};

export function StudentFormModal({
  editing,
  name,
  personalCode,
  phone,
  email,
  note,
  isMinor,
  payerName,
  payerRole,
  payerRoleOptions,
  payerRoleLabel,
  onNameChange,
  onPersonalCodeChange,
  onPhoneChange,
  onEmailChange,
  onNoteChange,
  onIsMinorChange,
  onPayerNameChange,
  onPayerRoleChange,
  onSave,
  onCancel,
  duplicateCheckResult,
  onOpenExistingStudent,
  onCreateAnyway,
  t,
}: StudentFormModalProps) {
  const exactMatch = duplicateCheckResult?.exactMatch;
  const possibleMatches = duplicateCheckResult?.possibleMatches ?? [];

  return (
    <div className="modal">
      <div className="modalBody">
        <h3>{editing ? t("modal.editStudent") : t("modal.addStudent")}</h3>
        <div className="formRow">
          <label>{t("field.name")}</label>
          <input value={name} onChange={(e) => onNameChange(e.target.value)} />
        </div>
        <div className="formRow">
          <label>{t("field.personalCode")}</label>
          <input value={personalCode} onChange={(e) => onPersonalCodeChange(e.target.value)} />
        </div>
        <div className="formRow">
          <label>{isMinor ? t("student.parentPhone") : t("field.phone")}</label>
          <input value={phone} onChange={(e) => onPhoneChange(e.target.value)} />
        </div>
        <div className="formRow">
          <label>{isMinor ? t("student.parentEmail") : t("field.email")}</label>
          <input value={email} onChange={(e) => onEmailChange(e.target.value)} />
        </div>
        <div className="formRow">
          <label>{t("field.note")}</label>
          <input value={note} onChange={(e) => onNoteChange(e.target.value)} />
        </div>
        <div className="formRow">
          <label>{t("field.studentType")}</label>
          <label className="inline">
            <input
              type="checkbox"
              checked={isMinor}
              onChange={(e) => onIsMinorChange(e.target.checked)}
            />
            {t("student.minor")}
          </label>
        </div>
        {isMinor && (
          <>
            <div className="formRow">
              <label>{t("field.payerName")}</label>
              <input value={payerName} onChange={(e) => onPayerNameChange(e.target.value)} />
            </div>
            <div className="formRow">
              <label>{t("field.payerRole")}</label>
              <select value={payerRole} onChange={(e) => onPayerRoleChange(e.target.value)}>
                <option value="">{t("filter.selectRole")}</option>
                {payerRoleOptions.map((role) => (
                  <option key={role} value={role}>
                    {payerRoleLabel(role)}
                  </option>
                ))}
              </select>
            </div>
          </>
        )}

        {(exactMatch || possibleMatches.length > 0) && (
          <div className="formRow">
            <div
              style={{
                width: "100%",
                padding: "12px",
                borderRadius: "8px",
                backgroundColor: "#fff4e5",
                border: "1px solid #f59e0b",
              }}
            >
              <div style={{ fontWeight: 600, marginBottom: "8px" }}>
                {exactMatch ? t("student.duplicateExactTitle") : t("student.duplicatePossibleTitle")}
              </div>
              <div style={{ marginBottom: "12px", lineHeight: "1.5" }}>
                {exactMatch ? t("msg.studentDuplicateExact") : t("msg.studentDuplicatePossible")}
              </div>
              <div style={{ display: "grid", gap: "8px" }}>
                {(exactMatch ? [exactMatch] : possibleMatches).map((student) => (
                  <div
                    key={student.id}
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      gap: "12px",
                      alignItems: "center",
                      padding: "10px",
                      borderRadius: "6px",
                      backgroundColor: "white",
                    }}
                  >
                    <div>
                      <div style={{ fontWeight: 600 }}>{student.fullName}</div>
                      <div style={{ fontSize: "0.95em", color: "#555" }}>
                        {[student.personalCode, student.phone, student.email].filter(Boolean).join(" · ")}
                      </div>
                      <div style={{ fontSize: "0.9em", color: "#777" }}>
                        {student.isActive ? t("student.statusActive") : t("student.statusInactive")}
                      </div>
                    </div>
                    <button type="button" onClick={() => onOpenExistingStudent(student.id)}>
                      {t("button.openStudent")}
                    </button>
                  </div>
                ))}
              </div>
              {!exactMatch && possibleMatches.length > 0 && (
                <div style={{ marginTop: "12px", display: "flex", justifyContent: "flex-end" }}>
                  <button type="button" onClick={onCreateAnyway}>
                    {t("button.createAnyway")}
                  </button>
                </div>
              )}
            </div>
          </div>
        )}

        <div className="modalActions">
          <button onClick={onSave}>{t("button.save")}</button>
          <button onClick={onCancel}>{t("button.cancel")}</button>
        </div>
      </div>
    </div>
  );
}
