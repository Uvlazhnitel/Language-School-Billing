import type { TableSortController } from "../hooks/useTableSort";
import type { TranslateFn } from "../lib/i18n";

type Controls<T> = { sorting: TableSortController<T>; t: TranslateFn };

export function SortableHeader<T>({
  sorting,
  column,
  t,
  align = "left",
  className,
}: Controls<T> & {
  column: string;
  align?: "left" | "right";
  className?: string;
}) {
  const label = sorting.columns.find((item) => item.key === column)?.label ?? column;
  const active = sorting.sort?.key === column;
  const direction = active ? sorting.sort?.direction : undefined;
  return (
    <th
      className={className}
      style={{ textAlign: align }}
      aria-sort={active ? (direction === "asc" ? "ascending" : "descending") : "none"}
    >
      <button
        type="button"
        className={`tableSortHeader${align === "right" ? " tableSortHeader--right" : ""}`}
        onClick={() => sorting.choose(column)}
        aria-label={t(direction === "asc" ? "sort.nextDescending" : "sort.nextAscending", {
          column: label,
        })}
      >
        <span>{label}</span>
        <span
          className={active ? "tableSortArrow tableSortArrow--active" : "tableSortArrow"}
          aria-hidden="true"
        >
          {active ? (direction === "asc" ? "↑" : "↓") : "↕"}
        </span>
      </button>
    </th>
  );
}

export function TableSortTools<T>({
  sorting,
  t,
  mobileSelector = false,
}: Controls<T> & { mobileSelector?: boolean }) {
  return (
    <div className="tableSortTools">
      {mobileSelector && (
        <div className="tableSortMobile">
          <label>
            <span>{t("sort.by")}</span>
            <select
              aria-label={t("sort.by")}
              value={sorting.sort?.key ?? ""}
              onChange={(event) =>
                event.target.value
                  ? sorting.set({ key: event.target.value, direction: "asc" })
                  : sorting.reset()
              }
            >
              <option value="">{t("sort.original")}</option>
              {sorting.columns.map((column) => (
                <option key={column.key} value={column.key}>
                  {column.label}
                </option>
              ))}
            </select>
          </label>
          <select
            aria-label={t("sort.direction")}
            disabled={!sorting.sort}
            value={sorting.sort?.direction ?? "asc"}
            onChange={(event) =>
              sorting.sort &&
              sorting.set({
                ...sorting.sort,
                direction: event.target.value === "desc" ? "desc" : "asc",
              })
            }
          >
            <option value="asc">↑ {t("sort.ascending")}</option>
            <option value="desc">↓ {t("sort.descending")}</option>
          </select>
        </div>
      )}
      <button
        type="button"
        className="tableSortReset"
        disabled={sorting.isDefault}
        onClick={sorting.reset}
      >
        {t("sort.reset")}
      </button>
    </div>
  );
}
