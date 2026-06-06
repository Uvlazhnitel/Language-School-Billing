import { useEffect, useLayoutEffect, useMemo, useState, useCallback, useRef, type FormEvent } from "react";
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
import { AppShell, type AppTab } from "./components/AppShell";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { LoginScreen } from "./components/LoginScreen";
import { NotificationToast } from "./components/NotificationToast";
import { DebtDetailsModal } from "./components/modals/DebtDetailsModal";
import { InvoiceDetailsModal } from "./components/modals/InvoiceDetailsModal";
import { PaymentModal } from "./components/modals/PaymentModal";
import { StudentCardModal } from "./components/modals/StudentCardModal";
import { useConfirmDialog } from "./hooks/useConfirmDialog";
import { useNotifications } from "./hooks/useNotifications";
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
import {
  AppTabId,
  billingModeLabel,
  buildDebtReminderMessage,
  buildTabMeta,
  copyTextToClipboard,
  courseTypeLabel,
  decimalOrZero,
  formatEUR,
  formatHoursValue,
  intOrUndef,
  invoiceStatusLabel,
  normalizeHoursDraftInput,
  normalizeMoneyInput,
  normalizeQuarterHours,
  numOrZero,
  payerRoleLabel,
  payerRoleOptions,
  paymentMethodLabel,
  subscriptionTotal,
} from "./lib/appUi";
import { AttendanceScreen } from "./screens/AttendanceScreen";
import { AuditScreen } from "./screens/AuditScreen";
import { CoursesScreen } from "./screens/CoursesScreen";
import { DashboardScreen } from "./screens/DashboardScreen";
import { DebtorsScreen } from "./screens/DebtorsScreen";
import { EnrollmentsScreen } from "./screens/EnrollmentsScreen";
import { InvoicesScreen } from "./screens/InvoicesScreen";
import { SettingsScreen } from "./screens/SettingsScreen";
import { StudentsScreen } from "./screens/StudentsScreen";

type Tab = AppTabId;
type InvoiceMenuTarget = { kind: "row" | "modal"; invoiceId: number };
type InvoiceMenuPosition = { top: number; left: number; openUpward: boolean };
type UserDraft = { username: string; role: string; isActive: boolean };

function auditActionLabel(action: string): string {
  return action.replaceAll("_", " ").replaceAll(".", " ");
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
  const { message, showMessage, clearMessage } = useNotifications();
  const { confirmDialog, showConfirm, handleConfirmYes, handleConfirmNo } = useConfirmDialog();

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

  const tabs = useMemo<AppTab[]>(
    () => [
      { id: "dashboard", label: t("tabs.dashboard"), ...tabMeta.dashboard },
      { id: "students", label: t("tabs.students"), ...tabMeta.students },
      { id: "courses", label: t("tabs.courses"), ...tabMeta.courses },
      { id: "enrollments", label: t("tabs.enrollments"), ...tabMeta.enrollments },
      { id: "attendance", label: t("tabs.attendance"), ...tabMeta.attendance },
      { id: "invoice", label: t("tabs.invoice"), ...tabMeta.invoice },
      { id: "debtors", label: t("tabs.debtors"), ...tabMeta.debtors },
      ...(canViewAuditLog ? [{ id: "audit", label: t("tabs.audit"), ...tabMeta.audit }] : []),
      { id: "settings", label: t("tabs.settings"), ...tabMeta.settings },
    ],
    [canViewAuditLog, t, tabMeta]
  );

  const secondaryActions = useMemo(
    () =>
      [
        {
          id: "files",
          label: t("button.filesAndCopies"),
          onClick: () => setTab("settings"),
        },
        authRequired && !transportCapabilities.isDesktop
          ? {
              id: "logout",
              label: t("auth.logout"),
              onClick: () => {
                void handleLogout();
              },
            }
          : null,
      ].filter((value): value is { id: string; label: string; onClick: () => void } => value !== null),
    [authRequired, handleLogout, t, transportCapabilities.isDesktop]
  );

  const adjustSubscriptionLessons = useCallback(
    async (courseId: number, nextValue: number) => {
      try {
        setSubscriptionMonthSaving((prev) => ({ ...prev, [courseId]: true }));
        const updated = await saveCourseMonthSubscriptionLessons(courseId, year, month, nextValue);
        setSubscriptionMonthLessons((prev) => ({
          ...prev,
          [courseId]: updated.lessonsHeld,
        }));
        await loadAttendance();
      } catch (e: any) {
        showMessage(
          t("msg.errorGeneric", {
            message: String(e?.message ?? e),
          }),
          "error"
        );
      } finally {
        setSubscriptionMonthSaving((prev) => {
          const next = { ...prev };
          delete next[courseId];
          return next;
        });
      }
    },
    [loadAttendance, month, showMessage, t, year]
  );

  return (
    <div className="container">
      {message && <NotificationToast message={message} onDismiss={clearMessage} t={t} />}
      {confirmDialog?.isOpen && (
        <ConfirmDialog
          dialog={confirmDialog}
          onConfirm={handleConfirmYes}
          onCancel={handleConfirmNo}
          t={t}
        />
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
          <AppShell
            tabs={tabs}
            activeTab={tab}
            secondaryActions={secondaryActions}
            onTabChange={(nextTab) => setTab(nextTab as Tab)}
            currentMeta={tabs.find((item) => item.id === tab) ?? tabs[0]}
            month={month}
            year={year}
            monthLabels={uiMonths}
            onMonthChange={setMonth}
            onYearChange={setYear}
            showMonthPicker={showMonthPicker}
          >
            {tab === "dashboard" && (
              <DashboardScreen
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
                onOpenPaymentQueueStudent={openDebtorPaymentModalByStudentId}
                onCopyDebtQueueRu={(studentId) => void copyDebtMessageForStudentId(studentId, "ru")}
                onCopyDebtQueueLv={(studentId) => void copyDebtMessageForStudentId(studentId, "lv")}
                recentPayments={recentPayments}
                actionQueue={debtorActionQueue}
              />
            )}

            {tab === "students" && (
              <StudentsScreen
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
                studentModalOpen={studentModalOpen}
                editingStudent={Boolean(editingStudent)}
                sfName={sfName}
                sfPersonalCode={sfPersonalCode}
                sfPhone={sfPhone}
                sfEmail={sfEmail}
                sfNote={sfNote}
                sfIsMinor={sfIsMinor}
                sfPayerName={sfPayerName}
                sfPayerRole={sfPayerRole}
                payerRoleOptions={payerRoleOptions}
                onSfNameChange={setSfName}
                onSfPersonalCodeChange={setSfPersonalCode}
                onSfPhoneChange={setSfPhone}
                onSfEmailChange={setSfEmail}
                onSfNoteChange={setSfNote}
                onSfIsMinorChange={setSfIsMinor}
                onSfPayerNameChange={setSfPayerName}
                onSfPayerRoleChange={setSfPayerRole}
                onSaveStudent={() => void saveStudent()}
                onCloseStudentModal={() => setStudentModalOpen(false)}
              />
            )}

            {tab === "courses" && (
              <CoursesScreen
                loading={courseLoading}
                courses={courseList}
                query={courseQ}
                canDeleteCourses={canDeleteCourses}
                courseTypeLabel={localizedCourseTypeLabel}
                formatEUR={formatEUR}
                onQueryChange={setCourseQ}
                onRefresh={() => void loadCourses()}
                onAddCourse={openAddCourse}
                onEditCourse={openEditCourse}
                onDeleteCourse={(courseId) => void removeCourse(courseId)}
                courseModalOpen={courseModalOpen}
                editingCourse={Boolean(editingCourse)}
                cfName={cfName}
                cfTeacherSearch={cfTeacherSearch}
                cfTeacherId={cfTeacherId}
                cfTeacherPickerOpen={cfTeacherPickerOpen}
                selectedCourseTeacher={selectedCourseTeacher}
                filteredTeachers={filteredTeachers}
                exactTeacherMatch={exactTeacherMatch}
                cfTeacherCreating={cfTeacherCreating}
                cfType={cfType}
                cfLessonPrice={cfLessonPrice}
                cfSubscriptionPrice={cfSubscriptionPrice}
                cfTeacherComboRef={cfTeacherComboRef}
                onCfNameChange={setCfName}
                onCfTeacherSearchChange={setCfTeacherSearch}
                onCfTeacherIdChange={setCfTeacherId}
                onCfTeacherPickerOpenChange={setCfTeacherPickerOpen}
                onAddTeacherFromCourseForm={addTeacherFromCourseForm}
                onCfTypeChange={setCfType}
                onCfLessonPriceChange={(value) => handleCoursePriceChange(value, setCfLessonPrice)}
                onCfSubscriptionPriceChange={(value) =>
                  handleCoursePriceChange(value, setCfSubscriptionPrice)
                }
                onSaveCourse={() => void saveCourse()}
                onCloseCourseModal={() => setCourseModalOpen(false)}
                t={t}
              />
            )}

            {tab === "enrollments" && (
              <EnrollmentsScreen
                loading={enrLoading}
                enrollments={enrollments}
                studentFilter={enrStudentFilter}
                courseFilter={enrCourseFilter}
                allStudents={allStudents}
                allCourses={allCourses}
                billingModeLabel={localizedBillingModeLabel}
                onStudentFilterChange={setEnrStudentFilter}
                onCourseFilterChange={setEnrCourseFilter}
                onRefresh={() => void loadEnrollments()}
                onAddEnrollment={openAddEnrollment}
                onOpenStudent={(studentId) => void openStudentCardById(studentId)}
                onEditEnrollment={openEditEnrollment}
                enrollmentModalOpen={enrModalOpen}
                editingEnrollment={editingEnr}
                studentSearch={efStudentSearch}
                studentId={efStudentId}
                studentPickerOpen={efStudentPickerOpen}
                filteredStudents={filteredEnrollmentStudents}
                selectedStudent={selectedEnrollmentStudent}
                enrollmentCourseId={efCourseId}
                enrollmentMode={efMode}
                enrollmentDiscount={efDiscount}
                enrollmentSubscriptionDiscount={efSubscriptionDiscount}
                enrollmentNote={efNote}
                studentComboRef={efStudentComboRef}
                onStudentSearchChange={setEfStudentSearch}
                onStudentIdChange={(value) => setEfStudentId(value ?? 0)}
                onStudentPickerOpenChange={setEfStudentPickerOpen}
                onEnrollmentCourseIdChange={setEfCourseId}
                onEnrollmentModeChange={setEfMode}
                onEnrollmentDiscountChange={setEfDiscount}
                onEnrollmentSubscriptionDiscountChange={setEfSubscriptionDiscount}
                onEnrollmentNoteChange={setEfNote}
                onSaveEnrollment={() => void saveEnrollment()}
                onCloseEnrollmentModal={() => setEnrModalOpen(false)}
                t={t}
              />
            )}

            {tab === "attendance" && (
              <AttendanceScreen
                attendanceSummary={attendanceSummary}
                courseFilter={courseFilter}
                allCourses={allCourses}
                query={attQ}
                filter={attFilter}
                rows={rows}
                filteredRows={filteredAttendanceRows}
                loading={loadingAtt}
                attendanceSavingRows={attendanceSavingRows}
                attendancePendingSelectRef={attendancePendingSelectRef}
                subscriptionLeadEnrollmentIds={subscriptionLeadEnrollmentIds}
                subscriptionMonthLessons={subscriptionMonthLessons}
                subscriptionMonthSaving={subscriptionMonthSaving}
                year={year}
                month={month}
                perLessonTotal={perLessonTotal}
                courseTypeLabel={localizedCourseTypeLabel}
                formatEUR={formatEUR}
                normalizeHoursDraftInput={normalizeHoursDraftInput}
                getAttendanceStepBase={getAttendanceStepBase}
                getAttendanceInputValue={getAttendanceInputValue}
                getSubscriptionMonthLessonsValue={getSubscriptionMonthLessonsValue}
                setAttendanceDraft={setAttendanceDraft}
                clearAttendanceDraft={clearAttendanceDraft}
                commitAttendanceDraft={commitAttendanceDraft}
                onChangeHours={onChangeHours}
                setSubscriptionMonthLessonsDraft={setSubscriptionMonthLessonsDraft}
                clearSubscriptionMonthLessonsDraft={clearSubscriptionMonthLessonsDraft}
                commitSubscriptionMonthLessonsDraft={commitSubscriptionMonthLessonsDraft}
                onAdjustSubscriptionLessons={adjustSubscriptionLessons}
                onRefresh={() => void loadAttendance()}
                onOpenInvoices={() => setTab("invoice")}
                onCourseFilterChange={setCourseFilter}
                onQueryChange={setAttQ}
                onFilterChange={setAttFilter}
                onOpenStudent={(studentId) => void openStudentCardById(studentId)}
                onDeleteEnrollmentFromSheet={(enrollmentId) => void onDeleteEnrollmentFromSheet(enrollmentId)}
                t={t}
              />
            )}

            {tab === "invoice" && (
              <InvoicesScreen
                currentMonthLabel={currentMonthLabel}
                status={invStatus}
                query={invQ}
                loading={loadingInv}
                items={filteredInvItems}
                months={uiMonths}
                invoiceStatusLabel={localizedInvoiceStatusLabel}
                formatEUR={formatEUR}
                renderInvoiceActionsMenu={(invoice) => renderInvoiceActionsMenu(invoice)}
                onStatusChange={setInvStatus}
                onQueryChange={setInvQ}
                onRefresh={() => void loadInvoices()}
                onOpenStudent={(studentId) => void openStudentCardById(studentId)}
                onOpenInvoice={(invoiceId) => void onOpenInvoice(invoiceId)}
                onIssueOne={(invoiceId) => void onIssueOne(invoiceId)}
                onDownloadPdf={(invoiceId) => void onDownloadPdf(invoiceId)}
                onOpenPaymentModal={(invoiceId) => void openPaymentModalForInvoice(invoiceId)}
                t={t}
              />
            )}

            {tab === "debtors" && (
              <DebtorsScreen
                loading={debtorsLoading}
                debtors={debtors}
                actionQueue={debtorActionQueue}
                formatEUR={formatEUR}
                onRefresh={() => void loadDebtors()}
                onOpenStudent={(studentId) => void openStudentCardById(studentId)}
                onOpenStudentWorkspace={(studentId) => void openStudentInWorkspaceById(studentId)}
                onOpenPaymentForStudent={openDebtorPaymentModalByStudentId}
                onOpenPaymentForDebtor={openDebtorPaymentModal}
                onOpenDebtDetails={openDebtDetails}
                onCopyDebtForStudentRu={(studentId) => void copyDebtMessageForStudentId(studentId, "ru")}
                onCopyDebtForStudentLv={(studentId) => void copyDebtMessageForStudentId(studentId, "lv")}
                onCopyDebtForDebtorRu={(debtor) => void copyDebtMessageForDebtor(debtor, "ru")}
                onCopyDebtForDebtorLv={(debtor) => void copyDebtMessageForDebtor(debtor, "lv")}
                t={t}
              />
            )}

            {tab === "audit" && canViewAuditLog && (
              <AuditScreen
                loading={auditLoading}
                items={auditItems}
                total={auditTotal}
                page={auditPage}
                pageSize={auditPageSize}
                expandedId={auditExpandedId}
                q={auditQ}
                actorFilter={auditActorFilter}
                entityTypeFilter={auditEntityTypeFilter}
                actionFilter={auditActionFilter}
                dateFrom={auditDateFrom}
                dateTo={auditDateTo}
                onQChange={setAuditQ}
                onActorFilterChange={setAuditActorFilter}
                onEntityTypeFilterChange={setAuditEntityTypeFilter}
                onActionFilterChange={setAuditActionFilter}
                onDateFromChange={setAuditDateFrom}
                onDateToChange={setAuditDateTo}
                onRefresh={() => {
                  setAuditPage(1);
                  void loadAuditLog();
                }}
                onToggleExpanded={(id) =>
                  setAuditExpandedId((current) => (current === id ? null : id))
                }
                onPrevPage={() => setAuditPage((page) => Math.max(1, page - 1))}
                onNextPage={() => setAuditPage((page) => page + 1)}
                actionLabel={auditActionLabel}
                t={t}
              />
            )}

            {tab === "settings" && (
              <SettingsScreen
                uiLocale={uiLocale}
                canManageSettings={canManageSettings}
                canCreateBackups={canCreateBackups}
                creatingBackup={creatingBackup}
                transportCapabilities={transportCapabilities}
                appDirs={appDirs}
                canManageUsers={canManageUsers}
                usersLoading={usersLoading}
                users={users}
                creatingUser={creatingUser}
                newUserUsername={newUserUsername}
                newUserPassword={newUserPassword}
                newUserRole={newUserRole}
                userDrafts={userDrafts}
                userPasswordDrafts={userPasswordDrafts}
                currentSessionUser={currentSessionUser}
                onLocaleChange={handleLocaleChange}
                onCreateBackup={createManualBackup}
                onOpenAppFolder={openAppFolder}
                onSetTab={setTab}
                onNewUserUsernameChange={setNewUserUsername}
                onNewUserPasswordChange={setNewUserPassword}
                onNewUserRoleChange={setNewUserRole}
                onCreateUser={handleCreateUser}
                onRefreshUsers={loadUsers}
                onUserDraftsChange={setUserDrafts}
                onUserPasswordDraftsChange={setUserPasswordDrafts}
                onSaveUser={handleSaveUser}
                onResetUserPassword={handleResetUserPassword}
                onDeleteUser={handleDeleteUser}
                t={t}
              />
            )}
          </AppShell>

          {paymentModalOpen && (
            <PaymentModal
              studentId={paymentStudentId}
              studentName={paymentStudentName}
              invoiceId={paymentInvoiceId ?? null}
              amount={paymentAmount}
              method={paymentMethod}
              note={paymentNote}
              onAmountChange={setPaymentAmount}
              onMethodChange={setPaymentMethod}
              onNoteChange={setPaymentNote}
              onCancel={closePaymentModal}
              onSubmit={() => void handleCreatePayment()}
              t={t}
            />
          )}

          {invoiceDetailsOpen && selectedInv && (
            <InvoiceDetailsModal
              invoice={selectedInv}
              summary={invSummary}
              months={uiMonths}
              pdfReady={selectedInvPdfReady}
              transportCapabilities={transportCapabilities}
              invoiceStatusLabel={localizedInvoiceStatusLabel}
              formatEUR={formatEUR}
              formatHoursValue={formatHoursValue}
              onOpenStudent={(studentId) => void openStudentCardById(studentId)}
              onIssue={(invoiceId) => void onIssueOne(invoiceId)}
              onDownloadPdf={(invoiceId) => void onDownloadPdf(invoiceId)}
              onAddPayment={() => openPaymentModal(selectedInv, invSummary)}
              onReopenToDraft={(invoiceId) => void onReopenToDraft(invoiceId)}
              onRevealInvoiceFile={(invoiceId) => void onRevealInvoiceFile(invoiceId)}
              onGeneratePdf={onGeneratePdf}
              onClose={() => setInvoiceDetailsOpen(false)}
              t={t}
            />
          )}

          {debtDetailsOpen && selectedDebtor && (
            <DebtDetailsModal
              debtor={selectedDebtor}
              details={debtDetails}
              loading={debtDetailsLoading}
              months={uiMonths}
              invoiceStatusLabel={localizedInvoiceStatusLabel}
              formatEUR={formatEUR}
              onOpenStudent={(studentId) => void openStudentCardById(studentId)}
              onRecordPayment={openPaymentFromDebtDetails}
              onCopyRu={() => void copyDebtMessage("ru")}
              onCopyLv={() => void copyDebtMessage("lv")}
              onClose={() => setDebtDetailsOpen(false)}
              t={t}
            />
          )}

          {studentCardOpen && selectedStudentCard && (
            <StudentCardModal
              student={selectedStudentCard}
              loading={studentCardLoading}
              enrollments={studentCardEnrollments}
              balance={studentCardBalance}
              debts={studentCardDebts}
              payments={studentCardPayments}
              monthInvoices={studentCardMonthInvoices}
              nextAction={studentNextAction}
              activity={studentActivity}
              deletingPaymentId={studentCardDeletingPaymentId}
              canDeletePayment={canDeletePayments}
              payerRoleLabel={localizedPayerRoleLabel}
              billingModeLabel={localizedBillingModeLabel}
              paymentMethodLabel={localizedPaymentMethodLabel}
              invoiceStatusLabel={localizedInvoiceStatusLabel}
              formatEUR={formatEUR}
              months={uiMonths}
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
              onClose={() => setStudentCardOpen(false)}
              t={t}
            />
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
