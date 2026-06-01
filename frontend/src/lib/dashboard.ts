import { MonthOverview, RecentPayments } from "../../wailsjs/go/main/App";

export type MonthOverviewDTO = {
  year: number;
  month: number;
  activeStudents: number;
  activeCourses: number;
  enrollments: number;
  perLessonEnrollments: number;
  attendanceFilled: number;
  attendanceMissing: number;
  draftInvoices: number;
  issuedInvoices: number;
  paidInvoices: number;
  totalIssued: number;
  totalPaid: number;
  debtorsCount: number;
  totalDebt: number;
};

export type RecentPaymentDTO = {
  id: number;
  studentId: number;
  studentName: string;
  invoiceId?: number;
  amount: number;
  method: string;
  paidAt: string;
  note: string;
};

export async function loadMonthOverview(year: number, month: number) {
  return (await MonthOverview(year, month)) as MonthOverviewDTO;
}

export async function loadRecentPayments(limit = 8) {
  return (await RecentPayments(limit)) as RecentPaymentDTO[];
}
