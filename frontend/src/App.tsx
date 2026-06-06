import { Fragment, useEffect, useLayoutEffect, useMemo, useState, useCallback, useRef, type FormEvent } from "react";
import "./App.css";

import {
  fetchRows,
  saveHours,
  deleteEnrollment,
  listCourseMonthSubscriptions,
  saveCourseMonthSubscriptionLessons,
  Row,
} from "./lib/attendance";

import {
  listInvoices,
  getInvoice,
  genDrafts,
  issueOne,
  reopenToDraft,
  rebuildStudentDraft,
  ensurePdf,
  hasPdf,
  InvoiceListItemView,
  InvoiceDTO,
} from "./lib/invoices";

import {
  listStudents,
  getStudent,
  createStudent,
  updateStudent,
  setStudentActive,
  deleteStudent,
  StudentDTO,
} from "./lib/students";

import { listCourses, createCourse, updateCourse, deleteCourse, CourseDTO } from "./lib/courses";
import { listTeachers, createTeacher, TeacherDTO } from "./lib/teachers";
import {
  BillingModePerLesson,
  BillingModeSubscription,
  InvoiceStatusCanceled,
  InvoiceStatusIssued,
  InvoiceStatusPaid,
} from "./lib/constants";

import {
  listEnrollments,
  createEnrollment,
  updateEnrollment,
  EnrollmentDTO,
} from "./lib/enrollments";

import {
  listDebtors,
  DebtorDTO,
  createPayment,
  deletePayment,
  invoiceSummary,
  InvoiceSummaryDTO,
  studentDebtDetails,
  DebtInvoiceDTO,
  studentBalance,
  BalanceDTO,
  paymentListForStudent,
  PaymentDTO,
} from "./lib/payments";
import {
  loadMonthOverview,
  loadRecentPayments,
  MonthOverviewDTO,
  RecentPaymentDTO,
} from "./lib/dashboard";
import { AuditLogItem, listAuditLogs } from "./lib/audit";
import { getTransport, type TransportCapabilities, type UserDTO } from "./lib/api";
import { AUTH_REQUIRED_EVENT } from "./lib/api/shared";
import { DashboardOverview } from "./components/DashboardOverview";
import { LoginScreen } from "./components/LoginScreen";
import { StudentWorkspace } from "./components/StudentWorkspace";
import { StudentDetailPanel } from "./components/StudentDetailPanel";
import {
  buildDebtorActionQueue,
  buildStudentActivity,
  buildStudentNextAction,
  DebtorActionQueueItem,
  StudentActivityItem,
  StudentNextAction,
} from "./lib/studentActivity";
import { canShowInvoiceFolderAction, canShowSettingsFilesCard } from "./lib/uiCapabilities";
import { createTranslator, getMonthNames, normalizeLocale, TranslateFn, UiLocale } from "./lib/i18n";

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

const payerRoleOptions = [
  "mother",
  "father",
  "grandmother",
  "grandfather",
  "guardian",
  "other",
] as const;

function payerRoleLabel(relation: string, t: TranslateFn): string {
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

function courseTypeLabel(type: string, t: TranslateFn): string {
  switch (type) {
    case "group":
      return t("course.group");
    case "individual":
      return t("course.individual");
    default:
      return type;
  }
}

function billingModeLabel(mode: string, t: TranslateFn): string {
  switch (mode) {
    case "per_lesson":
      return t("billing.perLesson");
    case "subscription":
      return t("billing.subscription");
    default:
      return mode;
  }
}

function paymentMethodLabel(method: string, t: TranslateFn): string {
  switch (method) {
    case "cash":
      return t("payment.cash");
    case "bank":
      return t("payment.bank");
    default:
      return method;
  }
}

function invoiceStatusLabel(status: string, t: TranslateFn): string {
  switch (status) {
    case "draft":
      return t("status.draft");
    case "issued":
      return t("status.issued");
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

function auditActionLabel(action: string): string {
  return action.replaceAll("_", " ").replaceAll(".", " ");
}

type Tab =
  | "dashboard"
  | "students"
  | "courses"
  | "enrollments"
  | "attendance"
  | "invoice"
  | "debtors"
  | "audit"
  | "settings";
type InvoiceMenuTarget = { kind: "row" | "modal"; invoiceId: number };
type InvoiceMenuPosition = { top: number; left: number; openUpward: boolean };
type UserDraft = { username: string; role: string; isActive: boolean };

function buildTabMeta(t: TranslateFn): Record<Tab, { eyebrow: string; title: string }> {
  return {
    dashboard: {
      eyebrow: t("eyebrow.dashboard"),
      title: t("title.dashboard"),
    },
    students: {
      eyebrow: t("eyebrow.students"),
      title: t("title.students"),
    },
    courses: {
      eyebrow: t("eyebrow.courses"),
      title: t("title.courses"),
    },
    enrollments: {
      eyebrow: t("eyebrow.students"),
      title: t("button.manageEnrollments"),
    },
    attendance: {
      eyebrow: t("eyebrow.attendance"),
      title: t("title.attendance"),
    },
    invoice: {
      eyebrow: t("eyebrow.invoice"),
      title: t("title.invoice"),
    },
    debtors: {
      eyebrow: t("eyebrow.debtors"),
      title: t("title.debtors"),
    },
    audit: {
      eyebrow: t("eyebrow.audit"),
      title: t("title.audit"),
    },
    settings: {
      eyebrow: t("eyebrow.settings"),
      title: t("title.settings"),
    },
  };
}

function numOrZero(s: string): number {
  if (s.trim() === "") return 0;
  const n = Number(s);
  return Number.isFinite(n) ? n : 0;
}

function intOrUndef(s: string): number | undefined {
  if (s.trim() === "") return undefined;
  const n = Number(s);
  return Number.isFinite(n) ? Math.trunc(n) : undefined;
}

function decimalOrZero(s: string): number {
  if (s.trim() === "") return 0;
  const n = Number(s);
  return Number.isFinite(n) ? n : 0;
}

function normalizeMoneyInput(value: string): string | null {
  const normalized = value.replace(",", ".");
  if (normalized === "") return "";
  if (/^\d+(\.\d{0,2})?$/.test(normalized)) return normalized;
  return null;
}

function formatEUR(value: number): string {
  return `€${value.toFixed(2)}`;
}

async function copyTextToClipboard(text: string) {
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

function normalizeQuarterHours(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0;
  return Math.round(value * 4) / 4;
}

function formatHoursValue(value: number): string {
  if (Math.abs(value - Math.round(value)) < 0.0001) {
    return String(Math.round(value));
  }
  return value.toFixed(2).replace(/\.?0+$/, "");
}

function clampPct(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, value));
}

function subscriptionTotal(row: Row, lessonsHeld: number): number {
  const totalDiscountPct = clampPct(row.discountPct + row.subscriptionDiscountPct);
  const base = row.lessonPrice * lessonsHeld;
  return Math.round(base * (1 - totalDiscountPct / 100) * 100) / 100;
}

function normalizeHoursDraftInput(value: string): string | null {
  const normalized = value.replace(",", ".");
  if (normalized === "") return "";
  if (/^\d*(\.\d{0,2})?$/.test(normalized)) return normalized;
  return null;
}

function debtMonthLabel(month: number, year: number, locale: "ru" | "lv"): string {
  const labels = locale === "ru" ? monthsRu : monthsLv;
  return `${labels[month - 1]} ${year}`;
}

function buildDebtReminderMessage(
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

export default function App() {
  const now = new Date();
  const [transportCapabilities, setTransportCapabilities] = useState<TransportCapabilities>({
    isDesktop: false,
    canOpenLocalFiles: false,
    canOpenFolders: false,
    canDownloadPdf: true,
  });
  const [tab, setTab] = useState<Tab>("dashboard");
  const [appReady, setAppReady] = useState(false);
  const [authLoading, setAuthLoading] = useState(true);
  const [authRequired, setAuthRequired] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(true);
  const [currentSessionUser, setCurrentSessionUser] = useState<{ id: number; username: string; role: string } | null>(null);
  const [sessionCapabilities, setSessionCapabilities] = useState<Record<string, boolean>>({});
  const [loginUsername, setLoginUsername] = useState("");
  const [loginPassword, setLoginPassword] = useState("");
  const [loginRememberMe, setLoginRememberMe] = useState(true);
  const [loginPending, setLoginPending] = useState(false);
  const [loginError, setLoginError] = useState<string | null>(null);
  const [sessionExpired, setSessionExpired] = useState(false);
  const [uiLocale, setUiLocale] = useState<UiLocale>("en-US");
  const [appDirs, setAppDirs] = useState<Record<string, string> | null>(null);
  const [creatingBackup, setCreatingBackup] = useState(false);
  const [users, setUsers] = useState<UserDTO[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [creatingUser, setCreatingUser] = useState(false);
  const [newUserUsername, setNewUserUsername] = useState("");
  const [newUserPassword, setNewUserPassword] = useState("");
  const [newUserRole, setNewUserRole] = useState("staff");
  const [userDrafts, setUserDrafts] = useState<Record<number, UserDraft>>({});
  const [userPasswordDrafts, setUserPasswordDrafts] = useState<Record<number, string>>({});

  // Global message display
  const [message, setMessage] = useState<{ text: string; type: "success" | "error" } | null>(null);
  const messageTimeoutRef = useRef<number | null>(null);

  // Global confirmation dialog
  const [confirmDialog, setConfirmDialog] = useState<{
    isOpen: boolean;
    message: string;
    onConfirm: () => void | Promise<void>;
    confirmButtonLabel?: string;
  } | null>(null);

  const showConfirm = useCallback((
    messageText: string,
    onConfirm: () => void | Promise<void>,
    confirmButtonLabel?: string
  ) => {
    setConfirmDialog({ isOpen: true, message: messageText, onConfirm, confirmButtonLabel });
  }, []);

  const handleConfirmYes = async () => {
    try {
      if (confirmDialog?.onConfirm) {
        await confirmDialog.onConfirm();
      }
    } finally {
      setConfirmDialog(null);
    }
  };

  const handleConfirmNo = () => {
    setConfirmDialog(null);
  };

  const showMessage = useCallback((text: string, type: "success" | "error" = "success") => {
    console.log(`[${type.toUpperCase()}] ${text}`);

    // Clear any existing timeout
    if (messageTimeoutRef.current) {
      clearTimeout(messageTimeoutRef.current);
      messageTimeoutRef.current = null;
    }

    setMessage({ text, type });

    // Auto-dismiss success messages after 5 seconds
    if (type === "success") {
      messageTimeoutRef.current = window.setTimeout(() => {
        setMessage(null);
        messageTimeoutRef.current = null;
      }, 5000);
    }
  }, []);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (messageTimeoutRef.current) {
        clearTimeout(messageTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    const bootstrap = async () => {
      try {
        const transport = await getTransport();
        const bootstrapResult = await transport.bootstrap();
        if (cancelled) return;
        setAppDirs(bootstrapResult.appDirs);
        setTransportCapabilities(bootstrapResult.capabilities);
        setUiLocale(normalizeLocale(bootstrapResult.locale));
        setAuthRequired(bootstrapResult.authRequired);
        setIsAuthenticated(bootstrapResult.session.authenticated);
        setCurrentSessionUser(bootstrapResult.session.user ?? null);
        setSessionCapabilities(bootstrapResult.session.capabilities ?? {});
        setAppReady(bootstrapResult.ready && (!bootstrapResult.authRequired || bootstrapResult.session.authenticated));
        setAuthLoading(false);
      } catch (e: any) {
        if (!cancelled) {
          setAuthLoading(false);
          showMessage(
            createTranslator("en-US")("msg.loadingFoldersError", {
              message: String(e?.message ?? e),
            }),
            "error"
          );
        }
      }
    };

    void bootstrap();

    return () => {
      cancelled = true;
    };
  }, [showMessage]);

  useEffect(() => {
    const onAuthRequired = () => {
      setIsAuthenticated(false);
      setCurrentSessionUser(null);
      setSessionCapabilities({});
      setAppReady(false);
      setLoginPassword("");
      setLoginError(null);
      setSessionExpired(true);
    };

    window.addEventListener(AUTH_REQUIRED_EVENT, onAuthRequired);
    return () => {
      window.removeEventListener(AUTH_REQUIRED_EVENT, onAuthRequired);
    };
  }, []);

  const t = useMemo(() => createTranslator(uiLocale), [uiLocale]);
  const uiMonths = useMemo(() => getMonthNames(uiLocale), [uiLocale]);
  const tabMeta = useMemo(() => buildTabMeta(t), [t]);
  const canManageUsers = Boolean(sessionCapabilities.manageUsers) || transportCapabilities.isDesktop;
  const canManageSettings = Boolean(sessionCapabilities.manageSettings) || transportCapabilities.isDesktop;
  const canCreateBackups = Boolean(sessionCapabilities.backups) || transportCapabilities.isDesktop;
  const canDeleteStudents = Boolean(sessionCapabilities.deleteStudents) || transportCapabilities.isDesktop;
  const canDeleteCourses = Boolean(sessionCapabilities.deleteCourses) || transportCapabilities.isDesktop;
  const canDeletePayments = Boolean(sessionCapabilities.deletePayments) || transportCapabilities.isDesktop;
  const canViewAuditLog = Boolean(sessionCapabilities.viewAuditLog) || transportCapabilities.isDesktop;

  const localizedPayerRoleLabel = useCallback(
    (relation: string) => payerRoleLabel(relation, t),
    [t]
  );
  const localizedCourseTypeLabel = useCallback((type: string) => courseTypeLabel(type, t), [t]);
  const localizedBillingModeLabel = useCallback(
    (mode: string) => billingModeLabel(mode, t),
    [t]
  );
  const localizedPaymentMethodLabel = useCallback(
    (method: string) => paymentMethodLabel(method, t),
    [t]
  );
  const localizedInvoiceStatusLabel = useCallback(
    (status: string) => invoiceStatusLabel(status, t),
    [t]
  );

  // Shared month/year for Attendance + Invoices
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);
  const currentMeta = tabMeta[tab];
  const currentMonthLabel = `${uiMonths[month - 1]} ${year}`;
  const [overview, setOverview] = useState<MonthOverviewDTO | null>(null);
  const [overviewLoading, setOverviewLoading] = useState(false);
  const [recentPayments, setRecentPayments] = useState<RecentPaymentDTO[]>([]);
  const [auditItems, setAuditItems] = useState<AuditLogItem[]>([]);
  const [auditLoading, setAuditLoading] = useState(false);
  const [auditQ, setAuditQ] = useState("");
  const [auditActorFilter, setAuditActorFilter] = useState("");
  const [auditEntityTypeFilter, setAuditEntityTypeFilter] = useState("");
  const [auditActionFilter, setAuditActionFilter] = useState("");
  const [auditDateFrom, setAuditDateFrom] = useState("");
  const [auditDateTo, setAuditDateTo] = useState("");
  const [auditPage, setAuditPage] = useState(1);
  const [auditPageSize] = useState(50);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditExpandedId, setAuditExpandedId] = useState<number | null>(null);

  // ---------------- Students ----------------
  const [studentList, setStudentList] = useState<StudentDTO[]>([]);
  const [allStudents, setAllStudents] = useState<StudentDTO[]>([]);
  const [studentQ, setStudentQ] = useState("");
  const [includeInactive, setIncludeInactive] = useState(false);
  const [studentLoading, setStudentLoading] = useState(false);

  const [studentModalOpen, setStudentModalOpen] = useState(false);
  const [editingStudent, setEditingStudent] = useState<StudentDTO | null>(null);
  const [sfName, setSfName] = useState("");
  const [sfPersonalCode, setSfPersonalCode] = useState("");
  const [sfPhone, setSfPhone] = useState("");
  const [sfEmail, setSfEmail] = useState("");
  const [sfNote, setSfNote] = useState("");
  const [sfIsMinor, setSfIsMinor] = useState(false);
  const [sfPayerName, setSfPayerName] = useState("");
  const [sfPayerRole, setSfPayerRole] = useState("");

  // ---------------- Student Card ----------------
  const [studentCardOpen, setStudentCardOpen] = useState(false);
  const [selectedStudentCard, setSelectedStudentCard] = useState<StudentDTO | null>(null);
  const [studentCardLoading, setStudentCardLoading] = useState(false);
  const [studentCardEnrollments, setStudentCardEnrollments] = useState<EnrollmentDTO[]>([]);
  const [studentCardBalance, setStudentCardBalance] = useState<BalanceDTO | null>(null);
  const [studentCardDebts, setStudentCardDebts] = useState<DebtInvoiceDTO[]>([]);
  const [studentCardPayments, setStudentCardPayments] = useState<PaymentDTO[]>([]);
  const [studentCardMonthInvoices, setStudentCardMonthInvoices] = useState<InvoiceListItemView[]>(
    []
  );
  const [studentCardDeletingPaymentId, setStudentCardDeletingPaymentId] = useState<number | null>(
    null
  );
  const [studentNextAction, setStudentNextAction] = useState<StudentNextAction | null>(null);
  const [studentActivity, setStudentActivity] = useState<StudentActivityItem[]>([]);
  const [debtorActionQueue, setDebtorActionQueue] = useState<DebtorActionQueueItem[]>([]);

  const loadStudents = useCallback(async () => {
    setStudentLoading(true);
    try {
      const data = await listStudents(studentQ, includeInactive);
      setStudentList(data);
    } finally {
      setStudentLoading(false);
    }
  }, [studentQ, includeInactive]);

  const loadAllStudents = useCallback(async () => {
    const data = await listStudents("", true);
    setAllStudents(data);
    return data;
  }, []);

  useEffect(() => {
    if (!appReady) return;
    if (tab === "students") loadStudents();
  }, [appReady, tab, loadStudents]);

  useEffect(() => {
    if (!appReady) return;
    void loadAllStudents();
  }, [appReady, loadAllStudents]);

  const loadDashboard = useCallback(async () => {
    setOverviewLoading(true);
    try {
      const [snapshot, payments, debtorsSnapshot] = await Promise.all([
        loadMonthOverview(year, month),
        loadRecentPayments(8),
        listDebtors(),
      ]);
      setOverview(snapshot);
      setRecentPayments(payments);
      setDebtors(debtorsSnapshot);
      setDebtorActionQueue(buildDebtorActionQueue(debtorsSnapshot, payments, t));
    } catch (e: any) {
      showMessage(t("msg.dashboardLoadError", { message: String(e?.message ?? e) }), "error");
    } finally {
      setOverviewLoading(false);
    }
  }, [month, showMessage, t, year]);

  useEffect(() => {
    if (!appReady) return;
    if (tab === "dashboard") {
      void loadDashboard();
    }
  }, [appReady, loadDashboard, tab]);

  const loadAuditLog = useCallback(async () => {
    if (!canViewAuditLog) return;
    setAuditLoading(true);
    try {
      const result = await listAuditLogs({
        q: auditQ,
        actorLabel: auditActorFilter,
        entityType: auditEntityTypeFilter,
        action: auditActionFilter,
        dateFrom: auditDateFrom,
        dateTo: auditDateTo,
        page: auditPage,
        pageSize: auditPageSize,
      });
      setAuditItems(result.items);
      setAuditTotal(result.total);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setAuditLoading(false);
    }
  }, [
    canViewAuditLog,
    auditQ,
    auditActorFilter,
    auditEntityTypeFilter,
    auditActionFilter,
    auditDateFrom,
    auditDateTo,
    auditPage,
    auditPageSize,
    showMessage,
    t,
  ]);

  useEffect(() => {
    if (!appReady || tab !== "audit") return;
    void loadAuditLog();
  }, [appReady, tab, loadAuditLog]);

  function openAddStudent() {
    setEditingStudent(null);
    setSfName("");
    setSfPersonalCode("");
    setSfPhone("");
    setSfEmail("");
    setSfNote("");
    setSfIsMinor(false);
    setSfPayerName("");
    setSfPayerRole("");
    setStudentModalOpen(true);
  }

  function openEditStudent(s: StudentDTO) {
    setEditingStudent(s);
    setSfName(s.fullName);
    setSfPersonalCode(s.personalCode ?? "");
    setSfPhone(s.phone);
    setSfEmail(s.email);
    setSfNote(s.note);
    setSfIsMinor(s.isMinor);
    setSfPayerName(s.payerName ?? "");
    setSfPayerRole(s.payerRole ?? "");
    setStudentModalOpen(true);
  }

  async function saveStudent() {
    if (!sfName.trim()) {
      showMessage(t("msg.studentNameRequired"), "error");
      return;
    }
    if (sfIsMinor && !sfPayerName.trim()) {
      showMessage(t("msg.studentPayerRequired"), "error");
      return;
    }
    if (sfIsMinor && !sfPayerRole) {
      showMessage(t("msg.studentPayerRoleRequired"), "error");
      return;
    }
    try {
      if (editingStudent) {
        // Update existing student
        await updateStudent(
          editingStudent.id,
          sfName,
          sfPersonalCode,
          sfPhone,
          sfEmail,
          sfNote,
          sfIsMinor,
          sfPayerName,
          sfPayerRole
        );
      } else {
        // Create new student
        await createStudent(
          sfName,
          sfPersonalCode,
          sfPhone,
          sfEmail,
          sfNote,
          sfIsMinor,
          sfPayerName,
          sfPayerRole
        );
      }
      setStudentModalOpen(false);
      await Promise.all([loadStudents(), loadAllStudents()]);
      showMessage(editingStudent ? t("msg.studentUpdated") : t("msg.studentCreated"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function toggleStudentActive(s: StudentDTO) {
    try {
      await setStudentActive(s.id, !s.isActive);
      await Promise.all([loadStudents(), loadAllStudents()]);
      showMessage(s.isActive ? t("msg.studentDeactivated") : t("msg.studentActivated"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function removeStudent(id: number) {
    showConfirm(
      t("msg.studentDeleteConfirm"),
      async () => {
        try {
          await deleteStudent(id);
          await Promise.all([loadStudents(), loadAllStudents()]);
          showMessage(t("msg.studentDeleted"));
        } catch (e: any) {
          showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
        }
      }
    );
  }

  const refreshStudentCardData = useCallback(async (studentId: number) => {
    try {
      const [enr, bal, debts, payments, monthInvoices] = await Promise.all([
        listEnrollments(studentId, undefined),
        studentBalance(studentId),
        studentDebtDetails(studentId),
        paymentListForStudent(studentId),
        listInvoices(year, month, "all"),
      ]);
      const studentMonthInvoices = monthInvoices.filter(
        (invoice) => invoice.studentId === studentId
      );
      setStudentCardEnrollments(enr);
      setStudentCardBalance(bal);
      setStudentCardDebts(debts);
      setStudentCardPayments(payments);
      setStudentCardMonthInvoices(studentMonthInvoices);
      setStudentNextAction(
        buildStudentNextAction({
          debt: bal?.debt ?? 0,
          enrollments: enr,
          debts,
          payments,
          monthInvoices: studentMonthInvoices,
          t,
        })
      );
      setStudentActivity(
        buildStudentActivity({
          enrollments: enr,
          payments,
          debts,
          monthInvoices: studentMonthInvoices,
          months: uiMonths,
          t,
          paymentMethodLabel: localizedPaymentMethodLabel,
          billingModeLabel: localizedBillingModeLabel,
        })
      );
    } catch (e: any) {
      showMessage(t("msg.studentCardLoadError", { message: String(e?.message ?? e) }), "error");
    }
  }, [
    localizedBillingModeLabel,
    localizedPaymentMethodLabel,
    month,
    showMessage,
    t,
    uiMonths,
    year,
  ]);

  const openStudentCard = useCallback(async (s: StudentDTO, options?: { inline?: boolean }) => {
    setSelectedStudentCard(s);
    setStudentCardOpen(!(options?.inline || tab === "students"));
    setStudentCardLoading(true);
    setStudentCardEnrollments([]);
    setStudentCardBalance(null);
    setStudentCardDebts([]);
    setStudentCardPayments([]);
    setStudentCardMonthInvoices([]);
    setStudentNextAction(null);
    setStudentActivity([]);
    try {
      await refreshStudentCardData(s.id);
    } finally {
      setStudentCardLoading(false);
    }
  }, [refreshStudentCardData, tab]);

  useEffect(() => {
    if (tab !== "students" || studentLoading || studentList.length === 0) return;
    if (
      !selectedStudentCard ||
      !studentList.some((student) => student.id === selectedStudentCard.id)
    ) {
      void openStudentCard(studentList[0], { inline: true });
    }
  }, [openStudentCard, tab, studentLoading, studentList, selectedStudentCard]);

  async function openStudentCardById(studentId: number) {
    const existing = allStudents.find((s) => s.id === studentId);
    try {
      const student = existing ?? (await getStudent(studentId));
      if (tab !== "students") {
        setStudentCardOpen(true);
      }
      await openStudentCard(student);
    } catch (e: any) {
      showMessage(t("msg.studentCardLoadError", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function openStudentInWorkspaceById(studentId: number) {
    const existing = allStudents.find((s) => s.id === studentId);
    try {
      const student = existing ?? (await getStudent(studentId));
      setTab("students");
      await openStudentCard(student, { inline: true });
    } catch (e: any) {
      showMessage(`Ошибка загрузки карточки ученика: ${String(e?.message ?? e)}`, "error");
    }
  }

  useEffect(() => {
    if (!selectedStudentCard) return;
    if (tab !== "students" && !studentCardOpen) return;
    void refreshStudentCardData(selectedStudentCard.id);
  }, [month, refreshStudentCardData, selectedStudentCard, studentCardOpen, tab, year]);

  async function resolveDebtReminderRecipient(studentId: number, studentName: string) {
    const student =
      selectedStudentCard?.id === studentId ? selectedStudentCard : await getStudent(studentId);
    if (!student.isMinor) return student.fullName;
    return student.payerName?.trim() || studentName;
  }

  async function copyStudentCardDebtMessage(locale: "ru" | "lv") {
    if (!selectedStudentCard || studentCardDebts.length === 0 || !studentCardBalance) return;
    try {
      const debtorLike: DebtorDTO = {
        studentId: selectedStudentCard.id,
        studentName: selectedStudentCard.fullName,
        debt: studentCardBalance.debt,
        totalInvoiced: studentCardBalance.totalInvoiced,
        totalPaid: studentCardBalance.totalPaid,
      };
      const recipientName = await resolveDebtReminderRecipient(
        selectedStudentCard.id,
        selectedStudentCard.fullName
      );
      const text = buildDebtReminderMessage(locale, debtorLike, studentCardDebts, recipientName);
      await copyTextToClipboard(text);
      showMessage(locale === "ru" ? t("msg.debtReminderRuCopied") : t("msg.debtReminderLvCopied"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  function deleteStudentPayment(payment: PaymentDTO) {
    if (!selectedStudentCard) return;

    const amountLabel = formatEUR(payment.amount);
    const dateLabel = payment.paidAt.slice(0, 10);

    showConfirm(
      t("msg.paymentDeleteConfirm", { amount: amountLabel, date: dateLabel }),
      async () => {
        try {
          setStudentCardDeletingPaymentId(payment.id);
          await deletePayment(payment.id);
          await Promise.all([refreshStudentCardData(selectedStudentCard.id), loadDebtors()]);
          showMessage(t("msg.paymentDeleted"));
        } catch (e: any) {
          showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
        } finally {
          setStudentCardDeletingPaymentId(null);
        }
      }
    );
  }

  // ---------------- Courses ----------------
  const [courseList, setCourseList] = useState<CourseDTO[]>([]);
  const [allCourses, setAllCourses] = useState<CourseDTO[]>([]);
  const [courseQ, setCourseQ] = useState("");
  const [courseLoading, setCourseLoading] = useState(false);

  const [courseModalOpen, setCourseModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState<CourseDTO | null>(null);
  const [cfName, setCfName] = useState("");
  const [cfTeacherId, setCfTeacherId] = useState<number | undefined>(undefined);
  const [cfTeacherSearch, setCfTeacherSearch] = useState("");
  const [cfTeacherPickerOpen, setCfTeacherPickerOpen] = useState(false);
  const [cfTeacherCreating, setCfTeacherCreating] = useState(false);
  const [allTeachers, setAllTeachers] = useState<TeacherDTO[]>([]);
  const cfTeacherComboRef = useRef<HTMLDivElement | null>(null);
  const [cfType, setCfType] = useState<"group" | "individual">("group");
  const [cfLessonPrice, setCfLessonPrice] = useState("0.00");
  const [cfSubscriptionPrice, setCfSubscriptionPrice] = useState("0.00");

  const handleCoursePriceChange = (value: string, setter: (value: string) => void) => {
    const next = normalizeMoneyInput(value);
    if (next !== null) setter(next);
  };

  const loadCourses = useCallback(async () => {
    setCourseLoading(true);
    try {
      const data = await listCourses(courseQ);
      setCourseList(data);
    } finally {
      setCourseLoading(false);
    }
  }, [courseQ]);

  const loadAllCourses = useCallback(async () => {
    const data = await listCourses("");
    setAllCourses(data);
    return data;
  }, []);

  const loadAllTeachers = useCallback(async () => {
    const data = await listTeachers("");
    setAllTeachers(data);
    return data;
  }, []);

  useEffect(() => {
    if (!appReady) return;
    if (tab === "courses") loadCourses();
  }, [appReady, tab, loadCourses]);

  useEffect(() => {
    if (!appReady) return;
    void loadAllCourses();
  }, [appReady, loadAllCourses]);

  useEffect(() => {
    if (!appReady) return;
    void loadAllTeachers();
  }, [appReady, loadAllTeachers]);

  const selectedCourseTeacher = useMemo(
    () => allTeachers.find((t) => t.id === cfTeacherId) ?? null,
    [allTeachers, cfTeacherId]
  );

  const filteredTeachers = useMemo(() => {
    const q = cfTeacherSearch.trim().toLowerCase();
    if (!q) return allTeachers;
    return allTeachers.filter((t) => t.fullName.toLowerCase().includes(q));
  }, [allTeachers, cfTeacherSearch]);

  const exactTeacherMatch = useMemo(() => {
    const q = cfTeacherSearch.trim().toLowerCase();
    if (!q) return null;
    return allTeachers.find((t) => t.fullName.trim().toLowerCase() === q) ?? null;
  }, [allTeachers, cfTeacherSearch]);

  useEffect(() => {
    if (!cfTeacherPickerOpen) return;

    const handlePointerDown = (event: MouseEvent) => {
      if (!cfTeacherComboRef.current?.contains(event.target as Node)) {
        setCfTeacherPickerOpen(false);
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
    };
  }, [cfTeacherPickerOpen]);

  function openAddCourse() {
    setEditingCourse(null);
    setCfName("");
    setCfTeacherId(undefined);
    setCfTeacherSearch("");
    setCfTeacherPickerOpen(false);
    setCfType("group");
    setCfLessonPrice("");
    setCfSubscriptionPrice("");
    setCourseModalOpen(true);
  }

  function openEditCourse(c: CourseDTO) {
    setEditingCourse(c);
    setCfName(c.name);
    setCfTeacherId(c.teacherId);
    setCfTeacherSearch(c.teacherName);
    setCfTeacherPickerOpen(false);
    setCfType(c.type);
    setCfLessonPrice(c.lessonPrice.toFixed(2));
    setCfSubscriptionPrice(c.subscriptionPrice.toFixed(2));
    setCourseModalOpen(true);
  }

  async function addTeacherFromCourseForm() {
    const name = cfTeacherSearch.trim();
    if (!name) return;

    try {
      setCfTeacherCreating(true);
      const created = await createTeacher(name);
      setAllTeachers((prev) => {
        const withoutSame = prev.filter((t) => t.id !== created.id);
        return [...withoutSame, created].sort((a, b) => a.fullName.localeCompare(b.fullName));
      });
      setCfTeacherId(created.id);
      setCfTeacherSearch(created.fullName);
      setCfTeacherPickerOpen(false);
      showMessage(t("msg.teacherAdded"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setCfTeacherCreating(false);
    }
  }

  async function saveCourse() {
    const lessonPrice = decimalOrZero(cfLessonPrice);
    const subscriptionPrice = decimalOrZero(cfSubscriptionPrice);
    const trimmedTeacherSearch = cfTeacherSearch.trim();

    if (!cfName.trim()) {
      showMessage(t("msg.courseNameRequired"), "error");
      return;
    }
    if (lessonPrice < 0 || subscriptionPrice < 0) {
      showMessage(t("msg.coursePricesNonNegative"), "error");
      return;
    }

    let teacherId = cfTeacherId;
    if (!teacherId && exactTeacherMatch) {
      teacherId = exactTeacherMatch.id;
    }
    if (trimmedTeacherSearch && !teacherId) {
      showMessage(t("msg.courseTeacherRequired"), "error");
      return;
    }

    try {
      if (editingCourse) {
        await updateCourse(
          editingCourse.id,
          cfName,
          teacherId,
          cfType,
          lessonPrice,
          subscriptionPrice
        );
      } else {
        await createCourse(cfName, teacherId, cfType, lessonPrice, subscriptionPrice);
      }

      setCourseModalOpen(false);
      await Promise.all([loadCourses(), loadAllCourses()]);
      showMessage(editingCourse ? t("msg.courseUpdated") : t("msg.courseCreated"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function removeCourse(id: number) {
    showConfirm(
      t("msg.courseDeleteConfirm"),
      async () => {
        try {
          await deleteCourse(id);
          await Promise.all([loadCourses(), loadAllCourses()]);
          showMessage(t("msg.courseDeleted"));
        } catch (e: any) {
          showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
        }
      }
    );
  }

  // ---------------- Enrollments ----------------
  const [enrollments, setEnrollments] = useState<EnrollmentDTO[]>([]);
  const [enrStudentFilter, setEnrStudentFilter] = useState<number | undefined>(undefined);
  const [enrCourseFilter, setEnrCourseFilter] = useState<number | undefined>(undefined);
  const [enrLoading, setEnrLoading] = useState(false);

  const [enrModalOpen, setEnrModalOpen] = useState(false);
  const [editingEnr, setEditingEnr] = useState<EnrollmentDTO | null>(null);
  const [efStudentId, setEfStudentId] = useState<number>(0);
  const [efStudentSearch, setEfStudentSearch] = useState("");
  const [efStudentPickerOpen, setEfStudentPickerOpen] = useState(false);
  const efStudentComboRef = useRef<HTMLDivElement | null>(null);
  const [efCourseId, setEfCourseId] = useState<number>(0);
  const [efMode, setEfMode] = useState<"subscription" | "per_lesson">("per_lesson");
  const [efDiscount, setEfDiscount] = useState(0);
  const [efSubscriptionDiscount, setEfSubscriptionDiscount] = useState(20);
  const [efNote, setEfNote] = useState("");

  const activeStudents = useMemo(() => allStudents.filter((s) => s.isActive), [allStudents]);
  const selectedEnrollmentStudent = useMemo(
    () => allStudents.find((s) => s.id === efStudentId) ?? null,
    [allStudents, efStudentId]
  );
  const filteredEnrollmentStudents = useMemo(() => {
    const q = efStudentSearch.trim().toLowerCase();
    if (!q) return activeStudents;
    return activeStudents.filter((s) => {
      const haystack = `${s.fullName} ${s.phone} ${s.email}`.toLowerCase();
      return haystack.includes(q);
    });
  }, [activeStudents, efStudentSearch]);

  useEffect(() => {
    if (!efStudentPickerOpen) return;

    const handlePointerDown = (event: MouseEvent) => {
      if (!efStudentComboRef.current?.contains(event.target as Node)) {
        setEfStudentPickerOpen(false);
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
    };
  }, [efStudentPickerOpen]);

  const loadEnrollments = useCallback(async () => {
    setEnrLoading(true);
    try {
      await Promise.all([
        allStudents.length === 0 ? loadAllStudents() : Promise.resolve(),
        allCourses.length === 0 ? loadAllCourses() : Promise.resolve(),
      ]);
      const data = await listEnrollments(enrStudentFilter, enrCourseFilter);
      setEnrollments(data);
    } finally {
      setEnrLoading(false);
    }
  }, [
    enrStudentFilter,
    enrCourseFilter,
    allStudents.length,
    allCourses.length,
    loadAllStudents,
    loadAllCourses,
  ]);

  useEffect(() => {
    if (tab === "enrollments") loadEnrollments();
  }, [tab, loadEnrollments]);

  function openAddEnrollment() {
    if (activeStudents.length === 0) {
      showMessage(t("msg.noActiveStudents"), "error");
      setTab("students");
      return;
    }
    if (allCourses.length === 0) {
      showMessage(t("msg.noAvailableCourses"), "error");
      setTab("courses");
      return;
    }

    const initialCourseId = allCourses[0]?.id ?? 0;

    setEditingEnr(null);
    setEfStudentId(0);
    setEfStudentSearch("");
    setEfStudentPickerOpen(false);
    setEfCourseId(initialCourseId);
    setEfMode("per_lesson");
    setEfDiscount(0);
    setEfSubscriptionDiscount(20);
    setEfNote("");
    setEnrModalOpen(true);
  }

  function openEditEnrollment(e: EnrollmentDTO) {
    setEditingEnr(e);
    setEfStudentId(e.studentId);
    setEfStudentSearch(e.studentName);
    setEfStudentPickerOpen(false);
    setEfCourseId(e.courseId);
    setEfMode(e.billingMode);
    setEfDiscount(e.discountPct);
    setEfSubscriptionDiscount(e.subscriptionDiscountPct);
    setEfNote(e.note);
    setEnrModalOpen(true);
  }

  async function saveEnrollment() {
    if (efStudentId <= 0 || efCourseId <= 0) {
      showMessage(t("msg.chooseStudentAndCourse"), "error");
      return;
    }
    if (efDiscount < 0 || efDiscount > 100) {
      showMessage(t("msg.discountRange"), "error");
      return;
    }
    if (efSubscriptionDiscount < 0 || efSubscriptionDiscount > 100) {
      showMessage(t("msg.discountRange"), "error");
      return;
    }

    try {
      let result: EnrollmentDTO;
      if (editingEnr) {
        result = await updateEnrollment(
          editingEnr.id,
          efMode,
          efDiscount,
          efMode === "subscription" ? efSubscriptionDiscount : 0,
          efNote
        );
        showMessage(t("msg.enrollmentUpdated"));
      } else {
        result = await createEnrollment(
          efStudentId,
          efCourseId,
          efMode,
          efDiscount,
          efMode === "subscription" ? efSubscriptionDiscount : 0,
          efNote
        );

        const matchesFilters =
          (enrStudentFilter === undefined || enrStudentFilter === result.studentId) &&
          (enrCourseFilter === undefined || enrCourseFilter === result.courseId);

        if (matchesFilters) {
          showMessage(
            t("msg.enrollmentCreated", {
              student: result.studentName,
              course: result.courseName,
            })
          );
        } else {
          showMessage(
            t("msg.enrollmentCreatedFiltered", {
              student: result.studentName,
              course: result.courseName,
            })
          );
        }
      }

      setEnrModalOpen(false);
      await loadEnrollments();
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  // ---------------- Attendance ----------------
  const [rows, setRows] = useState<Row[]>([]);
  const [loadingAtt, setLoadingAtt] = useState(false);
  const [courseFilter, setCourseFilter] = useState<number | undefined>(undefined);
  const [attQ, setAttQ] = useState("");
  const [attFilter, setAttFilter] = useState<"all" | "missing" | "filled" | "zero">("all");
  const [attendanceSavingRows, setAttendanceSavingRows] = useState<Record<number, boolean>>({});
  const attendanceSavingRowsRef = useRef<Record<number, boolean>>({});
  const [attendanceInputDrafts, setAttendanceInputDrafts] = useState<Record<number, string>>({});
  const attendancePendingSelectRef = useRef<number | null>(null);
  const [subscriptionMonthLessons, setSubscriptionMonthLessons] = useState<Record<number, number>>({});
  const [subscriptionMonthDrafts, setSubscriptionMonthDrafts] = useState<Record<number, string>>({});
  const [subscriptionMonthSaving, setSubscriptionMonthSaving] = useState<Record<number, boolean>>({});

  // For search by phone we need students list (shared with invoices and attendance)
  const studentIndex = useMemo(() => {
    const m = new Map<number, StudentDTO>();
    for (const s of allStudents) m.set(s.id, s);
    return m;
  }, [allStudents]);

  const ensureStudentsLoaded = useCallback(async () => {
    if (allStudents.length > 0) return;
    await loadAllStudents();
  }, [allStudents.length, loadAllStudents]);

  const ensureCoursesLoaded = useCallback(async () => {
    if (allCourses.length > 0) return;
    await loadAllCourses();
  }, [allCourses.length, loadAllCourses]);

  const loadAttendance = useCallback(async () => {
    setLoadingAtt(true);
    try {
      await ensureStudentsLoaded();
      await ensureCoursesLoaded();
      const [data, subscriptionData] = await Promise.all([
        fetchRows(year, month, courseFilter),
        listCourseMonthSubscriptions(year, month, courseFilter),
      ]);
      setRows(data);
      setSubscriptionMonthLessons(
        Object.fromEntries(subscriptionData.map((item) => [item.courseId, item.lessonsHeld]))
      );
      setSubscriptionMonthDrafts({});
    } finally {
      setLoadingAtt(false);
    }
  }, [year, month, courseFilter, ensureStudentsLoaded, ensureCoursesLoaded]);

  useEffect(() => {
    if (!appReady) return;
    if (tab === "attendance") loadAttendance();
  }, [appReady, tab, loadAttendance]);

  useEffect(() => {
    attendanceSavingRowsRef.current = attendanceSavingRows;
  }, [attendanceSavingRows]);

  const perLessonTotal = useMemo(
    () =>
      rows.reduce(
        (s, r) => s + (r.billingMode === BillingModePerLesson ? r.hours * r.lessonPrice : 0),
        0
      ),
    [rows]
  );

  const filteredAttendanceRows = useMemo(() => {
    const q = attQ.trim().toLowerCase();
    let filtered = rows;

    if (q) {
      filtered = filtered.filter((r) => {
        const s = studentIndex.get(r.studentId);
        const studentName = (r.studentName ?? "").toLowerCase();
        const courseName = (r.courseName ?? "").toLowerCase();
        const phone = (s?.phone ?? "").toLowerCase();
        return studentName.includes(q) || courseName.includes(q) || phone.includes(q);
      });
    }

    if (attFilter === "missing") {
      filtered = filtered.filter((r) => r.billingMode === BillingModePerLesson && !r.hasRecord);
    } else if (attFilter === "filled") {
      filtered = filtered.filter((r) => r.billingMode === BillingModePerLesson && r.hasRecord);
    } else if (attFilter === "zero") {
      filtered = filtered.filter(
        (r) => r.billingMode === BillingModePerLesson && r.hasRecord && r.hours === 0
      );
    }

    return [...filtered].sort((a, b) => {
      const aSubscription = a.billingMode === BillingModeSubscription;
      const bSubscription = b.billingMode === BillingModeSubscription;
      if (aSubscription !== bSubscription) {
        return aSubscription ? 1 : -1;
      }

      const studentCompare = a.studentName.localeCompare(b.studentName);
      if (studentCompare !== 0) return studentCompare;

      const courseCompare = a.courseName.localeCompare(b.courseName);
      if (courseCompare !== 0) return courseCompare;

      return a.enrollmentId - b.enrollmentId;
    });
  }, [rows, attQ, attFilter, studentIndex]);

  const attendanceSummary = useMemo(() => {
    const editableRows = rows.filter((r) => r.billingMode === BillingModePerLesson);
    const filled = editableRows.filter((r) => r.hasRecord).length;
    const missing = editableRows.filter((r) => !r.hasRecord).length;
    const zero = editableRows.filter((r) => r.hasRecord && r.hours === 0).length;
    return { filled, missing, zero, total: editableRows.length };
  }, [rows]);

  const subscriptionLeadEnrollmentIds = useMemo(() => {
    const seen = new Set<number>();
    const leadIds = new Set<number>();
    for (const row of filteredAttendanceRows) {
      if (row.billingMode !== BillingModeSubscription) continue;
      if (seen.has(row.courseId)) continue;
      seen.add(row.courseId);
      leadIds.add(row.enrollmentId);
    }
    return leadIds;
  }, [filteredAttendanceRows]);

  const onChangeHours = useCallback(async (r: Row, v: number) => {
    if (r.billingMode !== BillingModePerLesson) return;
    if (r.attendanceLocked) {
      showMessage(
        t("msg.attendanceLocked", {
          status: localizedInvoiceStatusLabel(r.invoiceStatus ?? "issued"),
        }),
        "error"
      );
      return;
    }
    if (!Number.isFinite(v)) return;
    const n = normalizeQuarterHours(v);
    if (attendanceSavingRowsRef.current[r.enrollmentId]) return;

    try {
      setAttendanceSavingRows((prev) => ({ ...prev, [r.enrollmentId]: true }));
      await saveHours(r.studentId, r.courseId, year, month, n);
      setRows((prev) =>
        prev.map((x) =>
          x.enrollmentId === r.enrollmentId ? { ...x, hours: n, hasRecord: true } : x
        )
      );
      try {
        await rebuildStudentDraft(r.studentId, year, month);
      } catch (invoiceError: any) {
        showMessage(
          t("msg.attendanceSavedDraftError", {
            message: String(invoiceError?.message ?? invoiceError),
          }),
          "error"
        );
      }
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setAttendanceSavingRows((prev) => {
        const next = { ...prev };
        delete next[r.enrollmentId];
        return next;
      });
    }
  }, [localizedInvoiceStatusLabel, month, showMessage, t, year]);

  const setAttendanceDraft = useCallback((enrollmentId: number, value: string) => {
    setAttendanceInputDrafts((prev) => ({ ...prev, [enrollmentId]: value }));
  }, []);

  const clearAttendanceDraft = useCallback((enrollmentId: number) => {
    setAttendanceInputDrafts((prev) => {
      if (!(enrollmentId in prev)) return prev;
      const next = { ...prev };
      delete next[enrollmentId];
      return next;
    });
  }, []);

  const commitAttendanceDraft = useCallback(
    async (r: Row) => {
      const draft = attendanceInputDrafts[r.enrollmentId];
      if (draft === undefined) return;

      const trimmed = draft.trim();
      if (trimmed === "") {
        clearAttendanceDraft(r.enrollmentId);
        return;
      }

      const parsed = Number(trimmed.replace(",", "."));
      if (!Number.isFinite(parsed)) {
        clearAttendanceDraft(r.enrollmentId);
        showMessage(t("msg.errorGeneric", { message: "Invalid hours value" }), "error");
        return;
      }

      if (normalizeQuarterHours(parsed) === r.hours) {
        clearAttendanceDraft(r.enrollmentId);
        return;
      }

      try {
        await onChangeHours(r, parsed);
      } finally {
        clearAttendanceDraft(r.enrollmentId);
      }
    },
    [attendanceInputDrafts, clearAttendanceDraft, onChangeHours, showMessage, t]
  );

  const getAttendanceInputValue = useCallback(
    (r: Row) => attendanceInputDrafts[r.enrollmentId] ?? formatHoursValue(r.hours),
    [attendanceInputDrafts]
  );

  const getAttendanceStepBase = useCallback(
    (r: Row) => {
      const draft = attendanceInputDrafts[r.enrollmentId];
      if (draft === undefined || draft.trim() === "") return r.hours;
      const parsed = Number(draft.replace(",", "."));
      return Number.isFinite(parsed) ? parsed : r.hours;
    },
    [attendanceInputDrafts]
  );

  const getSubscriptionMonthLessonsValue = useCallback(
    (courseId: number) => {
      const draft = subscriptionMonthDrafts[courseId];
      if (draft !== undefined) return draft;
      return formatHoursValue(subscriptionMonthLessons[courseId] ?? 0);
    },
    [subscriptionMonthDrafts, subscriptionMonthLessons]
  );

  const setSubscriptionMonthLessonsDraft = useCallback((courseId: number, value: string) => {
    setSubscriptionMonthDrafts((prev) => ({ ...prev, [courseId]: value }));
  }, []);

  const clearSubscriptionMonthLessonsDraft = useCallback((courseId: number) => {
    setSubscriptionMonthDrafts((prev) => {
      if (!(courseId in prev)) return prev;
      const next = { ...prev };
      delete next[courseId];
      return next;
    });
  }, []);

  const commitSubscriptionMonthLessonsDraft = useCallback(
    async (row: Row) => {
      const draft = subscriptionMonthDrafts[row.courseId];
      if (draft === undefined) return;
      const trimmed = draft.trim();
      if (trimmed === "") {
        clearSubscriptionMonthLessonsDraft(row.courseId);
        return;
      }
      const parsed = Number(trimmed.replace(",", "."));
      if (!Number.isFinite(parsed) || parsed < 0) {
        clearSubscriptionMonthLessonsDraft(row.courseId);
        showMessage(t("msg.errorGeneric", { message: "Invalid lessons value" }), "error");
        return;
      }
      const normalized = normalizeQuarterHours(parsed);
      if (normalized === (subscriptionMonthLessons[row.courseId] ?? 0)) {
        clearSubscriptionMonthLessonsDraft(row.courseId);
        return;
      }

      try {
        setSubscriptionMonthSaving((prev) => ({ ...prev, [row.courseId]: true }));
        const updated = await saveCourseMonthSubscriptionLessons(row.courseId, year, month, normalized);
        setSubscriptionMonthLessons((prev) => ({ ...prev, [row.courseId]: updated.lessonsHeld }));
        await loadAttendance();
      } catch (e: any) {
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      } finally {
        clearSubscriptionMonthLessonsDraft(row.courseId);
        setSubscriptionMonthSaving((prev) => {
          const next = { ...prev };
          delete next[row.courseId];
          return next;
        });
      }
    },
    [
      clearSubscriptionMonthLessonsDraft,
      loadAttendance,
      month,
      showMessage,
      subscriptionMonthDrafts,
      subscriptionMonthLessons,
      t,
      year,
    ]
  );

  const onDeleteEnrollmentFromSheet = async (id: number) => {
    showConfirm(
      t("msg.enrollmentDeleteConfirm"),
      async () => {
        try {
          await deleteEnrollment(id);
          await loadAttendance();
          showMessage(t("msg.enrollmentDeleted"));
        } catch (e: any) {
          showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
        }
      }
    );
  };

  // ---------------- Invoices ----------------
  const [invStatus, setInvStatus] = useState<string>("all");
  const [invItems, setInvItems] = useState<InvoiceListItemView[]>([]);
  const [selectedInv, setSelectedInv] = useState<InvoiceDTO | null>(null);
  const [invoiceDetailsOpen, setInvoiceDetailsOpen] = useState(false);
  const [loadingInv, setLoadingInv] = useState(false);
  const [invQ, setInvQ] = useState("");
  const [invSummary, setInvSummary] = useState<InvoiceSummaryDTO | null>(null);
  const pendingInvoiceScrollRestoreRef = useRef<number | null>(null);
  const [openInvoiceMenu, setOpenInvoiceMenu] = useState<InvoiceMenuTarget | null>(null);
  const invoiceMenuRef = useRef<HTMLDivElement | null>(null);
  const activeInvoiceMenuTriggerRef = useRef<HTMLButtonElement | null>(null);
  const [invoiceMenuPosition, setInvoiceMenuPosition] = useState<InvoiceMenuPosition | null>(null);

  // Payment modal state
  const [paymentModalOpen, setPaymentModalOpen] = useState(false);
  const [paymentAmount, setPaymentAmount] = useState("");
  const [paymentMethod, setPaymentMethod] = useState<"cash" | "bank">("cash");
  const [paymentNote, setPaymentNote] = useState("");
  const [paymentStudentId, setPaymentStudentId] = useState<number>(0);
  const [paymentStudentName, setPaymentStudentName] = useState("");
  const [paymentInvoiceId, setPaymentInvoiceId] = useState<number | undefined>(undefined);
  const [returnToDebtDetailsAfterPayment, setReturnToDebtDetailsAfterPayment] = useState(false);
  const [returnToStudentCardAfterPayment, setReturnToStudentCardAfterPayment] = useState(false);

  const syncDraftInvoices = useCallback(
    async (showFeedback = true) => {
      const beforeDrafts = await listInvoices(year, month, "draft");
      const res = await genDrafts(year, month);
      const afterDrafts = await listInvoices(year, month, "draft");

      const afterIds = new Set(afterDrafts.map((item) => item.id));
      const removed = beforeDrafts.filter((item) => !afterIds.has(item.id)).length;

      if (showFeedback && (res.created > 0 || res.updated > 0 || removed > 0)) {
        const parts = [];
        if (res.created > 0) parts.push(t("msg.createdCount", { count: res.created }));
        if (res.updated > 0) parts.push(t("msg.updatedCount", { count: res.updated }));
        if (removed > 0) parts.push(t("msg.deletedCount", { count: removed }));
        showMessage(t("msg.invoiceSyncSummary", { parts: parts.join(", ") }));
      }
    },
    [year, month, showMessage, t]
  );

  const loadInvoices = useCallback(
    async (options?: { syncDrafts?: boolean; showSyncFeedback?: boolean }) => {
      setLoadingInv(true);
      try {
        await ensureStudentsLoaded();
        if (options?.syncDrafts !== false) {
          await syncDraftInvoices(options?.showSyncFeedback ?? true);
        }
        const li = await listInvoices(year, month, invStatus);
        const pdfReadyById = new Map<number, boolean>();
        await Promise.all(
          li
            .filter((item) => item.status !== "draft")
            .map(async (item) => {
              try {
                pdfReadyById.set(item.id, await hasPdf(item.id));
              } catch {
                pdfReadyById.set(item.id, false);
              }
            })
        );
        setInvItems(
          li.map((item) => ({
            ...item,
            pdfReady: item.status !== "draft" ? (pdfReadyById.get(item.id) ?? false) : false,
          }))
        );
      } catch (e: any) {
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      } finally {
        setLoadingInv(false);
      }
    },
    [year, month, invStatus, ensureStudentsLoaded, showMessage, syncDraftInvoices, t]
  );

  useEffect(() => {
    if (tab === "invoice") {
      void loadInvoices();
    }
  }, [tab, loadInvoices]);

  useLayoutEffect(() => {
    const scrollY = pendingInvoiceScrollRestoreRef.current;
    if (scrollY === null) return;

    pendingInvoiceScrollRestoreRef.current = null;
    requestAnimationFrame(() => {
      window.scrollTo({ top: scrollY, behavior: "auto" });
    });
  }, [invItems]);

  // ---------------- Debtors ----------------
  const [debtors, setDebtors] = useState<DebtorDTO[]>([]);
  const [debtorsLoading, setDebtorsLoading] = useState(false);
  const [debtDetailsOpen, setDebtDetailsOpen] = useState(false);
  const [selectedDebtor, setSelectedDebtor] = useState<DebtorDTO | null>(null);
  const [debtDetails, setDebtDetails] = useState<DebtInvoiceDTO[]>([]);
  const [debtDetailsLoading, setDebtDetailsLoading] = useState(false);

  const loadDebtors = useCallback(async () => {
    setDebtorsLoading(true);
    try {
      const data = await listDebtors();
      setDebtors(data);
      setDebtorActionQueue(buildDebtorActionQueue(data, recentPayments, t));
      return data;
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      return [];
    } finally {
      setDebtorsLoading(false);
    }
  }, [recentPayments, showMessage, t]);

  useEffect(() => {
    if (!appReady) return;
    if (tab === "debtors") loadDebtors();
  }, [appReady, tab, loadDebtors]);

  useEffect(() => {
    setDebtorActionQueue(buildDebtorActionQueue(debtors, recentPayments, t));
  }, [debtors, recentPayments, t]);

  async function openDebtDetails(debtor: DebtorDTO) {
    setSelectedDebtor(debtor);
    setDebtDetailsOpen(true);
    setDebtDetails([]);
    setDebtDetailsLoading(true);

    try {
      const details = await studentDebtDetails(debtor.studentId);
      setDebtDetails(details);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    } finally {
      setDebtDetailsLoading(false);
    }
  }

  async function copyDebtMessage(locale: "ru" | "lv") {
    if (!selectedDebtor || debtDetailsLoading || debtDetails.length === 0) return;

    try {
      const recipientName = await resolveDebtReminderRecipient(
        selectedDebtor.studentId,
        selectedDebtor.studentName
      );
      const text = buildDebtReminderMessage(locale, selectedDebtor, debtDetails, recipientName);
      await copyTextToClipboard(text);
      showMessage(locale === "ru" ? t("msg.debtReminderRuCopied") : t("msg.debtReminderLvCopied"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function copyDebtMessageForDebtor(debtor: DebtorDTO, locale: "ru" | "lv") {
    try {
      const [details, recipientName] = await Promise.all([
        studentDebtDetails(debtor.studentId),
        resolveDebtReminderRecipient(debtor.studentId, debtor.studentName),
      ]);
      if (details.length === 0) {
        showMessage(t("msg.noOpenDebtsStudent"), "error");
        return;
      }
      const text = buildDebtReminderMessage(locale, debtor, details, recipientName);
      await copyTextToClipboard(text);
      showMessage(locale === "ru" ? t("msg.copyRu") : t("msg.copyLv"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  function openDebtorPaymentModalByStudentId(studentId: number) {
    const debtor = debtors.find((item) => item.studentId === studentId);
    if (!debtor) return;
    openDebtorPaymentModal(debtor);
  }

  async function copyDebtMessageForStudentId(studentId: number, locale: "ru" | "lv") {
    const debtor = debtors.find((item) => item.studentId === studentId);
    if (!debtor) return;
    await copyDebtMessageForDebtor(debtor, locale);
  }

  const filteredInvItems = useMemo(() => {
    const q = invQ.trim().toLowerCase();
    if (!q) return invItems;

    return invItems.filter((it) => {
      const s = studentIndex.get(it.studentId);
      const name = (it.studentName ?? "").toLowerCase();
      const number = (it.number ?? "").toLowerCase();
      const email = (s?.email ?? "").toLowerCase();
      const phone = (s?.phone ?? "").toLowerCase();
      return name.includes(q) || number.includes(q) || email.includes(q) || phone.includes(q);
    });
  }, [invItems, invQ, studentIndex]);

  const loadInvoiceDetails = useCallback(async (id: number) => {
    const iv = await getInvoice(id);
    setSelectedInv(iv);
    if (iv.status !== "draft") {
      const summary = await invoiceSummary(id);
      setInvSummary(summary);
      return { invoice: iv, summary };
    } else {
      setInvSummary(null);
      return { invoice: iv, summary: null };
    }
  }, []);

  const onOpenInvoice = async (id: number) => {
    try {
      setOpenInvoiceMenu(null);
      await loadInvoiceDetails(id);
      setInvoiceDetailsOpen(true);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  };

  const closeInvoiceMenu = useCallback(() => {
    activeInvoiceMenuTriggerRef.current = null;
    setOpenInvoiceMenu(null);
    setInvoiceMenuPosition(null);
  }, []);

  const openPaymentModal = (inv?: InvoiceDTO, summary?: InvoiceSummaryDTO | null) => {
    const currentInv = inv ?? selectedInv;
    const currentSummary = summary !== undefined ? summary : invSummary;
    if (!currentInv) return;
    const remaining = currentSummary ? currentSummary.remaining : currentInv.total;
    setPaymentStudentId(currentInv.studentId);
    setPaymentStudentName(currentInv.studentName);
    setPaymentInvoiceId(currentInv.id);
    setPaymentAmount(remaining.toFixed(2));
    setPaymentMethod("cash");
    setPaymentNote("");
    setPaymentModalOpen(true);
  };

  const openPaymentModalForInvoice = async (id: number) => {
    try {
      const { invoice, summary } = await loadInvoiceDetails(id);
      openPaymentModal(invoice, summary);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  };

  const openDebtorPaymentModal = (debtor: DebtorDTO, returnToDebtDetails = false) => {
    setPaymentStudentId(debtor.studentId);
    setPaymentStudentName(debtor.studentName);
    setPaymentInvoiceId(undefined);
    setPaymentAmount(debtor.debt.toFixed(2));
    setPaymentMethod("cash");
    setPaymentNote("");
    setReturnToDebtDetailsAfterPayment(returnToDebtDetails);
    setPaymentModalOpen(true);
  };

  const closePaymentModal = () => {
    setPaymentModalOpen(false);
    setReturnToDebtDetailsAfterPayment(false);
    setReturnToStudentCardAfterPayment(false);
  };

  const openStudentCardPaymentModal = () => {
    if (!selectedStudentCard) return;
    const debt = studentCardBalance?.debt ?? 0;
    setPaymentStudentId(selectedStudentCard.id);
    setPaymentStudentName(selectedStudentCard.fullName);
    setPaymentInvoiceId(undefined);
    setPaymentAmount(debt > 0 ? debt.toFixed(2) : "");
    setPaymentMethod("cash");
    setPaymentNote("");
    setReturnToStudentCardAfterPayment(true);
    setPaymentModalOpen(true);
  };

  const openPaymentFromDebtDetails = () => {
    if (!selectedDebtor) return;
    setDebtDetailsOpen(false);
    openDebtorPaymentModal(selectedDebtor, true);
  };

  const handleCreatePayment = async () => {
    const amount = parseFloat(paymentAmount);
    if (paymentStudentId <= 0) {
      showMessage(t("msg.paymentStudentMissing"), "error");
      return;
    }
    if (isNaN(amount) || amount <= 0) {
      showMessage(t("msg.paymentAmountInvalid"), "error");
      return;
    }

    try {
      const today = new Date().toISOString().split("T")[0]; // YYYY-MM-DD
      await createPayment(
        paymentStudentId,
        paymentInvoiceId,
        amount,
        paymentMethod,
        today,
        paymentNote
      );

      setPaymentModalOpen(false);
      showMessage(t("msg.paymentRecorded"));

      if (paymentInvoiceId) {
        await loadInvoices({ syncDrafts: false });
        if (invoiceDetailsOpen && selectedInv?.id === paymentInvoiceId) {
          await loadInvoiceDetails(paymentInvoiceId);
        }
      }
      const updatedDebtors = await loadDebtors();

      if (returnToDebtDetailsAfterPayment && selectedDebtor?.studentId === paymentStudentId) {
        const updatedDetails = await studentDebtDetails(paymentStudentId);
        const matchedDebtor = updatedDebtors.find((d) => d.studentId === paymentStudentId);
        const refreshedDebt = updatedDetails.reduce((sum, item) => sum + item.remaining, 0);

        setSelectedDebtor(
          matchedDebtor ?? {
            ...selectedDebtor,
            debt: refreshedDebt,
          }
        );
        setDebtDetails(updatedDetails);
        setDebtDetailsLoading(false);
        setDebtDetailsOpen(true);
      }

      if (returnToStudentCardAfterPayment && selectedStudentCard?.id === paymentStudentId) {
        try {
          await refreshStudentCardData(paymentStudentId);
        } catch {
          // refreshStudentCardData handles its own errors via showMessage
        }
      }

      setReturnToDebtDetailsAfterPayment(false);
      setReturnToStudentCardAfterPayment(false);
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onIssueOne = async (id: number) => {
    try {
      closeInvoiceMenu();
      const scrollY = window.scrollY;
      const res = await issueOne(id);
      pendingInvoiceScrollRestoreRef.current = scrollY;
      await loadInvoices({ syncDrafts: false });
      if (invoiceDetailsOpen && selectedInv?.id === id) {
        await loadInvoiceDetails(id);
      }
      showMessage(t("msg.invoiceIssued", { number: res.number }));
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onReopenToDraft = useCallback(async (id: number) => {
    closeInvoiceMenu();
    showConfirm(
      t("msg.invoiceReopenConfirm"),
      async () => {
        try {
          await reopenToDraft(id);
          await loadInvoices({ syncDrafts: false });
          if (invoiceDetailsOpen && selectedInv?.id === id) {
            await loadInvoiceDetails(id);
          }
          showMessage(t("msg.invoiceReopened"));
        } catch (e: any) {
          showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
        }
      },
      t("button.reopenDraft")
    );
  }, [closeInvoiceMenu, invoiceDetailsOpen, loadInvoiceDetails, loadInvoices, selectedInv, showConfirm, showMessage, t]);

  const onGeneratePdf = useCallback(async (id: number) => {
    try {
      closeInvoiceMenu();
      const pdf = await ensurePdf(id);
      setInvItems((prev) =>
        prev.map((item) => (item.id === id ? { ...item, pdfReady: true } : item))
      );
      showMessage(t("msg.pdfReady", { path: pdf.localPath ?? pdf.filename }));
      if (!transportCapabilities.isDesktop && pdf.downloadUrl) {
        window.open(pdf.downloadUrl, "_blank", "noopener,noreferrer");
      }
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [closeInvoiceMenu, showMessage, t, transportCapabilities.isDesktop]);

  const onDownloadPdf = useCallback(async (id: number) => {
    try {
      closeInvoiceMenu();
      const pdf = await ensurePdf(id);
      setInvItems((prev) =>
        prev.map((item) => (item.id === id ? { ...item, pdfReady: true } : item))
      );

      if (transportCapabilities.isDesktop) {
        if (!pdf.localPath) {
          showMessage(t("msg.errorGeneric", { message: t("msg.pdfDownloadUnavailable") }), "error");
          return;
        }
        const transport = await getTransport();
        await transport.openLocalPath(pdf.localPath);
        showMessage(t("msg.pdfReady", { path: pdf.localPath }));
        return;
      }

      if (!pdf.downloadUrl) {
        showMessage(t("msg.errorGeneric", { message: t("msg.pdfDownloadUnavailable") }), "error");
        return;
      }

      const link = document.createElement("a");
      link.href = pdf.downloadUrl;
      link.download = pdf.filename;
      link.rel = "noopener";
      link.style.display = "none";
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      showMessage(t("msg.pdfDownloaded", { filename: pdf.filename }));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [closeInvoiceMenu, showMessage, t, transportCapabilities.isDesktop]);

  const onRevealInvoiceFile = useCallback(async (id: number) => {
    try {
      closeInvoiceMenu();
      const pdf = await ensurePdf(id);
      if (!pdf.localPath) {
        showMessage(t("msg.folderUnavailable", { label: t("tabs.invoice").toLowerCase() }), "error");
        return;
      }
      const transport = await getTransport();
      await transport.openLocalPath(pdf.localPath);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [closeInvoiceMenu, showMessage, t]);

  const buildInvoiceMenuItems = useCallback(
    (invoice: Pick<InvoiceDTO, "id" | "status"> & { pdfReady?: boolean }) => {
      const menuItems: Array<{ label: string; onClick: () => void }> = [];

      if (invoice.status === "issued") {
        menuItems.push({
          label: t("button.reopenDraft"),
          onClick: () => void onReopenToDraft(invoice.id),
        });
      }
      if (invoice.status !== "draft") {
        if (canShowInvoiceFolderAction(transportCapabilities)) {
          menuItems.push({
            label: t("button.showInFolder"),
            onClick: () => void onRevealInvoiceFile(invoice.id),
          });
        }
        if (transportCapabilities.isDesktop && !invoice.pdfReady) {
          menuItems.push({
            label: t("button.createPdf"),
            onClick: () => void onGeneratePdf(invoice.id),
          });
        }
      }

      return menuItems;
    },
    [onGeneratePdf, onReopenToDraft, onRevealInvoiceFile, t, transportCapabilities]
  );

  const openInvoiceMenuAtTrigger = useCallback(
    (
      kind: InvoiceMenuTarget["kind"],
      invoice: Pick<InvoiceDTO, "id" | "status"> & { pdfReady?: boolean },
      trigger: HTMLButtonElement
    ) => {
      const menuItems = buildInvoiceMenuItems(invoice);
      if (menuItems.length === 0) {
        closeInvoiceMenu();
        return;
      }

      const triggerRect = trigger.getBoundingClientRect();
      const menuWidth = 190;
      const menuGap = 8;
      const menuHeight = menuItems.length * 48 + 16;
      const openUpward =
        window.innerHeight - triggerRect.bottom < menuHeight + menuGap &&
        triggerRect.top > menuHeight + menuGap;
      const top = openUpward ? triggerRect.top - menuGap : triggerRect.bottom + menuGap;
      const left = Math.min(
        window.innerWidth - menuWidth - 12,
        Math.max(12, triggerRect.right - menuWidth)
      );

      activeInvoiceMenuTriggerRef.current = trigger;
      setInvoiceMenuPosition({ top, left, openUpward });
      setOpenInvoiceMenu({ kind, invoiceId: invoice.id });
    },
    [buildInvoiceMenuItems, closeInvoiceMenu]
  );

  const toggleInvoiceMenu = useCallback(
    (
      kind: InvoiceMenuTarget["kind"],
      invoice: Pick<InvoiceDTO, "id" | "status"> & { pdfReady?: boolean },
      trigger: HTMLButtonElement
    ) => {
      if (openInvoiceMenu?.kind === kind && openInvoiceMenu.invoiceId === invoice.id) {
        closeInvoiceMenu();
        return;
      }
      openInvoiceMenuAtTrigger(kind, invoice, trigger);
    },
    [closeInvoiceMenu, openInvoiceMenu, openInvoiceMenuAtTrigger]
  );

  useEffect(() => {
    if (!openInvoiceMenu) return;

    const handlePointerDown = (event: MouseEvent) => {
      const target = event.target as Node;
      if (invoiceMenuRef.current?.contains(target)) return;
      if (activeInvoiceMenuTriggerRef.current?.contains(target)) return;
      closeInvoiceMenu();
    };

    const handleViewportChange = () => {
      closeInvoiceMenu();
    };

    document.addEventListener("mousedown", handlePointerDown);
    window.addEventListener("resize", handleViewportChange);
    window.addEventListener("scroll", handleViewportChange, true);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      window.removeEventListener("resize", handleViewportChange);
      window.removeEventListener("scroll", handleViewportChange, true);
    };
  }, [closeInvoiceMenu, openInvoiceMenu]);

  const previousInvoiceDetailsOpenRef = useRef(invoiceDetailsOpen);
  const previousTabRef = useRef(tab);

  useEffect(() => {
    const invoiceDetailsChanged = previousInvoiceDetailsOpenRef.current !== invoiceDetailsOpen;
    const tabChanged = previousTabRef.current !== tab;

    previousInvoiceDetailsOpenRef.current = invoiceDetailsOpen;
    previousTabRef.current = tab;

    if ((invoiceDetailsChanged || tabChanged) && openInvoiceMenu) {
      closeInvoiceMenu();
    }
  }, [closeInvoiceMenu, invoiceDetailsOpen, openInvoiceMenu, tab]);

  const renderInvoiceActionsMenu = (
    invoice: Pick<InvoiceDTO, "id" | "status"> & { pdfReady?: boolean },
    options?: { kind?: InvoiceMenuTarget["kind"] }
  ) => {
    const kind = options?.kind ?? "row";
    const isOpen = openInvoiceMenu?.kind === kind && openInvoiceMenu.invoiceId === invoice.id;
    const menuItems = buildInvoiceMenuItems(invoice);

    if (menuItems.length === 0) return null;

    return (
      <div className="invoiceActionsMenu">
        <button
          type="button"
          className="invoiceActionsMenuTrigger"
          aria-haspopup="menu"
          aria-expanded={isOpen}
          onMouseDown={(event) => {
            event.stopPropagation();
          }}
          onClick={(event) => {
            event.stopPropagation();
            toggleInvoiceMenu(kind, invoice, event.currentTarget);
          }}
        >
          {t("msg.more")}
        </button>
      </div>
    );
  };

  const openAppFolder = async (path: string | undefined, label: string) => {
    if (!path) {
      showMessage(t("msg.folderUnavailable", { label }), "error");
      return;
    }
    try {
      const transport = await getTransport();
      await transport.openLocalPath(path);
    } catch (e: any) {
      showMessage(
        t("msg.folderOpenError", { label, message: String(e?.message ?? e) }),
        "error"
      );
    }
  };

  const createManualBackup = async () => {
    try {
      setCreatingBackup(true);
      const transport = await getTransport();
      const backup = await transport.createBackup();
      showMessage(t("msg.backupCreated", { path: backup.path ?? backup.filename }));
    } catch (e: any) {
      showMessage(t("msg.backupCreateError", { message: String(e?.message ?? e) }), "error");
    } finally {
      setCreatingBackup(false);
    }
  };

  const loadUsers = useCallback(async () => {
    if (!canManageUsers) return;
    try {
      setUsersLoading(true);
      const transport = await getTransport();
      const items = await transport.listUsers();
      setUsers(items);
      setUserDrafts(
        Object.fromEntries(
          items.map((item) => [item.id, { username: item.username, role: item.role, isActive: item.isActive }])
        )
      );
    } catch (e: any) {
      showMessage(String(e?.message ?? e), "error");
    } finally {
      setUsersLoading(false);
    }
  }, [canManageUsers, showMessage]);

  useEffect(() => {
    if (isAuthenticated && canManageUsers) {
      void loadUsers();
    }
  }, [isAuthenticated, canManageUsers, loadUsers]);

  const handleCreateUser = async () => {
    try {
      setCreatingUser(true);
      const transport = await getTransport();
      const created = await transport.createUser(newUserUsername, newUserPassword, newUserRole);
      setUsers((prev) => [...prev, created]);
      setUserDrafts((prev) => ({
        ...prev,
        [created.id]: { username: created.username, role: created.role, isActive: created.isActive },
      }));
      setNewUserUsername("");
      setNewUserPassword("");
      setNewUserRole("staff");
      showMessage("User created");
    } catch (e: any) {
      showMessage(String(e?.message ?? e), "error");
    } finally {
      setCreatingUser(false);
    }
  };

  const handleSaveUser = async (userId: number) => {
    const draft = userDrafts[userId];
    if (!draft) return;
    try {
      const transport = await getTransport();
      const updated = await transport.updateUser(userId, draft.username, draft.role, draft.isActive);
      setUsers((prev) => prev.map((item) => (item.id === userId ? updated : item)));
      setUserDrafts((prev) => ({ ...prev, [userId]: { username: updated.username, role: updated.role, isActive: updated.isActive } }));
      showMessage("User updated");
    } catch (e: any) {
      showMessage(String(e?.message ?? e), "error");
    }
  };

  const handleDeleteUser = async (userId: number) => {
    const target = users.find((item) => item.id === userId);
    if (!target) return;
    showConfirm(
      `Delete user "${target.username}"? This cannot be undone.`,
      async () => {
        try {
          const transport = await getTransport();
          await transport.deleteUser(userId);
          setUsers((prev) => prev.filter((item) => item.id !== userId));
          setUserDrafts((prev) => {
            const next = { ...prev };
            delete next[userId];
            return next;
          });
          setUserPasswordDrafts((prev) => {
            const next = { ...prev };
            delete next[userId];
            return next;
          });
          showMessage("User deleted");
        } catch (e: any) {
          showMessage(String(e?.message ?? e), "error");
        }
      },
      "Delete user"
    );
  };

  const handleResetUserPassword = async (userId: number) => {
    const password = userPasswordDrafts[userId]?.trim() ?? "";
    if (!password) {
      showMessage("Password is required", "error");
      return;
    }
    try {
      const transport = await getTransport();
      await transport.setUserPassword(userId, password);
      setUserPasswordDrafts((prev) => ({ ...prev, [userId]: "" }));
      showMessage("Password reset");
    } catch (e: any) {
      showMessage(String(e?.message ?? e), "error");
    }
  };

  const handleLocaleChange = async (nextLocale: UiLocale) => {
    const previousLocale = uiLocale;
    setUiLocale(nextLocale);
    try {
      const transport = await getTransport();
      await transport.setLocale(nextLocale);
      showMessage(createTranslator(nextLocale)("settings.languageSaved"));
    } catch (e: any) {
      setUiLocale(previousLocale);
      showMessage(
        createTranslator(previousLocale)("settings.languageSaveError") +
          `: ${String(e?.message ?? e)}`,
        "error"
      );
    }
  };

  // ---------------- Render ----------------
  const showMonthPicker = tab === "dashboard" || tab === "attendance" || tab === "invoice";
  const selectedInvPdfReady = selectedInv
    ? (invItems.find((item) => item.id === selectedInv.id)?.pdfReady ?? false)
    : false;
  const openInvoiceMenuItems = useMemo(() => {
    if (!openInvoiceMenu) return [];

    if (openInvoiceMenu.kind === "modal" && selectedInv) {
      return buildInvoiceMenuItems({ ...selectedInv, pdfReady: selectedInvPdfReady });
    }

    const rowInvoice = invItems.find((item) => item.id === openInvoiceMenu.invoiceId);
    return rowInvoice ? buildInvoiceMenuItems(rowInvoice) : [];
  }, [buildInvoiceMenuItems, invItems, openInvoiceMenu, selectedInv, selectedInvPdfReady]);

  const handleLogin = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      setLoginPending(true);
      setLoginError(null);
      try {
        const transport = await getTransport();
        const session = await transport.login(loginUsername, loginPassword, loginRememberMe);
        setUiLocale(normalizeLocale(session.locale));
        setCurrentSessionUser(session.user ?? null);
        setSessionCapabilities(session.capabilities ?? {});
        setTransportCapabilities({
          isDesktop: false,
          canOpenLocalFiles: false,
          canOpenFolders: false,
          canDownloadPdf: Boolean(session.capabilities?.pdfDownload),
        });
        setIsAuthenticated(session.authenticated);
        setAppReady(session.ready && session.authenticated);
        setLoginError(null);
        setLoginPassword("");
        setSessionExpired(false);
      } catch (e: any) {
        setLoginError(String(e?.message ?? e));
      } finally {
        setLoginPending(false);
      }
    },
    [loginRememberMe, loginPassword, loginUsername]
  );

  const handleLogout = useCallback(async () => {
    try {
      const transport = await getTransport();
      await transport.logout();
    } catch (error) {
      void error;
    }
    setIsAuthenticated(false);
    setCurrentSessionUser(null);
    setSessionCapabilities({});
    setAppReady(false);
    setLoginPassword("");
    setLoginError(null);
    setSessionExpired(false);
  }, []);

  return (
    <div className="container">
      {/* Global message display */}
      {message && (
        <div
          className={`messageToast ${message.type}`}
          style={{
            position: "fixed",
            top: "20px",
            right: "20px",
            padding: "16px 24px",
            backgroundColor: message.type === "success" ? "#4caf50" : "#f44336",
            color: "white",
            borderRadius: "4px",
            boxShadow: "0 2px 8px rgba(0,0,0,0.2)",
            zIndex: 10000,
            maxWidth: "400px",
            fontSize: "14px",
            lineHeight: "1.5",
          }}
          role={message.type === "error" ? "alert" : "status"}
          aria-live={message.type === "error" ? "assertive" : "polite"}
          onClick={() => setMessage(null)}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "flex-start",
              gap: "12px",
            }}
          >
            <span>{message.text}</span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setMessage(null);
                }}
              aria-label={t("msg.closeNotification")}
              style={{
                background: "none",
                border: "none",
                color: "white",
                cursor: "pointer",
                fontSize: "18px",
                padding: "0",
                lineHeight: "1",
              }}
            >
              ×
            </button>
          </div>
        </div>
      )}

      {/* Global confirmation dialog */}
      {confirmDialog?.isOpen && (
        <div
          style={{
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: "rgba(0, 0, 0, 0.5)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 10100,
          }}
        >
          <div
            style={{
              backgroundColor: "white",
              padding: "24px",
              borderRadius: "8px",
              maxWidth: "500px",
              boxShadow: "0 4px 16px rgba(0,0,0,0.3)",
            }}
          >
            <h3 style={{ marginTop: 0, marginBottom: "16px" }}>{t("modal.confirm")}</h3>
            <p style={{ marginBottom: "24px", lineHeight: "1.5" }}>{confirmDialog.message}</p>
            <div style={{ display: "flex", gap: "12px", justifyContent: "flex-end" }}>
              <button onClick={handleConfirmNo} style={{ padding: "8px 16px" }}>
                {t("button.cancel")}
              </button>
              <button
                onClick={handleConfirmYes}
                style={{
                  padding: "8px 16px",
                  backgroundColor: "#f44336",
                  color: "white",
                  border: "none",
                  borderRadius: "4px",
                  cursor: "pointer",
                }}
              >
                {confirmDialog.confirmButtonLabel ?? t("msg.confirmDelete")}
              </button>
            </div>
          </div>
        </div>
      )}

      {authLoading ? (
        <div className="authShell">
          <section className="authCard">
            <div className="workspaceEyebrow">{t("auth.eyebrow")}</div>
            <h1>{t("label.loading")}</h1>
          </section>
        </div>
      ) : authRequired && !isAuthenticated ? (
        <LoginScreen
          username={loginUsername}
          password={loginPassword}
          rememberMe={loginRememberMe}
          pending={loginPending}
          error={loginError}
          sessionExpired={sessionExpired}
          onUsernameChange={setLoginUsername}
          onPasswordChange={setLoginPassword}
          onRememberMeChange={setLoginRememberMe}
          onSubmit={handleLogin}
          t={t}
        />
      ) : (
        <>
      <div className="appShell">
        <section className="workspaceCard">
          <div className="workspaceTopbar">
            <div className="workspaceHeading">
              <div className="workspaceEyebrow">{currentMeta.eyebrow}</div>
              <h1>{currentMeta.title}</h1>
            </div>
            {showMonthPicker && (
              <div className="monthpickers monthpickersTopbar">
                <select value={month} onChange={(e) => setMonth(parseInt(e.target.value))}>
                  {uiMonths.map((m, i) => (
                    <option key={m} value={i + 1}>
                      {m}
                    </option>
                  ))}
                </select>
                <select value={year} onChange={(e) => setYear(parseInt(e.target.value))}>
                  {[year - 1, year, year + 1].map((y) => (
                    <option key={y} value={y}>
                      {y}
                    </option>
                  ))}
                </select>
              </div>
            )}
            <div className="workspaceActions" aria-label={t("msg.systemSectionsNav")}>
              <button
                type="button"
                className="workspaceActionButton"
                onClick={() => setTab("settings")}
              >
                {t("button.filesAndCopies")}
              </button>
              {authRequired && !transportCapabilities.isDesktop && (
                <button
                  type="button"
                  className="workspaceActionButton"
                  onClick={() => void handleLogout()}
                >
                  {t("auth.logout")}
                </button>
              )}
            </div>
          </div>

          <nav className="tabs">
            <button
              className={tab === "dashboard" ? "active" : ""}
              onClick={() => setTab("dashboard")}
            >
              {t("tabs.dashboard")}
            </button>
            <button
              className={tab === "students" ? "active" : ""}
              onClick={() => setTab("students")}
            >
              {t("tabs.students")}
            </button>
            <button className={tab === "courses" ? "active" : ""} onClick={() => setTab("courses")}>
              {t("tabs.courses")}
            </button>
            <button
              className={tab === "enrollments" ? "active" : ""}
              onClick={() => setTab("enrollments")}
            >
              {t("tabs.enrollments")}
            </button>
            <button
              className={tab === "attendance" ? "active" : ""}
              onClick={() => setTab("attendance")}
            >
              {t("tabs.attendance")}
            </button>
            <button className={tab === "invoice" ? "active" : ""} onClick={() => setTab("invoice")}>
              {t("tabs.invoice")}
            </button>
            <button className={tab === "debtors" ? "active" : ""} onClick={() => setTab("debtors")}>
              {t("tabs.debtors")}
            </button>
            {canViewAuditLog && (
              <button className={tab === "audit" ? "active" : ""} onClick={() => setTab("audit")}>
                {t("tabs.audit")}
              </button>
            )}
            <button
              className={tab === "settings" ? "active" : ""}
              onClick={() => setTab("settings")}
            >
              {t("tabs.settings")}
            </button>
          </nav>

          {tab === "dashboard" && (
            <DashboardOverview
              overview={overview}
              loading={overviewLoading}
              monthLabel={currentMonthLabel}
              t={t}
              formatEUR={formatEUR}
              paymentMethodLabel={localizedPaymentMethodLabel}
              onOpenAttendance={() => setTab("attendance")}
              onOpenInvoices={() => setTab("invoice")}
              onOpenDebtors={() => setTab("debtors")}
              onOpenStudents={() => setTab("students")}
              onOpenStudent={(studentId) => void openStudentInWorkspaceById(studentId)}
              onOpenPaymentQueueStudent={(studentId) =>
                openDebtorPaymentModalByStudentId(studentId)
              }
              onCopyDebtQueueRu={(studentId) => void copyDebtMessageForStudentId(studentId, "ru")}
              onCopyDebtQueueLv={(studentId) => void copyDebtMessageForStudentId(studentId, "lv")}
              recentPayments={recentPayments}
              actionQueue={debtorActionQueue}
            />
          )}

          {/* ---------------- Students ---------------- */}
          {tab === "students" && (
            <>
              <StudentWorkspace
                students={studentList}
                loading={studentLoading}
                query={studentQ}
                includeInactive={includeInactive}
                selectedStudent={selectedStudentCard}
                detailLoading={studentCardLoading}
                detailEnrollments={studentCardEnrollments}
                detailBalance={studentCardBalance}
                detailDebts={studentCardDebts}
                detailPayments={studentCardPayments}
                detailMonthInvoices={studentCardMonthInvoices}
                detailNextAction={studentNextAction}
                detailActivity={studentActivity}
                deletingPaymentId={studentCardDeletingPaymentId}
                t={t}
                onQueryChange={setStudentQ}
                onIncludeInactiveChange={setIncludeInactive}
                onRefresh={() => void loadStudents()}
                onAddStudent={openAddStudent}
                onSelectStudent={(student) => void openStudentCard(student, { inline: true })}
                onEditStudent={openEditStudent}
                onToggleActive={(student) => void toggleStudentActive(student)}
                onDeleteStudent={(studentId) => void removeStudent(studentId)}
                onAddPayment={openStudentCardPaymentModal}
                onCopyDebtRu={() => void copyStudentCardDebtMessage("ru")}
                onCopyDebtLv={() => void copyStudentCardDebtMessage("lv")}
                onDeletePayment={deleteStudentPayment}
                onManageEnrollments={() => setTab("enrollments")}
                onOpenInvoices={() => setTab("invoice")}
                canDeleteStudent={canDeleteStudents}
                canDeletePayment={canDeletePayments}
                payerRoleLabel={localizedPayerRoleLabel}
                billingModeLabel={localizedBillingModeLabel}
                paymentMethodLabel={localizedPaymentMethodLabel}
                invoiceStatusLabel={localizedInvoiceStatusLabel}
                formatEUR={formatEUR}
                months={uiMonths}
              />

              {studentModalOpen && (
                <div className="modal">
                  <div className="modalBody">
                    <h3>{editingStudent ? t("modal.editStudent") : t("modal.addStudent")}</h3>
                    <div className="formRow">
                      <label>{t("field.name")}</label>
                      <input value={sfName} onChange={(e) => setSfName(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>{t("field.personalCode")}</label>
                      <input
                        value={sfPersonalCode}
                        onChange={(e) => setSfPersonalCode(e.target.value)}
                      />
                    </div>
                    <div className="formRow">
                      <label>{sfIsMinor ? t("student.parentPhone") : t("field.phone")}</label>
                      <input value={sfPhone} onChange={(e) => setSfPhone(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>{sfIsMinor ? t("student.parentEmail") : t("field.email")}</label>
                      <input value={sfEmail} onChange={(e) => setSfEmail(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>{t("field.note")}</label>
                      <input value={sfNote} onChange={(e) => setSfNote(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>{t("field.studentType")}</label>
                      <label className="inline">
                        <input
                          type="checkbox"
                          checked={sfIsMinor}
                          onChange={(e) => setSfIsMinor(e.target.checked)}
                        />
                        {t("student.minor")}
                      </label>
                    </div>
                    {sfIsMinor && (
                      <>
                        <div className="formRow">
                          <label>{t("field.payerName")}</label>
                          <input
                            value={sfPayerName}
                            onChange={(e) => setSfPayerName(e.target.value)}
                          />
                        </div>
                        <div className="formRow">
                          <label>{t("field.payerRole")}</label>
                          <select
                            value={sfPayerRole}
                            onChange={(e) => setSfPayerRole(e.target.value)}
                          >
                            <option value="">{t("filter.selectRole")}</option>
                            {payerRoleOptions.map((role) => (
                              <option key={role} value={role}>
                                {localizedPayerRoleLabel(role)}
                              </option>
                            ))}
                          </select>
                        </div>
                      </>
                    )}

                    <div className="modalActions">
                      <button onClick={saveStudent}>{t("button.save")}</button>
                      <button onClick={() => setStudentModalOpen(false)}>{t("button.cancel")}</button>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}

          {/* ---------------- Courses ---------------- */}
          {tab === "courses" && (
            <>
              <div className="controls">
                <button onClick={openAddCourse}>{t("button.addCourse")}</button>
                <input
                  className="searchField"
                  placeholder={t("msg.searchPlaceholderCourse")}
                  value={courseQ}
                  onChange={(e) => setCourseQ(e.target.value)}
                />
                <button onClick={loadCourses}>{t("button.refresh")}</button>
              </div>

              {courseLoading ? (
                <div>{t("label.loading")}</div>
              ) : courseList.length === 0 ? (
                <div className="empty">{t("msg.noCoursesYet")}</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>{t("field.name")}</th>
                      <th>{t("field.teacher")}</th>
                      <th>{t("field.type")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.lessonPrice")} (EUR)</th>
                      <th style={{ textAlign: "right" }}>{t("field.subscriptionPrice")} (EUR)</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {courseList.map((c) => (
                      <tr key={c.id}>
                        <td>{c.name}</td>
                        <td>{c.teacherName || "—"}</td>
                        <td>{localizedCourseTypeLabel(c.type)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(c.lessonPrice)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(c.subscriptionPrice)}</td>
                        <td>
                          <button onClick={() => openEditCourse(c)}>{t("button.edit")}</button>
                          {canDeleteCourses && (
                            <button onClick={() => removeCourse(c.id)}>{t("button.delete")}</button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {courseModalOpen && (
                <div className="modal">
                  <div className="modalBody">
                    <h3>{editingCourse ? t("modal.editCourse") : t("modal.addCourse")}</h3>

                    <div className="formRow">
                      <label>{t("field.name")}</label>
                      <input value={cfName} onChange={(e) => setCfName(e.target.value)} />
                    </div>

                    <div className="formRow">
                      <label>{t("field.teacher")}</label>
                      <div className="comboBox" ref={cfTeacherComboRef}>
                        <input
                          value={selectedCourseTeacher?.fullName ?? cfTeacherSearch}
                          onChange={(e) => {
                            setCfTeacherSearch(e.target.value);
                            setCfTeacherId(undefined);
                            setCfTeacherPickerOpen(true);
                          }}
                          onFocus={() => setCfTeacherPickerOpen(true)}
                          onKeyDown={(e) => {
                            if (e.key === "Escape") {
                              setCfTeacherPickerOpen(false);
                            }
                          }}
                          placeholder={t("filter.selectTeacher")}
                        />
                        {cfTeacherPickerOpen && (
                          <div className="comboBoxMenu">
                            {filteredTeachers.map((t) => (
                              <button
                                key={t.id}
                                type="button"
                                className={`comboBoxOption ${t.id === cfTeacherId ? "active" : ""}`}
                                onClick={() => {
                                  setCfTeacherId(t.id);
                                  setCfTeacherSearch(t.fullName);
                                  setCfTeacherPickerOpen(false);
                                }}
                              >
                                <span className="comboBoxPrimary">{t.fullName}</span>
                              </button>
                            ))}
                            {!exactTeacherMatch && cfTeacherSearch.trim() && (
                              <button
                                type="button"
                                className="comboBoxOption"
                                onClick={() => void addTeacherFromCourseForm()}
                                disabled={cfTeacherCreating}
                              >
                                <span className="comboBoxPrimary">
                                  {cfTeacherCreating
                                    ? `${t("field.teacher")}...`
                                    : `${t("button.addCourse")}: ${cfTeacherSearch.trim()}`}
                                </span>
                                <span className="comboBoxMeta">
                                  {t("field.teacher")}
                                </span>
                              </button>
                            )}
                            {filteredTeachers.length === 0 && !cfTeacherSearch.trim() && (
                              <div className="comboBoxEmpty">{t("msg.noTeachers")}</div>
                            )}
                          </div>
                        )}
                      </div>
                    </div>

                    <div className="formRow">
                      <label>{t("field.type")}</label>
                      <select value={cfType} onChange={(e) => setCfType(e.target.value as any)}>
                        <option value="group">{t("course.group")}</option>
                        <option value="individual">{t("course.individual")}</option>
                      </select>
                    </div>

                    <div className="formRow">
                      <label>{t("field.lessonPrice")} (EUR)</label>
                      <input
                        type="text"
                        inputMode="decimal"
                        min={0}
                        step="0.01"
                        value={cfLessonPrice}
                        onChange={(e) => handleCoursePriceChange(e.target.value, setCfLessonPrice)}
                      />
                    </div>

                    <div className="formRow">
                      <label>{t("field.subscriptionPrice")} (EUR)</label>
                      <input
                        type="text"
                        inputMode="decimal"
                        min={0}
                        step="0.01"
                        value={cfSubscriptionPrice}
                        onChange={(e) =>
                          handleCoursePriceChange(e.target.value, setCfSubscriptionPrice)
                        }
                      />
                    </div>

                    <div className="modalActions">
                      <button onClick={saveCourse}>{t("button.save")}</button>
                      <button onClick={() => setCourseModalOpen(false)}>{t("button.cancel")}</button>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}

          {/* ---------------- Enrollments ---------------- */}
          {tab === "enrollments" && (
            <>
              <div className="controls">
                <button onClick={openAddEnrollment}>{t("button.addEnrollment")}</button>

                <select
                  value={enrStudentFilter ?? ""}
                  onChange={(e) => setEnrStudentFilter(intOrUndef(e.target.value))}
                >
                  <option value="">{t("filter.allStudents")}</option>
                  {allStudents.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.fullName}
                    </option>
                  ))}
                </select>

                <select
                  value={enrCourseFilter ?? ""}
                  onChange={(e) => setEnrCourseFilter(intOrUndef(e.target.value))}
                >
                  <option value="">{t("filter.allCourses")}</option>
                  {allCourses.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.teacherName ? `${c.name} — ${c.teacherName}` : c.name}
                    </option>
                  ))}
                </select>

                <button onClick={loadEnrollments}>{t("button.refresh")}</button>
              </div>

              {enrLoading ? (
                <div>{t("label.loading")}</div>
              ) : enrollments.length === 0 ? (
                <div className="empty">{t("msg.noEnrollmentsYet")}</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>{t("field.student")}</th>
                      <th>{t("field.course")}</th>
                      <th>{t("field.teacher")}</th>
                      <th>{t("field.billing")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.discount")}</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {enrollments.map((e) => (
                      <tr key={e.id}>
                        <td>
                          <button
                            className="linkButton"
                            onClick={() => void openStudentCardById(e.studentId)}
                          >
                            {e.studentName}
                          </button>
                        </td>
                        <td>{e.courseName}</td>
                        <td>{e.teacherName || "—"}</td>
                        <td>{localizedBillingModeLabel(e.billingMode)}</td>
                        <td style={{ textAlign: "right" }}>{e.discountPct.toFixed(1)}%</td>
                        <td>
                          <button onClick={() => openEditEnrollment(e)}>{t("button.edit")}</button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {enrModalOpen && (
                <div className="modal">
                  <div className="modalBody">
                    <h3>{editingEnr ? t("modal.editEnrollment") : t("modal.addEnrollment")}</h3>

                    <div className="formRow">
                      <label>{t("field.student")}</label>
                      {editingEnr ? (
                        <input
                          value={selectedEnrollmentStudent?.fullName ?? efStudentSearch}
                          disabled
                        />
                      ) : (
                        <div className="comboBox" ref={efStudentComboRef}>
                          <input
                            value={efStudentSearch}
                            onChange={(e) => {
                              setEfStudentSearch(e.target.value);
                              setEfStudentPickerOpen(true);
                            }}
                            onFocus={() => setEfStudentPickerOpen(true)}
                            onKeyDown={(e) => {
                              if (e.key === "Escape") {
                                setEfStudentPickerOpen(false);
                              }
                            }}
                            placeholder={t("msg.searchPlaceholderStudent")}
                          />
                          {efStudentPickerOpen && (
                            <div className="comboBoxMenu">
                              {filteredEnrollmentStudents.length === 0 ? (
                                <div className="comboBoxEmpty">{t("msg.noStudentsFound")}</div>
                              ) : (
                                filteredEnrollmentStudents.map((s) => (
                                  <button
                                    key={s.id}
                                    type="button"
                                    className={`comboBoxOption ${s.id === efStudentId ? "active" : ""}`}
                                    onClick={() => {
                                      setEfStudentId(s.id);
                                      setEfStudentSearch(s.fullName);
                                      setEfStudentPickerOpen(false);
                                    }}
                                  >
                                    <span className="comboBoxPrimary">{s.fullName}</span>
                                    <span className="comboBoxMeta">
                                      {[s.phone, s.email].filter(Boolean).join(" · ")}
                                    </span>
                                  </button>
                                ))
                              )}
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    <div className="formRow">
                      <label>{t("field.course")}</label>
                      <select
                        value={efCourseId}
                        disabled={!!editingEnr}
                        onChange={(e) => setEfCourseId(parseInt(e.target.value))}
                      >
                        {allCourses.map((c) => (
                          <option key={c.id} value={c.id}>
                            {c.teacherName ? `${c.name} — ${c.teacherName}` : c.name}
                          </option>
                        ))}
                      </select>
                    </div>

                    <div className="formRow">
                      <label>{t("field.billing")}</label>
                      <select value={efMode} onChange={(e) => setEfMode(e.target.value as any)}>
                        <option value="per_lesson">{t("billing.perLesson")}</option>
                        <option value="subscription">{t("billing.subscription")}</option>
                      </select>
                    </div>

                    <div className="formRow">
                      <label>{t("field.discount")} %</label>
                      <input
                        type="number"
                        min={0}
                        max={100}
                        step="0.1"
                        value={efDiscount}
                        onChange={(e) => setEfDiscount(numOrZero(e.target.value))}
                      />
                    </div>

                    {efMode === "subscription" && (
                      <div className="formRow">
                        <label>{t("field.subscriptionDiscount")} %</label>
                        <input
                          type="number"
                          min={0}
                          max={100}
                          step="0.1"
                          value={efSubscriptionDiscount}
                          onChange={(e) => setEfSubscriptionDiscount(numOrZero(e.target.value))}
                        />
                      </div>
                    )}

                    <div className="formRow">
                      <label>{t("field.note")}</label>
                      <input value={efNote} onChange={(e) => setEfNote(e.target.value)} />
                    </div>

                    <div className="modalActions">
                      <button onClick={saveEnrollment}>{t("button.save")}</button>
                      <button onClick={() => setEnrModalOpen(false)}>{t("button.cancel")}</button>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}

          {/* ---------------- Attendance ---------------- */}
          {tab === "attendance" && (
            <>
              <div className="sectionBanner">
                <div>
                  <div className="dashboardCardEyebrow">{t("msg.monthStatus")}</div>
                  <strong>
                    {attendanceSummary.missing > 0
                      ? t("msg.monthStatusMissing", { count: attendanceSummary.missing })
                      : attendanceSummary.total > 0
                        ? t("msg.monthStatusDone")
                        : t("msg.monthStatusEmpty")}
                  </strong>
                </div>
                <div className="sectionBannerActions">
                  <button className="workspaceActionButton" onClick={() => void loadAttendance()}>
                    {t("msg.refreshSheet")}
                  </button>
                  <button
                    className="workspaceActionButton workspaceActionButtonPrimary"
                    onClick={() => setTab("invoice")}
                    disabled={attendanceSummary.total === 0}
                  >
                    {t("msg.openMonthInvoices")}
                  </button>
                </div>
              </div>

              <div className="controls">
                <select
                  value={courseFilter ?? ""}
                  onChange={(e) => setCourseFilter(intOrUndef(e.target.value))}
                >
                  <option value="">{t("filter.allGroups")}</option>
                  {allCourses.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.teacherName ? `${c.name} — ${c.teacherName}` : c.name}
                    </option>
                  ))}
                </select>

                <input
                  className="searchField"
                  placeholder={t("msg.searchPlaceholderAttendance")}
                  value={attQ}
                  onChange={(e) => setAttQ(e.target.value)}
                />

                <select
                  value={attFilter}
                  onChange={(e) => setAttFilter(e.target.value as typeof attFilter)}
                >
                  <option value="all">{t("status.showAll")}</option>
                  <option value="missing">{t("status.onlyMissing")}</option>
                  <option value="filled">{t("status.onlyFilled")}</option>
                  <option value="zero">{t("status.zeroLessons")}</option>
                </select>

                <button onClick={loadAttendance}>{t("button.refresh")}</button>
              </div>

              {rows.length > 0 && (
                <div className="attSummary">
                  {t("msg.attFilled")}: {attendanceSummary.filled} / {attendanceSummary.total}
                  &nbsp;·&nbsp;{t("msg.attMissing")}: {attendanceSummary.missing}
                  &nbsp;·&nbsp;{t("msg.attZero")}: {attendanceSummary.zero}
                </div>
              )}

              {loadingAtt ? (
                <div>{t("label.loading")}</div>
              ) : filteredAttendanceRows.length === 0 ? (
                <div className="empty">
                  {attQ.trim() || attFilter !== "all"
                    ? t("msg.noSearchResults")
                    : t("msg.noAttendanceRows")}
                </div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>{t("field.student")}</th>
                      <th>{t("field.course")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.lessonPrice")} (EUR)</th>
                      <th style={{ textAlign: "right" }}>{t("field.quantity")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.totalEur")}</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredAttendanceRows.map((r) => (
                      <tr key={r.enrollmentId}>
                        <td>
                          <button
                            className="linkButton"
                            onClick={() => void openStudentCardById(r.studentId)}
                          >
                            {r.studentName}
                          </button>
                        </td>
                        <td>
                          {r.courseName} ({localizedCourseTypeLabel(r.courseType)})
                          {r.billingMode === BillingModeSubscription && (
                            <>
                              {" "}
                              <span className="attBadge attBadge--subscription">{t("billing.subscription")}</span>
                            </>
                          )}
                        </td>
                        <td style={{ textAlign: "right" }}>
                          {formatEUR(r.lessonPrice)}
                        </td>
                        <td style={{ textAlign: "right" }}>
                          {r.billingMode === BillingModePerLesson && !r.hasRecord && (
                            <span className="attBadge attBadge--missing">{t("msg.attMissing")}</span>
                          )}
                          {r.billingMode === BillingModePerLesson &&
                            r.hasRecord &&
                            r.hours === 0 && (
                              <span className="attBadge attBadge--zero">0h</span>
                            )}
                          {r.billingMode === BillingModePerLesson && !r.attendanceLocked ? (
                            <div className="attendanceStepper">
                              <button
                                type="button"
                                className="attendanceStepperButton"
                                onClick={() =>
                                  onChangeHours(r, Math.max(0, getAttendanceStepBase(r) - 1))
                                }
                                disabled={
                                  attendanceSavingRows[r.enrollmentId] ||
                                  getAttendanceStepBase(r) <= 0
                                }
                                aria-label={`Decrease hours for ${r.studentName}`}
                              >
                                −
                              </button>
                              <input
                                type="text"
                                inputMode="decimal"
                                value={getAttendanceInputValue(r)}
                                disabled={attendanceSavingRows[r.enrollmentId]}
                                onChange={(e) => {
                                  const nextValue = normalizeHoursDraftInput(e.target.value);
                                  if (nextValue !== null) {
                                    setAttendanceDraft(r.enrollmentId, nextValue);
                                  }
                                }}
                                onPointerDown={() => {
                                  attendancePendingSelectRef.current = r.enrollmentId;
                                }}
                                onFocus={(e) => {
                                  if (attendancePendingSelectRef.current !== r.enrollmentId) {
                                    e.currentTarget.select();
                                  }
                                }}
                                onMouseUp={(e) => {
                                  if (attendancePendingSelectRef.current === r.enrollmentId) {
                                    e.preventDefault();
                                    e.currentTarget.select();
                                    attendancePendingSelectRef.current = null;
                                  }
                                }}
                                onBlur={() => {
                                  if (attendancePendingSelectRef.current === r.enrollmentId) {
                                    attendancePendingSelectRef.current = null;
                                  }
                                  void commitAttendanceDraft(r);
                                }}
                                onKeyDown={(e) => {
                                  if (e.key === "Enter") {
                                    e.preventDefault();
                                    void commitAttendanceDraft(r);
                                  }
                                  if (e.key === "Escape") {
                                    e.preventDefault();
                                    clearAttendanceDraft(r.enrollmentId);
                                    e.currentTarget.blur();
                                  }
                                }}
                                className="attendanceStepperInput"
                                aria-label={`Hours for ${r.studentName}`}
                              />
                              <button
                                type="button"
                                className="attendanceStepperButton"
                                onClick={() => onChangeHours(r, getAttendanceStepBase(r) + 1)}
                                disabled={attendanceSavingRows[r.enrollmentId]}
                                aria-label={`Increase hours for ${r.studentName}`}
                              >
                                +
                              </button>
                            </div>
                          ) : r.billingMode === BillingModeSubscription ? (
                            subscriptionLeadEnrollmentIds.has(r.enrollmentId) ? (
                              <div className="attendanceStepper">
                                <button
                                  type="button"
                                  className="attendanceStepperButton"
                                  onClick={() => {
                                    const current = subscriptionMonthLessons[r.courseId] ?? 0;
                                    void saveCourseMonthSubscriptionLessons(
                                      r.courseId,
                                      year,
                                      month,
                                      Math.max(0, current - 1)
                                    )
                                      .then((updated) => {
                                        setSubscriptionMonthLessons((prev) => ({
                                          ...prev,
                                          [r.courseId]: updated.lessonsHeld,
                                        }));
                                        return loadAttendance();
                                      })
                                      .catch((e: any) => {
                                        showMessage(
                                          t("msg.errorGeneric", {
                                            message: String(e?.message ?? e),
                                          }),
                                          "error"
                                        );
                                      });
                                  }}
                                  disabled={
                                    subscriptionMonthSaving[r.courseId] ||
                                    (subscriptionMonthLessons[r.courseId] ?? 0) <= 0
                                  }
                                  aria-label={`Decrease subscription lessons for ${r.courseName}`}
                                >
                                  −
                                </button>
                                <input
                                  type="text"
                                  inputMode="decimal"
                                  value={getSubscriptionMonthLessonsValue(r.courseId)}
                                  disabled={subscriptionMonthSaving[r.courseId]}
                                  onChange={(e) => {
                                    const nextValue = normalizeHoursDraftInput(e.target.value);
                                    if (nextValue !== null) {
                                      setSubscriptionMonthLessonsDraft(r.courseId, nextValue);
                                    }
                                  }}
                                  onFocus={(e) => e.currentTarget.select()}
                                  onBlur={() => {
                                    void commitSubscriptionMonthLessonsDraft(r);
                                  }}
                                  onKeyDown={(e) => {
                                    if (e.key === "Enter") {
                                      e.preventDefault();
                                      void commitSubscriptionMonthLessonsDraft(r);
                                    }
                                    if (e.key === "Escape") {
                                      e.preventDefault();
                                      clearSubscriptionMonthLessonsDraft(r.courseId);
                                      e.currentTarget.blur();
                                    }
                                  }}
                                  className="attendanceStepperInput"
                                  aria-label={`Subscription lessons held for ${r.courseName}`}
                                />
                                <button
                                  type="button"
                                  className="attendanceStepperButton"
                                  onClick={() => {
                                    const current = subscriptionMonthLessons[r.courseId] ?? 0;
                                    void saveCourseMonthSubscriptionLessons(
                                      r.courseId,
                                      year,
                                      month,
                                      current + 1
                                    )
                                      .then((updated) => {
                                        setSubscriptionMonthLessons((prev) => ({
                                          ...prev,
                                          [r.courseId]: updated.lessonsHeld,
                                        }));
                                        return loadAttendance();
                                      })
                                      .catch((e: any) => {
                                        showMessage(
                                          t("msg.errorGeneric", {
                                            message: String(e?.message ?? e),
                                          }),
                                          "error"
                                        );
                                      });
                                  }}
                                  disabled={subscriptionMonthSaving[r.courseId]}
                                  aria-label={`Increase subscription lessons for ${r.courseName}`}
                                >
                                  +
                                </button>
                              </div>
                            ) : (
                              <div className="attendanceReadOnly">
                                <span className="attBadge attBadge--subscription">
                                  {t("msg.readOnly")}
                                </span>
                                <span className="mutedInline">{t("msg.subscriptionSharedValue")}</span>
                              </div>
                            )
                          ) : (
                            <div className="attendanceReadOnly">
                              <span className="attBadge attBadge--subscription">{t("msg.readOnly")}</span>
                              <span className="mutedInline">
                                {r.invoiceStatus === InvoiceStatusIssued
                                  ? t("msg.lockedIssuedInvoice")
                                  : r.invoiceStatus === InvoiceStatusPaid
                                    ? t("msg.lockedPaidInvoice")
                                    : r.invoiceStatus === InvoiceStatusCanceled
                                      ? t("msg.lockedCanceledInvoice")
                                      : t("msg.lockedUntilDraft")}
                              </span>
                            </div>
                          )}
                        </td>
                        <td style={{ textAlign: "right" }}>
                          {r.billingMode === BillingModePerLesson
                            ? formatEUR(r.hours * r.lessonPrice)
                            : formatEUR(subscriptionTotal(r, subscriptionMonthLessons[r.courseId] ?? 0))}
                        </td>
                        <td>
                          {r.billingMode === BillingModePerLesson &&
                            !r.attendanceLocked &&
                            !r.hasRecord && (
                              <button
                                onClick={() => onChangeHours(r, 0)}
                                disabled={attendanceSavingRows[r.enrollmentId]}
                                style={{ marginRight: "0.5rem" }}
                              >
                                {t("msg.setZeroHours")}
                              </button>
                            )}
                          {r.canDelete ? (
                            <button onClick={() => onDeleteEnrollmentFromSheet(r.enrollmentId)}>
                              {t("msg.deleteEnrollment")}
                            </button>
                          ) : (
                            <span className="mutedInline">
                              {t("msg.deleteEnrollmentBlocked")}
                            </span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <td colSpan={4} style={{ textAlign: "right" }}>
                        {t("msg.lessonsTotalEur")}:
                      </td>
                      <td style={{ textAlign: "right" }}>{formatEUR(perLessonTotal)}</td>
                      <td></td>
                    </tr>
                  </tfoot>
                </table>
              )}
            </>
          )}

          {/* ---------------- Invoices ---------------- */}
          {tab === "invoice" && (
            <>
              <div className="sectionBanner">
                <div>
                  <div className="dashboardCardEyebrow">{t("msg.billing")}</div>
                  <strong>{currentMonthLabel}</strong>
                  <span className="mutedInline">
                    {t("title.invoice")}
                  </span>
                </div>
                <div className="sectionBannerActions">
                  <button className="workspaceActionButton" onClick={() => void loadInvoices()}>
                    {t("button.sync")}
                  </button>
                </div>
              </div>

              <div className="controls">
                <select value={invStatus} onChange={(e) => setInvStatus(e.target.value)}>
                  <option value="draft">{t("filter.selectStatusDraft")}</option>
                  <option value="issued">{t("filter.selectStatusIssued")}</option>
                  <option value="paid">{t("filter.selectStatusPaid")}</option>
                  <option value="all">{t("filter.selectStatusAll")}</option>
                </select>

                <input
                  className="searchField searchFieldWide"
                  placeholder={t("msg.searchPlaceholderInvoice")}
                  value={invQ}
                  onChange={(e) => setInvQ(e.target.value)}
                />

                <button onClick={() => void loadInvoices()}>{t("button.refresh")}</button>
              </div>

              {loadingInv ? (
                <div>{t("label.loading")}</div>
              ) : filteredInvItems.length === 0 ? (
                <div className="empty">{t("msg.noInvoiceResults")}</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>{t("field.student")}</th>
                      <th>{t("field.period")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.amount")} (EUR)</th>
                      <th>{t("field.status")}</th>
                      <th>{t("field.number")}</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredInvItems.map((it) => (
                      <tr key={it.id}>
                        <td>
                          <button
                            className="linkButton"
                            onClick={() => void openStudentCardById(it.studentId)}
                          >
                            {it.studentName}
                          </button>
                        </td>
                        <td>
                          {uiMonths[it.month - 1]} {it.year}
                        </td>
                        <td style={{ textAlign: "right" }}>{formatEUR(it.total)}</td>
                        <td>
                          <span className={`statusPill statusPill--${it.status}`}>
                            {localizedInvoiceStatusLabel(it.status)}
                          </span>
                        </td>
                        <td>
                          {it.number ?? ""}
                          {it.pdfReady && (
                            <div className="badgeRow">
                              <span className="attBadge attBadge--pdfReady">PDF</span>
                            </div>
                          )}
                        </td>
                        <td>
                          <div className="invoiceRowActions">
                            <button onClick={() => onOpenInvoice(it.id)}>{t("button.open")}</button>
                            {it.status === "draft" && (
                              <button
                                className="workspaceActionButtonPrimary workspaceActionButton invoicePrimaryAction"
                                onClick={() => onIssueOne(it.id)}
                              >
                                {t("button.issue")}
                              </button>
                            )}
                            {it.status !== "draft" && (
                              <button onClick={() => void onDownloadPdf(it.id)}>
                                {t("button.downloadPdf")}
                              </button>
                            )}
                            {it.status !== "draft" && (
                              <button onClick={() => void openPaymentModalForInvoice(it.id)}>
                                {t("button.recordPayment")}
                              </button>
                            )}
                            {renderInvoiceActionsMenu(it)}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </>
          )}

          {/* ---------------- Debtors ---------------- */}
          {tab === "debtors" && (
            <>
              <div className="sectionBanner">
                <div>
                  <div className="dashboardCardEyebrow">{t("msg.collection")}</div>
                  <strong>
                    {t("label.needsAction")}
                  </strong>
                </div>
                <div className="sectionBannerActions">
                  <button className="workspaceActionButton" onClick={loadDebtors}>
                    {t("button.refresh")}
                  </button>
                </div>
              </div>

              {debtorActionQueue.length > 0 && (
                <div className="detailCard detailCard--wide actionQueuePanel">
                  <div className="detailCardHeader">
                    <h3>{t("label.needsAction")}</h3>
                    <span className="statusPill warning">{t("msg.queueCount", { count: debtorActionQueue.length })}</span>
                  </div>
                  <div className="actionQueueList">
                    {debtorActionQueue.map((item) => (
                      <div key={item.studentId} className="actionQueueItem">
                        <div>
                          <strong>{item.studentName}</strong>
                          <span>{item.subtitle}</span>
                        </div>
                        <div className="actionQueueMeta">
                          <strong>{formatEUR(item.debt)}</strong>
                          <div className="inlineActions">
                            <button
                              className="workspaceActionButton workspaceActionButtonPrimary"
                              onClick={() => openDebtorPaymentModalByStudentId(item.studentId)}
                            >
                              {t("button.takePayment")}
                            </button>
                            <button
                              className="secondaryActionButton"
                              onClick={() => void openStudentInWorkspaceById(item.studentId)}
                            >
                              {t("button.card")}
                            </button>
                            <button
                              className="secondaryActionButton"
                              onClick={() => void copyDebtMessageForStudentId(item.studentId, "ru")}
                            >
                              RU
                            </button>
                            <button
                              className="secondaryActionButton"
                              onClick={() => void copyDebtMessageForStudentId(item.studentId, "lv")}
                            >
                              LV
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {debtorsLoading ? (
                <div>{t("label.loading")}</div>
              ) : debtors.length === 0 ? (
                <div className="empty">{t("msg.noDebtors")}</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>{t("field.student")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.debtEur")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.totalEur")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.paidEur")}</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {debtors.map((d) => (
                      <tr key={d.studentId}>
                        <td>
                          <button
                            className="linkButton"
                            onClick={() => void openStudentCardById(d.studentId)}
                          >
                            {d.studentName}
                          </button>
                        </td>
                        <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                          {formatEUR(d.debt)}
                        </td>
                        <td style={{ textAlign: "right" }}>{formatEUR(d.totalInvoiced)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(d.totalPaid)}</td>
                        <td>
                          <button
                            className="workspaceActionButton workspaceActionButtonPrimary"
                            onClick={() => openDebtorPaymentModal(d)}
                          >
                            {t("button.takePayment")}
                          </button>
                          <button onClick={() => openDebtDetails(d)}>{t("modal.debtBreakdown")}</button>
                          <button onClick={() => void copyDebtMessageForDebtor(d, "ru")}>
                            {t("button.copyRu")}
                          </button>
                          <button onClick={() => void copyDebtMessageForDebtor(d, "lv")}>
                            {t("button.copyLv")}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <td style={{ fontWeight: "bold" }}>{t("field.debtEur")}:</td>
                      <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                        {formatEUR(debtors.reduce((sum, d) => sum + d.debt, 0))}
                      </td>
                      <td colSpan={3}></td>
                    </tr>
                  </tfoot>
                </table>
              )}
            </>
          )}

          {tab === "audit" && canViewAuditLog && (
            <>
              <div className="sectionBanner">
                <div>
                  <div className="dashboardCardEyebrow">{t("eyebrow.audit")}</div>
                  <strong>{t("title.audit")}</strong>
                  <span className="mutedInline">{t("audit.subtitle")}</span>
                </div>
                <div className="sectionBannerActions">
                  <button className="workspaceActionButton" onClick={() => void loadAuditLog()}>
                    {t("button.refresh")}
                  </button>
                </div>
              </div>

              <div className="controls controlsWrap">
                <input
                  className="searchField searchFieldWide"
                  placeholder={t("audit.searchPlaceholder")}
                  value={auditQ}
                  onChange={(e) => setAuditQ(e.target.value)}
                />
                <input
                  className="searchField"
                  placeholder={t("audit.actorPlaceholder")}
                  value={auditActorFilter}
                  onChange={(e) => setAuditActorFilter(e.target.value)}
                />
                <select
                  value={auditEntityTypeFilter}
                  onChange={(e) => setAuditEntityTypeFilter(e.target.value)}
                >
                  <option value="">{t("audit.allEntities")}</option>
                  <option value="invoice">{t("tabs.invoice")}</option>
                  <option value="payment">{t("field.payment")}</option>
                  <option value="invoice_batch">{t("audit.batchEntity")}</option>
                </select>
                <select value={auditActionFilter} onChange={(e) => setAuditActionFilter(e.target.value)}>
                  <option value="">{t("audit.allActions")}</option>
                  <option value="invoice.generate_drafts">invoice.generate_drafts</option>
                  <option value="invoice.rebuild_student_draft">invoice.rebuild_student_draft</option>
                  <option value="invoice.delete_draft">invoice.delete_draft</option>
                  <option value="invoice.reopen_draft">invoice.reopen_draft</option>
                  <option value="invoice.issue">invoice.issue</option>
                  <option value="invoice.issue_all">invoice.issue_all</option>
                  <option value="payment.create">payment.create</option>
                  <option value="payment.allocate_or_credit">payment.allocate_or_credit</option>
                  <option value="payment.delete">payment.delete</option>
                </select>
                <input type="date" value={auditDateFrom} onChange={(e) => setAuditDateFrom(e.target.value)} />
                <input type="date" value={auditDateTo} onChange={(e) => setAuditDateTo(e.target.value)} />
                <button
                  onClick={() => {
                    setAuditPage(1);
                    void loadAuditLog();
                  }}
                >
                  {t("button.refresh")}
                </button>
              </div>

              {auditLoading ? (
                <div>{t("label.loading")}</div>
              ) : auditItems.length === 0 ? (
                <div className="empty">{t("audit.empty")}</div>
              ) : (
                <>
                  <table>
                    <thead>
                      <tr>
                        <th>{t("field.date")}</th>
                        <th>{t("field.user")}</th>
                        <th>{t("field.entity")}</th>
                        <th>{t("field.action")}</th>
                        <th>{t("field.summary")}</th>
                        <th></th>
                      </tr>
                    </thead>
                    <tbody>
                      {auditItems.map((item) => (
                        <Fragment key={item.id}>
                          <tr key={item.id}>
                            <td>{new Date(item.createdAt).toLocaleString()}</td>
                            <td>{item.actorLabel || "system"}</td>
                            <td>
                              {item.entityType}
                              {typeof item.entityId === "number" ? ` #${item.entityId}` : ""}
                            </td>
                            <td>{auditActionLabel(item.action)}</td>
                            <td>{item.summary}</td>
                            <td>
                              <button
                                onClick={() =>
                                  setAuditExpandedId((current) => (current === item.id ? null : item.id))
                                }
                              >
                                {auditExpandedId === item.id ? t("button.hide") : t("button.open")}
                              </button>
                            </td>
                          </tr>
                          {auditExpandedId === item.id && (
                            <tr>
                              <td colSpan={6}>
                                <div className="auditDetails">
                                  <div className="auditDetailMeta">
                                    <span>{t("field.studentId")}: {item.studentId ?? "—"}</span>
                                    <span>{t("field.invoiceId")}: {item.invoiceId ?? "—"}</span>
                                  </div>
                                  <div className="auditJsonGrid">
                                    <div>
                                      <h4>{t("audit.before")}</h4>
                                      <pre>{item.beforeJson || "{}"}</pre>
                                    </div>
                                    <div>
                                      <h4>{t("audit.after")}</h4>
                                      <pre>{item.afterJson || "{}"}</pre>
                                    </div>
                                  </div>
                                </div>
                              </td>
                            </tr>
                          )}
                        </Fragment>
                      ))}
                    </tbody>
                  </table>

                  <div className="auditPager">
                    <span>{t("audit.totalRows", { count: auditTotal })}</span>
                    <div className="inlineActions">
                      <button
                        disabled={auditPage <= 1}
                        onClick={() => setAuditPage((page) => Math.max(1, page - 1))}
                      >
                        {t("button.prev")}
                      </button>
                      <span>{t("audit.pageLabel", { page: auditPage })}</span>
                      <button
                        disabled={auditPage * auditPageSize >= auditTotal}
                        onClick={() => setAuditPage((page) => page + 1)}
                      >
                        {t("button.next")}
                      </button>
                    </div>
                  </div>
                </>
              )}
            </>
          )}

          {tab === "settings" && (
            <div className="settingsGrid">
              <section className="detailCard">
                <div className="detailCardHeader">
                  <h3>{t("settings.languageTitle")}</h3>
                </div>
                <p className="mutedInline">{t("settings.languageDesc")}</p>
                <div className="formRow">
                  <label>{t("settings.locale")}</label>
                  <select
                    value={uiLocale}
                    disabled={!canManageSettings}
                    onChange={(e) => void handleLocaleChange(e.target.value as UiLocale)}
                  >
                    <option value="en-US">{t("settings.languageEnglish")}</option>
                    <option value="ru-RU">{t("settings.languageRussian")}</option>
                  </select>
                </div>
              </section>

              <section className="detailCard">
                <div className="detailCardHeader">
                  <h3>{t("settings.backupsTitle")}</h3>
                </div>
                <p className="mutedInline">{t("settings.backupDesc")}</p>
                <div className="settingsActions">
                  <button
                    type="button"
                    className="workspaceActionButton workspaceActionButtonPrimary"
                    onClick={() => void createManualBackup()}
                    disabled={creatingBackup || !canCreateBackups}
                  >
                    {creatingBackup ? `${t("button.createBackup")}...` : t("button.createBackup")}
                  </button>
                  {canShowInvoiceFolderAction(transportCapabilities) && (
                    <button
                      type="button"
                      className="workspaceActionButton"
                      onClick={() => void openAppFolder(appDirs?.backups, t("field.backups").toLowerCase())}
                      disabled={!appDirs?.backups}
                    >
                      {t("button.backupsFolder")}
                    </button>
                  )}
                </div>
              </section>

              {canShowSettingsFilesCard(transportCapabilities) && (
                <section className="detailCard">
                  <div className="detailCardHeader">
                    <h3>{t("settings.filesTitle")}</h3>
                  </div>
                  <p className="mutedInline">{t("settings.filesDesc")}</p>
                  <div className="settingsActions">
                    <button
                      type="button"
                      className="workspaceActionButton"
                      onClick={() => void openAppFolder(appDirs?.invoices, t("tabs.invoice").toLowerCase())}
                      disabled={!appDirs?.invoices}
                    >
                      {t("button.invoicesFolder")}
                    </button>
                    <button
                      type="button"
                      className="workspaceActionButton"
                      onClick={() => void openAppFolder(appDirs?.exports, "exports")}
                      disabled={!appDirs?.exports}
                    >
                      {t("button.exportsFolder")}
                    </button>
                    <button
                      type="button"
                      className="workspaceActionButton"
                      onClick={() => void openAppFolder(appDirs?.data, "data")}
                      disabled={!appDirs?.data}
                    >
                      {t("button.dataFolder")}
                    </button>
                  </div>
                </section>
              )}

              {canManageUsers && (
                <section className="detailCard detailCard--wide">
                  <div className="detailCardHeader">
                    <h3>Users</h3>
                  </div>
                  <p className="mutedInline">Manage admin and staff accounts for the web app.</p>

                  <div className="formRow">
                    <label>Username</label>
                    <input value={newUserUsername} onChange={(e) => setNewUserUsername(e.target.value)} />
                  </div>
                  <div className="formRow">
                    <label>Password</label>
                    <input
                      type="password"
                      value={newUserPassword}
                      onChange={(e) => setNewUserPassword(e.target.value)}
                    />
                  </div>
                  <div className="formRow">
                    <label>Role</label>
                    <select value={newUserRole} onChange={(e) => setNewUserRole(e.target.value)}>
                      <option value="staff">staff</option>
                      <option value="admin">admin</option>
                    </select>
                  </div>
                  <div className="settingsActions">
                    <button
                      type="button"
                      className="workspaceActionButton workspaceActionButtonPrimary"
                      onClick={() => void handleCreateUser()}
                      disabled={creatingUser}
                    >
                      {creatingUser ? "Create..." : "Create user"}
                    </button>
                    <button type="button" className="workspaceActionButton" onClick={() => void loadUsers()}>
                      {t("button.refresh")}
                    </button>
                  </div>

                  {usersLoading ? (
                    <div className="empty">{t("label.loading")}</div>
                  ) : (
                    <div className="tableWrap">
                      <table>
                        <thead>
                          <tr>
                            <th>Username</th>
                            <th>Role</th>
                            <th>Active</th>
                            <th>Password reset</th>
                            <th>{t("field.actions")}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {users.map((user) => {
                            const draft = userDrafts[user.id] ?? {
                              username: user.username,
                              role: user.role,
                              isActive: user.isActive,
                            };
                            return (
                              <tr key={user.id}>
                                <td>
                                  <input
                                    value={draft.username}
                                    onChange={(e) =>
                                      setUserDrafts((prev) => ({
                                        ...prev,
                                        [user.id]: { ...draft, username: e.target.value },
                                      }))
                                    }
                                  />
                                </td>
                                <td>
                                  <select
                                    value={draft.role}
                                    onChange={(e) =>
                                      setUserDrafts((prev) => ({
                                        ...prev,
                                        [user.id]: { ...draft, role: e.target.value },
                                      }))
                                    }
                                  >
                                    <option value="staff">staff</option>
                                    <option value="admin">admin</option>
                                  </select>
                                </td>
                                <td>
                                  <input
                                    type="checkbox"
                                    checked={draft.isActive}
                                    onChange={(e) =>
                                      setUserDrafts((prev) => ({
                                        ...prev,
                                        [user.id]: { ...draft, isActive: e.target.checked },
                                      }))
                                    }
                                  />
                                </td>
                                <td>
                                  <input
                                    type="password"
                                    value={userPasswordDrafts[user.id] ?? ""}
                                    onChange={(e) =>
                                      setUserPasswordDrafts((prev) => ({ ...prev, [user.id]: e.target.value }))
                                    }
                                    placeholder="New password"
                                  />
                                </td>
                                <td>
                                  <button onClick={() => void handleSaveUser(user.id)}>{t("button.save")}</button>
                                  <button onClick={() => void handleResetUserPassword(user.id)}>Reset password</button>
                                  <button
                                    onClick={() => void handleDeleteUser(user.id)}
                                    disabled={currentSessionUser?.id === user.id}
                                  >
                                    Delete
                                  </button>
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}
                </section>
              )}
            </div>
          )}
        </section>
      </div>

      {paymentModalOpen && paymentStudentId > 0 && (
        <div className="modal" onClick={() => setPaymentModalOpen(false)}>
          <div className="modalBody" onClick={(e) => e.stopPropagation()}>
            <h3>{t("modal.paymentTitle")}</h3>
            <div className="formRow">
              <label>{t("tabs.students")}</label>
              <input value={paymentStudentName} disabled />
            </div>
            {paymentInvoiceId && (
              <div className="formRow">
                <label>{t("field.course")}</label>
                <input value={`Счёт #${paymentInvoiceId}`} disabled />
              </div>
            )}
            <div className="formRow">
              <label>{t("field.amount")} (EUR):</label>
              <input
                type="number"
                step="0.01"
                value={paymentAmount}
                onChange={(e) => setPaymentAmount(e.target.value)}
                autoFocus
              />
            </div>
            <div className="formRow">
              <label>{t("field.method")}:</label>
              <select
                value={paymentMethod}
                onChange={(e) => setPaymentMethod(e.target.value as "cash" | "bank")}
              >
                <option value="cash">{t("payment.cash")}</option>
                <option value="bank">{t("payment.bank")}</option>
              </select>
            </div>
            <div className="formRow">
              <label>{t("field.note")}:</label>
              <input
                type="text"
                value={paymentNote}
                onChange={(e) => setPaymentNote(e.target.value)}
                placeholder={t("field.note")}
              />
            </div>
            <div className="modalActions">
              <button onClick={closePaymentModal}>{t("button.cancel")}</button>
              <button onClick={handleCreatePayment}>{t("button.recordPayment")}</button>
            </div>
          </div>
        </div>
      )}

      {invoiceDetailsOpen && selectedInv && (
        <div className="modal" onClick={() => setInvoiceDetailsOpen(false)}>
          <div className="modalBody modalBodyWide" onClick={(e) => e.stopPropagation()}>
            <div style={{ marginBottom: "1rem" }}>
              <h3>
                {t("modal.invoiceTitle")} {selectedInv.number ? `#${selectedInv.number}` : ""} —{" "}
                <button
                  className="linkButton"
                  onClick={() => void openStudentCardById(selectedInv.studentId)}
                >
                  {selectedInv.studentName}
                </button>{" "}
                — {uiMonths[selectedInv.month - 1]} {selectedInv.year}
              </h3>
            </div>

            {invSummary && selectedInv.status !== "draft" && (
              <div className="invSummary">
                <div className="invSummaryRow">
                  <span>{t("field.recipient")}:</span>
                  <span>{selectedInv.recipientName || selectedInv.studentName}</span>
                </div>
                {selectedInv.studentPersonalCode && (
                  <div className="invSummaryRow">
                    <span>
                      {selectedInv.isMinor ? `${t("field.personalCode")} child:` : `${t("field.personalCode")}:`}
                    </span>
                    <span>{selectedInv.studentPersonalCode}</span>
                  </div>
                )}
                {selectedInv.isMinor && (
                  <div className="invSummaryRow">
                    <span>{t("field.forChild")}:</span>
                    <span>{selectedInv.childName}</span>
                  </div>
                )}
                <div className="invSummaryRow">
                  <span>{t("field.amount")}:</span>
                  <span className="money">{formatEUR(invSummary.total)}</span>
                </div>

                <div className="invSummaryRow">
                  <span>{t("label.paid")}:</span>
                  <span className="money good">{formatEUR(invSummary.paid)}</span>
                </div>

                <div className="invSummaryRow">
                  <span>{t("field.remaining")}:</span>
                  <span className={`money ${invSummary.remaining > 0 ? "bad" : "good"}`}>
                    {formatEUR(invSummary.remaining)}
                  </span>
                </div>

                <div className="invSummaryRow">
                  <span>{t("field.status")}:</span>
                  <span className="money">{localizedInvoiceStatusLabel(invSummary.status)}</span>
                </div>
              </div>
            )}

            <div style={{ overflowX: "auto" }}>
              <table>
                <thead>
                  <tr>
                    <th>{t("field.description")}</th>
                    <th style={{ textAlign: "right" }}>{t("field.quantity")}</th>
                    <th style={{ textAlign: "right" }}>{t("field.lessonPrice")} (EUR)</th>
                    <th style={{ textAlign: "right" }}>{t("field.amount")} (EUR)</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedInv.lines.map((l, idx) => (
                    <tr key={idx}>
                      <td>{l.description}</td>
                      <td style={{ textAlign: "right" }}>{formatHoursValue(l.qty)}</td>
                      <td style={{ textAlign: "right" }}>{formatEUR(l.unitPrice)}</td>
                      <td style={{ textAlign: "right" }}>{formatEUR(l.amount)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={3} style={{ textAlign: "right" }}>
                      {t("field.totalEur")}:
                    </td>
                    <td style={{ textAlign: "right" }}>{formatEUR(selectedInv.total)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>

            <div className="modalActions">
              {selectedInv.status === "draft" && (
                <button onClick={() => onIssueOne(selectedInv.id)}>{t("button.issue")}</button>
              )}
              {selectedInv.status !== "draft" && (
                <button onClick={() => void onDownloadPdf(selectedInv.id)}>
                  {t("button.downloadPdf")}
                </button>
              )}
              {selectedInv.status !== "draft" && (
                <button onClick={() => openPaymentModal(selectedInv, invSummary)}>
                  {t("button.recordPayment")}
                </button>
              )}
              {selectedInv.status === "issued" && (
                <button onClick={() => void onReopenToDraft(selectedInv.id)}>
                  {t("button.reopenDraft")}
                </button>
              )}
              {selectedInv.status !== "draft" && canShowInvoiceFolderAction(transportCapabilities) && (
                <button onClick={() => void onRevealInvoiceFile(selectedInv.id)}>
                  {t("button.showInFolder")}
                </button>
              )}
              {transportCapabilities.isDesktop && selectedInv.status !== "draft" && !selectedInvPdfReady && (
                <button onClick={() => onGeneratePdf(selectedInv.id)}>{t("button.createPdf")}</button>
              )}
              <button onClick={() => setInvoiceDetailsOpen(false)}>{t("button.close")}</button>
            </div>
          </div>
        </div>
      )}

      {debtDetailsOpen && selectedDebtor && (
        <div className="modal" onClick={() => setDebtDetailsOpen(false)}>
          <div className="modalBody" onClick={(e) => e.stopPropagation()}>
            <h3>{t("modal.debtBreakdown")}</h3>

            <div className="invSummary">
              <div className="invSummaryRow">
                <span>{t("field.student")}</span>
                <button
                  className="linkButton"
                  onClick={() => void openStudentCardById(selectedDebtor.studentId)}
                >
                  {selectedDebtor.studentName}
                </button>
              </div>
              <div className="invSummaryRow">
                <span>{t("field.debtEur")}</span>
                <strong className="money bad">{formatEUR(selectedDebtor.debt)}</strong>
              </div>
            </div>

            {debtDetailsLoading ? (
              <div>{t("label.loading")}</div>
            ) : debtDetails.length === 0 ? (
              <div className="empty">{t("msg.noOpenDebts")}</div>
            ) : (
              <div style={{ overflowX: "auto" }}>
                <table>
                  <thead>
                    <tr>
                      <th>{t("field.month")}</th>
                      <th>{t("field.number")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.amount")}</th>
                      <th style={{ textAlign: "right" }}>{t("label.paid")}</th>
                      <th style={{ textAlign: "right" }}>{t("field.remaining")}</th>
                      <th>{t("field.status")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {debtDetails.map((x) => (
                      <tr key={x.invoiceId}>
                        <td>
                          {uiMonths[x.month - 1]} {x.year}
                        </td>
                        <td>{x.number ?? t("msg.noInvoiceNumber")}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(x.total)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(x.paid)}</td>
                        <td style={{ textAlign: "right" }}>
                          <strong className="money bad">{formatEUR(x.remaining)}</strong>
                        </td>
                        <td>{localizedInvoiceStatusLabel(x.status)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <div className="modalActions">
              {!debtDetailsLoading && debtDetails.length > 0 && (
                <>
                  <button onClick={openPaymentFromDebtDetails}>{t("button.recordPayment")}</button>
                  <button onClick={() => void copyDebtMessage("ru")}>{t("button.copyRu")}</button>
                  <button onClick={() => void copyDebtMessage("lv")}>{t("button.copyLv")}</button>
                </>
              )}
              <button onClick={() => setDebtDetailsOpen(false)}>{t("button.close")}</button>
            </div>
          </div>
        </div>
      )}

      {/* Student Card Modal */}
      {studentCardOpen && selectedStudentCard && (
        <div className="modal" onClick={() => setStudentCardOpen(false)}>
          <div className="modalBody modalBodyWide" onClick={(e) => e.stopPropagation()}>
            <StudentDetailPanel
              student={selectedStudentCard}
              loading={studentCardLoading}
              enrollments={studentCardEnrollments}
              balance={studentCardBalance}
              debts={studentCardDebts}
              payments={studentCardPayments}
              monthInvoices={studentCardMonthInvoices}
              nextAction={studentNextAction}
              activity={studentActivity}
              t={t}
              payerRoleLabel={localizedPayerRoleLabel}
              billingModeLabel={localizedBillingModeLabel}
              paymentMethodLabel={localizedPaymentMethodLabel}
              invoiceStatusLabel={localizedInvoiceStatusLabel}
              formatEUR={formatEUR}
              months={uiMonths}
              deletingPaymentId={studentCardDeletingPaymentId}
              canDeletePayment={canDeletePayments}
              onEditStudent={() => {
                setStudentCardOpen(false);
                openEditStudent(selectedStudentCard);
              }}
              onAddPayment={openStudentCardPaymentModal}
              onCopyDebtRu={() => void copyStudentCardDebtMessage("ru")}
              onCopyDebtLv={() => void copyStudentCardDebtMessage("lv")}
              onDeletePayment={deleteStudentPayment}
              onManageEnrollments={() => {
                setStudentCardOpen(false);
                setTab("enrollments");
              }}
              onOpenInvoices={() => {
                setStudentCardOpen(false);
                setTab("invoice");
              }}
              footer={
                <div className="modalActions">
                  <button onClick={() => setStudentCardOpen(false)}>{t("button.close")}</button>
                </div>
              }
            />
          </div>
        </div>
      )}

      {openInvoiceMenu && invoiceMenuPosition && openInvoiceMenuItems.length > 0 && (
        <div
          ref={invoiceMenuRef}
          className={`invoiceActionsMenuPanel ${invoiceMenuPosition.openUpward ? "invoiceActionsMenuPanelUpward" : ""}`}
          role="menu"
          onMouseDown={(event) => {
            event.stopPropagation();
          }}
          style={{
            position: "fixed",
            top: invoiceMenuPosition.top,
            left: invoiceMenuPosition.left,
          }}
        >
          {openInvoiceMenuItems.map((item) => (
            <button
              key={item.label}
              type="button"
              className="invoiceActionsMenuItem"
              role="menuitem"
              onClick={() => {
                closeInvoiceMenu();
                item.onClick();
              }}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
        </>
      )}
    </div>
  );
}
