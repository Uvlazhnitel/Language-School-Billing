import { PaymentMethod } from "./constants";
import { getTransport } from "./api";
export type { BalanceDTO, DebtInvoiceDTO, DebtorDTO, InvoiceSummaryDTO, PaymentDTO } from "./api";

export async function createPayment(
  studentId: number,
  invoiceId: number | undefined,
  amount: number,
  method: PaymentMethod,
  paidAt: string, // "YYYY-MM-DD"
  note: string
) {
  const transport = await getTransport();
  return transport.createPayment(studentId, invoiceId, amount, method, paidAt, note);
}

export async function deletePayment(paymentId: number) {
  const transport = await getTransport();
  return transport.deletePayment(paymentId);
}

export async function listDebtors() {
  const transport = await getTransport();
  return transport.listDebtors();
}

export async function invoiceSummary(invoiceId: number) {
  const transport = await getTransport();
  return transport.invoiceSummary(invoiceId);
}

export async function studentDebtDetails(studentId: number) {
  const transport = await getTransport();
  return transport.studentDebtDetails(studentId);
}

export async function studentBalance(studentId: number) {
  const transport = await getTransport();
  return transport.studentBalance(studentId);
}

export async function paymentListForStudent(studentId: number) {
  const transport = await getTransport();
  return transport.paymentListForStudent(studentId);
}
