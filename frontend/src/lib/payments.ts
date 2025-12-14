import {
    PaymentCreate,
    PaymentDelete,
    PaymentListForStudent,
    StudentBalance,
    DebtorsList,
    InvoicePaymentSummary,
    PaymentQuickCash
    } from "../../wailsjs/go/main/App";
  
  export type PaymentDTO = {
    id: number;
    studentId: number;
    invoiceId?: number;
    paidAt: string;     // RFC3339
    amount: number;
    method: "cash" | "bank";
    note: string;
    createdAt: string;  // RFC3339
  };
  
  export type BalanceDTO = {
    studentId: number;
    studentName: string;
    totalInvoiced: number;
    totalPaid: number;
    balance: number; // paid - invoiced
    debt: number;
  };
  
  export type DebtorDTO = {
    studentId: number;
    studentName: string;
    debt: number;
    totalInvoiced: number;
    totalPaid: number;
  };
  
  export type InvoiceSummaryDTO = {
    invoiceId: number;
    total: number;
    paid: number;
    remaining: number;
    status: string;
    number?: string;
  };
  
  export async function createPayment(
    studentId: number,
    invoiceId: number | undefined,
    amount: number,
    method: "cash" | "bank",
    paidAt: string, // "YYYY-MM-DD"
    note: string
  ) {
    const inv = invoiceId ? invoiceId : undefined;
    return (await PaymentCreate(studentId, inv, amount, method, paidAt, note)) as PaymentDTO;
  }
  
  export async function deletePayment(id: number) {
    return await PaymentDelete(id);
  }
  
  export async function listPayments(studentId: number) {
    return (await PaymentListForStudent(studentId)) as PaymentDTO[];
  }
  
  export async function studentBalance(studentId: number) {
    return (await StudentBalance(studentId)) as BalanceDTO;
  }
  
  export async function listDebtors() {
    return (await DebtorsList()) as DebtorDTO[];
  }
  
  export async function invoiceSummary(invoiceId: number) {
    return (await InvoicePaymentSummary(invoiceId)) as InvoiceSummaryDTO;
  }
  
  export async function quickCash(studentId: number, amount: number, note: string) {
    return (await PaymentQuickCash(studentId, amount, note)) as PaymentDTO;
  }
  