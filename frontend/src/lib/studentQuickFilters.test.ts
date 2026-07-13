import { describe, expect, it } from "vitest";

import type { StudentListControls } from "./studentListControls";
import {
  hasActiveAdvancedStudentFilters,
  isStudentQuickFilterActive,
  studentQuickFilterDefaults,
  toggleStudentQuickFilter,
} from "./studentQuickFilters";

const baseControls: StudentListControls = {
  ...studentQuickFilterDefaults,
};

describe("studentQuickFilters", () => {
  it("treats the default active filter as an active quick chip", () => {
    expect(isStudentQuickFilterActive("active", baseControls)).toBe(true);
    expect(isStudentQuickFilterActive("debt", baseControls)).toBe(false);
    expect(isStudentQuickFilterActive("recent", baseControls)).toBe(true);
  });

  it("toggles quick chips on and off", () => {
    expect(toggleStudentQuickFilter("active", baseControls).statusFilter).toBe("all");
    expect(toggleStudentQuickFilter("debt", baseControls).debtFilter).toBe("debt_only");
    expect(toggleStudentQuickFilter("recent", baseControls).sortOption).toBe("name_asc");

    expect(
      toggleStudentQuickFilter("recent", { ...baseControls, sortOption: "created_desc" }).sortOption
    ).toBe("name_asc");
  });

  it("treats non-chip filter values as advanced filters", () => {
    expect(hasActiveAdvancedStudentFilters(baseControls)).toBe(false);
    expect(hasActiveAdvancedStudentFilters({ ...baseControls, statusFilter: "inactive" })).toBe(true);
    expect(hasActiveAdvancedStudentFilters({ ...baseControls, balanceFilter: "credit_only" })).toBe(
      true
    );
    expect(hasActiveAdvancedStudentFilters({ ...baseControls, ageFilter: "minors_only" })).toBe(true);
    expect(hasActiveAdvancedStudentFilters({ ...baseControls, sortOption: "created_desc" })).toBe(false);
    expect(hasActiveAdvancedStudentFilters({ ...baseControls, sortOption: "name_asc" })).toBe(true);
  });
});
