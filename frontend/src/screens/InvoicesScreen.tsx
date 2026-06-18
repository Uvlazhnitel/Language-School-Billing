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
  renderInvoiceActionsMenu: (invoice: InvoiceListItemView) => ReactNode;
  onStatusChange: (value: string) => void;
  onQueryChange: (value: string) => void;
  onRefresh: () => void;
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
  renderInvoiceActionsMenu,
  onStatusChange,
  onQueryChange,
  onRefresh,
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
  const hasActiveFilters = Boolean(query.trim() || status !== "all");

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
            <option value="issued">{t("filter.selectStatusIssued")}</option>
            <option value="paid">{t("filter.selectStatusPaid")}</option>
            <option value="all">{t("filter.selectStatusAll")}</option>
          </select>
        }
        hasActiveFilters={hasActiveFilters}
        onClearFilters={onResetFilters}
        clearLabel={t("button.clearFilters")}
        secondaryActions={
          <button className="workspaceActionButton" onClick={onRefresh}>
            {t("button.sync")}
          </button>
        }
      />

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
              <th>{t("field.student")}</th>
              <th>{t("field.period")}</th>
              <th style={{ textAlign: "right" }}>{t("field.amount")} (EUR)</th>
              <th>{t("field.status")}</th>
              <th>{t("field.number")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td>
                  <button className="linkButton" onClick={() => void onOpenStudent(item.studentId)}>
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
                </td>
                <td>
                  <div className="invoiceRowActions">
                    <button
                      className="workspaceActionButton"
                      onClick={() => void onOpenInvoice(item.id)}
                    >
                      {t("button.open")}
                    </button>
                    {item.status === "draft" && (
                      <button
                        className="workspaceActionButtonPrimary workspaceActionButton invoicePrimaryAction"
                        onClick={() => void onIssueOne(item.id)}
                      >
                        {t("button.issue")}
                      </button>
                    )}
                    {item.status === "issued" && (
                      !item.pdfReady ? (
                        <button
                          className="workspaceActionButtonPrimary workspaceActionButton invoicePrimaryAction"
                          onClick={() => void onGeneratePdf(item.id)}
                        >
                          {t("button.createPdf")}
                        </button>
                      ) : (
                      <button
                        className="workspaceActionButtonPrimary workspaceActionButton invoicePrimaryAction"
                        onClick={() => void onOpenPaymentModal(item.id)}
                      >
                        {t("button.recordPayment")}
                      </button>
                      )
                    )}
                    {item.status === "paid" && (
                      !item.pdfReady ? (
                        <button
                          className="workspaceActionButtonPrimary workspaceActionButton invoicePrimaryAction"
                          onClick={() => void onGeneratePdf(item.id)}
                        >
                          {t("button.createPdf")}
                        </button>
                      ) : (
                        <button
                          className="workspaceActionButtonPrimary workspaceActionButton invoicePrimaryAction"
                          onClick={() => void onDownloadPdf(item.id)}
                        >
                          {t("button.downloadPdf")}
                        </button>
                      )
                    )}
                    {item.status === "issued" && (
                      item.pdfReady ? (
                        <button
                          className="secondaryActionButton"
                          onClick={() => void onDownloadPdf(item.id)}
                        >
                          {t("button.downloadPdf")}
                        </button>
                      ) : (
                        <button
                          className="secondaryActionButton"
                          onClick={() => void onOpenPaymentModal(item.id)}
                        >
                          {t("button.recordPayment")}
                        </button>
                      )
                    )}
                    {renderInvoiceActionsMenu(item)}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
