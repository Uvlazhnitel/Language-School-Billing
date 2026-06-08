import { BalanceDTO, DebtInvoiceDTO, PaymentDTO } from "../lib/payments";
import { EnrollmentDTO } from "../lib/enrollments";
import { TranslateFn } from "../lib/i18n";
import { StudentDTO } from "../lib/students";
import { InvoiceListItemView } from "../lib/invoices";
import { StudentActivityItem, StudentNextAction } from "../lib/studentActivity";
import { StudentDetailPanel } from "./StudentDetailPanel";

type StudentWorkspaceProps = {
  students: StudentDTO[];
  loading: boolean;
  query: string;
  includeInactive: boolean;
  selectedStudent: StudentDTO | null;
  detailLoading: boolean;
  detailEnrollments: EnrollmentDTO[];
  detailBalance: BalanceDTO | null;
  detailDebts: DebtInvoiceDTO[];
  detailPayments: PaymentDTO[];
  detailMonthInvoices: InvoiceListItemView[];
  detailNextAction: StudentNextAction | null;
  detailActivity: StudentActivityItem[];
  t: TranslateFn;
  deletingPaymentId: number | null;
  onQueryChange: (value: string) => void;
  onIncludeInactiveChange: (value: boolean) => void;
  onRefresh: () => void;
  onAddStudent: () => void;
  onSelectStudent: (student: StudentDTO) => void;
  onEditStudent: (student: StudentDTO) => void;
  onToggleActive: (student: StudentDTO) => void;
  onDeleteStudent: (studentId: number) => void;
  onAddPayment: () => void;
  onCopyDebtRu: () => void;
  onCopyDebtLv: () => void;
  onDeletePayment: (payment: PaymentDTO) => void;
  canDeleteStudent: boolean;
  canDeletePayment: boolean;
  onManageEnrollments: () => void;
  onOpenInvoices: () => void;
  payerRoleLabel: (relation: string) => string;
  billingModeLabel: (mode: string) => string;
  paymentMethodLabel: (method: string) => string;
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  months: string[];
};

export function StudentWorkspace({
  students,
  loading,
  query,
  includeInactive,
  selectedStudent,
  detailLoading,
  detailEnrollments,
  detailBalance,
  detailDebts,
  detailPayments,
  detailMonthInvoices,
  detailNextAction,
  detailActivity,
  t,
  deletingPaymentId,
  onQueryChange,
  onIncludeInactiveChange,
  onRefresh,
  onAddStudent,
  onSelectStudent,
  onEditStudent,
  onToggleActive,
  onDeleteStudent,
  onAddPayment,
  onCopyDebtRu,
  onCopyDebtLv,
  onDeletePayment,
  canDeleteStudent,
  canDeletePayment,
  onManageEnrollments,
  onOpenInvoices,
  payerRoleLabel,
  billingModeLabel,
  paymentMethodLabel,
  invoiceStatusLabel,
  formatEUR,
  months,
}: StudentWorkspaceProps) {
  return (
    <div className="studentWorkspace">
      <div className="studentSidebar">
        <div className="controls controls--sidebar">
          <button onClick={onAddStudent}>{t("button.addStudent")}</button>
          <input
            className="searchField"
            placeholder={t("msg.searchPlaceholderStudent")}
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
          />
          <label className="inline">
            <input
              type="checkbox"
              checked={includeInactive}
              onChange={(e) => onIncludeInactiveChange(e.target.checked)}
            />
            {t("label.showInactive")}
          </label>
          <button onClick={onRefresh}>{t("button.refresh")}</button>
        </div>

        {loading ? (
          <div className="empty">{t("label.loading")}</div>
        ) : students.length === 0 ? (
          <div className="empty">{t("msg.noStudents")}</div>
        ) : (
          <div className="studentListPane">
            {students.map((student) => {
              const selected = selectedStudent?.id === student.id;
              return (
                <button
                  key={student.id}
                  type="button"
                  className={`studentListItem ${selected ? "active" : ""}`}
                  onClick={() => onSelectStudent(student)}
                >
                  <div className="studentListItemTop">
                    <strong>{student.fullName}</strong>
                    <span className={`statusDot ${student.isActive ? "active" : "inactive"}`}>
                      {student.isActive ? t("status.active") : t("status.inactive")}
                    </span>
                  </div>
                  <div className="studentListItemMeta">
                    <span>{student.phone || t("student.noPhone")}</span>
                    <span>{student.email || t("student.noEmail")}</span>
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>

      <div className="studentMainPanel">
        <StudentDetailPanel
          student={selectedStudent}
          loading={detailLoading}
          enrollments={detailEnrollments}
          balance={detailBalance}
          debts={detailDebts}
          payments={detailPayments}
          monthInvoices={detailMonthInvoices}
          nextAction={detailNextAction}
          activity={detailActivity}
          t={t}
          payerRoleLabel={payerRoleLabel}
          billingModeLabel={billingModeLabel}
          paymentMethodLabel={paymentMethodLabel}
          invoiceStatusLabel={invoiceStatusLabel}
          formatEUR={formatEUR}
          months={months}
          deletingPaymentId={deletingPaymentId}
          canDeletePayment={canDeletePayment}
          onEditStudent={() => selectedStudent && onEditStudent(selectedStudent)}
          onToggleActive={selectedStudent ? () => onToggleActive(selectedStudent) : undefined}
          onDeleteStudent={
            selectedStudent ? () => onDeleteStudent(selectedStudent.id) : undefined
          }
          canDeleteStudent={canDeleteStudent}
          onAddPayment={onAddPayment}
          onCopyDebtRu={onCopyDebtRu}
          onCopyDebtLv={onCopyDebtLv}
          onDeletePayment={onDeletePayment}
          onManageEnrollments={onManageEnrollments}
          onOpenInvoices={onOpenInvoices}
        />
      </div>
    </div>
  );
}
