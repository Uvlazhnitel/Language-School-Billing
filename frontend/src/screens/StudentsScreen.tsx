import { StudentWorkspace } from "../components/StudentWorkspace";
import { StudentFormModal } from "../components/modals/StudentFormModal";
import type { EnrollmentDTO } from "../lib/enrollments";
import type { InvoiceListItemView } from "../lib/invoices";
import type { BalanceDTO, DebtInvoiceDTO, PaymentDTO } from "../lib/payments";
import type { StudentDTO, StudentDuplicateCheckResult } from "../lib/students";
import type { StudentActivityItem, StudentNextAction } from "../lib/studentActivity";
import type { TranslateFn } from "../lib/i18n";
import type { CourseDTO } from "../lib/courses";
import type {
  StudentOnboardingEnrollmentRow,
  StudentOnboardingEnrollmentRowPatch,
} from "../lib/studentOnboarding";
import type {
  StudentAgeFilter,
  StudentBalanceFilter,
  StudentDebtFilter,
  StudentSortOption,
  StudentStatusFilter,
} from "../lib/studentListControls";

type StudentsScreenProps = {
  students: StudentDTO[];
  loading: boolean;
  query: string;
  statusFilter: StudentStatusFilter;
  debtFilter: StudentDebtFilter;
  balanceFilter: StudentBalanceFilter;
  ageFilter: StudentAgeFilter;
  sortOption: StudentSortOption;
  hasActiveStudentFilters: boolean;
  selectedStudent: StudentDTO | null;
  detailLoading: boolean;
  detailEnrollments: EnrollmentDTO[];
  detailBalance: BalanceDTO | null;
  detailDebts: DebtInvoiceDTO[];
  detailPayments: PaymentDTO[];
  detailMonthInvoices: InvoiceListItemView[];
  detailNextAction: StudentNextAction | null;
  detailActivity: StudentActivityItem[];
  deletingPaymentId: number | null;
  canDeleteStudent: boolean;
  canDeletePayment: boolean;
  payerRoleLabel: (role: string) => string;
  billingModeLabel: (mode: string) => string;
  paymentMethodLabel: (method: string) => string;
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  months: string[];
  onQueryChange: (value: string) => void;
  onStatusFilterChange: (value: StudentStatusFilter) => void;
  onDebtFilterChange: (value: StudentDebtFilter) => void;
  onBalanceFilterChange: (value: StudentBalanceFilter) => void;
  onAgeFilterChange: (value: StudentAgeFilter) => void;
  onSortOptionChange: (value: StudentSortOption) => void;
  onResetStudentFilters: () => void;
  onAddStudent: () => void;
  onSelectStudent: (student: StudentDTO) => void | Promise<void>;
  onEditStudent: (student: StudentDTO) => void;
  onToggleActive: (student: StudentDTO) => void | Promise<void>;
  onDeleteStudent: (studentId: number) => void | Promise<void>;
  onAddPayment: () => void;
  onCopyDebtRu: () => void | Promise<void>;
  onCopyDebtLv: () => void | Promise<void>;
  onDeletePayment: (payment: PaymentDTO) => void | Promise<void>;
  onManageEnrollments: () => void;
  onOpenInvoices: () => void;
  studentModalOpen: boolean;
  editingStudent: boolean;
  sfName: string;
  sfPersonalCode: string;
  sfPhone: string;
  sfEmail: string;
  sfNote: string;
  sfIsMinor: boolean;
  sfPayerName: string;
  sfPayerRole: string;
  studentDuplicateCheckResult: StudentDuplicateCheckResult | null;
  payerRoleOptions: readonly string[];
  allCourses: CourseDTO[];
  onboardingRows: StudentOnboardingEnrollmentRow[];
  onSfNameChange: (value: string) => void;
  onSfPersonalCodeChange: (value: string) => void;
  onSfPhoneChange: (value: string) => void;
  onSfEmailChange: (value: string) => void;
  onSfNoteChange: (value: string) => void;
  onSfIsMinorChange: (value: boolean) => void;
  onSfPayerNameChange: (value: string) => void;
  onSfPayerRoleChange: (value: string) => void;
  onAddOnboardingRow: () => void;
  onRemoveOnboardingRow: (rowId: number) => void;
  onOnboardingCourseChange: (rowId: number, courseId: number) => void;
  onOnboardingModeChange: (rowId: number, value: EnrollmentDTO["billingMode"]) => void;
  onOnboardingRowChange: (
    rowId: number,
    patch: StudentOnboardingEnrollmentRowPatch
  ) => void;
  onSaveStudent: () => void;
  onSaveStudentAndAddAnother: () => void;
  onOpenExistingDuplicateStudent: (studentId: number) => void;
  onEnrollExistingDuplicateStudent: (student: StudentDTO) => void;
  onCreateStudentAnyway: () => void;
  onCloseStudentModal: () => void;
  t: TranslateFn;
};

export function StudentsScreen({
  studentModalOpen,
  editingStudent,
  sfName,
  sfPersonalCode,
  sfPhone,
  sfEmail,
  sfNote,
  sfIsMinor,
  sfPayerName,
  sfPayerRole,
  studentDuplicateCheckResult,
  payerRoleOptions,
  allCourses,
  onboardingRows,
  onSfNameChange,
  onSfPersonalCodeChange,
  onSfPhoneChange,
  onSfEmailChange,
  onSfNoteChange,
  onSfIsMinorChange,
  onSfPayerNameChange,
  onSfPayerRoleChange,
  onAddOnboardingRow,
  onRemoveOnboardingRow,
  onOnboardingCourseChange,
  onOnboardingModeChange,
  onOnboardingRowChange,
  onSaveStudent,
  onSaveStudentAndAddAnother,
  onOpenExistingDuplicateStudent,
  onEnrollExistingDuplicateStudent,
  onCreateStudentAnyway,
  onCloseStudentModal,
  t,
  ...workspaceProps
}: StudentsScreenProps) {
  return (
    <>
      <StudentWorkspace {...workspaceProps} t={t} />
      {studentModalOpen && (
        <StudentFormModal
          editing={editingStudent}
          name={sfName}
          personalCode={sfPersonalCode}
          phone={sfPhone}
          email={sfEmail}
          note={sfNote}
          isMinor={sfIsMinor}
          payerName={sfPayerName}
          payerRole={sfPayerRole}
          payerRoleOptions={payerRoleOptions}
          payerRoleLabel={workspaceProps.payerRoleLabel}
          allCourses={allCourses}
          enrollmentRows={onboardingRows}
          formatEUR={workspaceProps.formatEUR}
          onNameChange={onSfNameChange}
          onPersonalCodeChange={onSfPersonalCodeChange}
          onPhoneChange={onSfPhoneChange}
          onEmailChange={onSfEmailChange}
          onNoteChange={onSfNoteChange}
          onIsMinorChange={onSfIsMinorChange}
          onPayerNameChange={onSfPayerNameChange}
          onPayerRoleChange={onSfPayerRoleChange}
          onAddEnrollmentRow={onAddOnboardingRow}
          onRemoveEnrollmentRow={onRemoveOnboardingRow}
          onEnrollmentCourseChange={onOnboardingCourseChange}
          onEnrollmentModeChange={onOnboardingModeChange}
          onEnrollmentRowChange={onOnboardingRowChange}
          onSave={onSaveStudent}
          onSaveAndAddAnother={onSaveStudentAndAddAnother}
          onCancel={onCloseStudentModal}
          duplicateCheckResult={studentDuplicateCheckResult}
          onOpenExistingStudent={onOpenExistingDuplicateStudent}
          onEnrollExistingStudent={onEnrollExistingDuplicateStudent}
          onCreateAnyway={onCreateStudentAnyway}
          t={t}
        />
      )}
    </>
  );
}
