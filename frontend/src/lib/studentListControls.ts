import type { StudentDTO } from "./api";

export type StudentStatusFilter = "active" | "inactive" | "all";
export type StudentDebtFilter = "all" | "debt_only" | "no_debt";
export type StudentBalanceFilter = "all" | "credit_only" | "zero_or_debt";
export type StudentAgeFilter = "all" | "minors_only" | "adults_only";
export type StudentSortOption =
  | "name_asc"
  | "name_desc"
  | "created_desc"
  | "created_asc"
  | "debt_desc"
  | "balance_desc";

export type StudentListControls = {
  statusFilter: StudentStatusFilter;
  debtFilter: StudentDebtFilter;
  balanceFilter: StudentBalanceFilter;
  ageFilter: StudentAgeFilter;
  sortOption: StudentSortOption;
};

function compareNameAsc(a: StudentDTO, b: StudentDTO): number {
  return a.fullName.localeCompare(b.fullName) || a.id - b.id;
}

function compareNameDesc(a: StudentDTO, b: StudentDTO): number {
  return b.fullName.localeCompare(a.fullName) || b.id - a.id;
}

function parseCreatedAt(value: string): number {
  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

export function applyStudentListControls(
  students: StudentDTO[],
  controls: StudentListControls
): StudentDTO[] {
  const filtered = students.filter((student) => {
    if (controls.statusFilter === "active" && !student.isActive) return false;
    if (controls.statusFilter === "inactive" && student.isActive) return false;

    if (controls.debtFilter === "debt_only" && student.debt <= 0) return false;
    if (controls.debtFilter === "no_debt" && student.debt > 0) return false;

    if (controls.balanceFilter === "credit_only" && student.balance <= 0) return false;
    if (controls.balanceFilter === "zero_or_debt" && student.balance > 0) return false;

    if (controls.ageFilter === "minors_only" && !student.isMinor) return false;
    if (controls.ageFilter === "adults_only" && student.isMinor) return false;

    return true;
  });

  return [...filtered].sort((a, b) => {
    switch (controls.sortOption) {
      case "name_desc":
        return compareNameDesc(a, b);
      case "created_desc":
        return parseCreatedAt(b.createdAt) - parseCreatedAt(a.createdAt) || b.id - a.id;
      case "created_asc":
        return parseCreatedAt(a.createdAt) - parseCreatedAt(b.createdAt) || a.id - b.id;
      case "debt_desc":
        return b.debt - a.debt || compareNameAsc(a, b);
      case "balance_desc":
        return b.balance - a.balance || compareNameAsc(a, b);
      case "name_asc":
      default:
        return compareNameAsc(a, b);
    }
  });
}
