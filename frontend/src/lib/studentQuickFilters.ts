import type {
  StudentAgeFilter,
  StudentBalanceFilter,
  StudentDebtFilter,
  StudentListControls,
  StudentSortOption,
  StudentStatusFilter,
} from "./studentListControls";

export type StudentQuickFilterKey = "active" | "debt" | "recent";

type StudentQuickFilterDefaults = {
  statusFilter: StudentStatusFilter;
  debtFilter: StudentDebtFilter;
  balanceFilter: StudentBalanceFilter;
  ageFilter: StudentAgeFilter;
  sortOption: StudentSortOption;
};

export const studentQuickFilterDefaults: StudentQuickFilterDefaults = {
  statusFilter: "active",
  debtFilter: "all",
  balanceFilter: "all",
  ageFilter: "all",
  sortOption: "created_desc",
};

export function isStudentQuickFilterActive(
  key: StudentQuickFilterKey,
  controls: StudentListControls
): boolean {
  switch (key) {
    case "active":
      return controls.statusFilter === "active";
    case "debt":
      return controls.debtFilter === "debt_only";
    case "recent":
      return controls.sortOption === "created_desc";
    default:
      return false;
  }
}

export function toggleStudentQuickFilter(
  key: StudentQuickFilterKey,
  controls: StudentListControls
): StudentListControls {
  switch (key) {
    case "active":
      return {
        ...controls,
        statusFilter: controls.statusFilter === "active" ? "all" : "active",
      };
    case "debt":
      return {
        ...controls,
        debtFilter: controls.debtFilter === "debt_only" ? "all" : "debt_only",
      };
    case "recent":
      return {
        ...controls,
        sortOption: controls.sortOption === "created_desc" ? "name_asc" : "created_desc",
      };
    default:
      return controls;
  }
}

export function hasActiveAdvancedStudentFilters(controls: StudentListControls): boolean {
  return (
    (controls.statusFilter !== studentQuickFilterDefaults.statusFilter &&
      controls.statusFilter !== "all") ||
    (controls.debtFilter !== studentQuickFilterDefaults.debtFilter &&
      controls.debtFilter !== "debt_only") ||
    controls.balanceFilter !== studentQuickFilterDefaults.balanceFilter ||
    controls.ageFilter !== studentQuickFilterDefaults.ageFilter ||
    (controls.sortOption !== studentQuickFilterDefaults.sortOption &&
      controls.sortOption !== "created_desc")
  );
}
