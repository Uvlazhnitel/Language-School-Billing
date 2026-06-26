import type { IssueResult } from "./invoices";
import type { TranslateFn } from "./i18n";

export function buildIssueFeedback(
  result: IssueResult,
  t: TranslateFn
): { text: string; type: "success" | "warning" } {
  if (result.pdfReady) {
    return {
      text: t("msg.invoiceIssuedPdfReady", { number: result.number }),
      type: "success",
    };
  }

  return {
    text: t("msg.invoiceIssuedPdfPending", { number: result.number }),
    type: "warning",
  };
}
