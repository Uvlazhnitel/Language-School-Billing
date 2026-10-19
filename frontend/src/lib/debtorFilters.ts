import type { DebtorDTO } from "./payments";

export type DebtorActivityFilter = "all" | "active" | "inactive";

export function filterDebtorsByActivity(
  debtors: DebtorDTO[],
  filter: DebtorActivityFilter
): DebtorDTO[] {
  return debtors.filter((debtor) => filter === "all" || debtor.isActive === (filter === "active"));
}
