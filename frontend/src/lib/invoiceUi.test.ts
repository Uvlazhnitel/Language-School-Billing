import { describe, expect, it } from "vitest";

import { getInvoiceMenuActions } from "./invoiceUi";

describe("invoiceUi", () => {
  it("keeps return-to-draft available for issued invoices after pdf is ready", () => {
    expect(
      getInvoiceMenuActions({ status: "issued", pdfReady: false, canReopenDraft: true })
    ).toEqual(["reopenDraft"]);
    expect(
      getInvoiceMenuActions({ status: "issued", pdfReady: true, canReopenDraft: true })
    ).toEqual(["reopenDraft"]);
  });

  it("offers return-to-draft for paid invoices covered only by credit", () => {
    expect(getInvoiceMenuActions({ status: "paid", pdfReady: true, canReopenDraft: true })).toEqual(
      ["reopenDraft"]
    );
    expect(
      getInvoiceMenuActions({ status: "paid", pdfReady: true, canReopenDraft: false })
    ).toEqual([]);
  });
});

import { getInvoiceRowActions } from "./invoiceUi";
import type { InvoiceStatus } from "./api";

describe("invoice row actions", () => {
  it.each([
    ["draft", false, "issue", ["open"]],
    ["issued_pending_pdf", false, "createPdf", ["open", "recordPayment"]],
    ["issued", false, "createPdf", ["open", "recordPayment"]],
    ["issued", true, "recordPayment", ["open", "downloadPdf"]],
    ["issued_pending_pdf", true, "recordPayment", ["open", "downloadPdf"]],
    ["paid_pending_pdf", false, "createPdf", ["open"]],
    ["paid", false, "createPdf", ["open"]],
    ["paid", true, "downloadPdf", ["open"]],
    ["paid_pending_pdf", true, "downloadPdf", ["open"]],
    ["canceled", false, "open", []],
    ["canceled", true, "open", []],
  ] as const)("%s with pdfReady=%s has one primary action", (status, pdfReady, primary, menu) => {
    expect(getInvoiceRowActions({ status, pdfReady })).toEqual({ primary, menu: [...menu] });
  });

  it.each([
    "draft",
    "issued",
    "issued_pending_pdf",
    "paid",
    "paid_pending_pdf",
    "canceled",
  ] as InvoiceStatus[])("uses the server permission for return to draft on %s", (status) => {
    const allowed = getInvoiceRowActions({ status, pdfReady: true, canReopenDraft: true });
    const denied = getInvoiceRowActions({ status, pdfReady: true, canReopenDraft: false });
    expect(allowed.menu).toEqual([...denied.menu, "reopenDraft"]);
    expect(allowed.menu).not.toContain(allowed.primary);
    expect(new Set(allowed.menu).size).toBe(allowed.menu.length);
  });
});
