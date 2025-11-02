import {
  InvoiceGenerateDrafts,
  InvoiceListDrafts,
  InvoiceGet,
  InvoiceDeleteDraft,
  InvoiceIssue,
  InvoiceIssueAll
} from "../../wailsjs/go/main/App"; // или "@wailsjs/go/main/App"

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

// NEW: типы результатов
export type IssueResult = { number: string; pdfPath: string };
export type IssueAllResult = { count: number; pdfPaths: string[] };

export async function genDrafts(year: number, month: number) {
  return await InvoiceGenerateDrafts(year, month);
}
export async function listDrafts(year: number, month: number) {
  return (await InvoiceListDrafts(year, month)) as InvoiceListItem[];
}
export async function getInvoice(id: number) {
  return (await InvoiceGet(id)) as InvoiceDTO;
}
export async function deleteDraft(id: number) {
  return await InvoiceDeleteDraft(id);
}
export async function issueOne(id: number): Promise<IssueResult> {
  return (await InvoiceIssue(id)) as IssueResult;
}
export async function issueAll(year: number, month: number): Promise<IssueAllResult> {
  return (await InvoiceIssueAll(year, month)) as IssueAllResult;
}
