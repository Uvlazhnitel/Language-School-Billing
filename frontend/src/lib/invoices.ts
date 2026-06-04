import { getTransport } from "./api";
export type {
  EnsurePdfResult,
  GenerateResult,
  InvoiceDTO,
  InvoiceListItem,
  InvoiceListItemView,
  IssueAllResult,
  IssueResult,
} from "./api";

export async function genDrafts(year: number, month: number) {
  const transport = await getTransport();
  return transport.generateDrafts(year, month);
}

export async function listInvoices(year: number, month: number, status: string) {
  const transport = await getTransport();
  return transport.listInvoices(year, month, status);
}

export async function getInvoice(id: number) {
  const transport = await getTransport();
  return transport.getInvoice(id);
}

export async function deleteDraft(id: number) {
  const transport = await getTransport();
  return transport.deleteDraft(id);
}

export async function reopenToDraft(id: number) {
  const transport = await getTransport();
  return transport.reopenToDraft(id);
}

export async function issueOne(id: number) {
  const transport = await getTransport();
  return transport.issueInvoice(id);
}

export async function rebuildStudentDraft(studentId: number, year: number, month: number) {
  const transport = await getTransport();
  return transport.rebuildStudentDraft(studentId, year, month);
}

export async function ensurePdf(invoiceId: number) {
  const transport = await getTransport();
  return transport.ensurePdf(invoiceId);
}

export async function hasPdf(invoiceId: number) {
  const transport = await getTransport();
  return transport.hasPdf(invoiceId);
}
