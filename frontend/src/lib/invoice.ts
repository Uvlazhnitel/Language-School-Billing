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
  const result = await InvoiceList(year, month, status);
  return result.map(item => ({
    ...item,
    status: item.status as "draft" | "issued" | "paid" | "canceled"
  }));
}

export async function getInvoice(id: number): Promise<InvoiceDTO> {
  const result = await InvoiceGet(id);
  return {
    ...result,
    status: result.status as "draft" | "issued" | "paid" | "canceled"
  };
}

export async function deleteDraft(id: number): Promise<void> {
  return InvoiceDeleteDraft(id);
}

export async function issueOne(id: number): Promise<IssueResult> {
  return InvoiceIssue(id);
}

export async function issueAll(year: number, month: number): Promise<IssueAllResult> {
  return InvoiceIssueAll(year, month);
}

export async function ensurePdf(id: number): Promise<string> {
  return InvoiceEnsurePDF(id);
}
