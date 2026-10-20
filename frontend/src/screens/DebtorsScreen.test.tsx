import { SortContext } from "../lib/tableSortContext";
import { TableSortPreferences } from "../lib/tableSort";
import { Children, isValidElement, type ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { DebtorsScreen } from "./DebtorsScreen";
import { createTranslator } from "../lib/i18n";
import type { DebtorDTO } from "../lib/payments";

const debtors: DebtorDTO[] = [
  {
    studentId: 1,
    studentName: "Active Student",
    isActive: true,
    debt: 100,
    totalInvoiced: 150,
    totalPaid: 50,
  },
  {
    studentId: 2,
    studentName: "Inactive Student",
    isActive: false,
    debt: 70,
    totalInvoiced: 100,
    totalPaid: 30,
  },
];
const props = {
  loading: false,
  debtors,
  recentPayments: [],
  activityFilter: "all" as const,
  onActivityFilterChange: vi.fn(),
  formatEUR: (value: number) => `€${value.toFixed(2)}`,
  onRefresh: vi.fn(),
  onOpenStudent: vi.fn(),
  onOpenPaymentForStudent: vi.fn(),
  onOpenPaymentForDebtor: vi.fn(),
  onOpenDebtDetails: vi.fn(),
  onCopyDebtForStudentRu: vi.fn(),
  onCopyDebtForStudentLv: vi.fn(),
  onCopyDebtForDebtorRu: vi.fn(),
  onCopyDebtForDebtorLv: vi.fn(),
  t: createTranslator("en-US"),
};

describe("debtors screen", () => {
  it.each(["all", "active", "inactive"] as const)(
    "uses one %s filter for table, queue, and total",
    (activityFilter) => {
      const html = renderToStaticMarkup(
        <DebtorsScreen {...props} activityFilter={activityFilter} />
      );
      expect(html.includes("Active Student")).toBe(activityFilter !== "inactive");
      expect(html.includes("Inactive Student")).toBe(activityFilter !== "active");
      const footer = html.slice(html.indexOf("<tfoot>"));
      expect(footer).toContain(
        activityFilter === "all" ? "€170.00" : activityFilter === "active" ? "€100.00" : "€70.00"
      );
      expect(html).toContain(`value="${activityFilter}" selected=""`);
      expect(html).toContain('class="actionQueueList"');
      expect(html).toContain(activityFilter === "inactive" ? ">Inactive</span>" : ">Active</span>");
    }
  );
  it("keeps payment, card, breakdown and reminder actions for inactive debtors", () => {
    const html = renderToStaticMarkup(<DebtorsScreen {...props} activityFilter="inactive" />);
    expect(html).toContain(props.t("button.takePayment"));
    expect(html).toContain(props.t("button.card"));
    expect(html).toContain(props.t("modal.debtBreakdown"));
    expect(html).toContain(props.t("button.copyRu"));
    expect(html).toContain(props.t("button.copyLv"));
  });
  it("distinguishes an empty collection from empty filtered results", () => {
    const empty = renderToStaticMarkup(<DebtorsScreen {...props} debtors={[]} />);
    expect(empty).toContain(props.t("msg.noDebtors"));
    expect(empty).not.toContain(props.t("button.clearFilters"));
    const filtered = renderToStaticMarkup(
      <DebtorsScreen {...props} debtors={[debtors[0]]} activityFilter="inactive" />
    );
    expect(filtered).toContain(props.t("msg.noDebtorsFiltered"));
    expect(filtered).toContain(props.t("button.clearFilters"));
    expect(filtered).not.toContain("Active Student");
  });
  it("does not show stale action queue while loading", () => {
    const html = renderToStaticMarkup(<DebtorsScreen {...props} loading />);
    expect(html).toContain(props.t("label.loading"));
    expect(html).not.toContain("Active Student");
  });
  it.each(["en-US", "ru-RU", "lv-LV"] as const)("translates new labels in %s", (locale) => {
    const t = createTranslator(locale);
    const html = renderToStaticMarkup(
      <DebtorsScreen {...props} t={t} debtors={[debtors[0]]} activityFilter="inactive" />
    );
    for (const key of [
      "filter.debtorActivity",
      "msg.noDebtorsFiltered",
      "msg.noDebtorsFilteredDescription",
    ]) {
      expect(html).toContain(t(key));
      expect(html).not.toContain(key);
    }
  });
  it("sorts the filtered table without changing the priority queue or total", () => {
    const preferences = new TableSortPreferences("review", () => undefined);
    preferences.write("debtors", { key: "debt", direction: "asc" });
    const html = renderToStaticMarkup(
      <SortContext.Provider value={{locale: "en-US", preferences}}>
        <DebtorsScreen {...props} />
      </SortContext.Provider>
    );
    const queue = html.slice(html.indexOf('class="actionQueueList"'), html.indexOf("<table"));
    const table = html.slice(html.indexOf("<tbody"), html.indexOf("</tbody>"));
    expect(queue.indexOf("Active Student")).toBeLessThan(queue.indexOf("Inactive Student"));
    expect(table.indexOf("Inactive Student")).toBeLessThan(table.indexOf("Active Student"));
    expect(html.slice(html.indexOf("<tfoot"))).toContain("€170.00");
    expect(html).toContain('aria-sort="ascending"');
  });
  it("opens the selected inactive student's card directly from the queue", () => {
    const onOpenStudent = vi.fn();
    let tree: ReactNode;
    function CaptureScreen() {
      tree = DebtorsScreen({ ...props, activityFilter: "inactive", onOpenStudent });
      return tree;
    }
    renderToStaticMarkup(<CaptureScreen />);
    function clickCard(node: ReactNode): boolean {
      for (const child of Children.toArray(node)) {
        if (!isValidElement<{ children?: ReactNode; onClick?: () => void }>(child)) continue;
        if (child.type === "button" && child.props.children === props.t("button.card")) {
          child.props.onClick?.();
          return true;
        }
        if (clickCard(child.props.children)) return true;
      }
      return false;
    }
    expect(clickCard(tree)).toBe(true);
    expect(onOpenStudent).toHaveBeenCalledWith(2);
  });
});
