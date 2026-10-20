import { useTableSort } from "../hooks/useTableSort";
import { SortableHeader, TableSortTools } from "../components/SortableHeader";
import { invoiceSortColumns } from "../lib/tableSortColumns";
import { InvoiceRowActions } from "../components/InvoiceRowActions";
import type { ReactNode } from "react";
import { EmptyState } from "../components/EmptyState";
import type { InvoiceListItemView } from "../lib/invoices";
import type { TranslateFn } from "../lib/i18n";
import { FilterToolbar } from "../components/FilterToolbar";

type InvoicesScreenProps = {
  currentMonthLabel: string;
  status: string;
  query: string;
  loading: boolean;
  items: InvoiceListItemView[];
  months: string[];
  invoiceStatusLabel: (status: string) => string;
  formatEUR: (value: number) => string;
  formatDateTime: (value: string) => string;
  renderInvoiceActionsMenu: (invoice: InvoiceListItemView) => ReactNode;
  onStatusChange: (value: string) => void;
  onQueryChange: (value: string) => void;
  onRefresh: () => void;
  onEnsureAllPdfs: () => void;
  onResetFilters: () => void;
  onOpenAttendance: () => void;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onOpenInvoice: (invoiceId: number) => void | Promise<void>;
  onIssueOne: (invoiceId: number) => void | Promise<void>;
  onGeneratePdf: (invoiceId: number) => void | Promise<void>;
  onDownloadPdf: (invoiceId: number) => void | Promise<void>;
  onOpenPaymentModal: (invoiceId: number) => void | Promise<void>;
  t: TranslateFn;
};

export function InvoicesScreen({
  currentMonthLabel,
  status,
  query,
  loading,
  items,
  months,
  invoiceStatusLabel,
  formatEUR,
  formatDateTime,
  renderInvoiceActionsMenu,
  onStatusChange,
  onQueryChange,
  onRefresh,
  onEnsureAllPdfs,
  onResetFilters,
  onOpenAttendance,
  onOpenStudent,
  onOpenInvoice,
  onIssueOne,
  onGeneratePdf,
  onDownloadPdf,
  onOpenPaymentModal,
  t,
}: InvoicesScreenProps) {
  const sorting = useTableSort<InvoiceListItemView>(
    "invoices",
    invoiceSortColumns(t, invoiceStatusLabel)
  );
  const hasActiveFilters = Boolean(query.trim() || status !== "all");
  const isDraft = (invoiceStatus: InvoiceListItemView["status"]) => invoiceStatus === "draft";
  const renderEmailStatus = (item: InvoiceListItemView) => {
    if (isDraft(item.status)) {
      return null;
    }
    if (item.emailCommunicationStatus === "failed") {
      return (
        <div className="mutedInline" style={{ marginTop: "0.35rem" }}>
          {t("msg.invoiceEmailStatusFailed")}
          {item.lastEmailError ? ` · ${item.lastEmailError}` : ""}
        </div>
      );
    }
    if (item.emailCommunicationStatus === "not_sent") {
      return (
        <div className="mutedInline" style={{ marginTop: "0.35rem" }}>
          {t("msg.invoiceEmailStatusNotSent")}
        </div>
      );
    }
    if (item.lastEmailedAt && item.lastEmailedTo) {
      return (
        <div className="mutedInline" style={{ marginTop: "0.35rem" }}>
          {t(
            item.emailCommunicationStatus === "stale"
              ? "msg.invoiceEmailStatusStale"
              : "msg.invoiceEmailStatusSent"
          )}{" "}
          · {t("msg.invoiceEmailLastSentTo", { email: item.lastEmailedTo })} ·{" "}
          {formatDateTime(item.lastEmailedAt)}
        </div>
      );
    }
    return (
      <div className="mutedInline" style={{ marginTop: "0.35rem" }}>
        {t(
          item.emailCommunicationStatus === "stale"
            ? "msg.invoiceEmailStatusStale"
            : "msg.invoiceEmailStatusSent"
        )}
      </div>
    );
  };

  return (
    <>
      <div className="sectionBanner">
        <div>
          <div className="dashboardCardEyebrow">{t("msg.billing")}</div>
          <strong>{currentMonthLabel}</strong>
          <span className="mutedInline">{t("title.invoice")}</span>
        </div>
      </div>

      <FilterToolbar
        search={
          <input
            className="searchField searchFieldWide"
            placeholder={t("msg.searchPlaceholderInvoice")}
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
          />
        }
        filters={
          <select value={status} onChange={(e) => onStatusChange(e.target.value)}>
            <option value="draft">{t("filter.selectStatusDraft")}</option>
            <option value="issued_pending_pdf">{t("filter.selectStatusIssuedPendingPdf")}</option>
            <option value="issued">{t("filter.selectStatusIssued")}</option>
            <option value="paid_pending_pdf">{t("filter.selectStatusPaidPendingPdf")}</option>
            <option value="paid">{t("filter.selectStatusPaid")}</option>
            <option value="all">{t("filter.selectStatusAll")}</option>
          </select>
        }
        hasActiveFilters={hasActiveFilters}
        onClearFilters={onResetFilters}
        clearLabel={t("button.clearFilters")}
        secondaryActions={
          <>
            <button className="workspaceActionButton" onClick={onEnsureAllPdfs}>
              {t("button.ensureAllPdfs")}
            </button>
            <button className="workspaceActionButton" onClick={onRefresh}>
              {t("button.sync")}
            </button>
          </>
        }
      />

      <TableSortTools sorting={sorting} t={t} />
      {loading ? (
        <div>{t("label.loading")}</div>
      ) : items.length === 0 ? (
        hasActiveFilters ? (
          <EmptyState
            title={t("msg.noInvoiceSearchTitle")}
            description={t("msg.noInvoiceSearchDescription")}
            actionLabel={t("button.clearFilters")}
            onAction={onResetFilters}
          />
        ) : (
          <EmptyState
            title={t("msg.noInvoicesTitle")}
            description={t("msg.noInvoicesDescription")}
            actionLabel={t("button.sync")}
            onAction={onRefresh}
            secondaryActionLabel={t("button.openAttendance")}
            onSecondaryAction={onOpenAttendance}
          />
        )
      ) : (
        <table>
          <thead>
            <tr>
              <SortableHeader sorting={sorting} column="student" t={t} />
              <SortableHeader sorting={sorting} column="period" t={t} />
              <SortableHeader sorting={sorting} column="total" t={t} align="right" />
              <SortableHeader sorting={sorting} column="status" t={t} />
              <SortableHeader sorting={sorting} column="number" t={t} />
              <th></th>
            </tr>
          </thead>
          <tbody>
            {sorting
              .rows(items, (row) => row.id)
              .map((item) => (
                <tr key={item.id}>
                  <td>
                    <button
                      className="linkButton"
                      onClick={() => void onOpenStudent(item.studentId)}
                    >
                      {item.studentName}
                    </button>
                  </td>
                  <td>
                    {months[item.month - 1]} {item.year}
                  </td>
                  <td style={{ textAlign: "right" }}>{formatEUR(item.total)}</td>
                  <td>
                    <span className={`statusPill statusPill--${item.status}`}>
                      {invoiceStatusLabel(item.status)}
                    </span>
                  </td>
                  <td>
                    {item.number ?? ""}
                    {item.pdfReady && (
                      <div className="badgeRow">
                        <span className="attBadge attBadge--pdfReady">PDF</span>
                      </div>
                    )}
                    {renderEmailStatus(item)}
                  </td>
                  <td>
                    <InvoiceRowActions
                      invoice={item}
                      menu={renderInvoiceActionsMenu(item)}
                      onAction={(action, id) => {
                        if (action === "open") void onOpenInvoice(id);
                        if (action === "issue") void onIssueOne(id);
                        if (action === "createPdf") void onGeneratePdf(id);
                        if (action === "recordPayment") void onOpenPaymentModal(id);
                        if (action === "downloadPdf") void onDownloadPdf(id);
                      }}
                      t={t}
                    />
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      )}
    </>
  );
}
