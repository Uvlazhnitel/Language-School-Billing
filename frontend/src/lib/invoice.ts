import {
  InvoiceGenerateDrafts,
  InvoiceList,
  InvoiceGet,
  InvoiceDeleteDraft,
  InvoiceIssue,
  InvoiceIssueAll,
  InvoiceEnsurePDF
} from "../../wailsjs/go/main/App";

export type InvoiceListItem = {
  id: number; studentId: number; studentName: string;
  year: number; month: number; total: number;
  status: "draft" | "issued" | "paid" | "canceled";
  linesCount: number; number?: string;
};

export type InvoiceLine = {
  enrollmentId: number; description: string; qty: number; unitPrice: number; amount: number;
};

export type InvoiceDTO = {
  id: number; studentId: number; studentName: string;
  year: number; month: number; total: number;
  status: "draft" | "issued" | "paid" | "canceled";
  number?: string; lines: InvoiceLine[];
};

export type GenerateResult = {
  created: number; updated: number; skippedHasInvoice: number; skippedNoLines: number;
};

export type IssueResult = { number: string; pdfPath: string };
export type IssueAllResult = { count: number; pdfPaths: string[] };

export async function genDrafts(year: number, month: number): Promise<GenerateResult> {
  try {
    const result = await InvoiceGenerateDrafts(year, month);
    return result;
  } catch (error) {
    console.error("Error generating drafts:", error);
    throw error; // Re-throw the error after logging
  }
}

export async function listInvoices(
  year: number, month: number,
  status: "draft" | "issued" | "paid" | "canceled" | "all"
): Promise<InvoiceListItem[]> {
  return (await InvoiceList(year, month, status)) as any;
}
export async function getInvoice(id: number) { return (await InvoiceGet(id)) as any; }
export async function deleteDraft(id: number) { return await InvoiceDeleteDraft(id); }
export async function issueOne(id: number) { return (await InvoiceIssue(id)) as any; }
export async function issueAll(year: number, month: number) { return (await InvoiceIssueAll(year, month)) as any; }
export async function ensurePdf(id: number) { return (await InvoiceEnsurePDF(id)) as any; }
