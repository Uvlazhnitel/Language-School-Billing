import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { SortableHeader, TableSortTools } from "./SortableHeader";
import type { TableSortController } from "../hooks/useTableSort";
import { createTranslator, type UiLocale } from "../lib/i18n";

const sorting: TableSortController<{ value: number }> = {
  sort: { key: "amount", direction: "desc" },
  columns: [{ key: "amount", label: "Amount", value: (row) => row.value }],
  choose: vi.fn(),
  set: vi.fn(),
  reset: vi.fn(),
  isDefault: false,
  rows: (rows) => [...rows],
};
describe("sortable headings", () => {
  it.each(["ru-RU", "en-US", "lv-LV"] as UiLocale[])(
    "announces direction and next action in %s",
    (locale) => {
      const t = createTranslator(locale);
      const html = renderToStaticMarkup(
        <table>
          <thead>
            <tr>
              <SortableHeader sorting={sorting} column="amount" t={t} />
            </tr>
          </thead>
        </table>
      );
      expect(html).toContain('aria-sort="descending"');
      expect(html).toContain(t("sort.nextAscending", { column: "Amount" }));
      expect(html).toContain('type="button"');
      expect(html).toContain('aria-hidden="true">↓');
      expect(html).not.toContain("sort.nextAscending");
    }
  );
  it("shows the ascending state and mobile column and direction selectors", () => {
    const t = createTranslator("en-US");
    const controls = { ...sorting, sort: { key: "amount", direction: "asc" as const } };
    const html = renderToStaticMarkup(
      <>
        <table>
          <thead>
            <tr>
              <SortableHeader sorting={controls} column="amount" t={t} />
            </tr>
          </thead>
        </table>
        <TableSortTools sorting={controls} t={t} mobileSelector />
      </>
    );
    expect(html).toContain('aria-sort="ascending"');
    expect(html).toContain(t("sort.nextDescending", { column: "Amount" }));
    expect(html.match(/<select/g)).toHaveLength(2);
    expect(html).toContain(t("sort.reset"));
  });
  it("keeps unsorted headings usable and disables reset at the default", () => {
    const t = createTranslator("en-US");
    const html = renderToStaticMarkup(
      <>
        <table>
          <thead>
            <tr>
              <SortableHeader sorting={{ ...sorting, sort: null }} column="amount" t={t} />
            </tr>
          </thead>
        </table>
        <TableSortTools
          sorting={{ ...sorting, sort: null, isDefault: true }}
          t={t}
          mobileSelector
        />
      </>
    );
    expect(html).toContain('aria-sort="none"');
    expect(html).toContain('aria-hidden="true">↕');
    expect(html).toContain('class="tableSortReset" disabled=""');
    expect(html).toContain('aria-label="Sort direction" disabled=""');
  });
});
