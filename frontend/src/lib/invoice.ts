import {
    InvoiceGenerateDrafts,
    InvoiceListDrafts,
    InvoiceGet,
    InvoiceDeleteDraft
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
  