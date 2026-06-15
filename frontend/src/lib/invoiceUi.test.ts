import { describe, expect, it } from "vitest";

import { getInvoiceMenuActions } from "./invoiceUi";

describe("invoiceUi", () => {
  it("keeps return-to-draft available for issued invoices after pdf is ready", () => {
    expect(getInvoiceMenuActions({ status: "issued", pdfReady: false })).toEqual([
      "reopenDraft",
      "createPdf",
    ]);
    expect(getInvoiceMenuActions({ status: "issued", pdfReady: true })).toEqual(["reopenDraft"]);
  });

  it("does not offer return-to-draft for paid invoices", () => {
    expect(getInvoiceMenuActions({ status: "paid", pdfReady: true })).toEqual([]);
  });
});
