import { describe, expect, it } from "vitest";
import { createTranslator } from "./i18n";
import {
  debtorSortColumns,
  courseSortColumns,
  invoiceSortColumns,
  enrollmentSortColumns,
} from "./tableSortColumns";
import { sortTableRows, type SortColumn } from "./tableSort";
import type { DebtorDTO } from "./payments";
import type { CourseDTO } from "./courses";
import type { InvoiceListItemView } from "./invoices";
import type { EnrollmentDTO } from "./enrollments";

const t = createTranslator("en-US");
function checkBothDirections<T>(columns: SortColumn<T>[], rows: T[], id: (row: T) => number) {
  for (const column of columns) {
    const ascending = sortTableRows(
      rows,
      { key: column.key, direction: "asc" },
      columns,
      "en-US",
      id
    );
    const descending = sortTableRows(
      rows,
      { key: column.key, direction: "desc" },
      columns,
      "en-US",
      id
    );
    expect(ascending.map(id), column.key).toEqual([1, 2]);
    expect(descending.map(id), column.key).toEqual([2, 1]);
  }
}

describe("table column values", () => {
  it("sorts each debtor column by its own saved amount or name", () => {
    const rows: DebtorDTO[] = [
      {
        studentId: 2,
        studentName: "Zoe",
        isActive: false,
        debt: 100,
        totalInvoiced: 150,
        totalPaid: 50,
      },
      {
        studentId: 1,
        studentName: "Anna",
        isActive: true,
        debt: 9,
        totalInvoiced: 10,
        totalPaid: 1,
      },
    ];
    checkBothDirections(debtorSortColumns(t), rows, (row) => row.studentId);
  });
  it("sorts invoice periods chronologically across a year boundary and localized statuses", () => {
    const base = {
      version: 1,
      studentId: 1,
      month: 12,
      year: 2025,
      pdfReady: false,
      linesCount: 1,
      eventDate: "",
      emailCommunicationStatus: "not_sent" as const,
    };
    const rows: InvoiceListItemView[] = [
      {
        ...base,
        id: 2,
        studentName: "Zoe",
        number: "LS-10",
        total: 100,
        status: "paid",
        year: 2026,
        month: 1,
      },
      { ...base, id: 1, studentName: "Anna", number: "LS-2", total: 9, status: "draft" },
    ];
    const labels: Record<string, string> = { draft: "A draft", paid: "Z paid" };
    checkBothDirections(
      invoiceSortColumns(t, (status) => labels[status]),
      rows,
      (row) => row.id
    );
  });
  it("sorts both course prices independently and uses the displayed format", () => {
    const rows: CourseDTO[] = [
      {
        id: 2,
        version: 1,
        name: "Same subject",
        teacherName: "Zoe",
        type: "individual",
        lessonPrice: 100,
        subscriptionPrice: 10,
      },
      {
        id: 1,
        version: 1,
        name: "Same subject",
        teacherName: "Anna",
        type: "group",
        lessonPrice: 9,
        subscriptionPrice: 0,
      },
    ];
    checkBothDirections(
      courseSortColumns(t, (type) => (type === "group" ? "A group" : "Z individual")),
      rows,
      (row) => row.id
    );
    const columns = courseSortColumns(t, (type) => type);
    const unassigned = { ...rows[1], id: 3, teacherName: "" };
    expect(
      sortTableRows(
        [...rows, unassigned],
        { key: "teacher", direction: "desc" },
        columns,
        "en-US",
        (row) => row.id
      ).map((row) => row.id)
    ).toEqual([2, 1, 3]);
  });
  it("sorts enrolments by their own billing price, not the unused rate", () => {
    const base = {
      version: 1,
      studentId: 1,
      courseId: 1,
      courseName: "Course",
      courseType: "group" as const,
      teacherName: "Teacher",
      createdAt: "",
    };
    const rows: EnrollmentDTO[] = [
      {
        ...base,
        id: 2,
        studentName: "Zoe",
        billingMode: "subscription",
        subscriptionLessonPrice: 100,
        lessonPriceOverride: 0,
        chargeMaterials: true,
        note: "Z note",
      },
      {
        ...base,
        id: 1,
        studentName: "Anna",
        billingMode: "per_lesson",
        subscriptionLessonPrice: 999,
        lessonPriceOverride: 9,
        chargeMaterials: false,
        note: "A note",
      },
    ];
    checkBothDirections(
      enrollmentSortColumns(t, (mode) => (mode === "per_lesson" ? "A hourly" : "Z subscription")),
      rows,
      (row) => row.id
    );
  });
});
