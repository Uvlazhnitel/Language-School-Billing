import { useTableSort } from "../hooks/useTableSort";
import { SortableHeader, TableSortTools } from "../components/SortableHeader";
import { debtorSortColumns } from "../lib/tableSortColumns";
import type { DebtorDTO } from "../lib/payments";
import { buildDebtorActionQueue } from "../lib/studentActivity";
import { filterDebtorsByActivity, type DebtorActivityFilter } from "../lib/debtorFilters";
import { EmptyState } from "../components/EmptyState";
import type { TranslateFn } from "../lib/i18n";

type DebtorsScreenProps = {
  loading: boolean;
  debtors: DebtorDTO[];
  recentPayments: Array<{ studentId: number }>;
  formatEUR: (value: number) => string;
  onRefresh: () => void;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onOpenPaymentForStudent: (studentId: number) => void;
  onOpenPaymentForDebtor: (debtor: DebtorDTO) => void;
  onOpenDebtDetails: (debtor: DebtorDTO) => void;
  onCopyDebtForStudentRu: (studentId: number) => void | Promise<void>;
  onCopyDebtForStudentLv: (studentId: number) => void | Promise<void>;
  onCopyDebtForDebtorRu: (debtor: DebtorDTO) => void | Promise<void>;
  onCopyDebtForDebtorLv: (debtor: DebtorDTO) => void | Promise<void>;
  t: TranslateFn;
  activityFilter: DebtorActivityFilter;
  onActivityFilterChange: (filter: DebtorActivityFilter) => void;
};

export function DebtorsScreen({
  loading,
  debtors,
  recentPayments,
  formatEUR,
  onRefresh,
  onOpenStudent,
  onOpenPaymentForStudent,
  onOpenPaymentForDebtor,
  onOpenDebtDetails,
  onCopyDebtForStudentRu,
  onCopyDebtForStudentLv,
  onCopyDebtForDebtorRu,
  onCopyDebtForDebtorLv,
  t,
  activityFilter,
  onActivityFilterChange,
}: DebtorsScreenProps) {
  const sorting = useTableSort<DebtorDTO>("debtors", debtorSortColumns(t), {
    key: "debt",
    direction: "desc",
  });
  const visibleDebtors = filterDebtorsByActivity(debtors, activityFilter);
  const actionQueue = buildDebtorActionQueue(visibleDebtors, recentPayments, t);
  return (
    <>
      <div className="sectionBanner">
        <div>
          <div className="dashboardCardEyebrow">{t("msg.collection")}</div>
          <strong>{t("label.needsAction")}</strong>
        </div>
        <div className="sectionBannerActions">
          <label className="debtorActivityFilter">
            <span>{t("filter.debtorActivity")}</span>
            <select
              value={activityFilter}
              onChange={(event) =>
                onActivityFilterChange(event.target.value as DebtorActivityFilter)
              }
            >
              <option value="all">{t("filter.studentStatusAll")}</option>
              <option value="active">{t("filter.studentStatusActive")}</option>
              <option value="inactive">{t("filter.studentStatusInactive")}</option>
            </select>
          </label>
          <button className="workspaceActionButton" onClick={onRefresh}>
            {t("button.refresh")}
          </button>
        </div>
      </div>

      {!loading && actionQueue.length > 0 && (
        <div className="detailCard detailCard--wide actionQueuePanel">
          <div className="detailCardHeader">
            <h3>{t("label.needsAction")}</h3>
            <span className="statusPill warning">
              {t("msg.queueCount", { count: actionQueue.length })}
            </span>
          </div>
          <div className="actionQueueList">
            {actionQueue.map((item) => (
              <div key={item.studentId} className="actionQueueItem">
                <div>
                  <div className="debtorIdentity">
                    <strong>{item.studentName}</strong>
                    <span className={`statusPill ${item.isActive ? "success" : "warning"}`}>
                      {t(item.isActive ? "status.active" : "status.inactive")}
                    </span>
                  </div>
                  <span>{item.subtitle}</span>
                </div>
                <div className="actionQueueMeta">
                  <strong>{formatEUR(item.debt)}</strong>
                  <div className="actionQueueActions">
                    <button
                      className="workspaceActionButton workspaceActionButtonPrimary"
                      onClick={() => onOpenPaymentForStudent(item.studentId)}
                    >
                      {t("button.takePayment")}
                    </button>
                    <button
                      className="secondaryActionButton"
                      onClick={() => void onOpenStudent(item.studentId)}
                    >
                      {t("button.card")}
                    </button>
                    <div className="actionQueueSecondaryGroup">
                      <button
                        className="secondaryActionButton secondaryActionButton--mini"
                        onClick={() => void onCopyDebtForStudentRu(item.studentId)}
                      >
                        RU
                      </button>
                      <button
                        className="secondaryActionButton secondaryActionButton--mini"
                        onClick={() => void onCopyDebtForStudentLv(item.studentId)}
                      >
                        LV
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <TableSortTools sorting={sorting} t={t} />
      {loading ? (
        <div>{t("label.loading")}</div>
      ) : debtors.length === 0 ? (
        <div className="empty">{t("msg.noDebtors")}</div>
      ) : visibleDebtors.length === 0 ? (
        <EmptyState
          title={t("msg.noDebtorsFiltered")}
          description={t("msg.noDebtorsFilteredDescription")}
          actionLabel={t("button.clearFilters")}
          onAction={() => onActivityFilterChange("all")}
        />
      ) : (
        <table>
          <thead>
            <tr>
              <SortableHeader sorting={sorting} column="student" t={t} align="left" />
              <SortableHeader sorting={sorting} column="debt" t={t} align="right" />
              <SortableHeader sorting={sorting} column="total" t={t} align="right" />
              <SortableHeader sorting={sorting} column="paid" t={t} align="right" />
              <th></th>
            </tr>
          </thead>
          <tbody>
            {sorting
              .rows(visibleDebtors, (row) => row.studentId)
              .map((debtor) => (
                <tr key={debtor.studentId}>
                  <td>
                    <div className="debtorIdentity">
                      <button
                        className="linkButton"
                        onClick={() => void onOpenStudent(debtor.studentId)}
                      >
                        {debtor.studentName}
                      </button>
                      <span className={`statusPill ${debtor.isActive ? "success" : "warning"}`}>
                        {t(debtor.isActive ? "status.active" : "status.inactive")}
                      </span>
                    </div>
                  </td>
                  <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                    {formatEUR(debtor.debt)}
                  </td>
                  <td style={{ textAlign: "right" }}>{formatEUR(debtor.totalInvoiced)}</td>
                  <td style={{ textAlign: "right" }}>{formatEUR(debtor.totalPaid)}</td>
                  <td>
                    <div className="invoiceRowActions">
                      <button
                        className="workspaceActionButton workspaceActionButtonPrimary invoicePrimaryAction"
                        onClick={() => onOpenPaymentForDebtor(debtor)}
                      >
                        {t("button.takePayment")}
                      </button>
                      <button
                        className="workspaceActionButton"
                        onClick={() => onOpenDebtDetails(debtor)}
                      >
                        {t("modal.debtBreakdown")}
                      </button>
                      <button
                        className="secondaryActionButton"
                        onClick={() => void onCopyDebtForDebtorRu(debtor)}
                      >
                        {t("button.copyRu")}
                      </button>
                      <button
                        className="secondaryActionButton"
                        onClick={() => void onCopyDebtForDebtorLv(debtor)}
                      >
                        {t("button.copyLv")}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
          </tbody>
          <tfoot>
            <tr>
              <td style={{ fontWeight: "bold" }}>{t("field.debtEur")}:</td>
              <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                {formatEUR(visibleDebtors.reduce((sum, debtor) => sum + debtor.debt, 0))}
              </td>
              <td colSpan={3}></td>
            </tr>
          </tfoot>
        </table>
      )}
    </>
  );
}
