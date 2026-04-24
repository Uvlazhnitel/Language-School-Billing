import { MonthOverview } from "../../wailsjs/go/main/App";

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

export async function getMonthOverview(year: number, month: number): Promise<MonthOverviewDTO> {
  return (await MonthOverview(year, month)) as MonthOverviewDTO;
}
