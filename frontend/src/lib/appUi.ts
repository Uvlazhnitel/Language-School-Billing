import type { Row } from "./attendance";
import type { DebtorDTO, DebtInvoiceDTO } from "./payments";
import type { TranslateFn } from "./i18n";

const monthsRu = [
  "Январь",
  "Февраль",
  "Март",
  "Апрель",
  "Май",
  "Июнь",
  "Июль",
  "Август",
  "Сентябрь",
  "Октябрь",
  "Ноябрь",
  "Декабрь",
];

const monthsLv = [
  "Janvāris",
  "Februāris",
  "Marts",
  "Aprīlis",
  "Maijs",
  "Jūnijs",
  "Jūlijs",
  "Augusts",
  "Septembris",
  "Oktobris",
  "Novembris",
  "Decembris",
];

export const payerRoleOptions = [
  "mother",
  "father",
  "grandmother",
  "grandfather",
  "guardian",
  "other",
] as const;

export type AppTabId =
  | "dashboard"
  | "students"
  | "courses"
  | "enrollments"
  | "attendance"
  | "invoice"
  | "debtors"
  | "audit"
  | "settings";

export function payerRoleLabel(relation: string, t: TranslateFn): string {
  switch (relation) {
    case "mother":
      return t("student.mother");
    case "father":
      return t("student.father");
    case "grandmother":
      return t("student.grandmother");
    case "grandfather":
      return t("student.grandfather");
    case "guardian":
      return t("student.guardian");
    default:
      return t("student.other");
  }
}

export function courseTypeLabel(type: string, t: TranslateFn): string {
  switch (type) {
    case "group":
      return t("course.group");
    case "individual":
      return t("course.individual");
    default:
      return type;
  }
}

export function billingModeLabel(mode: string, t: TranslateFn): string {
  switch (mode) {
    case "per_lesson":
      return t("billing.perLesson");
    case "subscription":
      return t("billing.subscription");
    default:
      return mode;
  }
}

export function paymentMethodLabel(method: string, t: TranslateFn): string {
  switch (method) {
    case "cash":
      return t("payment.cash");
    case "bank":
      return t("payment.bank");
    default:
      return method;
  }
}

export function invoiceStatusLabel(status: string, t: TranslateFn): string {
  switch (status) {
    case "draft":
      return t("status.draft");
    case "issued_pending_pdf":
      return t("status.issuedPendingPdf");
    case "issued":
      return t("status.issued");
    case "paid_pending_pdf":
      return t("status.paidPendingPdf");
    case "paid":
      return t("status.paid");
    case "canceled":
      return t("status.canceled");
    case "all":
      return t("status.all");
    default:
      return status;
  }
}

export function buildTabMeta(t: TranslateFn): Record<AppTabId, { eyebrow: string; title: string }> {
  return {
    dashboard: { eyebrow: t("eyebrow.dashboard"), title: t("title.dashboard") },
    students: { eyebrow: t("eyebrow.students"), title: t("title.students") },
    courses: { eyebrow: t("eyebrow.courses"), title: t("title.courses") },
    enrollments: { eyebrow: t("eyebrow.students"), title: t("button.manageEnrollments") },
    attendance: { eyebrow: t("eyebrow.attendance"), title: t("title.attendance") },
    invoice: { eyebrow: t("eyebrow.invoice"), title: t("title.invoice") },
    debtors: { eyebrow: t("eyebrow.debtors"), title: t("title.debtors") },
    audit: { eyebrow: t("eyebrow.audit"), title: t("title.audit") },
    settings: { eyebrow: t("eyebrow.settings"), title: t("title.settings") },
  };
}

export function numOrZero(s: string): number {
  if (s.trim() === "") return 0;
  const n = Number(s);
  return Number.isFinite(n) ? n : 0;
}

export function intOrUndef(s: string): number | undefined {
  if (s.trim() === "") return undefined;
  const n = Number(s);
  return Number.isFinite(n) ? Math.trunc(n) : undefined;
}

export function decimalOrZero(s: string): number {
  if (s.trim() === "") return 0;
  const n = Number(s);
  return Number.isFinite(n) ? n : 0;
}

export function normalizeMoneyInput(value: string): string | null {
  const normalized = value.replace(",", ".");
  if (normalized === "") return "";
  if (/^\d+(\.\d{0,2})?$/.test(normalized)) return normalized;
  return null;
}

export function formatEUR(value: number): string {
  return `€${value.toFixed(2)}`;
}

export async function copyTextToClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.top = "-9999px";
  textarea.style.left = "-9999px";
  document.body.appendChild(textarea);
  textarea.select();
  textarea.setSelectionRange(0, textarea.value.length);

  try {
    const copied = document.execCommand("copy");
    if (!copied) {
      throw new Error("Clipboard copy is unavailable");
    }
  } finally {
    document.body.removeChild(textarea);
  }
}

export function normalizeQuarterHours(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0;
  return Math.round(value * 4) / 4;
}

export function formatHoursValue(value: number): string {
  if (Math.abs(value - Math.round(value)) < 0.0001) {
    return String(Math.round(value));
  }
  return value.toFixed(2).replace(/\.?0+$/, "");
}

export function clampPct(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, value));
}

export function subscriptionTotal(row: Row, lessonsHeld: number): number {
  return Math.round(row.subscriptionLessonPrice * lessonsHeld * 100) / 100;
}

export function normalizeHoursDraftInput(value: string): string | null {
  const normalized = value.replace(",", ".");
  if (normalized === "") return "";
  if (/^\d*(\.\d{0,2})?$/.test(normalized)) return normalized;
  return null;
}

function debtMonthLabel(month: number, year: number, locale: "ru" | "lv"): string {
  const labels = locale === "ru" ? monthsRu : monthsLv;
  return `${labels[month - 1]} ${year}`;
}

export function buildDebtReminderMessage(
  locale: "ru" | "lv",
  debtor: DebtorDTO,
  details: DebtInvoiceDTO[],
  recipientName?: string
): string {
  const intro =
    locale === "ru"
      ? "Здравствуйте! Напоминаю об оплате за занятия."
      : "Sveiki! Atgādinu par apmaksu par nodarbībām.";

  const lines = details.map(
    (item) => `${debtMonthLabel(item.month, item.year, locale)}: ${formatEUR(item.remaining)}`
  );

  const totalLine =
    locale === "ru"
      ? `Итого к оплате: ${formatEUR(debtor.debt)}`
      : `Kopā apmaksai: ${formatEUR(debtor.debt)}`;

  const closing = locale === "ru" ? "Спасибо! ArtLab" : "Paldies! ArtLab";
  const recipientLine = recipientName?.trim()
    ? locale === "ru"
      ? `Получатель: ${recipientName.trim()}`
      : `Saņēmējs: ${recipientName.trim()}`
    : null;

  return [intro, recipientLine, recipientLine ? "" : null, ...lines, "", totalLine, "", closing]
    .filter((value): value is string => value !== null)
    .join("\n");
}
