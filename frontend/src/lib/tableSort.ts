export type SortDirection = "asc" | "desc";
export type TableSort = { key: string; direction: SortDirection } | null;
export type SortValue = string | number | boolean | null | undefined;
export type SortColumn<T> = { key: string; label: string; value: (row: T) => SortValue };

export function nextTableSort(current: TableSort, key: string): TableSort {
  return { key, direction: current?.key === key && current.direction === "asc" ? "desc" : "asc" };
}

export function parseTableSort(raw: unknown, keys: string[], fallback: TableSort): TableSort {
  if (
    raw &&
    typeof raw === "object" &&
    "key" in raw &&
    "direction" in raw &&
    typeof raw.key === "string" &&
    keys.includes(raw.key) &&
    (raw.direction === "asc" || raw.direction === "desc")
  ) {
    return { key: raw.key, direction: raw.direction };
  }
  return fallback;
}

function empty(value: SortValue): boolean {
  return (
    value == null ||
    (typeof value === "string" && value.trim() === "") ||
    (typeof value === "number" && !Number.isFinite(value))
  );
}

export function sortTableRows<T>(
  rows: readonly T[],
  sort: TableSort,
  columns: SortColumn<T>[],
  locale: string,
  id: (row: T) => number
): T[] {
  const column = columns.find((item) => item.key === sort?.key);
  if (!sort || !column) return [...rows];
  const collator = new Intl.Collator(locale, { numeric: true, sensitivity: "base" });
  return [...rows].sort((a, b) => {
    const left = column.value(a);
    const right = column.value(b);
    const leftEmpty = empty(left),
      rightEmpty = empty(right);
    if (leftEmpty !== rightEmpty) return leftEmpty ? 1 : -1;
    if (leftEmpty && rightEmpty) return id(a) - id(b);
    const comparison =
      typeof left === "number" && typeof right === "number"
        ? left - right
        : typeof left === "boolean" && typeof right === "boolean"
          ? Number(left) - Number(right)
          : collator.compare(String(left).trim(), String(right).trim());
    return comparison * (sort.direction === "asc" ? 1 : -1) || id(a) - id(b);
  });
}

export function priceInCents(value: number): number {
  return Math.round(value * 100);
}

type SortStorage = Pick<Storage, "getItem" | "setItem">;
export class TableSortPreferences {
  private memory = new Map<string, unknown>();
  constructor(
    private account: string,
    private storage: () => SortStorage | undefined
  ) {}
  private key(table: string) {
    return `langschool.table-sort.v1.${encodeURIComponent(this.account)}.${table}`;
  }
  read(table: string, keys: string[], fallback: TableSort): TableSort {
    let raw = this.memory.get(table);
    if (!this.memory.has(table)) {
      try {
        const saved = this.storage()?.getItem(this.key(table));
        raw = saved ? JSON.parse(saved) : undefined;
      } catch {
        /* Sorting also works without browser storage. */
      }
    }
    return parseTableSort(raw, keys, fallback);
  }
  write(table: string, sort: TableSort) {
    this.memory.set(table, sort);
    try {
      this.storage()?.setItem(this.key(table), JSON.stringify(sort));
    } catch {
      /* Keep the in-memory choice for this session. */
    }
  }
}
