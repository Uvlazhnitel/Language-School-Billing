import { useContext, useState } from "react";
import { nextTableSort, sortTableRows, type SortColumn, type TableSort } from "../lib/tableSort";
import { SortContext } from "../lib/tableSortContext";

export type TableSortController<T> = {
  sort: TableSort;
  choose: (key: string) => void;
  set: (sort: TableSort) => void;
  reset: () => void;
  isDefault: boolean;
  columns: SortColumn<T>[];
  rows: (rows: readonly T[], id: (row: T) => number) => T[];
};

export function useTableSort<T>(
  table: string,
  columns: SortColumn<T>[],
  initial: TableSort = null
): TableSortController<T> {
  const { locale, preferences } = useContext(SortContext);
  const [sort, setSort] = useState<TableSort>(
    () =>
      preferences?.read(
        table,
        columns.map((c) => c.key),
        initial
      ) ?? initial
  );
  const set = (next: TableSort) => {
    setSort(next);
    preferences?.write(table, next);
  };
  return {
    sort,
    columns,
    set,
    choose: (key) => set(nextTableSort(sort, key)),
    reset: () => set(initial),
    isDefault: sort?.key === initial?.key && sort?.direction === initial?.direction,
    rows: (rows, id) => sortTableRows(rows, sort, columns, locale, id),
  };
}
