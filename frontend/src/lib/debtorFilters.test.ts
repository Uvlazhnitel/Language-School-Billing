import { describe, expect, it } from "vitest";
import { filterDebtorsByActivity } from "./debtorFilters";
import { buildDebtorActionQueue } from "./studentActivity";
import { createTranslator } from "./i18n";
import type { DebtorDTO } from "./payments";

const debtors: DebtorDTO[] = Array.from({ length: 8 }, (_, i) => ({
  studentId: i + 1,
  studentName: `Student ${i + 1}`,
  isActive: i < 6,
  debt: 100 - i * 10,
  totalInvoiced: 100 - i * 10,
  totalPaid: 0,
}));

describe("debtor activity filter", () => {
  it("keeps all debtors by default and preserves debt sorting", () => {
    expect(filterDebtorsByActivity(debtors, "all")).toEqual(debtors);
    expect(filterDebtorsByActivity(debtors, "active").map((d) => d.studentId)).toEqual([
      1, 2, 3, 4, 5, 6,
    ]);
    expect(filterDebtorsByActivity(debtors, "inactive").map((d) => d.studentId)).toEqual([7, 8]);
  });
  it("filters before selecting the action queue and keeps totals consistent", () => {
    const filtered = filterDebtorsByActivity(debtors, "inactive");
    const queue = buildDebtorActionQueue(filtered, [], createTranslator("ru-RU"));
    expect(queue.map((d) => d.studentId)).toEqual([7, 8]);
    expect(queue.every((d) => !d.isActive)).toBe(true);
    expect(filtered.reduce((sum, d) => sum + d.debt, 0)).toBe(70);
    expect(debtors).toHaveLength(8);
  });
  it("handles no debtors or no matching activity", () => {
    expect(filterDebtorsByActivity([], "all")).toEqual([]);
    expect(filterDebtorsByActivity(debtors.slice(0, 6), "inactive")).toEqual([]);
  });
});
