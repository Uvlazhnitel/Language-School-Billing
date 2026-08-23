import type { InvoiceStatus } from "./api";

export type InvoiceMenuAction = "reopenDraft";

export function getInvoiceMenuActions(
  invoice: Pick<
    { status: InvoiceStatus; pdfReady?: boolean; canReopenDraft?: boolean },
    "status" | "pdfReady" | "canReopenDraft"
  >
): InvoiceMenuAction[] {
  const actions: InvoiceMenuAction[] = [];

  if (invoice.canReopenDraft === true) {
    actions.push("reopenDraft");
  }

  return actions;
}
