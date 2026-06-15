import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import { InvoiceDetailsModal } from "./InvoiceDetailsModal";
import { createTranslator, getMonthNames } from "../../lib/i18n";

describe("InvoiceDetailsModal", () => {
  it("shows return to draft for issued invoices", () => {
    const markup = renderToStaticMarkup(
      <InvoiceDetailsModal
        invoice={{
          id: 12,
          version: 3,
          studentId: 5,
          studentName: "Alice",
          recipientName: "Alice",
          recipientPhone: "",
          recipientEmail: "",
          childName: "",
          studentPersonalCode: "",
          isMinor: false,
          year: 2026,
          month: 6,
          total: 42,
          status: "issued",
          number: "LS-202606-001",
          lines: [],
        }}
        summary={{
          invoiceId: 12,
          total: 42,
          paid: 0,
          remaining: 42,
          status: "issued",
          number: "LS-202606-001",
        }}
        months={getMonthNames("en-US")}
        invoiceStatusLabel={(value) => value}
        formatEUR={(value) => value.toFixed(2)}
        formatHoursValue={(value) => String(value)}
        onOpenStudent={vi.fn()}
        onIssue={vi.fn()}
        onDownloadPdf={vi.fn()}
        onAddPayment={vi.fn()}
        onReopenToDraft={vi.fn()}
        onClose={vi.fn()}
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Return to draft");
  });
});
