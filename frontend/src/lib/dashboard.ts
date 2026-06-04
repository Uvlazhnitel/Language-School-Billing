import { getTransport } from "./api";
export type { MonthOverviewDTO, RecentPaymentDTO } from "./api";

export async function loadMonthOverview(year: number, month: number) {
  const transport = await getTransport();
  return transport.loadMonthOverview(year, month);
}

export async function loadRecentPayments(limit = 8) {
  const transport = await getTransport();
  return transport.loadRecentPayments(limit);
}
