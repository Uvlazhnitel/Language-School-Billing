import { useEffect, useLayoutEffect, useMemo, useState, useCallback, useRef } from "react";
import "./App.css";

import { fetchRows, saveHours, deleteEnrollment, Row } from "./lib/attendance";

import {
  listInvoices,
  getInvoice,
  genDrafts,
  issueOne,
  reopenToDraft,
  ensurePdf,
  ensureAllPdfs,
  previewInvoiceEmail,
  sendInvoiceEmail,
  EnsureAllPDFsResult,
  InvoiceEmailPreviewResult,
  InvoiceListItemView,
  InvoiceDTO,
} from "./lib/invoices";
import { buildIssueFeedback } from "./lib/invoiceIssueFeedback";
import { getInvoiceMenuActions } from "./lib/invoiceUi";

import {
  listStudents,
  getStudent,
  checkStudentDuplicates,
  createStudentWithEnrollment,
  updateStudent,
  setStudentActive,
  deleteStudent,
  StudentDTO,
  StudentDuplicateCheckResult,
  type EnrollmentCreateInput,
} from "./lib/students";

import { listCourses, createCourse, updateCourse, deleteCourse, CourseDTO } from "./lib/courses";
import { listTeachers, createTeacher, TeacherDTO } from "./lib/teachers";
import { BillingModePerLesson, BillingModeSubscription } from "./lib/constants";

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
import { isConflictError } from "./lib/api/shared";
import { AppShell, type AppTab } from "./components/AppShell";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { LoginScreen } from "./components/LoginScreen";
import { NotificationToast } from "./components/NotificationToast";
import { DebtDetailsModal } from "./components/modals/DebtDetailsModal";
import { InvoiceDetailsModal } from "./components/modals/InvoiceDetailsModal";
import { InvoiceEnsureAllPDFsModal } from "./components/modals/InvoiceEnsureAllPDFsModal";
import { InvoiceEmailModal } from "./components/modals/InvoiceEmailModal";
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
import { createTranslator, getMonthNames } from "./lib/i18n";
import {
  applyStudentListControls,
  type StudentAgeFilter,
  type StudentBalanceFilter,
  type StudentDebtFilter,
  type StudentSortOption,
  type StudentStatusFilter,
} from "./lib/studentListControls";
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
  invoiceStatusLabel,
  normalizeHoursDraftInput,
  normalizeMoneyInput,
  normalizeQuarterHours,
  payerRoleLabel,
  payerRoleOptions,
  paymentMethodLabel,
} from "./lib/appUi";
import { validatePersonName, validateStudentForm } from "./lib/inputValidation";
import { buildNextStudentDraft } from "./lib/studentFormState";
import { defaultLessonPriceForCourseType } from "./lib/courseDefaults";

const staleRevisionMessage = "record was changed or deleted by another user";
type StudentCreateFlow = "save" | "save_and_add_another";

function isStaleRevisionError(error: unknown): boolean {
  const message = String(
    (error as { message?: string } | undefined)?.message ?? error ?? ""
  ).toLowerCase();
  return (
    (isConflictError(error) || message.includes("conflict")) &&
    message.includes(staleRevisionMessage)
  );
}
import { AttendanceScreen, type AttendanceFocusTarget } from "./screens/AttendanceScreen";
import { AuditScreen } from "./screens/AuditScreen";
import { CoursesScreen } from "./screens/CoursesScreen";
import { DashboardScreen } from "./screens/DashboardScreen";
import { DebtorsScreen } from "./screens/DebtorsScreen";
import { EnrollmentsScreen } from "./screens/EnrollmentsScreen";
import { InvoicesScreen } from "./screens/InvoicesScreen";
import { SettingsScreen } from "./screens/SettingsScreen";
import { StudentsScreen } from "./screens/StudentsScreen";
import { useAuthController } from "./hooks/app/useAuthController";
import { useSettingsController } from "./hooks/app/useSettingsController";

type Tab = AppTabId;
type InvoiceMenuTarget = { kind: "row" | "modal"; invoiceId: number };
type InvoiceMenuPosition = { top: number; left: number; openUpward: boolean };

function auditActionLabel(action: string): string {
  return action.replaceAll("_", " ").replaceAll(".", " ");
}

export default function App() {
  const now = new Date();
  const [tab, setTab] = useState<Tab>("dashboard");
  const { message, showMessage, clearMessage } = useNotifications();
  const { confirmDialog, showConfirm, handleConfirmYes, handleConfirmNo } = useConfirmDialog();
  const {
    appReady,
    authLoading,
    authRequired,
    isAuthenticated,
    currentSessionUser,
    sessionCapabilities,
    loginUsername,
    loginPassword,
    loginRememberMe,
    loginPending,
    loginError,
    sessionExpired,
    uiLocale,
    setUiLocale,
    setLoginUsername,
    setLoginPassword,
    setLoginRememberMe,
    handleLogin,
    handleLogout,
  } = useAuthController({ showMessage });

  const t = useMemo(() => createTranslator(uiLocale), [uiLocale]);
  const uiMonths = useMemo(() => getMonthNames(uiLocale), [uiLocale]);
  const tabMeta = useMemo(() => buildTabMeta(t), [t]);
  const canManageUsers = Boolean(sessionCapabilities.manageUsers);
  const canManageSettings = Boolean(sessionCapabilities.manageSettings);
  const canCreateBackups = Boolean(sessionCapabilities.backups);
  const canViewInvoiceArchive = Boolean(sessionCapabilities.invoiceArchive);
  const canDeleteStudents = Boolean(sessionCapabilities.deleteStudents);
  const canDeleteCourses = Boolean(sessionCapabilities.deleteCourses);
  const canDeletePayments = Boolean(sessionCapabilities.deletePayments);
  const canViewAuditLog = Boolean(sessionCapabilities.viewAuditLog);
  const {
    creatingBackup,
    invoiceArchive,
    invoiceArchiveLoading,
    invoiceEmailSettings,
    invoiceEmailSettingsLoading,
    savingInvoiceEmailSettings,
    invoiceEmailSubjectTemplate,
    invoiceEmailBodyTemplate,
    invoiceEmailReplyTo,
    users,
    usersLoading,
    creatingUser,
    newUserUsername,
    newUserPassword,
    newUserRole,
    userDrafts,
    userPasswordDrafts,
    setInvoiceEmailSubjectTemplate,
    setInvoiceEmailBodyTemplate,
    setInvoiceEmailReplyTo,
    setNewUserUsername,
    setNewUserPassword,
    setNewUserRole,
    setUserDrafts,
    setUserPasswordDrafts,
    createManualBackup,
    loadInvoiceArchive,
    loadUsers,
    handleCreateUser,
    handleSaveUser,
    handleDeleteUser,
    handleResetUserPassword,
    handleLocaleChange,
    handleSaveInvoiceEmailSettings,
    handleResetInvoiceEmailSettings,
  } = useSettingsController({
    appReady,
    isAuthenticated,
    tab,
    canManageUsers,
    canManageSettings,
    canCreateBackups,
    canViewInvoiceArchive,
    uiLocale,
    setUiLocale,
    showMessage,
    showConfirm,
    t,
  });

  const localizedPayerRoleLabel = useCallback(
    (relation: string) => payerRoleLabel(relation, t),
    [t]
  );
  const localizedCourseTypeLabel = useCallback((type: string) => courseTypeLabel(type, t), [t]);
  const localizedBillingModeLabel = useCallback((mode: string) => billingModeLabel(mode, t), [t]);
  const localizedPaymentMethodLabel = useCallback(
    (method: string) => paymentMethodLabel(method, t),
    [t]
  );
  const localizedInvoiceStatusLabel = useCallback(
    (status: string) => invoiceStatusLabel(status, t),
    [t]
  );
  const formatDateTime = useCallback(
    (value: string) => new Date(value).toLocaleString(uiLocale),
    [uiLocale]
  );

  // Shared month/year for Attendance + Invoices
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);
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
  const resetAuditFilters = useCallback(() => {
    setAuditQ("");
    setAuditActorFilter("");
    setAuditEntityTypeFilter("");
    setAuditActionFilter("");
    setAuditDateFrom("");
    setAuditDateTo("");
    setAuditPage(1);
  }, []);

  // ---------------- Students ----------------
  const [studentList, setStudentList] = useState<StudentDTO[]>([]);
  const [allStudents, setAllStudents] = useState<StudentDTO[]>([]);
  const [studentQ, setStudentQ] = useState("");
  const [studentStatusFilter, setStudentStatusFilter] = useState<StudentStatusFilter>("active");
  const [studentDebtFilter, setStudentDebtFilter] = useState<StudentDebtFilter>("all");
  const [studentBalanceFilter, setStudentBalanceFilter] = useState<StudentBalanceFilter>("all");
  const [studentAgeFilter, setStudentAgeFilter] = useState<StudentAgeFilter>("all");
  const [studentSortOption, setStudentSortOption] = useState<StudentSortOption>("created_desc");
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
  const [sfCourseId, setSfCourseId] = useState(0);
  const [sfEnrollmentMode, setSfEnrollmentMode] =
    useState<EnrollmentDTO["billingMode"]>(BillingModePerLesson);
  const [sfEnrollmentChargeMaterials, setSfEnrollmentChargeMaterials] = useState(true);
  const [sfEnrollmentLessonPrice, setSfEnrollmentLessonPrice] = useState("0");
  const [sfEnrollmentSubscriptionPrice, setSfEnrollmentSubscriptionPrice] = useState("0");
  const [sfEnrollmentNote, setSfEnrollmentNote] = useState("");
  const [sfEnrollmentSettingsOpen, setSfEnrollmentSettingsOpen] = useState(false);
  const [studentDuplicateCheckResult, setStudentDuplicateCheckResult] =
    useState<StudentDuplicateCheckResult | null>(null);
  const [studentCreateFlow, setStudentCreateFlow] = useState<StudentCreateFlow>("save");

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
      const data = await listStudents(studentQ, true);
      setStudentList(data);
    } finally {
      setStudentLoading(false);
    }
  }, [studentQ]);

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

  const resetStudentDuplicateCheck = useCallback(() => {
    setStudentDuplicateCheckResult(null);
  }, []);

  const resetStudentFilters = useCallback(() => {
    setStudentStatusFilter("active");
    setStudentDebtFilter("all");
    setStudentBalanceFilter("all");
    setStudentAgeFilter("all");
    setStudentSortOption("created_desc");
  }, []);

  const filteredStudentList = useMemo(
    () =>
      applyStudentListControls(studentList, {
        statusFilter: studentStatusFilter,
        debtFilter: studentDebtFilter,
        balanceFilter: studentBalanceFilter,
        ageFilter: studentAgeFilter,
        sortOption: studentSortOption,
      }),
    [
      studentAgeFilter,
      studentBalanceFilter,
      studentDebtFilter,
      studentList,
      studentSortOption,
      studentStatusFilter,
    ]
  );

  const hasActiveStudentFilters = useMemo(
    () =>
      studentStatusFilter !== "active" ||
      studentDebtFilter !== "all" ||
      studentBalanceFilter !== "all" ||
      studentAgeFilter !== "all" ||
      studentSortOption !== "created_desc",
    [
      studentAgeFilter,
      studentBalanceFilter,
      studentDebtFilter,
      studentSortOption,
      studentStatusFilter,
    ]
  );

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
    resetStudentDuplicateCheck();
    setStudentCreateFlow("save");
    setEditingStudent(null);
    setSfName("");
    setSfPersonalCode("");
    setSfPhone("");
    setSfEmail("");
    setSfNote("");
    setSfIsMinor(false);
    setSfPayerName("");
    setSfPayerRole("");
    setSfCourseId(0);
    setSfEnrollmentMode(BillingModePerLesson);
    setSfEnrollmentChargeMaterials(true);
    setSfEnrollmentLessonPrice("0");
    setSfEnrollmentSubscriptionPrice("0");
    setSfEnrollmentNote("");
    setSfEnrollmentSettingsOpen(false);
    setStudentModalOpen(true);
  }

  function openEditStudent(s: StudentDTO) {
    resetStudentDuplicateCheck();
    setStudentCreateFlow("save");
    setEditingStudent(s);
    setSfName(s.fullName);
    setSfPersonalCode(s.personalCode ?? "");
    setSfPhone(s.phone);
    setSfEmail(s.email);
    setSfNote(s.note);
    setSfIsMinor(s.isMinor);
    setSfPayerName(s.payerName ?? "");
    setSfPayerRole(s.payerRole ?? "");
    setSfCourseId(0);
    setSfEnrollmentSettingsOpen(false);
    setStudentModalOpen(true);
  }

  function handleStudentOnboardingCourseChange(courseId: number) {
    resetStudentDuplicateCheck();
    setSfCourseId(courseId);
    setSfEnrollmentMode(BillingModePerLesson);
    setSfEnrollmentChargeMaterials(true);
    setSfEnrollmentNote("");
    setSfEnrollmentSettingsOpen(false);
    const course = allCourses.find((item) => item.id === courseId);
    setSfEnrollmentLessonPrice(String(course?.lessonPrice ?? 0));
    setSfEnrollmentSubscriptionPrice(String(course?.subscriptionPrice ?? 0));
  }

  function handleStudentOnboardingModeChange(mode: EnrollmentDTO["billingMode"]) {
    setSfEnrollmentMode(mode);
    const course = allCourses.find((item) => item.id === sfCourseId);
    if (mode === BillingModePerLesson && Number(sfEnrollmentLessonPrice) === 0) {
      setSfEnrollmentLessonPrice(String(course?.lessonPrice ?? 0));
    }
    if (mode === BillingModeSubscription && Number(sfEnrollmentSubscriptionPrice) === 0) {
      setSfEnrollmentSubscriptionPrice(String(course?.subscriptionPrice ?? 0));
    }
  }

  function getStudentOnboardingEnrollmentInput(): EnrollmentCreateInput | undefined {
    if (sfCourseId <= 0) return undefined;
    const lessonPrice = sfEnrollmentLessonPrice.trim() === "" ? 0 : Number(sfEnrollmentLessonPrice);
    const subscriptionPrice =
      sfEnrollmentSubscriptionPrice.trim() === "" ? 0 : Number(sfEnrollmentSubscriptionPrice);
    if (!Number.isFinite(lessonPrice) || lessonPrice < 0) {
      showMessage(t("msg.lessonPriceOverrideRange"), "error");
      return undefined;
    }
    if (!Number.isFinite(subscriptionPrice) || subscriptionPrice < 0) {
      showMessage(t("msg.subscriptionLessonPriceRange"), "error");
      return undefined;
    }
    return {
      courseId: sfCourseId,
      billingMode: sfEnrollmentMode,
      chargeMaterials: sfEnrollmentChargeMaterials,
      lessonPriceOverride: sfEnrollmentMode === BillingModePerLesson ? lessonPrice : 0,
      subscriptionLessonPrice:
        sfEnrollmentMode === BillingModeSubscription ? subscriptionPrice : 0,
      note: sfEnrollmentNote,
    };
  }

  function resetStudentFieldsForNextEntry() {
    const nextDraft = buildNextStudentDraft(sfIsMinor);
    setSfName(nextDraft.name);
    setSfPersonalCode(nextDraft.personalCode);
    setSfPhone(nextDraft.phone);
    setSfEmail(nextDraft.email);
    setSfNote(nextDraft.note);
    setSfIsMinor(nextDraft.isMinor);
    setSfPayerName(nextDraft.payerName);
    setSfPayerRole(nextDraft.payerRole);
  }

  async function saveStudent(
    skipDuplicateCheck = false,
    createFlow: StudentCreateFlow = "save"
  ) {
    const validationError = validateStudentForm({
      fullName: sfName,
      personalCode: sfPersonalCode,
      phone: sfPhone,
      email: sfEmail,
      isMinor: sfIsMinor,
      payerName: sfPayerName,
      payerRole: sfPayerRole,
    });
    if (validationError) {
      showMessage(t(validationError), "error");
      return;
    }
    const onboardingEnrollment = editingStudent
      ? undefined
      : getStudentOnboardingEnrollmentInput();
    if (!editingStudent && sfCourseId > 0 && !onboardingEnrollment) return;
    setStudentCreateFlow(createFlow);
    try {
      let savedStudent: StudentDTO;
      let savedEnrollment: EnrollmentDTO | undefined;
      if (editingStudent) {
        savedStudent = await updateStudent(
          editingStudent.id,
          editingStudent.version,
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
        if (!skipDuplicateCheck) {
          const duplicateResult = await checkStudentDuplicates(
            sfName,
            sfPersonalCode,
            sfPhone,
            sfEmail
          );
          if (duplicateResult.exactMatch || duplicateResult.possibleMatches.length > 0) {
            setStudentDuplicateCheckResult(duplicateResult);
            return;
          }
        }
        const onboardingResult = await createStudentWithEnrollment(
          {
            fullName: sfName,
            personalCode: sfPersonalCode,
            phone: sfPhone,
            email: sfEmail,
            note: sfNote,
            isMinor: sfIsMinor,
            payerName: sfPayerName,
            payerRole: sfPayerRole,
          },
          onboardingEnrollment
        );
        savedStudent = onboardingResult.student;
        savedEnrollment = onboardingResult.enrollment;
      }
      resetStudentDuplicateCheck();
      if (!editingStudent) {
        // Make the newly created student immediately visible at the top of the workspace list.
        setStudentQ("");
        setStudentStatusFilter("active");
        setStudentDebtFilter("all");
        setStudentBalanceFilter("all");
        setStudentAgeFilter("all");
        setStudentSortOption("created_desc");
      }
      await Promise.all([loadStudents(), loadAllStudents()]);
      if (!editingStudent && createFlow === "save_and_add_another") {
        resetStudentFieldsForNextEntry();
      } else if (!editingStudent && savedEnrollment) {
        setStudentModalOpen(false);
        openAttendanceForStudentCourse(savedStudent.id, savedEnrollment.courseId);
      } else if (!editingStudent) {
        setStudentModalOpen(false);
        await openStudentCard(savedStudent, { inline: true });
      } else if (selectedStudentCard?.id === savedStudent.id) {
        setStudentModalOpen(false);
        setSelectedStudentCard(savedStudent);
        void refreshStudentCardData(savedStudent.id);
      } else {
        setStudentModalOpen(false);
      }
      showMessage(editingStudent ? t("msg.studentUpdated") : t("msg.studentCreated"));
    } catch (e: any) {
      if (isStaleRevisionError(e)) {
        showMessage(t("msg.recordConflict"), "error");
        return;
      }
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function toggleStudentActive(s: StudentDTO) {
    try {
      await setStudentActive(s.id, s.version, !s.isActive);
      await Promise.all([loadStudents(), loadAllStudents()]);
      if (selectedStudentCard?.id === s.id) {
        const freshStudent = await getStudent(s.id);
        setSelectedStudentCard(freshStudent);
        void refreshStudentCardData(s.id);
      }
      showMessage(s.isActive ? t("msg.studentDeactivated") : t("msg.studentActivated"));
    } catch (e: any) {
      if (isStaleRevisionError(e)) {
        showMessage(t("msg.recordConflict"), "error");
        return;
      }
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function removeStudent(id: number) {
    showConfirm(t("msg.studentDeleteConfirm"), async () => {
      try {
        const currentStudent =
          studentList.find((item) => item.id === id) ?? allStudents.find((item) => item.id === id);
        if (!currentStudent) {
          throw new Error(t("msg.studentNotFound"));
        }
        await deleteStudent(id, currentStudent.version);
        await Promise.all([loadStudents(), loadAllStudents()]);
        showMessage(t("msg.studentDeleted"));
      } catch (e: any) {
        if (isStaleRevisionError(e)) {
          showMessage(t("msg.recordConflict"), "error");
          return;
        }
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      }
    });
  }

  const refreshStudentCardData = useCallback(
    async (studentId: number) => {
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
    },
    [localizedBillingModeLabel, localizedPaymentMethodLabel, month, showMessage, t, uiMonths, year]
  );

  const openStudentCard = useCallback(
    async (s: StudentDTO, options?: { inline?: boolean }) => {
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
    },
    [refreshStudentCardData, tab]
  );

  useEffect(() => {
    if (tab !== "students" || studentLoading) return;
    if (filteredStudentList.length === 0) {
      setSelectedStudentCard(null);
      return;
    }
    if (
      !selectedStudentCard ||
      !filteredStudentList.some((student) => student.id === selectedStudentCard.id)
    ) {
      void openStudentCard(filteredStudentList[0], { inline: true });
    }
  }, [filteredStudentList, openStudentCard, tab, studentLoading, selectedStudentCard]);

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
      showMessage(t("msg.studentCardLoadError", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function openExistingDuplicateStudent(studentId: number) {
    resetStudentDuplicateCheck();
    setStudentModalOpen(false);
    await openStudentInWorkspaceById(studentId);
  }

  async function enrollExistingDuplicateStudent(student: StudentDTO) {
    const enrollmentInput = getStudentOnboardingEnrollmentInput();
    if (!enrollmentInput) return;
    if (!student.isActive) {
      resetStudentDuplicateCheck();
      setStudentModalOpen(false);
      await openStudentInWorkspaceById(student.id);
      showMessage(t("msg.inactiveStudentCannotEnroll"), "error");
      return;
    }

    try {
      const existing = await listEnrollments(student.id, enrollmentInput.courseId);
      if (existing.length === 0) {
        await createEnrollment(
          student.id,
          enrollmentInput.courseId,
          enrollmentInput.billingMode,
          enrollmentInput.chargeMaterials,
          enrollmentInput.lessonPriceOverride,
          enrollmentInput.subscriptionLessonPrice,
          enrollmentInput.note
        );
      }
      resetStudentDuplicateCheck();
      await Promise.all([loadStudents(), loadAllStudents()]);
      if (studentCreateFlow === "save_and_add_another") {
        resetStudentFieldsForNextEntry();
        showMessage(t(existing.length > 0 ? "msg.studentAlreadyEnrolled" : "msg.enrollmentCreatedExisting"));
        return;
      }
      setStudentModalOpen(false);
      openAttendanceForStudentCourse(student.id, enrollmentInput.courseId);
      showMessage(t(existing.length > 0 ? "msg.studentAlreadyEnrolled" : "msg.enrollmentCreatedExisting"));
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
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
  const [allCourses, setAllCourses] = useState<CourseDTO[]>([]);
  const [courseQ, setCourseQ] = useState("");
  const [courseTypeFilter, setCourseTypeFilter] = useState<"" | "group" | "individual">("");
  const [courseTeacherFilter, setCourseTeacherFilter] = useState("all");
  const [coursePricingFilter, setCoursePricingFilter] = useState<
    "all" | "lesson" | "subscription" | "both" | "lesson_only" | "subscription_only"
  >("all");
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

  const handleCourseTypeChange = useCallback(
    (value: "group" | "individual") => {
      setCfType(value);
      if (!editingCourse) {
        setCfLessonPrice(defaultLessonPriceForCourseType(value));
      }
    },
    [editingCourse]
  );

  const handleCoursePriceChange = (value: string, setter: (value: string) => void) => {
    const next = normalizeMoneyInput(value);
    if (next !== null) setter(next);
  };

  const loadAllCourses = useCallback(async () => {
    const data = await listCourses("");
    setAllCourses(data);
    return data;
  }, []);

  const loadCourses = useCallback(async () => {
    setCourseLoading(true);
    try {
      return await loadAllCourses();
    } finally {
      setCourseLoading(false);
    }
  }, [loadAllCourses]);

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

  const courseTeacherOptions = useMemo(() => {
    const seen = new Map<number, string>();
    for (const course of allCourses) {
      if (!course.teacherId || !course.teacherName) continue;
      if (!seen.has(course.teacherId)) {
        seen.set(course.teacherId, course.teacherName);
      }
    }
    return Array.from(seen.entries())
      .map(([id, label]) => ({ value: String(id), label }))
      .sort((a, b) => a.label.localeCompare(b.label));
  }, [allCourses]);

  const courseList = useMemo(() => {
    const q = courseQ.trim().toLowerCase();
    return allCourses.filter((course) => {
      if (q) {
        const haystack = `${course.name} ${course.teacherName || ""}`.toLowerCase();
        if (!haystack.includes(q)) {
          return false;
        }
      }

      if (courseTypeFilter && course.type !== courseTypeFilter) {
        return false;
      }

      if (courseTeacherFilter === "none") {
        if (course.teacherId || course.teacherName?.trim()) {
          return false;
        }
      } else if (
        courseTeacherFilter !== "all" &&
        String(course.teacherId ?? "") !== courseTeacherFilter
      ) {
        return false;
      }

      const hasLessonPrice = course.lessonPrice > 0;
      const hasSubscriptionPrice = course.subscriptionPrice > 0;
      switch (coursePricingFilter) {
        case "lesson":
          if (!hasLessonPrice) return false;
          break;
        case "subscription":
          if (!hasSubscriptionPrice) return false;
          break;
        case "both":
          if (!hasLessonPrice || !hasSubscriptionPrice) return false;
          break;
        case "lesson_only":
          if (!hasLessonPrice || hasSubscriptionPrice) return false;
          break;
        case "subscription_only":
          if (!hasSubscriptionPrice || hasLessonPrice) return false;
          break;
      }

      return true;
    });
  }, [allCourses, coursePricingFilter, courseQ, courseTeacherFilter, courseTypeFilter]);

  const courseFiltersActive = Boolean(
    courseQ.trim() ||
    courseTypeFilter ||
    courseTeacherFilter !== "all" ||
    coursePricingFilter !== "all"
  );

  const clearCourseFilters = useCallback(() => {
    setCourseQ("");
    setCourseTypeFilter("");
    setCourseTeacherFilter("all");
    setCoursePricingFilter("all");
  }, []);

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
    setCfLessonPrice(defaultLessonPriceForCourseType("group"));
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
    const validationError = validatePersonName(name, "teacher", true);
    if (validationError) {
      showMessage(t(validationError), "error");
      return;
    }

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
          editingCourse.version,
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
      await loadCourses();
      showMessage(editingCourse ? t("msg.courseUpdated") : t("msg.courseCreated"));
    } catch (e: any) {
      if (isStaleRevisionError(e)) {
        showMessage(t("msg.recordConflict"), "error");
        return;
      }
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }

  async function removeCourse(id: number) {
    showConfirm(t("msg.courseDeleteConfirm"), async () => {
      try {
        const currentCourse =
          courseList.find((item) => item.id === id) ?? allCourses.find((item) => item.id === id);
        if (!currentCourse) {
          throw new Error(t("msg.courseNotFound"));
        }
        await deleteCourse(id, currentCourse.version);
        await loadCourses();
        showMessage(t("msg.courseDeleted"));
      } catch (e: any) {
        if (isStaleRevisionError(e)) {
          showMessage(t("msg.recordConflict"), "error");
          return;
        }
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      }
    });
  }

  // ---------------- Enrollments ----------------
  const [enrollments, setEnrollments] = useState<EnrollmentDTO[]>([]);
  const [enrStudentFilter, setEnrStudentFilter] = useState<number | undefined>(undefined);
  const [enrCourseFilter, setEnrCourseFilter] = useState<number | undefined>(undefined);
  const [enrLoading, setEnrLoading] = useState(false);

  const [enrModalOpen, setEnrModalOpen] = useState(false);
  const [editingEnr, setEditingEnr] = useState<EnrollmentDTO | null>(null);
  const [enrollmentStudentLocked, setEnrollmentStudentLocked] = useState(false);
  const [enrollmentOpenAttendanceAfterSave, setEnrollmentOpenAttendanceAfterSave] = useState(false);
  const [efStudentId, setEfStudentId] = useState<number>(0);
  const [efStudentSearch, setEfStudentSearch] = useState("");
  const [efStudentPickerOpen, setEfStudentPickerOpen] = useState(false);
  const efStudentComboRef = useRef<HTMLDivElement | null>(null);
  const [efCourseId, setEfCourseId] = useState<number>(0);
  const [efMode, setEfMode] = useState<"subscription" | "per_lesson">("per_lesson");
  const [efChargeMaterials, setEfChargeMaterials] = useState(true);
  const [efLessonPriceOverride, setEfLessonPriceOverride] = useState("0");
  const [efSubscriptionLessonPrice, setEfSubscriptionLessonPrice] = useState("0");
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

  const selectedEnrollmentCourse = useMemo(
    () => allCourses.find((course) => course.id === efCourseId) ?? null,
    [allCourses, efCourseId]
  );
  const enrollmentCourseOptions = useMemo(() => {
    if (!enrollmentStudentLocked || editingEnr || selectedStudentCard?.id !== efStudentId) {
      return allCourses;
    }
    const enrolledCourseIds = new Set(
      studentCardEnrollments.map((enrollment) => enrollment.courseId)
    );
    return allCourses.filter((course) => !enrolledCourseIds.has(course.id));
  }, [
    allCourses,
    editingEnr,
    efStudentId,
    enrollmentStudentLocked,
    selectedStudentCard?.id,
    studentCardEnrollments,
  ]);

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
    setEnrollmentStudentLocked(false);
    setEnrollmentOpenAttendanceAfterSave(false);
    setEfStudentId(0);
    setEfStudentSearch("");
    setEfStudentPickerOpen(false);
    setEfCourseId(initialCourseId);
    setEfMode("per_lesson");
    setEfChargeMaterials(true);
    setEfLessonPriceOverride(String(allCourses[0]?.lessonPrice ?? 0));
    setEfSubscriptionLessonPrice(String(allCourses[0]?.subscriptionPrice ?? 0));
    setEfNote("");
    setEnrModalOpen(true);
  }

  function openAddEnrollmentForStudent(student: StudentDTO) {
    if (!student.isActive) {
      showMessage(t("msg.inactiveStudentCannotEnroll"), "error");
      return;
    }
    if (allCourses.length === 0) {
      showMessage(t("msg.noAvailableCourses"), "error");
      setTab("courses");
      return;
    }

    const enrolledCourseIds = new Set(
      selectedStudentCard?.id === student.id
        ? studentCardEnrollments.map((enrollment) => enrollment.courseId)
        : []
    );
    const availableCourses = allCourses.filter((course) => !enrolledCourseIds.has(course.id));
    if (availableCourses.length === 0) {
      showMessage(t("msg.studentEnrolledInAllCourses"), "error");
      return;
    }

    const initialCourse = availableCourses[0];
    setEditingEnr(null);
    setEnrollmentStudentLocked(true);
    setEnrollmentOpenAttendanceAfterSave(true);
    setEfStudentId(student.id);
    setEfStudentSearch(student.fullName);
    setEfStudentPickerOpen(false);
    setEfCourseId(initialCourse.id);
    setEfMode(BillingModePerLesson);
    setEfChargeMaterials(true);
    setEfLessonPriceOverride(String(initialCourse.lessonPrice ?? 0));
    setEfSubscriptionLessonPrice(String(initialCourse.subscriptionPrice ?? 0));
    setEfNote("");
    setStudentCardOpen(false);
    setTab("enrollments");
    setEnrModalOpen(true);
  }

  function openEditEnrollment(e: EnrollmentDTO) {
    setEditingEnr(e);
    setEnrollmentStudentLocked(true);
    setEnrollmentOpenAttendanceAfterSave(false);
    setEfStudentId(e.studentId);
    setEfStudentSearch(e.studentName);
    setEfStudentPickerOpen(false);
    setEfCourseId(e.courseId);
    setEfMode(e.billingMode);
    setEfChargeMaterials(e.chargeMaterials);
    setEfLessonPriceOverride(String(e.lessonPriceOverride));
    setEfSubscriptionLessonPrice(String(e.subscriptionLessonPrice));
    setEfNote(e.note);
    setEnrModalOpen(true);
  }

  function handleEnrollmentCourseIdChange(value: number) {
    setEfCourseId(value);
    const course = allCourses.find((item) => item.id === value);
    if (course && efMode === "per_lesson") {
      setEfLessonPriceOverride(String(course.lessonPrice ?? 0));
    }
    if (course && efMode === "subscription") {
      setEfSubscriptionLessonPrice(String(course.subscriptionPrice ?? 0));
    }
  }

  function handleEnrollmentModeChange(value: "per_lesson" | "subscription") {
    setEfMode(value);
    if (
      value === "per_lesson" &&
      (efLessonPriceOverride.trim() === "" || Number(efLessonPriceOverride) === 0)
    ) {
      setEfLessonPriceOverride(String(selectedEnrollmentCourse?.lessonPrice ?? 0));
    }
    if (
      value === "subscription" &&
      (efSubscriptionLessonPrice.trim() === "" || Number(efSubscriptionLessonPrice) === 0)
    ) {
      setEfSubscriptionLessonPrice(String(selectedEnrollmentCourse?.subscriptionPrice ?? 0));
    }
  }

  async function saveEnrollment() {
    const lessonPriceOverrideValue =
      efLessonPriceOverride.trim() === "" ? 0 : Number(efLessonPriceOverride);
    const subscriptionLessonPriceValue =
      efSubscriptionLessonPrice.trim() === "" ? 0 : Number(efSubscriptionLessonPrice);

    if (efStudentId <= 0 || efCourseId <= 0) {
      showMessage(t("msg.chooseStudentAndCourse"), "error");
      return;
    }
    if (!Number.isFinite(lessonPriceOverrideValue) || lessonPriceOverrideValue < 0) {
      showMessage(t("msg.lessonPriceOverrideRange"), "error");
      return;
    }
    if (!Number.isFinite(subscriptionLessonPriceValue) || subscriptionLessonPriceValue < 0) {
      showMessage(t("msg.subscriptionLessonPriceRange"), "error");
      return;
    }

    try {
      let result: EnrollmentDTO;
      if (editingEnr) {
        result = await updateEnrollment(
          editingEnr.id,
          editingEnr.version,
          efMode,
          efChargeMaterials,
          efMode === "per_lesson" ? lessonPriceOverrideValue : 0,
          efMode === "subscription" ? subscriptionLessonPriceValue : 0,
          efNote
        );
        showMessage(t("msg.enrollmentUpdated"));
      } else {
        result = await createEnrollment(
          efStudentId,
          efCourseId,
          efMode,
          efChargeMaterials,
          efMode === "per_lesson" ? lessonPriceOverrideValue : 0,
          efMode === "subscription" ? subscriptionLessonPriceValue : 0,
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
      const shouldRefreshStudentCard =
        selectedStudentCard?.id === result.studentId && (tab === "students" || studentCardOpen);
      await Promise.all([
        loadEnrollments(),
        loadInvoices({ syncDrafts: false }),
        loadStudents(),
        loadAllStudents(),
        loadDebtors(),
        shouldRefreshStudentCard ? refreshStudentCardData(result.studentId) : Promise.resolve(),
      ]);
      const shouldOpenAttendance = !editingEnr && enrollmentOpenAttendanceAfterSave;
      setEnrollmentStudentLocked(false);
      setEnrollmentOpenAttendanceAfterSave(false);
      if (shouldOpenAttendance) {
        openAttendanceForStudentCourse(result.studentId, result.courseId);
      }
    } catch (e: any) {
      if (isStaleRevisionError(e)) {
        showMessage(t("msg.recordConflict"), "error");
        return;
      }
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
  const [attendanceFocusTarget, setAttendanceFocusTarget] =
    useState<AttendanceFocusTarget | null>(null);

  function openAttendanceForStudentCourse(studentId: number, targetCourseId: number) {
    setCourseFilter(targetCourseId);
    setAttQ("");
    setAttFilter("all");
    setAttendanceFocusTarget({ studentId, courseId: targetCourseId });
    setTab("attendance");
  }

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
      const data = await fetchRows(year, month, courseFilter);
      setRows(data);
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
        (s, r) =>
          s +
          (r.billingMode === BillingModePerLesson
            ? r.hours * r.lessonPrice
            : r.hours * r.subscriptionLessonPrice),
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
      filtered = filtered.filter((r) => !r.hasRecord);
    } else if (attFilter === "filled") {
      filtered = filtered.filter((r) => r.hasRecord);
    } else if (attFilter === "zero") {
      filtered = filtered.filter((r) => r.hasRecord && r.hours === 0);
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
    const editableRows = rows;
    const filled = editableRows.filter((r) => r.hasRecord).length;
    const missing = editableRows.filter((r) => !r.hasRecord).length;
    const zero = editableRows.filter((r) => r.hasRecord && r.hours === 0).length;
    return { filled, missing, zero, total: editableRows.length };
  }, [rows]);

  const onChangeHours = useCallback(
    async (r: Row, v: number) => {
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
      } catch (e: any) {
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      } finally {
        setAttendanceSavingRows((prev) => {
          const next = { ...prev };
          delete next[r.enrollmentId];
          return next;
        });
      }
    },
    [localizedInvoiceStatusLabel, month, showMessage, t, year]
  );

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
        showMessage(t("msg.errorGeneric", { message: t("msg.invalidHoursValue") }), "error");
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

  const onDeleteEnrollmentFromSheet = async (id: number, version: number) => {
    showConfirm(t("msg.enrollmentDeleteConfirm"), async () => {
      try {
        await deleteEnrollment(id, version);
        await loadAttendance();
        showMessage(t("msg.enrollmentDeleted"));
      } catch (e: any) {
        if (isStaleRevisionError(e)) {
          showMessage(t("msg.recordConflict"), "error");
          return;
        }
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      }
    });
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
  const [invoiceEmailDraft, setInvoiceEmailDraft] = useState<InvoiceEmailPreviewResult | null>(
    null
  );
  const [invoiceEmailInvoiceID, setInvoiceEmailInvoiceID] = useState<number | null>(null);
  const [invoiceEmailSending, setInvoiceEmailSending] = useState(false);

  // Payment modal state
  const [paymentModalOpen, setPaymentModalOpen] = useState(false);
  const [paymentAmount, setPaymentAmount] = useState("");
  // Payment method stays fixed for new UI-created payments to keep the flow simple.
  const [paymentMethod, setPaymentMethod] = useState<"cash" | "bank">("cash");
  const [paymentNote, setPaymentNote] = useState("");
  const [paymentStudentId, setPaymentStudentId] = useState<number>(0);
  const [paymentStudentName, setPaymentStudentName] = useState("");
  const [paymentInvoiceId, setPaymentInvoiceId] = useState<number | undefined>(undefined);
  const [returnToDebtDetailsAfterPayment, setReturnToDebtDetailsAfterPayment] = useState(false);
  const [returnToStudentCardAfterPayment, setReturnToStudentCardAfterPayment] = useState(false);
  const [ensureAllPDFsResult, setEnsureAllPDFsResult] = useState<EnsureAllPDFsResult | null>(null);

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
        setInvItems(li);
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
      void loadInvoices({ syncDrafts: false });
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
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  };

  const onIssueOne = async (id: number) => {
    try {
      closeInvoiceMenu();
      const scrollY = window.scrollY;
      const currentInvoice =
        invItems.find((item) => item.id === id) ??
        (selectedInv?.id === id ? selectedInv : undefined);
      if (!currentInvoice) {
        throw new Error(t("msg.invoiceNotFound"));
      }
      const res = await issueOne(id, currentInvoice.version);
      pendingInvoiceScrollRestoreRef.current = scrollY;
      await loadInvoices({ syncDrafts: false });
      if (invoiceDetailsOpen && selectedInv?.id === id) {
        await loadInvoiceDetails(id);
      }
      const feedback = buildIssueFeedback(res, t);
      showMessage(feedback.text, feedback.type);
    } catch (e: any) {
      if (isStaleRevisionError(e)) {
        showMessage(t("msg.recordConflict"), "error");
        return;
      }
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  };

  const onReopenToDraft = useCallback(
    async (id: number) => {
      closeInvoiceMenu();
      showConfirm(
        t("msg.invoiceReopenConfirm"),
        async () => {
          try {
            const currentInvoice =
              invItems.find((item) => item.id === id) ??
              (selectedInv?.id === id ? selectedInv : undefined);
            if (!currentInvoice) {
              throw new Error(t("msg.invoiceNotFound"));
            }
            await reopenToDraft(id, currentInvoice.version);
            await loadInvoices({ syncDrafts: false });
            if (invoiceDetailsOpen && selectedInv?.id === id) {
              await loadInvoiceDetails(id);
            }
            showMessage(t("msg.invoiceReopened"));
          } catch (e: any) {
            if (isStaleRevisionError(e)) {
              showMessage(t("msg.recordConflict"), "error");
              return;
            }
            showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
          }
        },
        t("button.reopenDraft")
      );
    },
    [
      closeInvoiceMenu,
      invoiceDetailsOpen,
      invItems,
      loadInvoiceDetails,
      loadInvoices,
      selectedInv,
      showConfirm,
      showMessage,
      t,
    ]
  );

  const onGeneratePdf = useCallback(
    async (id: number) => {
      try {
        closeInvoiceMenu();
        const pdf = await ensurePdf(id);
        await loadInvoices({ syncDrafts: false });
        if (invoiceDetailsOpen && selectedInv?.id === id) {
          await loadInvoiceDetails(id);
        }
        showMessage(t("msg.pdfReady", { path: pdf.localPath ?? pdf.filename }));
      } catch (e: any) {
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      }
    },
    [
      closeInvoiceMenu,
      invoiceDetailsOpen,
      loadInvoiceDetails,
      loadInvoices,
      selectedInv,
      showMessage,
      t,
    ]
  );

  const onEnsureAllPDFs = useCallback(async () => {
    try {
      const result = await ensureAllPdfs(year, month);
      await loadInvoices({ syncDrafts: false });
      if (invoiceDetailsOpen && selectedInv) {
        await loadInvoiceDetails(selectedInv.id);
      }
      setEnsureAllPDFsResult(result);
    } catch (e: any) {
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [
    invoiceDetailsOpen,
    loadInvoiceDetails,
    loadInvoices,
    month,
    selectedInv,
    showMessage,
    t,
    year,
  ]);

  const onDownloadPdf = useCallback(
    async (id: number) => {
      try {
        closeInvoiceMenu();
        const pdf = await ensurePdf(id);
        await loadInvoices({ syncDrafts: false });
        if (invoiceDetailsOpen && selectedInv?.id === id) {
          await loadInvoiceDetails(id);
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
    },
    [
      closeInvoiceMenu,
      invoiceDetailsOpen,
      loadInvoiceDetails,
      loadInvoices,
      selectedInv,
      showMessage,
      t,
    ]
  );

  const closeInvoiceEmailModal = useCallback(() => {
    setInvoiceEmailDraft(null);
    setInvoiceEmailInvoiceID(null);
    setInvoiceEmailSending(false);
  }, []);

  const onPreviewInvoiceEmail = useCallback(
    async (id: number) => {
      try {
        closeInvoiceMenu();
        const preview = await previewInvoiceEmail(id);
        setInvoiceEmailDraft(preview);
        setInvoiceEmailInvoiceID(id);
      } catch (e: any) {
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      }
    },
    [closeInvoiceMenu, showMessage, t]
  );

  const onSendInvoiceEmail = useCallback(async () => {
    if (!invoiceEmailDraft || invoiceEmailInvoiceID == null) return;
    if (invoiceEmailDraft.to.trim() === "") {
      showMessage(t("msg.invoiceEmailRecipientRequired"), "error");
      return;
    }
    try {
      setInvoiceEmailSending(true);
      const result = await sendInvoiceEmail(invoiceEmailInvoiceID, {
        to: invoiceEmailDraft.to,
        subject: invoiceEmailDraft.subject,
        body: invoiceEmailDraft.body,
      });
      await loadInvoices({ syncDrafts: false });
      if (invoiceDetailsOpen && selectedInv?.id === invoiceEmailInvoiceID) {
        await loadInvoiceDetails(invoiceEmailInvoiceID);
      }
      closeInvoiceEmailModal();
      showMessage(t("msg.invoiceEmailSent", { email: result.to }));
    } catch (e: any) {
      setInvoiceEmailSending(false);
      showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
    }
  }, [
    closeInvoiceEmailModal,
    invoiceDetailsOpen,
    invoiceEmailDraft,
    invoiceEmailInvoiceID,
    loadInvoiceDetails,
    loadInvoices,
    selectedInv,
    showMessage,
    t,
  ]);

  const buildInvoiceMenuItems = useCallback(
    (invoice: Pick<InvoiceDTO, "id" | "status"> & { pdfReady?: boolean }) => {
      const menuItems: Array<{ label: string; onClick: () => void }> = [];

      for (const action of getInvoiceMenuActions(invoice)) {
        if (action === "reopenDraft") {
          menuItems.push({
            label: t("button.reopenDraft"),
            onClick: () => void onReopenToDraft(invoice.id),
          });
          continue;
        }
        if (action === "createPdf") {
          menuItems.push({
            label: t("button.createPdf"),
            onClick: () => void onGeneratePdf(invoice.id),
          });
        }
      }

      return menuItems;
    },
    [onGeneratePdf, onReopenToDraft, t]
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

  const onGenerateArchivePdf = useCallback(
    async (id: number) => {
      try {
        const pdf = await ensurePdf(id);
        await loadInvoiceArchive();
        showMessage(t("msg.pdfReady", { path: pdf.localPath ?? pdf.filename }));
      } catch (e: any) {
        showMessage(t("msg.errorGeneric", { message: String(e?.message ?? e) }), "error");
      }
    },
    [loadInvoiceArchive, showMessage, t]
  );

  // ---------------- Render ----------------
  const showMonthPicker = tab === "dashboard" || tab === "attendance" || tab === "invoice";
  const selectedInvPdfReady = selectedInv?.pdfReady ?? false;
  const openInvoiceMenuItems = useMemo(() => {
    if (!openInvoiceMenu) return [];

    if (openInvoiceMenu.kind === "modal" && selectedInv) {
      return buildInvoiceMenuItems({ ...selectedInv, pdfReady: selectedInvPdfReady });
    }

    const rowInvoice = invItems.find((item) => item.id === openInvoiceMenu.invoiceId);
    return rowInvoice ? buildInvoiceMenuItems(rowInvoice) : [];
  }, [buildInvoiceMenuItems, invItems, openInvoiceMenu, selectedInv, selectedInvPdfReady]);

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
        authRequired
          ? {
              id: "logout",
              label: t("auth.logout"),
              onClick: () => {
                void handleLogout();
              },
            }
          : null,
      ].filter(
        (value): value is { id: string; label: string; onClick: () => void } => value !== null
      ),
    [authRequired, handleLogout, t]
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
                onOpenStudent={(studentId) => void openStudentInWorkspaceById(studentId)}
                recentPayments={recentPayments}
              />
            )}

            {tab === "students" && (
              <StudentsScreen
                students={filteredStudentList}
                loading={studentLoading}
                query={studentQ}
                statusFilter={studentStatusFilter}
                debtFilter={studentDebtFilter}
                balanceFilter={studentBalanceFilter}
                ageFilter={studentAgeFilter}
                sortOption={studentSortOption}
                hasActiveStudentFilters={hasActiveStudentFilters}
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
                onStatusFilterChange={setStudentStatusFilter}
                onDebtFilterChange={setStudentDebtFilter}
                onBalanceFilterChange={setStudentBalanceFilter}
                onAgeFilterChange={setStudentAgeFilter}
                onSortOptionChange={setStudentSortOption}
                onResetStudentFilters={resetStudentFilters}
                onAddStudent={openAddStudent}
                onSelectStudent={(student) => void openStudentCard(student, { inline: true })}
                onEditStudent={openEditStudent}
                onToggleActive={(student) => void toggleStudentActive(student)}
                onDeleteStudent={(studentId) => void removeStudent(studentId)}
                onAddPayment={openStudentCardPaymentModal}
                onCopyDebtRu={() => void copyStudentCardDebtMessage("ru")}
                onCopyDebtLv={() => void copyStudentCardDebtMessage("lv")}
                onDeletePayment={deleteStudentPayment}
                onManageEnrollments={() =>
                  selectedStudentCard && openAddEnrollmentForStudent(selectedStudentCard)
                }
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
                studentDuplicateCheckResult={studentDuplicateCheckResult}
                payerRoleOptions={payerRoleOptions}
                allCourses={allCourses}
                onboardingCourseId={sfCourseId}
                onboardingMode={sfEnrollmentMode}
                onboardingChargeMaterials={sfEnrollmentChargeMaterials}
                onboardingLessonPrice={sfEnrollmentLessonPrice}
                onboardingSubscriptionPrice={sfEnrollmentSubscriptionPrice}
                onboardingNote={sfEnrollmentNote}
                onboardingSettingsOpen={sfEnrollmentSettingsOpen}
                onSfNameChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfName(value);
                }}
                onSfPersonalCodeChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfPersonalCode(value);
                }}
                onSfPhoneChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfPhone(value);
                }}
                onSfEmailChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfEmail(value);
                }}
                onSfNoteChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfNote(value);
                }}
                onSfIsMinorChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfIsMinor(value);
                }}
                onSfPayerNameChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfPayerName(value);
                }}
                onSfPayerRoleChange={(value) => {
                  resetStudentDuplicateCheck();
                  setSfPayerRole(value);
                }}
                onOnboardingCourseIdChange={handleStudentOnboardingCourseChange}
                onOnboardingModeChange={handleStudentOnboardingModeChange}
                onOnboardingChargeMaterialsChange={setSfEnrollmentChargeMaterials}
                onOnboardingLessonPriceChange={setSfEnrollmentLessonPrice}
                onOnboardingSubscriptionPriceChange={setSfEnrollmentSubscriptionPrice}
                onOnboardingNoteChange={setSfEnrollmentNote}
                onOnboardingSettingsOpenChange={setSfEnrollmentSettingsOpen}
                onSaveStudent={() => void saveStudent()}
                onSaveStudentAndAddAnother={() => void saveStudent(false, "save_and_add_another")}
                onOpenExistingDuplicateStudent={(studentId) =>
                  void openExistingDuplicateStudent(studentId)
                }
                onEnrollExistingDuplicateStudent={(student) =>
                  void enrollExistingDuplicateStudent(student)
                }
                onCreateStudentAnyway={() => void saveStudent(true, studentCreateFlow)}
                onCloseStudentModal={() => {
                  resetStudentDuplicateCheck();
                  setStudentCreateFlow("save");
                  setStudentModalOpen(false);
                }}
              />
            )}

            {tab === "courses" && (
              <CoursesScreen
                loading={courseLoading}
                courses={courseList}
                query={courseQ}
                typeFilter={courseTypeFilter}
                teacherFilter={courseTeacherFilter}
                pricingFilter={coursePricingFilter}
                teacherOptions={courseTeacherOptions}
                hasActiveFilters={courseFiltersActive}
                canDeleteCourses={canDeleteCourses}
                courseTypeLabel={localizedCourseTypeLabel}
                formatEUR={formatEUR}
                onQueryChange={setCourseQ}
                onTypeFilterChange={setCourseTypeFilter}
                onTeacherFilterChange={setCourseTeacherFilter}
                onPricingFilterChange={setCoursePricingFilter}
                onClearFilters={clearCourseFilters}
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
                onCfTypeChange={handleCourseTypeChange}
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
                enrollmentCourseOptions={enrollmentCourseOptions}
                billingModeLabel={localizedBillingModeLabel}
                courseTypeLabel={localizedCourseTypeLabel}
                onStudentFilterChange={setEnrStudentFilter}
                onCourseFilterChange={setEnrCourseFilter}
                onAddEnrollment={openAddEnrollment}
                onOpenStudents={() => setTab("students")}
                onOpenCourses={() => setTab("courses")}
                onOpenStudent={(studentId) => void openStudentCardById(studentId)}
                onEditEnrollment={openEditEnrollment}
                enrollmentModalOpen={enrModalOpen}
                editingEnrollment={editingEnr}
                studentSearch={efStudentSearch}
                studentId={efStudentId}
                studentPickerOpen={efStudentPickerOpen}
                filteredStudents={filteredEnrollmentStudents}
                selectedStudent={selectedEnrollmentStudent}
                enrollmentStudentLocked={enrollmentStudentLocked}
                enrollmentSaveLabel={
                  enrollmentOpenAttendanceAfterSave
                    ? t("button.saveAndOpenAttendance")
                    : undefined
                }
                enrollmentCourseId={efCourseId}
                enrollmentMode={efMode}
                enrollmentChargeMaterials={efChargeMaterials}
                enrollmentLessonPriceOverride={efLessonPriceOverride}
                enrollmentSubscriptionLessonPrice={efSubscriptionLessonPrice}
                enrollmentNote={efNote}
                studentComboRef={efStudentComboRef}
                onStudentSearchChange={setEfStudentSearch}
                onStudentIdChange={(value) => setEfStudentId(value ?? 0)}
                onStudentPickerOpenChange={setEfStudentPickerOpen}
                onEnrollmentCourseIdChange={handleEnrollmentCourseIdChange}
                onEnrollmentModeChange={handleEnrollmentModeChange}
                onEnrollmentChargeMaterialsChange={setEfChargeMaterials}
                onEnrollmentLessonPriceOverrideChange={setEfLessonPriceOverride}
                onEnrollmentSubscriptionLessonPriceChange={setEfSubscriptionLessonPrice}
                onEnrollmentNoteChange={setEfNote}
                onSaveEnrollment={() => void saveEnrollment()}
                onCloseEnrollmentModal={() => {
                  setEnrModalOpen(false);
                  setEnrollmentStudentLocked(false);
                  setEnrollmentOpenAttendanceAfterSave(false);
                }}
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
                year={year}
                month={month}
                perLessonTotal={perLessonTotal}
                courseTypeLabel={localizedCourseTypeLabel}
                formatEUR={formatEUR}
                normalizeHoursDraftInput={normalizeHoursDraftInput}
                getAttendanceStepBase={getAttendanceStepBase}
                getAttendanceInputValue={getAttendanceInputValue}
                setAttendanceDraft={setAttendanceDraft}
                clearAttendanceDraft={clearAttendanceDraft}
                commitAttendanceDraft={commitAttendanceDraft}
                onChangeHours={onChangeHours}
                onRefresh={() => void loadAttendance()}
                onOpenInvoices={() => setTab("invoice")}
                onOpenEnrollments={() => setTab("enrollments")}
                onCourseFilterChange={setCourseFilter}
                onQueryChange={setAttQ}
                onFilterChange={setAttFilter}
                onOpenStudent={(studentId) => void openStudentCardById(studentId)}
                onDeleteEnrollmentFromSheet={(enrollmentId, enrollmentVersion) =>
                  void onDeleteEnrollmentFromSheet(enrollmentId, enrollmentVersion)
                }
                focusTarget={attendanceFocusTarget}
                onFocusTargetHandled={() => setAttendanceFocusTarget(null)}
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
                formatDateTime={formatDateTime}
                renderInvoiceActionsMenu={(invoice) => renderInvoiceActionsMenu(invoice)}
                onStatusChange={setInvStatus}
                onQueryChange={setInvQ}
                onRefresh={() => void loadInvoices({ syncDrafts: true, showSyncFeedback: true })}
                onEnsureAllPdfs={() => void onEnsureAllPDFs()}
                onResetFilters={() => {
                  setInvStatus("all");
                  setInvQ("");
                }}
                onOpenAttendance={() => setTab("attendance")}
                onOpenStudent={(studentId) => void openStudentCardById(studentId)}
                onOpenInvoice={(invoiceId) => void onOpenInvoice(invoiceId)}
                onIssueOne={(invoiceId) => void onIssueOne(invoiceId)}
                onGeneratePdf={(invoiceId) => void onGeneratePdf(invoiceId)}
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
                onCopyDebtForStudentRu={(studentId) =>
                  void copyDebtMessageForStudentId(studentId, "ru")
                }
                onCopyDebtForStudentLv={(studentId) =>
                  void copyDebtMessageForStudentId(studentId, "lv")
                }
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
                onResetFilters={resetAuditFilters}
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
                canCreateBackups={canCreateBackups}
                canManageSettings={canManageSettings}
                canViewInvoiceArchive={canViewInvoiceArchive}
                creatingBackup={creatingBackup}
                canManageUsers={canManageUsers}
                invoiceArchiveLoading={invoiceArchiveLoading}
                invoiceArchive={invoiceArchive}
                formatEUR={formatEUR}
                invoiceStatusLabel={(status) => invoiceStatusLabel(status, t)}
                invoiceEmailSettingsLoading={invoiceEmailSettingsLoading}
                savingInvoiceEmailSettings={savingInvoiceEmailSettings}
                invoiceEmailSettings={invoiceEmailSettings}
                invoiceEmailSubjectTemplate={invoiceEmailSubjectTemplate}
                invoiceEmailBodyTemplate={invoiceEmailBodyTemplate}
                invoiceEmailReplyTo={invoiceEmailReplyTo}
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
                onRefreshInvoiceArchive={loadInvoiceArchive}
                onSetTab={setTab}
                onOpenInvoice={onOpenInvoice}
                onGenerateInvoiceArchivePdf={onGenerateArchivePdf}
                onInvoiceEmailSubjectTemplateChange={setInvoiceEmailSubjectTemplate}
                onInvoiceEmailBodyTemplateChange={setInvoiceEmailBodyTemplate}
                onInvoiceEmailReplyToChange={setInvoiceEmailReplyTo}
                onSaveInvoiceEmailSettings={handleSaveInvoiceEmailSettings}
                onResetInvoiceEmailSettings={handleResetInvoiceEmailSettings}
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
              note={paymentNote}
              onAmountChange={setPaymentAmount}
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
              invoiceStatusLabel={localizedInvoiceStatusLabel}
              formatEUR={formatEUR}
              formatHoursValue={formatHoursValue}
              formatDateTime={formatDateTime}
              pdfReady={selectedInvPdfReady}
              onOpenStudent={(studentId) => void openStudentCardById(studentId)}
              onIssue={(invoiceId) => void onIssueOne(invoiceId)}
              onGeneratePdf={(invoiceId) => void onGeneratePdf(invoiceId)}
              onDownloadPdf={(invoiceId) => void onDownloadPdf(invoiceId)}
              onSendEmail={(invoiceId) => void onPreviewInvoiceEmail(invoiceId)}
              onAddPayment={() => openPaymentModal(selectedInv, invSummary)}
              onReopenToDraft={(invoiceId) => void onReopenToDraft(invoiceId)}
              onClose={() => setInvoiceDetailsOpen(false)}
              canSendEmail={Boolean(sessionCapabilities.emailSend)}
              t={t}
            />
          )}

          {ensureAllPDFsResult && (
            <InvoiceEnsureAllPDFsModal
              result={ensureAllPDFsResult}
              months={uiMonths}
              invoiceStatusLabel={localizedInvoiceStatusLabel}
              onClose={() => setEnsureAllPDFsResult(null)}
              t={t}
            />
          )}

          <InvoiceEmailModal
            isOpen={invoiceEmailDraft !== null}
            to={invoiceEmailDraft?.to ?? ""}
            subject={invoiceEmailDraft?.subject ?? ""}
            body={invoiceEmailDraft?.body ?? ""}
            attachmentFilename={invoiceEmailDraft?.attachmentFilename ?? ""}
            sending={invoiceEmailSending}
            onToChange={(value) =>
              setInvoiceEmailDraft((current) => (current ? { ...current, to: value } : current))
            }
            onSubjectChange={(value) =>
              setInvoiceEmailDraft((current) =>
                current ? { ...current, subject: value } : current
              )
            }
            onBodyChange={(value) =>
              setInvoiceEmailDraft((current) => (current ? { ...current, body: value } : current))
            }
            onCancel={closeInvoiceEmailModal}
            onSubmit={() => void onSendInvoiceEmail()}
            t={t}
          />

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
                openAddEnrollmentForStudent(selectedStudentCard);
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
