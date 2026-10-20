import { describe, expect, it } from "vitest";
import {
  nextTableSort,
  parseTableSort,
  priceInCents,
  sortTableRows,
  TableSortPreferences,
  type TableSort,
  type SortColumn,
} from "./tableSort";

type Row = { id: number; value?: string | number | boolean | null };
const columns: SortColumn<Row>[] = [{ key: "value", label: "Value", value: (row) => row.value }];
const order = (values: Row[], direction: "asc" | "desc", locale = "lv-LV") =>
  sortTableRows(values, { key: "value", direction }, columns, locale, (row) => row.id).map(
    (row) => row.id
  );

describe("table sorting", () => {
  it("toggles the current heading, and starts a new column ascending", () => {
    expect(nextTableSort(null, "debt")).toEqual({ key: "debt", direction: "asc" });
    expect(nextTableSort({ key: "debt", direction: "asc" }, "debt")).toEqual({
      key: "debt",
      direction: "desc",
    });
    expect(nextTableSort({ key: "debt", direction: "desc" }, "debt")).toEqual({
      key: "debt",
      direction: "asc",
    });
    expect(nextTableSort({ key: "debt", direction: "desc" }, "paid")).toEqual({
      key: "paid",
      direction: "asc",
    });
  });
  it("compares money numerically, keeps zero, and breaks ties by ascending ID", () => {
    const rows = Object.freeze([
      { id: 4, value: 100 },
      { id: 2, value: 80 },
      { id: 1, value: 9 },
      { id: 3, value: 80 },
      { id: 5, value: 0 },
    ]);
    expect(order([...rows], "asc")).toEqual([5, 1, 2, 3, 4]);
    expect(order([...rows], "desc")).toEqual([4, 2, 3, 1, 5]);
    expect(rows.map((row) => row.id)).toEqual([4, 2, 1, 3, 5]);
    expect(priceInCents(0.29)).toBe(29);
    expect(priceInCents(12.5)).toBe(1250);
  });
  it.each(["asc", "desc"] as const)(
    "keeps absent, blank and invalid values last in %s order",
    (direction) => {
      const rows: Row[] = [
        { id: 4, value: null },
        { id: 3, value: " " },
        { id: 5 },
        { id: 2, value: NaN },
        { id: 1, value: 0 },
      ];
      expect(order(rows, direction)).toEqual([1, 2, 3, 4, 5]);
    }
  );
  it("sorts natural invoice numbers and localized text", () => {
    expect(
      order(
        [
          { id: 1, value: "LS-10" },
          { id: 2, value: "LS-2" },
        ],
        "asc"
      )
    ).toEqual([2, 1]);
    expect(
      order(
        [
          { id: 1, value: "Борис" },
          { id: 2, value: "Анна" },
        ],
        "asc",
        "ru-RU"
      )
    ).toEqual([2, 1]);
    expect(
      order(
        [
          { id: 1, value: "Žanis" },
          { id: 2, value: "Ārija" },
          { id: 3, value: "Česlavs" },
        ],
        "asc"
      )
    ).toEqual([2, 3, 1]);
    expect(
      order(
        [
          { id: 2, value: "ANNA" },
          { id: 1, value: "anna" },
        ],
        "desc"
      )
    ).toEqual([1, 2]);
    expect(
      order(
        [
          { id: 1, value: true },
          { id: 2, value: false },
        ],
        "asc"
      )
    ).toEqual([2, 1]);
  });
  it("preserves the exact original order when reset or when the column is unknown", () => {
    const rows = [
      { id: 2, value: 1 },
      { id: 1, value: 2 },
    ];
    const result = sortTableRows(rows, null, columns, "en-US", (row) => row.id);
    expect(result).toEqual(rows);
    expect(result).not.toBe(rows);
    expect(
      sortTableRows(rows, { key: "removed", direction: "asc" }, columns, "en-US", (row) => row.id)
    ).toEqual(rows);
  });
});

describe("account-scoped sorting preferences", () => {
  const fallback: TableSort = { key: "value", direction: "desc" };
  function setup() {
    const data = new Map<string, string>();
    const storage = {
      getItem: (key: string) => data.get(key) ?? null,
      setItem: (key: string, value: string) => {
        data.set(key, value);
      },
    };
    return { data, storage };
  }
  it("survives recreation and isolates tables and accounts", () => {
    const { storage } = setup();
    const first = new TableSortPreferences("1", () => storage);
    first.write("debtors", { key: "value", direction: "asc" });
    expect(
      new TableSortPreferences("1", () => storage).read("debtors", ["value"], fallback)
    ).toEqual({ key: "value", direction: "asc" });
    expect(first.read("invoices", ["value"], null)).toBeNull();
    expect(
      new TableSortPreferences("2", () => storage).read("debtors", ["value"], fallback)
    ).toEqual(fallback);
    first.write("debtors", fallback);
    expect(new TableSortPreferences("1", () => storage).read("debtors", ["value"], null)).toEqual(
      fallback
    );
    first.write("invoices", null);
    expect(
      new TableSortPreferences("1", () => storage).read("invoices", ["value"], null)
    ).toBeNull();
  });
  it("keeps the in-memory choice if the storage getter or writes fail", () => {
    const getterFailure = new TableSortPreferences("1", () => {
      throw Error("denied");
    });
    expect(getterFailure.read("courses", ["value"], null)).toBeNull();
    getterFailure.write("courses", fallback);
    expect(getterFailure.read("courses", ["value"], null)).toEqual(fallback);
    const writeFailure = new TableSortPreferences("1", () => ({
      getItem: () => null,
      setItem: () => {
        throw Error("quota");
      },
    }));
    writeFailure.write("courses", fallback);
    expect(writeFailure.read("courses", ["value"], null)).toEqual(fallback);
  });
  it.each([
    "broken",
    "{}",
    '{"key":"removed","direction":"asc"}',
    '{"key":"value","direction":"wrong"}',
    "[]",
  ])("recovers from invalid saved data %s", (raw) => {
    const { data, storage } = setup();
    data.set("langschool.table-sort.v1.1.debtors", raw);
    expect(
      new TableSortPreferences("1", () => storage).read("debtors", ["value"], fallback)
    ).toEqual(fallback);
  });
  it("validates structured preferences", () => {
    expect(parseTableSort({ key: "value", direction: "asc" }, ["value"], fallback)).toEqual({
      key: "value",
      direction: "asc",
    });
  });
});
