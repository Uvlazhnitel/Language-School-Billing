import { Fragment } from "react";
import type { AuditLogItem } from "../lib/audit";
import type { TranslateFn } from "../lib/i18n";
import { EmptyState } from "../components/EmptyState";
import { FilterToolbar } from "../components/FilterToolbar";

type AuditScreenProps = {
  loading: boolean;
  items: AuditLogItem[];
  total: number;
  page: number;
  pageSize: number;
  expandedId: number | null;
  q: string;
  actorFilter: string;
  entityTypeFilter: string;
  actionFilter: string;
  dateFrom: string;
  dateTo: string;
  onQChange: (value: string) => void;
  onActorFilterChange: (value: string) => void;
  onEntityTypeFilterChange: (value: string) => void;
  onActionFilterChange: (value: string) => void;
  onDateFromChange: (value: string) => void;
  onDateToChange: (value: string) => void;
  onRefresh: () => void;
  onResetFilters: () => void;
  onToggleExpanded: (id: number) => void;
  onPrevPage: () => void;
  onNextPage: () => void;
  actionLabel: (action: string) => string;
  t: TranslateFn;
};

export function AuditScreen({
  loading,
  items,
  total,
  page,
  pageSize,
  expandedId,
  q,
  actorFilter,
  entityTypeFilter,
  actionFilter,
  dateFrom,
  dateTo,
  onQChange,
  onActorFilterChange,
  onEntityTypeFilterChange,
  onActionFilterChange,
  onDateFromChange,
  onDateToChange,
  onRefresh,
  onResetFilters,
  onToggleExpanded,
  onPrevPage,
  onNextPage,
  actionLabel,
  t,
}: AuditScreenProps) {
  const hasActiveFilters = Boolean(
    q.trim() || actorFilter.trim() || entityTypeFilter || actionFilter || dateFrom || dateTo
  );

  return (
    <>
      <div className="sectionBanner">
        <div>
          <div className="dashboardCardEyebrow">{t("eyebrow.audit")}</div>
          <strong>{t("title.audit")}</strong>
          <span className="mutedInline">{t("audit.subtitle")}</span>
        </div>
      </div>

      <FilterToolbar
        search={
          <input
            className="searchField searchFieldWide"
            placeholder={t("audit.searchPlaceholder")}
            value={q}
            onChange={(e) => onQChange(e.target.value)}
          />
        }
        filters={
          <>
            <input
              className="searchField"
              placeholder={t("audit.actorPlaceholder")}
              value={actorFilter}
              onChange={(e) => onActorFilterChange(e.target.value)}
            />
            <select
              value={entityTypeFilter}
              onChange={(e) => onEntityTypeFilterChange(e.target.value)}
            >
              <option value="">{t("audit.allEntities")}</option>
              <option value="invoice">invoice</option>
              <option value="payment">payment</option>
              <option value="invoice_batch">{t("audit.batchEntity")}</option>
            </select>
            <select value={actionFilter} onChange={(e) => onActionFilterChange(e.target.value)}>
              <option value="">{t("audit.allActions")}</option>
              <option value="invoice.generate_drafts">invoice.generate_drafts</option>
              <option value="invoice.rebuild_student_draft">invoice.rebuild_student_draft</option>
              <option value="invoice.delete_draft">invoice.delete_draft</option>
              <option value="invoice.issue">invoice.issue</option>
              <option value="invoice.issue_all">invoice.issue_all</option>
              <option value="invoice.reopen_draft">invoice.reopen_draft</option>
              <option value="payment.create">payment.create</option>
              <option value="payment.allocate_or_credit">payment.allocate_or_credit</option>
              <option value="payment.delete">payment.delete</option>
            </select>
            <input
              type="date"
              value={dateFrom}
              onChange={(e) => onDateFromChange(e.target.value)}
            />
            <input type="date" value={dateTo} onChange={(e) => onDateToChange(e.target.value)} />
          </>
        }
        hasActiveFilters={hasActiveFilters}
        onClearFilters={onResetFilters}
        clearLabel={t("button.clearFilters")}
        secondaryActions={
          <button className="workspaceActionButton" onClick={onRefresh}>
            {t("button.refresh")}
          </button>
        }
      />

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : items.length === 0 ? (
        hasActiveFilters ? (
          <EmptyState
            title={t("audit.emptyFiltered")}
            description={t("audit.subtitle")}
            actionLabel={t("button.clearFilters")}
            onAction={onResetFilters}
          />
        ) : (
          <div className="empty">{t("audit.empty")}</div>
        )
      ) : (
        <>
          <table>
            <thead>
              <tr>
                <th>{t("field.date")}</th>
                <th>{t("field.user")}</th>
                <th>{t("field.entity")}</th>
                <th>{t("field.action")}</th>
                <th>{t("field.summary")}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <Fragment key={item.id}>
                  <tr>
                    <td>{new Date(item.createdAt).toLocaleString()}</td>
                    <td>{item.actorLabel || "system"}</td>
                    <td>
                      {item.entityType}
                      {typeof item.entityId === "number" ? ` #${item.entityId}` : ""}
                    </td>
                    <td>{actionLabel(item.action)}</td>
                    <td>{item.summary}</td>
                    <td>
                      <button onClick={() => onToggleExpanded(item.id)}>
                        {expandedId === item.id ? t("button.hide") : t("button.open")}
                      </button>
                    </td>
                  </tr>
                  {expandedId === item.id && (
                    <tr>
                      <td colSpan={6}>
                        <div className="auditDetails">
                          <div className="auditDetailMeta">
                            <span>
                              {t("field.studentId")}: {item.studentId ?? "—"}
                            </span>
                            <span>
                              {t("field.invoiceId")}: {item.invoiceId ?? "—"}
                            </span>
                          </div>
                          <div className="auditJsonGrid">
                            <div>
                              <h4>{t("audit.before")}</h4>
                              <pre>{item.beforeJson || "{}"}</pre>
                            </div>
                            <div>
                              <h4>{t("audit.after")}</h4>
                              <pre>{item.afterJson || "{}"}</pre>
                            </div>
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>

          <div className="auditPager">
            <span>{t("audit.totalRows", { count: total })}</span>
            <div className="inlineActions">
              <button disabled={page <= 1} onClick={onPrevPage}>
                {t("button.prev")}
              </button>
              <span>{t("audit.pageLabel", { page })}</span>
              <button disabled={page * pageSize >= total} onClick={onNextPage}>
                {t("button.next")}
              </button>
            </div>
          </div>
        </>
      )}
    </>
  );
}
