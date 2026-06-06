import { DashboardOverview } from "../components/DashboardOverview";
import type { MonthOverviewDTO, RecentPaymentDTO } from "../lib/dashboard";
import type { DebtorActionQueueItem } from "../lib/studentActivity";
import type { TranslateFn } from "../lib/i18n";

type DashboardScreenProps = {
  overview: MonthOverviewDTO | null;
  loading: boolean;
  monthLabel: string;
  recentPayments: RecentPaymentDTO[];
  actionQueue: DebtorActionQueueItem[];
  formatEUR: (value: number) => string;
  paymentMethodLabel: (method: string) => string;
  onOpenAttendance: () => void;
  onOpenInvoices: () => void;
  onOpenDebtors: () => void;
  onOpenStudents: () => void;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onOpenPaymentQueueStudent: (studentId: number) => void;
  onCopyDebtQueueRu: (studentId: number) => void | Promise<void>;
  onCopyDebtQueueLv: (studentId: number) => void | Promise<void>;
  t: TranslateFn;
};

export function DashboardScreen(props: DashboardScreenProps) {
  return <DashboardOverview {...props} />;
}
