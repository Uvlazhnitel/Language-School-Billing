import { DashboardOverview } from "../components/DashboardOverview";
import type { MonthOverviewDTO, RecentPaymentDTO } from "../lib/dashboard";
import type { TranslateFn } from "../lib/i18n";

type DashboardScreenProps = {
  overview: MonthOverviewDTO | null;
  loading: boolean;
  monthLabel: string;
  recentPayments: RecentPaymentDTO[];
  formatEUR: (value: number) => string;
  paymentMethodLabel: (method: string) => string;
  onOpenAttendance: () => void;
  onOpenInvoices: () => void;
  onOpenDebtors: () => void;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  t: TranslateFn;
};

export function DashboardScreen(props: DashboardScreenProps) {
  return <DashboardOverview {...props} />;
}
