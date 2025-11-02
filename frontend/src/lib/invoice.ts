import {
  InvoiceGenerateDrafts,
  InvoiceList,          // универсальный список по статусу
  InvoiceGet,
  InvoiceDeleteDraft,
  InvoiceIssue,
  InvoiceIssueAll
} from "../../wailsjs/go/main/App";

export type InvoiceListItem = {
  id: number;
  studentId: number;
  studentName: string;
  year: number;
  month: number;
  total: number;
  status: "draft" | "issued" | "paid" | "canceled";
  linesCount: number;
  number?: string;
};

export type InvoiceLine = {
  enrollmentId: number;
  description: string;
  qty: number;
  unitPrice: number;
  amount: number;
};

export type InvoiceDTO = {
  id: number;
  studentId: number;
  studentName: string;
  year: number;
  month: number;
  total: number;
  status: "draft" | "issued" | "paid" | "canceled";
  number?: string;
  lines: InvoiceLine[];
};

export type IssueResult = { number: string; pdfPath: string };
export type IssueAllResult = { count: number; pdfPaths: string[] };

export async function genDrafts(year: number, month: number): Promise<number> {
  return await InvoiceGenerateDrafts(year, month);
}

export async function listInvoices(
  year: number,
  month: number,
  status: "draft" | "issued" | "paid" | "canceled" | "all"
): Promise<InvoiceListItem[]> {
  return (await InvoiceList(year, month, status)) as any;
}

export async function getInvoice(id: number): Promise<InvoiceDTO> {
  return (await InvoiceGet(id)) as any;
}

export async function deleteDraft(id: number): Promise<void> {
  return await InvoiceDeleteDraft(id);
}

export async function issueOne(id: number): Promise<IssueResult> {
  return (await InvoiceIssue(id)) as any;
}

export async function issueAll(year: number, month: number): Promise<IssueAllResult> {
  return (await InvoiceIssueAll(year, month)) as any;
}
