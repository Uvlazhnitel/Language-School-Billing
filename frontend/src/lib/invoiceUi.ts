import type { InvoiceStatus } from "./api";

export type InvoiceMenuAction = "reopenDraft" | "createPdf";

export function getInvoiceMenuActions(
  invoice: Pick<{ status: InvoiceStatus; pdfReady?: boolean }, "status" | "pdfReady">
): InvoiceMenuAction[] {
  const actions: InvoiceMenuAction[] = [];

  if (invoice.status === "issued") {
    actions.push("reopenDraft");
  }
  if (invoice.status !== "draft" && !invoice.pdfReady) {
    actions.push("createPdf");
  }

  return actions;
}
