import { useEffect, useRef, useState } from "react";
import { BalanceDTO, DebtInvoiceDTO, PaymentDTO } from "../lib/payments";
import { EnrollmentDTO } from "../lib/enrollments";
import { TranslateFn } from "../lib/i18n";
import { StudentDTO } from "../lib/students";
import { InvoiceListItemView } from "../lib/invoices";
import { StudentActivityItem, StudentNextAction } from "../lib/studentActivity";
import { StudentDetailPanel } from "./StudentDetailPanel";
import { EmptyState } from "./EmptyState";
import { FilterToolbar } from "./FilterToolbar";
import type {
  StudentAgeFilter,
  StudentBalanceFilter,
  StudentDebtFilter,
  StudentSortOption,
  StudentStatusFilter,
} from "../lib/studentListControls";
import {
  hasActiveAdvancedStudentFilters,
  isStudentQuickFilterActive,
  studentQuickFilterDefaults,
  toggleStudentQuickFilter,
  type StudentQuickFilterKey,
} from "../lib/studentQuickFilters";

type StudentWorkspaceProps = {
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
  t: TranslateFn;
  deletingPaymentId: number | null;
  onQueryChange: (value: string) => void;
  onStatusFilterChange: (value: StudentStatusFilter) => void;
  onDebtFilterChange: (value: StudentDebtFilter) => void;
  onBalanceFilterChange: (value: StudentBalanceFilter) => void;
  onAgeFilterChange: (value: StudentAgeFilter) => void;
  onSortOptionChange: (value: StudentSortOption) => void;
  onResetStudentFilters: () => void;
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
  statusFilter,
  debtFilter,
  balanceFilter,
  ageFilter,
  sortOption,
  hasActiveStudentFilters,
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
  onStatusFilterChange,
  onDebtFilterChange,
  onBalanceFilterChange,
  onAgeFilterChange,
  onSortOptionChange,
  onResetStudentFilters,
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
  const controls = {
    statusFilter,
    debtFilter,
    balanceFilter,
    ageFilter,
    sortOption,
  };
  const advancedFiltersActive = hasActiveAdvancedStudentFilters(controls);
  const [advancedFiltersOpen, setAdvancedFiltersOpen] = useState(advancedFiltersActive);
  const previousAdvancedFiltersActive = useRef(advancedFiltersActive);

  useEffect(() => {
    if (!previousAdvancedFiltersActive.current && advancedFiltersActive) {
      setAdvancedFiltersOpen(true);
    }
    previousAdvancedFiltersActive.current = advancedFiltersActive;
  }, [advancedFiltersActive]);

  useEffect(() => {
    if (
      statusFilter === studentQuickFilterDefaults.statusFilter &&
      debtFilter === studentQuickFilterDefaults.debtFilter &&
      balanceFilter === studentQuickFilterDefaults.balanceFilter &&
      ageFilter === studentQuickFilterDefaults.ageFilter &&
      sortOption === studentQuickFilterDefaults.sortOption
    ) {
      setAdvancedFiltersOpen(false);
    }
  }, [ageFilter, balanceFilter, debtFilter, sortOption, statusFilter]);

  const studentBalanceMeta = (student: StudentDTO) => {
    if (student.debt > 0) {
      return {
        label: t("label.debt"),
        value: formatEUR(student.debt),
        toneClass: "studentBalancePill studentBalancePill--danger",
      };
    }
    if (student.balance > 0) {
      return {
        label: t("label.creditOnAccount"),
        value: formatEUR(student.balance),
        toneClass: "studentBalancePill studentBalancePill--success",
      };
    }
    return {
      label: t("label.balance"),
      value: formatEUR(0),
      toneClass: "studentBalancePill studentBalancePill--neutral",
    };
  };

  const handleQuickFilterToggle = (key: StudentQuickFilterKey) => {
    const nextControls = toggleStudentQuickFilter(key, controls);
    if (nextControls.statusFilter !== statusFilter) {
      onStatusFilterChange(nextControls.statusFilter);
    }
    if (nextControls.debtFilter !== debtFilter) {
      onDebtFilterChange(nextControls.debtFilter);
    }
    if (nextControls.sortOption !== sortOption) {
      onSortOptionChange(nextControls.sortOption);
    }
  };

  return (
    <div className="studentWorkspace">
      <div className="studentSidebar">
        <FilterToolbar
          primaryAction={<button onClick={onAddStudent}>{t("button.addStudent")}</button>}
          search={
            <input
              className="searchField"
              placeholder={t("msg.searchPlaceholderStudent")}
              value={query}
              onChange={(e) => onQueryChange(e.target.value)}
            />
          }
          quickFilters={
            <div className="quickFilterChips">
              {(
                [
                  ["active", t("chip.studentActive")],
                  ["debt", t("chip.studentDebt")],
                  ["recent", t("chip.studentRecent")],
                ] as Array<[StudentQuickFilterKey, string]>
              ).map(([key, label]) => (
                <button
                  key={key}
                  type="button"
                  className={`quickFilterChip ${
                    isStudentQuickFilterActive(key, controls) ? "active" : ""
                  }`}
                  onClick={() => handleQuickFilterToggle(key)}
                >
                  {label}
                </button>
              ))}
            </div>
          }
          advancedFilters={
            <div className="studentAdvancedFilters">
              <select
                value={statusFilter}
                onChange={(e) => onStatusFilterChange(e.target.value as StudentStatusFilter)}
              >
                <option value="active">{t("filter.studentStatusActive")}</option>
                <option value="inactive">{t("filter.studentStatusInactive")}</option>
                <option value="all">{t("filter.studentStatusAll")}</option>
              </select>
              <select
                value={debtFilter}
                onChange={(e) => onDebtFilterChange(e.target.value as StudentDebtFilter)}
              >
                <option value="all">{t("filter.studentDebtAll")}</option>
                <option value="debt_only">{t("filter.studentDebtOnly")}</option>
                <option value="no_debt">{t("filter.studentDebtNone")}</option>
              </select>
              <select
                value={balanceFilter}
                onChange={(e) => onBalanceFilterChange(e.target.value as StudentBalanceFilter)}
              >
                <option value="all">{t("filter.studentBalanceAll")}</option>
                <option value="credit_only">{t("filter.studentBalanceCreditOnly")}</option>
                <option value="zero_or_debt">{t("filter.studentBalanceZeroOrDebt")}</option>
              </select>
              <select
                value={ageFilter}
                onChange={(e) => onAgeFilterChange(e.target.value as StudentAgeFilter)}
              >
                <option value="all">{t("filter.studentAgeAll")}</option>
                <option value="minors_only">{t("filter.studentAgeMinorsOnly")}</option>
                <option value="adults_only">{t("filter.studentAgeAdultsOnly")}</option>
              </select>
              <select
                value={sortOption}
                onChange={(e) => onSortOptionChange(e.target.value as StudentSortOption)}
              >
                <option value="name_asc">{t("filter.studentSortNameAsc")}</option>
                <option value="name_desc">{t("filter.studentSortNameDesc")}</option>
                <option value="created_desc">{t("filter.studentSortCreatedDesc")}</option>
                <option value="created_asc">{t("filter.studentSortCreatedAsc")}</option>
                <option value="debt_desc">{t("filter.studentSortDebtDesc")}</option>
                <option value="balance_desc">{t("filter.studentSortBalanceDesc")}</option>
              </select>
            </div>
          }
          advancedFiltersOpen={advancedFiltersOpen}
          onToggleAdvancedFilters={() => setAdvancedFiltersOpen((value) => !value)}
          advancedFiltersLabel={t("button.filters")}
          hasActiveAdvancedFilters={advancedFiltersActive}
          hasActiveFilters={hasActiveStudentFilters}
          onClearFilters={onResetStudentFilters}
          clearLabel={t("button.clearFilters")}
        />

        {loading ? (
          <div className="empty">{t("label.loading")}</div>
        ) : students.length === 0 ? (
          query.trim() || hasActiveStudentFilters ? (
            <EmptyState
              compact
              title={t("msg.noStudentsSearchTitle")}
              description={t("msg.noStudentsSearchDescription")}
              actionLabel={
                hasActiveStudentFilters ? t("button.clearFilters") : t("button.clearSearch")
              }
              onAction={hasActiveStudentFilters ? onResetStudentFilters : () => onQueryChange("")}
            />
          ) : (
            <EmptyState
              compact
              title={t("msg.noStudentsTitle")}
              description={t("msg.noStudentsDescription")}
              actionLabel={t("button.addStudent")}
              onAction={onAddStudent}
            />
          )
        ) : (
          <div className="studentListPane">
            {students.map((student) => {
              const selected = selectedStudent?.id === student.id;
              const balanceMeta = studentBalanceMeta(student);
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
                  <div className="studentListItemFooter">
                    <span className="studentListItemBalanceLabel">{balanceMeta.label}</span>
                    <span className={balanceMeta.toneClass}>{balanceMeta.value}</span>
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
          onDeleteStudent={selectedStudent ? () => onDeleteStudent(selectedStudent.id) : undefined}
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
