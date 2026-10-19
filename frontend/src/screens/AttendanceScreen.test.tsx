import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { AttendanceScreen } from "./AttendanceScreen";
import { AppShell } from "../components/AppShell";
import { AttendanceEditor } from "../lib/attendanceEditor";
import { attendanceRow } from "../lib/attendanceFixture";
import { createTranslator } from "../lib/i18n";
import type { Row } from "../lib/attendance";
function render(rows: Row[]) {
  const editor = new AttendanceEditor(2026, 9, vi.fn(), () => "Invalid");
  editor.snapshot = { rows, entries: {}, loading: false };
  const meta = { id: "attendance", label: "Attendance", title: "Attendance", eyebrow: "Tracking" };
  return renderToStaticMarkup(
    <AppShell
      tabs={[meta]}
      activeTab="attendance"
      currentMeta={meta}
      year={2026}
      month={9}
      monthLabels={["September"]}
      onTabChange={vi.fn()}
      onMonthChange={vi.fn()}
      onYearChange={vi.fn()}
      showMonthPicker
    >
      <AttendanceScreen
        editor={editor}
        snapshot={editor.snapshot}
        phones={new Map()}
        onRefresh={vi.fn()}
        onOpenInvoices={vi.fn()}
        onOpenEnrollments={vi.fn()}
        onOpenStudent={vi.fn()}
        t={createTranslator("en-US")}
        formatEUR={(n) => `€${n.toFixed(2)}`}
        courseTypeLabel={(s) => s}
      />
    </AppShell>
  );
}
describe("attendance in AppShell", () => {
  it("opens an actionable teacher/course, shows blank missing quantity and removes enrollment deletion", () => {
    const markup = render([attendanceRow()]);
    expect(markup).toContain('aria-expanded="true"');
    expect(markup).toContain('aria-pressed="true"');
    expect(markup).toContain('value=""');
    expect(markup).toContain("No lessons");
    expect(markup).toContain('aria-label="Hours · Anna"');
    expect(markup).toContain('button class="btnPrimary"');
    expect(markup).not.toContain("Delete enrollment");
    expect(markup).not.toContain("Cannot delete");
  });
  it("shows subscription units, individual prices, confirmed totals and invoice lock reason", () => {
    const markup = render([
      attendanceRow({
        billingMode: "subscription",
        hours: 1.25,
        hasRecord: true,
        attendanceLocked: true,
        invoiceStatus: "paid",
        subscriptionLessonPrice: 12,
      }),
    ]);
    expect(markup).toContain("Lessons");
    expect(markup).toContain("Price / lesson");
    expect(markup).toContain("€15.00");
    expect(markup).toContain("Invoice paid");
    expect(markup).toContain("1.25");
    expect(markup).not.toContain('aria-label="Lessons · Anna"');
  });
  it("renders an empty month", () => {
    expect(render([])).toContain("No enrollments for this month");
  });
  it.each(["ru-RU", "en-US", "lv-LV"] as const)(
    "provides attendance translations in %s",
    (locale) => {
      const t = createTranslator(locale);
      for (const key of [
        "missing",
        "retry",
        "discard",
        "invalid",
        "noLessons",
        "lockPaid",
        "lessonPrice",
      ])
        expect(t(`attendance.${key}`)).not.toBe(`attendance.${key}`);
    }
  );
});
