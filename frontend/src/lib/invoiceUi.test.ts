import { describe, expect, it } from "vitest";

import { getInvoiceMenuActions } from "./invoiceUi";

describe("invoiceUi", () => {
  it("keeps return-to-draft available for issued invoices after pdf is ready", () => {
    expect(getInvoiceMenuActions({ status: "issued", pdfReady: false, canReopenDraft: true })).toEqual(["reopenDraft"]);
    expect(getInvoiceMenuActions({ status: "issued", pdfReady: true, canReopenDraft: true })).toEqual(["reopenDraft"]);
  });

  it("offers return-to-draft for paid invoices covered only by credit", () => {
    expect(getInvoiceMenuActions({ status: "paid", pdfReady: true, canReopenDraft: true })).toEqual(["reopenDraft"]);
    expect(getInvoiceMenuActions({ status: "paid", pdfReady: true, canReopenDraft: false })).toEqual([]);
  });
});
