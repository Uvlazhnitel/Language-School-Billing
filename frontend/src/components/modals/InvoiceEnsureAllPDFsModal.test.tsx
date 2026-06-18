import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { InvoiceEnsureAllPDFsModal } from "./InvoiceEnsureAllPDFsModal";
import { createTranslator, getMonthNames } from "../../lib/i18n";

describe("InvoiceEnsureAllPDFsModal", () => {
  it("renders summary and per-item results", () => {
    const markup = renderToStaticMarkup(
      <InvoiceEnsureAllPDFsModal
        result={{
          year: 2026,
          month: 6,
          processed: 3,
          generatedCount: 1,
          alreadyReadyCount: 1,
          failedCount: 1,
          items: [
            {
              invoiceId: 1,
              number: "LS-202606-001",
              studentName: "Alice",
              status: "issued",
              result: "generated",
            },
            {
              invoiceId: 2,
              number: "LS-202606-002",
              studentName: "Bob",
              status: "paid",
              result: "already_ready",
            },
            {
              invoiceId: 3,
              number: "LS-202606-003",
              studentName: "Carol",
              status: "issued_pending_pdf",
              result: "failed",
              message: "disk error",
            },
          ],
        }}
        months={getMonthNames("en-US")}
        invoiceStatusLabel={(value) => value}
        onClose={vi.fn()}
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Create PDFs for all invoices");
    expect(markup).toContain("Processed 3: generated 1, already ready 1, failed 1");
    expect(markup).toContain("Alice");
    expect(markup).toContain("Already ready");
    expect(markup).toContain("disk error");
  });
});
