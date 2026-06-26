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
          pdfReady: false,
          lastEmailedAt: "2026-06-18T10:30:00Z",
          lastEmailedTo: "family@example.com",
          emailCommunicationStatus: "stale",
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
        formatDateTime={() => "6/18/2026, 10:30:00 AM"}
        pdfReady={false}
        onOpenStudent={vi.fn()}
        onIssue={vi.fn()}
        onGeneratePdf={vi.fn()}
        onDownloadPdf={vi.fn()}
        onSendEmail={vi.fn()}
        onAddPayment={vi.fn()}
        onReopenToDraft={vi.fn()}
        onClose={vi.fn()}
        canSendEmail
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Return to draft");
    expect(markup).toContain("Create PDF");
    expect(markup).toContain("Send Email");
    expect(markup).toContain("Record payment");
    expect(markup).toContain("Email status");
    expect(markup).toContain("Sent, but invoice changed after sending");
    expect(markup).toContain("Sent to family@example.com");
  });

  it("hides record payment for fully paid invoices", () => {
    const markup = renderToStaticMarkup(
      <InvoiceDetailsModal
        invoice={{
          id: 13,
          version: 3,
          studentId: 5,
          studentName: "Paid Student",
          recipientName: "Paid Student",
          recipientPhone: "",
          recipientEmail: "",
          childName: "",
          studentPersonalCode: "",
          isMinor: false,
          year: 2026,
          month: 6,
          total: 250,
          status: "paid",
          number: "LS-202606-008",
          pdfReady: true,
          lastEmailedAt: "",
          lastEmailedTo: "",
          emailCommunicationStatus: "not_sent",
          lines: [],
        }}
        summary={{
          invoiceId: 13,
          total: 250,
          paid: 250,
          remaining: 0,
          status: "paid",
          number: "LS-202606-008",
        }}
        months={getMonthNames("en-US")}
        invoiceStatusLabel={(value) => value}
        formatEUR={(value) => value.toFixed(2)}
        formatHoursValue={(value) => String(value)}
        formatDateTime={() => "6/18/2026, 10:30:00 AM"}
        pdfReady={true}
        onOpenStudent={vi.fn()}
        onIssue={vi.fn()}
        onGeneratePdf={vi.fn()}
        onDownloadPdf={vi.fn()}
        onSendEmail={vi.fn()}
        onAddPayment={vi.fn()}
        onReopenToDraft={vi.fn()}
        onClose={vi.fn()}
        canSendEmail
        t={createTranslator("en-US")}
      />
    );

    expect(markup).toContain("Download PDF");
    expect(markup).not.toContain("Record payment");
  });
});
