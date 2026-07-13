import type { ReactNode } from "react";

type FilterToolbarProps = {
  primaryAction?: ReactNode;
  search?: ReactNode;
  filters?: ReactNode;
  quickFilters?: ReactNode;
  advancedFilters?: ReactNode;
  advancedFiltersOpen?: boolean;
  onToggleAdvancedFilters?: () => void;
  advancedFiltersLabel?: string;
  hasActiveAdvancedFilters?: boolean;
  hasActiveFilters?: boolean;
  onClearFilters?: () => void;
  clearLabel?: string;
  secondaryActions?: ReactNode;
};

export function FilterToolbar({
  primaryAction,
  search,
  filters,
  quickFilters,
  advancedFilters,
  advancedFiltersOpen = false,
  onToggleAdvancedFilters,
  advancedFiltersLabel,
  hasActiveAdvancedFilters = false,
  hasActiveFilters = false,
  onClearFilters,
  clearLabel,
  secondaryActions,
}: FilterToolbarProps) {
  const effectiveAdvancedFilters = advancedFilters ?? filters;
  const shouldShowAdvancedFilters =
    advancedFilters !== undefined ? advancedFiltersOpen && effectiveAdvancedFilters : effectiveAdvancedFilters;

  return (
    <div className="controls controls--wrap filterToolbar">
      <div className="filterToolbarTopRow">
        {primaryAction}
        {search}
        {quickFilters}
        {advancedFilters === undefined && filters}
        <div className="filterToolbarTrailing">
          {advancedFiltersLabel && onToggleAdvancedFilters && (
            <button
              type="button"
              className={`filterToggleButton ${
                advancedFiltersOpen || hasActiveAdvancedFilters ? "active" : ""
              }`}
              onClick={onToggleAdvancedFilters}
            >
              {advancedFiltersLabel}
            </button>
          )}
          {hasActiveFilters && onClearFilters && clearLabel && (
            <button type="button" onClick={onClearFilters}>
              {clearLabel}
            </button>
          )}
          {secondaryActions}
        </div>
      </div>
      {advancedFilters !== undefined && shouldShowAdvancedFilters && (
        <div className="filterToolbarAdvancedRow">{advancedFilters}</div>
      )}
    </div>
  );
}
