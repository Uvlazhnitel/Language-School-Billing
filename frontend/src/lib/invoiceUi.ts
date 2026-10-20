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

export type InvoiceRowAction =
  | "open"
  | "issue"
  | "createPdf"
  | "recordPayment"
  | "downloadPdf"
  | "reopenDraft";

type InvoiceActionState = { status: InvoiceStatus; pdfReady?: boolean; canReopenDraft?: boolean };

export function getInvoiceRowActions(invoice: InvoiceActionState): {
  primary: InvoiceRowAction;
  menu: InvoiceRowAction[];
} {
  const issued = invoice.status === "issued" || invoice.status === "issued_pending_pdf";
  const paid = invoice.status === "paid" || invoice.status === "paid_pending_pdf";
  const primary: InvoiceRowAction =
    invoice.status === "draft"
      ? "issue"
      : issued || paid
        ? !invoice.pdfReady
          ? "createPdf"
          : issued
            ? "recordPayment"
            : "downloadPdf"
        : "open";
  const menu: InvoiceRowAction[] = primary === "open" ? [] : ["open"];
  if (issued) menu.push(invoice.pdfReady ? "downloadPdf" : "recordPayment");
  menu.push(...getInvoiceMenuActions(invoice));
  return { primary, menu };
}
