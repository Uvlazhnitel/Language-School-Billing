import type { InvoiceStatus } from "./api";

export type InvoiceMenuAction = "reopenDraft";

export function getInvoiceMenuActions(
  invoice: Pick<{ status: InvoiceStatus; pdfReady?: boolean }, "status" | "pdfReady">
): InvoiceMenuAction[] {
  const actions: InvoiceMenuAction[] = [];

  if (invoice.status === "issued" || invoice.status === "issued_pending_pdf") {
    actions.push("reopenDraft");
  }

  return actions;
}
