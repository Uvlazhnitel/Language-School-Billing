import { describe, expect, it } from "vitest";

import { buildStudentActivity } from "./studentActivity";
import { createTranslator, getMonthNames } from "./i18n";

describe("studentActivity", () => {
  it("uses real enrollment and invoice dates in CRM timeline", () => {
    const t = createTranslator("ru-RU");
    const months = getMonthNames("ru-RU");

    const items = buildStudentActivity({
      enrollments: [
        {
          id: 7,
          version: 1,
          studentId: 1,
          studentName: "Alice",
          courseId: 10,
          courseName: "Latviešu valoda",
          courseType: "group",
          teacherName: "Natalja",
          billingMode: "per_lesson",
          chargeMaterials: true,
          discountPct: 0,
          subscriptionLessonPrice: 0,
          note: "",
          createdAt: "2026-06-03T11:15:00Z",
        },
      ],
      payments: [],
      debts: [],
      monthInvoices: [
        {
          id: 5,
          version: 2,
          studentId: 1,
          studentName: "Alice",
          year: 2026,
          month: 6,
          total: 17.5,
          status: "draft",
          linesCount: 1,
          eventDate: "2026-06-02T09:30:00Z",
        },
      ],
      months,
      paymentMethodLabel: (method) => method,
      billingModeLabel: (mode) => mode,
      t,
    });

    expect(items.find((item) => item.kind === "enrollment")?.date).toBe("2026-06-03T11:15:00Z");
    expect(items.find((item) => item.kind === "invoice")?.date).toBe("2026-06-02T09:30:00Z");
    expect(items.some((item) => item.date.startsWith("9999-"))).toBe(false);
  });
});
