import { StudentDetailPanel } from "../StudentDetailPanel";
import type { EnrollmentDTO } from "../../lib/enrollments";
import type { InvoiceListItemView } from "../../lib/invoices";
import type { BalanceDTO, DebtInvoiceDTO, PaymentDTO } from "../../lib/payments";
import type { StudentDTO } from "../../lib/students";
import type { StudentActivityItem, StudentNextAction } from "../../lib/studentActivity";
import type { TranslateFn } from "../../lib/i18n";

type StudentCardModalProps = {
  student: StudentDTO;
  loading: boolean;
  enrollments: EnrollmentDTO[];
  balance: BalanceDTO | null;
  debts: DebtInvoiceDTO[];
  payments: PaymentDTO[];
  monthInvoices: InvoiceListItemView[];
  nextAction: StudentNextAction | null;
  activity: StudentActivityItem[];
  deletingPaymentId: number | null;
  canDeletePayment: boolean;
  payerRoleLabel: (role: string) => string;
  billingModeLabel: (mode: string) => string;
  paymentMethodLabel: (method: string) => string;
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  months: string[];
  onEditStudent: () => void;
  onAddPayment: () => void;
  onCopyDebtRu: () => void | Promise<void>;
  onCopyDebtLv: () => void | Promise<void>;
  onDeletePayment: (payment: PaymentDTO) => void | Promise<void>;
  onManageEnrollments: () => void;
  onOpenInvoices: () => void;
  onClose: () => void;
  t: TranslateFn;
};

export function StudentCardModal({
  student,
  loading,
  enrollments,
  balance,
  debts,
  payments,
  monthInvoices,
  nextAction,
  activity,
  deletingPaymentId,
  canDeletePayment,
  payerRoleLabel,
  billingModeLabel,
  paymentMethodLabel,
  invoiceStatusLabel,
  formatEUR,
  months,
  onEditStudent,
  onAddPayment,
  onCopyDebtRu,
  onCopyDebtLv,
  onDeletePayment,
  onManageEnrollments,
  onOpenInvoices,
  onClose,
  t,
}: StudentCardModalProps) {
  return (
    <div className="modal" onClick={onClose}>
      <div className="modalBody modalBodyWide" onClick={(e) => e.stopPropagation()}>
        <StudentDetailPanel
          student={student}
          loading={loading}
          enrollments={enrollments}
          balance={balance}
          debts={debts}
          payments={payments}
          monthInvoices={monthInvoices}
          nextAction={nextAction}
          activity={activity}
          t={t}
          payerRoleLabel={payerRoleLabel}
          billingModeLabel={billingModeLabel}
          paymentMethodLabel={paymentMethodLabel}
          invoiceStatusLabel={invoiceStatusLabel}
          formatEUR={formatEUR}
          months={months}
          deletingPaymentId={deletingPaymentId}
          canDeletePayment={canDeletePayment}
          onEditStudent={onEditStudent}
          onAddPayment={onAddPayment}
          onCopyDebtRu={() => void onCopyDebtRu()}
          onCopyDebtLv={() => void onCopyDebtLv()}
          onDeletePayment={(payment) => void onDeletePayment(payment)}
          onManageEnrollments={onManageEnrollments}
          onOpenInvoices={onOpenInvoices}
          footer={
            <div className="modalActions">
              <button onClick={onClose}>{t("button.close")}</button>
            </div>
          }
        />
      </div>
    </div>
  );
}
