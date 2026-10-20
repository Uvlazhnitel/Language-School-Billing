import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { InvoiceRowActions } from "./InvoiceRowActions";
import { createTranslator, type UiLocale } from "../lib/i18n";
import type { InvoiceListItemView } from "../lib/invoices";

const invoice: InvoiceListItemView = {
  id: 1,
  version: 1,
  studentId: 2,
  studentName: "Student",
  year: 2026,
  month: 10,
  total: 30,
  status: "issued",
  pdfReady: true,
  linesCount: 1,
  number: "LS-1",
  emailCommunicationStatus: "not_sent",
  eventDate: "2026-10-20T00:00:00Z",
};

describe("InvoiceRowActions", () => {
  it.each(["ru-RU", "en-US", "lv-LV"] as UiLocale[])(
    "shows only the primary action and menu slot in %s",
    (locale) => {
      const t = createTranslator(locale);
      const html = renderToStaticMarkup(
        <InvoiceRowActions invoice={invoice} menu={<button>⋯</button>} onAction={vi.fn()} t={t} />
      );
      expect(html.match(/<button/g)).toHaveLength(2);
      expect(html).toContain(t("button.recordPayment"));
      expect(html).not.toContain(t("button.downloadPdf"));
      expect(html).not.toContain(t("button.open"));
      expect(t("button.invoiceActions", { invoice: 1 })).not.toContain("button.");
      expect(t("button.invoiceActions", { invoice: 1 })).toContain("1");
    }
  );

  it("keeps the menu slot when a canceled invoice has no extra actions", () => {
    const t = createTranslator("en-US");
    const html = renderToStaticMarkup(
      <InvoiceRowActions
        invoice={{ ...invoice, status: "canceled" }}
        menu={null}
        onAction={vi.fn()}
        t={t}
      />
    );
    expect(html.match(/<button/g)).toHaveLength(1);
    expect(html).toContain(t("button.open"));
    expect(html).toContain('class="invoiceListMenuSlot"></div>');
  });
});
