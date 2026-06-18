import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { InvoicesScreen } from "./InvoicesScreen";
import { createTranslator, getMonthNames } from "../lib/i18n";

describe("InvoicesScreen", () => {
  it("hides record payment for paid invoices with ready PDF", () => {
    const markup = renderToStaticMarkup(
      <InvoicesScreen
        currentMonthLabel="June 2026"
        status="all"
        query=""
        loading={false}
        items={[
          {
            id: 8,
            version: 1,
            studentId: 10,
            studentName: "Marharyta Karnafel",
            year: 2026,
            month: 6,
            total: 250,
            status: "paid",
            linesCount: 1,
            number: "LS-202606-008",
            eventDate: "2026-06-15T00:00:00Z",
            pdfReady: true,
          },
          {
            id: 7,
            version: 1,
            studentId: 12,
            studentName: "Paid Missing PDF",
            year: 2026,
            month: 6,
            total: 100,
            status: "paid",
            linesCount: 1,
            number: "LS-202606-007",
            eventDate: "2026-06-15T00:00:00Z",
            pdfReady: false,
          },
          {
            id: 9,
            version: 1,
            studentId: 11,
            studentName: "Issued Student",
            year: 2026,
            month: 6,
            total: 30,
            status: "issued",
            linesCount: 1,
            number: "LS-202606-009",
            eventDate: "2026-06-15T00:00:00Z",
            pdfReady: true,
          },
        ]}
        months={getMonthNames("en-US")}
        invoiceStatusLabel={(value) => value}
        formatEUR={(value) => `€${value.toFixed(2)}`}
        renderInvoiceActionsMenu={() => null}
        onStatusChange={vi.fn()}
        onQueryChange={vi.fn()}
        onRefresh={vi.fn()}
        onEnsureAllPdfs={vi.fn()}
        onResetFilters={vi.fn()}
        onOpenAttendance={vi.fn()}
        onOpenStudent={vi.fn()}
        onOpenInvoice={vi.fn()}
        onIssueOne={vi.fn()}
        onGeneratePdf={vi.fn()}
        onDownloadPdf={vi.fn()}
        onOpenPaymentModal={vi.fn()}
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Download PDF");
    expect(markup).toContain("Create PDF");
    expect(markup).toContain("Create PDFs for month");
    expect(markup).toContain("Record payment");
    expect(markup).toContain("Issued Student");
    expect(markup).toContain("Marharyta Karnafel");
    expect(markup).toContain("Paid Missing PDF");
    expect(markup.match(/Record payment/g)?.length ?? 0).toBe(1);
  });
});
