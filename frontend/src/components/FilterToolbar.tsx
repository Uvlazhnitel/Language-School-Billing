import type { ReactNode } from "react";

type FilterToolbarProps = {
  primaryAction?: ReactNode;
  search?: ReactNode;
  filters?: ReactNode;
  hasActiveFilters?: boolean;
  onClearFilters?: () => void;
  clearLabel?: string;
  secondaryActions?: ReactNode;
};

export function FilterToolbar({
  primaryAction,
  search,
  filters,
  hasActiveFilters = false,
  onClearFilters,
  clearLabel,
  secondaryActions,
}: FilterToolbarProps) {
  return (
    <div className="controls controls--wrap filterToolbar">
      {primaryAction}
      {search}
      {filters}
      {(hasActiveFilters || secondaryActions) && (
        <div className="filterToolbarTrailing">
          {hasActiveFilters && onClearFilters && clearLabel && (
            <button onClick={onClearFilters}>{clearLabel}</button>
          )}
          {secondaryActions}
        </div>
      )}
    </div>
  );
}
