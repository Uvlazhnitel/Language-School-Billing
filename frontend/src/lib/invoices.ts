import {
    InvoiceGenerateDrafts,
    InvoiceList,
    InvoiceGet,
    InvoiceDeleteDraft,
    InvoiceIssue,
    InvoiceIssueAll,
    InvoiceEnsurePDF,
    OpenFile,
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
  
  export type GenerateResult = {
    created: number;
    updated: number;
    skippedHasInvoice: number;
    skippedNoLines: number;
  };
  
  export type IssueResult = { number: string; pdfPath: string };
  export type IssueAllResult = { count: number; pdfPaths: string[] };
  
  export async function genDrafts(year: number, month: number) {
    return (await InvoiceGenerateDrafts(year, month)) as GenerateResult;
  }
  
  export async function listInvoices(year: number, month: number, status: string) {
    return (await InvoiceList(year, month, status)) as InvoiceListItem[];
  }
  
  export async function getInvoice(id: number) {
    return (await InvoiceGet(id)) as InvoiceDTO;
  }
  
  export async function deleteDraft(id: number) {
    return await InvoiceDeleteDraft(id);
  }
  
  export async function issueOne(id: number) {
    return (await InvoiceIssue(id)) as IssueResult;
  }
  
  export async function issueAll(year: number, month: number) {
    return (await InvoiceIssueAll(year, month)) as IssueAllResult;
  }
  
  export async function ensurePdfAndOpen(invoiceId: number) {
    const path = await InvoiceEnsurePDF(invoiceId);
    await OpenFile(path);
    return path;
  }
  