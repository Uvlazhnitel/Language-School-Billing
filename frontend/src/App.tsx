import { useEffect, useMemo, useState, useCallback, useRef } from "react";
import "./App.css";

import { fetchRows, saveCount, addOneMass, deleteEnrollment, Row } from "./lib/attendance";

import {
  listInvoices,
  getInvoice,
  genDrafts,
  issueOne,
  issueAll,
  reopenToDraft,
  ensurePdfAndOpen,
  InvoiceListItem,
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
import { AppDirs, BackupNow, OpenFile } from "../wailsjs/go/main/App";

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

const months = [
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

function payerRoleLabel(relation: string): string {
  switch (relation) {
    case "mother":
      return "Мама";
    case "father":
      return "Папа";
    case "grandmother":
      return "Бабушка";
    case "grandfather":
      return "Дедушка";
    case "guardian":
      return "Опекун";
    default:
      return "Другое";
  }
}

function courseTypeLabel(type: string): string {
  switch (type) {
    case "group":
      return "группа";
    case "individual":
      return "индивидуально";
    default:
      return type;
  }
}

function billingModeLabel(mode: string): string {
  switch (mode) {
    case "per_lesson":
      return "по занятиям";
    case "subscription":
      return "абонемент";
    default:
      return mode;
  }
}

function paymentMethodLabel(method: string): string {
  switch (method) {
    case "cash":
      return "Наличные";
    case "bank":
      return "Банк";
    default:
      return method;
  }
}

function invoiceStatusLabel(status: string): string {
  switch (status) {
    case "draft":
      return "черновик";
    case "issued":
      return "выставлен";
    case "paid":
      return "оплачен";
    case "canceled":
      return "отменён";
    case "all":
      return "все";
    default:
      return status;
  }
}

type Tab = "students" | "courses" | "enrollments" | "attendance" | "invoice" | "debtors";

const TAB_META: Record<Tab, { eyebrow: string; title: string; description: string }> = {
  students: {
    eyebrow: "Люди",
    title: "Ученики",
    description: "Управляйте базой учеников, контактами и активным списком в одном месте.",
  },
  courses: {
    eyebrow: "Программы",
    title: "Курсы и цены",
    description: "Настраивайте предложения, держите цены в порядке и удобно редактируйте каталог.",
  },
  enrollments: {
    eyebrow: "Связи",
    title: "Зачисления",
    description: "Привязывайте учеников к курсам, задавайте скидки и не путайтесь в режимах оплаты.",
  },
  attendance: {
    eyebrow: "Учёт",
    title: "Посещаемость",
    description: "Быстро отмечайте количество занятий и готовьте точное начисление за месяц.",
  },
  invoice: {
    eyebrow: "Счета",
    title: "Счета и сводка",
    description: "Создавайте, выставляйте, проверяйте и оплачивайте счета в понятном помесячном потоке.",
  },
  debtors: {
    eyebrow: "Долги",
    title: "Должники",
    description: "Смотрите, кто ещё должен оплату, и записывайте платежи без потери истории.",
  },
};

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
  const [tab, setTab] = useState<Tab>("students");
  const [appDirs, setAppDirs] = useState<Record<string, string> | null>(null);
  const [creatingBackup, setCreatingBackup] = useState(false);

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

  const showConfirm = (
    messageText: string,
    onConfirm: () => void | Promise<void>,
    confirmButtonLabel?: string
  ) => {
    setConfirmDialog({ isOpen: true, message: messageText, onConfirm, confirmButtonLabel });
  };

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

    void AppDirs()
      .then((dirs) => {
        if (!cancelled) setAppDirs(dirs);
      })
      .catch((e: any) => {
        if (!cancelled) {
          showMessage(`Failed to load app folders: ${String(e?.message ?? e)}`, "error");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [showMessage]);

  // Shared month/year for Attendance + Invoices
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);

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
  const [studentCardDeletingPaymentId, setStudentCardDeletingPaymentId] = useState<number | null>(null);

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
    if (tab === "students") loadStudents();
  }, [tab, loadStudents]);

  useEffect(() => {
    void loadAllStudents();
  }, [loadAllStudents]);

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
      showMessage("Введите имя ученика", "error");
      return;
    }
    if (sfIsMinor && !sfPayerName.trim()) {
      showMessage("Для несовершеннолетнего ученика нужно указать имя плательщика", "error");
      return;
    }
    if (sfIsMinor && !sfPayerRole) {
      showMessage("Для несовершеннолетнего ученика нужно выбрать роль плательщика", "error");
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
      showMessage(
        editingStudent ? "Ученик успешно обновлён" : "Ученик успешно создан"
      );
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  }

  async function toggleStudentActive(s: StudentDTO) {
    try {
      await setStudentActive(s.id, !s.isActive);
      await Promise.all([loadStudents(), loadAllStudents()]);
      showMessage(s.isActive ? "Ученик деактивирован" : "Ученик активирован");
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  }

  async function removeStudent(id: number) {
    showConfirm(
      "Удалить ученика? Вместе с ним будут удалены зачисления и записи посещаемости. Удаление не сработает, если у ученика уже есть счета или оплаты. Это действие нельзя отменить.",
      async () => {
        try {
          await deleteStudent(id);
          await Promise.all([loadStudents(), loadAllStudents()]);
          showMessage("Ученик удалён");
        } catch (e: any) {
          showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
        }
      }
    );
  }

  async function refreshStudentCardData(studentId: number) {
    try {
      const [enr, bal, debts, payments] = await Promise.all([
        listEnrollments(studentId, undefined),
        studentBalance(studentId),
        studentDebtDetails(studentId),
        paymentListForStudent(studentId),
      ]);
      setStudentCardEnrollments(enr);
      setStudentCardBalance(bal);
      setStudentCardDebts(debts);
      setStudentCardPayments(payments);
    } catch (e: any) {
      showMessage(`Ошибка загрузки карточки ученика: ${String(e?.message ?? e)}`, "error");
    }
  }

  async function openStudentCard(s: StudentDTO) {
    setSelectedStudentCard(s);
    setStudentCardOpen(true);
    setStudentCardLoading(true);
    setStudentCardEnrollments([]);
    setStudentCardBalance(null);
    setStudentCardDebts([]);
    setStudentCardPayments([]);
    try {
      await refreshStudentCardData(s.id);
    } finally {
      setStudentCardLoading(false);
    }
  }

  async function openStudentCardById(studentId: number) {
    const existing = allStudents.find((s) => s.id === studentId);
    try {
      const student = existing ?? (await getStudent(studentId));
      await openStudentCard(student);
    } catch (e: any) {
      showMessage(`Ошибка загрузки карточки ученика: ${String(e?.message ?? e)}`, "error");
    }
  }

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
      const text = buildDebtReminderMessage(
        locale,
        debtorLike,
        studentCardDebts,
        recipientName
      );
      await navigator.clipboard.writeText(text);
      showMessage(locale === "ru" ? "Русское напоминание скопировано" : "Латышское напоминание скопировано");
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  }

  function deleteStudentPayment(payment: PaymentDTO) {
    if (!selectedStudentCard) return;

    const amountLabel = formatEUR(payment.amount);
    const dateLabel = payment.paidAt.slice(0, 10);

    showConfirm(
      `Удалить оплату ${amountLabel} от ${dateLabel}? Остаток по связанному счёту будет восстановлен. Это действие нельзя отменить.`,
      async () => {
        try {
          setStudentCardDeletingPaymentId(payment.id);
          await deletePayment(payment.id);
          await Promise.all([refreshStudentCardData(selectedStudentCard.id), loadDebtors()]);
          showMessage("Оплата удалена");
        } catch (e: any) {
          showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
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
    if (tab === "courses") loadCourses();
  }, [tab, loadCourses]);

  useEffect(() => {
    void loadAllCourses();
  }, [loadAllCourses]);

  useEffect(() => {
    void loadAllTeachers();
  }, [loadAllTeachers]);

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
      showMessage("Учитель добавлен");
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    } finally {
      setCfTeacherCreating(false);
    }
  }

  async function saveCourse() {
    const lessonPrice = decimalOrZero(cfLessonPrice);
    const subscriptionPrice = decimalOrZero(cfSubscriptionPrice);
    const trimmedTeacherSearch = cfTeacherSearch.trim();

    if (!cfName.trim()) {
      showMessage("Введите название курса", "error");
      return;
    }
    if (lessonPrice < 0 || subscriptionPrice < 0) {
      showMessage("Цены должны быть не меньше 0", "error");
      return;
    }

    let teacherId = cfTeacherId;
    if (!teacherId && exactTeacherMatch) {
      teacherId = exactTeacherMatch.id;
    }
    if (trimmedTeacherSearch && !teacherId) {
      showMessage("Выберите существующего учителя или добавьте нового", "error");
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
      showMessage(editingCourse ? "Курс успешно обновлён" : "Курс успешно создан");
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  }

  async function removeCourse(id: number) {
    showConfirm("Удалить курс? Если по нему есть зачисления, удаление будет заблокировано.", async () => {
      try {
        await deleteCourse(id);
        await Promise.all([loadCourses(), loadAllCourses()]);
        showMessage("Курс удалён");
      } catch (e: any) {
        showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
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
  const [efStudentId, setEfStudentId] = useState<number>(0);
  const [efStudentSearch, setEfStudentSearch] = useState("");
  const [efStudentPickerOpen, setEfStudentPickerOpen] = useState(false);
  const efStudentComboRef = useRef<HTMLDivElement | null>(null);
  const [efCourseId, setEfCourseId] = useState<number>(0);
  const [efMode, setEfMode] = useState<"subscription" | "per_lesson">("per_lesson");
  const [efDiscount, setEfDiscount] = useState(0);
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
      showMessage("Нет активных учеников. Сначала добавьте или активируйте ученика.", "error");
      setTab("students");
      return;
    }
    if (allCourses.length === 0) {
      showMessage("Нет доступных курсов. Сначала добавьте курс.", "error");
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
    setEfNote(e.note);
    setEnrModalOpen(true);
  }

  async function saveEnrollment() {
    if (efStudentId <= 0 || efCourseId <= 0) {
      showMessage("Выберите ученика и курс", "error");
      return;
    }
    if (efDiscount < 0 || efDiscount > 100) {
      showMessage("Скидка должна быть от 0 до 100", "error");
      return;
    }

    try {
      let result: EnrollmentDTO;
      if (editingEnr) {
        result = await updateEnrollment(editingEnr.id, efMode, efDiscount, efNote);
        showMessage("Зачисление обновлено");
      } else {
        result = await createEnrollment(efStudentId, efCourseId, efMode, efDiscount, efNote);

        const matchesFilters =
          (enrStudentFilter === undefined || enrStudentFilter === result.studentId) &&
          (enrCourseFilter === undefined || enrCourseFilter === result.courseId);

        if (matchesFilters) {
          showMessage(`Зачисление создано: ${result.studentName} → ${result.courseName}`);
        } else {
          showMessage(
            `Зачисление создано: ${result.studentName} → ${result.courseName}. Очистите фильтры, чтобы увидеть его в списке.`
          );
        }
      }

      setEnrModalOpen(false);
      await loadEnrollments();
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  }

  // ---------------- Attendance ----------------
  const [rows, setRows] = useState<Row[]>([]);
  const [loadingAtt, setLoadingAtt] = useState(false);
  const [courseFilter, setCourseFilter] = useState<number | undefined>(undefined);
  const [attQ, setAttQ] = useState("");
  const [attFilter, setAttFilter] = useState<"all" | "missing" | "filled" | "zero">("all");
  const [attendanceSavingRows, setAttendanceSavingRows] = useState<Record<number, boolean>>({});

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
    if (tab === "attendance") loadAttendance();
  }, [tab, loadAttendance]);

  const perLessonTotal = useMemo(
    () => rows.reduce((s, r) => s + r.count * r.lessonPrice, 0),
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
      filtered = filtered.filter((r) => r.hasRecord && r.count === 0);
    }

    return filtered;
  }, [rows, attQ, attFilter, studentIndex]);

  const attendanceSummary = useMemo(() => {
    const filled = rows.filter((r) => r.hasRecord).length;
    const missing = rows.filter((r) => !r.hasRecord).length;
    const zero = rows.filter((r) => r.hasRecord && r.count === 0).length;
    return { filled, missing, zero, total: rows.length };
  }, [rows]);

  const onChangeCount = async (r: Row, v: number) => {
    if (!Number.isFinite(v)) return;
    const n = v < 0 ? 0 : Math.trunc(v);
    if (attendanceSavingRows[r.enrollmentId]) return;

    try {
      setAttendanceSavingRows((prev) => ({ ...prev, [r.enrollmentId]: true }));
      await saveCount(r.studentId, r.courseId, year, month, n);
      setRows((prev) =>
        prev.map((x) =>
          x.enrollmentId === r.enrollmentId ? { ...x, count: n, hasRecord: true } : x
        )
      );
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    } finally {
      setAttendanceSavingRows((prev) => {
        const next = { ...prev };
        delete next[r.enrollmentId];
        return next;
      });
    }
  };

  const onAddAll = async () => {
    try {
      await addOneMass(year, month, courseFilter);
      await loadAttendance();
      showMessage("Ко всем видимым строкам добавлено +1");
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onDeleteEnrollmentFromSheet = async (id: number) => {
    showConfirm(
      "Удалить зачисление? Вместе с ним будут удалены связанные записи посещаемости. Это действие нельзя отменить.",
      async () => {
        try {
          await deleteEnrollment(id);
          await loadAttendance();
          showMessage("Зачисление удалено");
        } catch (e: any) {
          showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
        }
      }
    );
  };

  // ---------------- Invoices ----------------
  const [invStatus, setInvStatus] = useState<string>("all");
  const [invItems, setInvItems] = useState<InvoiceListItem[]>([]);
  const [selectedInv, setSelectedInv] = useState<InvoiceDTO | null>(null);
  const [invoiceModalOpen, setInvoiceModalOpen] = useState(false);
  const [loadingInv, setLoadingInv] = useState(false);
  const [invQ, setInvQ] = useState("");
  const [invSummary, setInvSummary] = useState<InvoiceSummaryDTO | null>(null);

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

  const loadInvoices = useCallback(async () => {
    setLoadingInv(true);
    try {
      await ensureStudentsLoaded();
      const li = await listInvoices(year, month, invStatus);
      setInvItems(li);
      setSelectedInv(null);
      setInvoiceModalOpen(false);
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    } finally {
      setLoadingInv(false);
    }
  }, [year, month, invStatus, ensureStudentsLoaded, showMessage]);

  useEffect(() => {
    if (tab === "invoice") loadInvoices();
  }, [tab, loadInvoices]);

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
      return data;
    } catch (e: any) {
      showMessage(`Ошибка загрузки должников: ${String(e?.message ?? e)}`, "error");
      return [];
    } finally {
      setDebtorsLoading(false);
    }
  }, [showMessage]);

  useEffect(() => {
    if (tab === "debtors") loadDebtors();
  }, [tab, loadDebtors]);

  async function openDebtDetails(debtor: DebtorDTO) {
    setSelectedDebtor(debtor);
    setDebtDetailsOpen(true);
    setDebtDetails([]);
    setDebtDetailsLoading(true);

    try {
      const details = await studentDebtDetails(debtor.studentId);
      setDebtDetails(details);
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
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
      await navigator.clipboard.writeText(text);
      showMessage(locale === "ru" ? "Русское напоминание скопировано" : "Латышское напоминание скопировано");
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
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

  const onOpenInvoice = async (id: number) => {
    try {
      const iv = await getInvoice(id);
      setSelectedInv(iv);
      setInvoiceModalOpen(true);
      // Load payment summary
      if (iv.status !== "draft") {
        const summary = await invoiceSummary(id);
        setInvSummary(summary);
      } else {
        setInvSummary(null);
      }
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const closeInvoiceModal = () => {
    setInvoiceModalOpen(false);
    setSelectedInv(null);
    setInvSummary(null);
  };

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
      showMessage("Для оплаты не выбран ученик", "error");
      return;
    }
    if (isNaN(amount) || amount <= 0) {
      showMessage("Введите корректную сумму", "error");
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
      showMessage("Оплата записана");

      if (paymentInvoiceId) {
        await onOpenInvoice(paymentInvoiceId);
        await loadInvoices();
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

  const onGenerateDrafts = async () => {
    try {
      const res = await genDrafts(year, month);
      showMessage(
        `Черновики сформированы: создано ${res.created}, обновлено ${res.updated}, пропущено ${res.skippedHasInvoice} (уже выставлены), пропущено ${res.skippedNoLines} (нет строк)`
      );
      await loadInvoices();
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onIssueOne = async (id: number) => {
    try {
      const res = await issueOne(id);
      showMessage(`Счёт выставлен: #${res.number}`);
      await loadInvoices();
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onReopenToDraft = async (id: number) => {
    showConfirm(
      "Вернуть этот выставленный счёт в черновик? Это разрешено только если по нему нет оплат. Старый номер счёта будет очищен.",
      async () => {
        try {
          await reopenToDraft(id);
          if (selectedInv?.id === id) {
            await onOpenInvoice(id);
          }
          await loadInvoices();
          showMessage("Счёт возвращён в черновик");
        } catch (e: any) {
          showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
        }
      },
      "Вернуть"
    );
  };

  const onIssueAll = async () => {
    try {
      const res = await issueAll(year, month);
      showMessage(`Выставлено счетов: ${res.count}. PDF-файлов создано: ${res.pdfPaths.length}`);
      await loadInvoices();
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const onOpenPdf = async (id: number) => {
    try {
      const path = await ensurePdfAndOpen(id);
      console.log("Открыт PDF:", path);
      showMessage("PDF-файл открыт");
    } catch (e: any) {
      showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
    }
  };

  const openAppFolder = async (path: string | undefined, label: string) => {
    if (!path) {
      showMessage(`Папка «${label}» недоступна`, "error");
      return;
    }
    try {
      await OpenFile(path);
    } catch (e: any) {
      showMessage(`Не удалось открыть папку «${label}»: ${String(e?.message ?? e)}`, "error");
    }
  };

  const createManualBackup = async () => {
    try {
      setCreatingBackup(true);
      const backupPath = await BackupNow();
      showMessage(`Резервная копия создана: ${backupPath}`);
    } catch (e: any) {
      showMessage(`Не удалось создать резервную копию: ${String(e?.message ?? e)}`, "error");
    } finally {
      setCreatingBackup(false);
    }
  };

  // ---------------- Render ----------------
  const showMonthPicker = tab === "attendance" || tab === "invoice";
  const currentMeta = TAB_META[tab];
  const dashboardStats = [
    { label: "Ученики", value: studentList.length },
    { label: "Курсы", value: courseList.length },
    { label: "Открытые долги", value: debtors.length },
  ];

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
              aria-label="Закрыть уведомление"
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
            <h3 style={{ marginTop: 0, marginBottom: "16px" }}>Подтверждение</h3>
            <p style={{ marginBottom: "24px", lineHeight: "1.5" }}>{confirmDialog.message}</p>
            <div style={{ display: "flex", gap: "12px", justifyContent: "flex-end" }}>
              <button onClick={handleConfirmNo} style={{ padding: "8px 16px" }}>
                Отмена
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
                {confirmDialog.confirmButtonLabel ?? "Удалить"}
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="appShell">
        <section className="workspaceCard">
          <div className="workspaceTopbar">
            <div className="workspaceHeading">
              <div className="workspaceEyebrow">{currentMeta.eyebrow}</div>
              <h1>{currentMeta.title}</h1>
              <p>{currentMeta.description}</p>
            </div>

            <div className="workspaceStats" aria-label="Обзор приложения">
              {(tab === "attendance" || tab === "invoice") && (
                <div className="workspaceStat workspaceStatFocus">
                  <span>Фокус</span>
                  <strong>
                    {months[month - 1]} {year}
                  </strong>
                </div>
              )}
              {dashboardStats.map((stat) => (
                <div key={stat.label} className="workspaceStat">
                  <span>{stat.label}</span>
                  <strong>{stat.value}</strong>
                </div>
              ))}
            </div>

            <div className="workspaceActions" aria-label="File and backup actions">
              <button
                type="button"
                className="workspaceActionButton workspaceActionButtonPrimary"
                onClick={() => void createManualBackup()}
                disabled={creatingBackup}
              >
                {creatingBackup ? "Создание копии..." : "Создать резервную копию"}
              </button>
              <button
                type="button"
                className="workspaceActionButton"
                onClick={() => void openAppFolder(appDirs?.backups, "резервных копий")}
                disabled={!appDirs?.backups}
              >
                Открыть папку резервных копий
              </button>
              <button
                type="button"
                className="workspaceActionButton"
                onClick={() => void openAppFolder(appDirs?.invoices, "счетов")}
                disabled={!appDirs?.invoices}
              >
                Открыть папку счетов
              </button>
            </div>
          </div>

          <nav className="tabs">
            <button
              className={tab === "students" ? "active" : ""}
              onClick={() => setTab("students")}
            >
              Ученики
            </button>
            <button className={tab === "courses" ? "active" : ""} onClick={() => setTab("courses")}>
              Курсы
            </button>
            <button
              className={tab === "enrollments" ? "active" : ""}
              onClick={() => setTab("enrollments")}
            >
              Зачисления
            </button>
            <button
              className={tab === "attendance" ? "active" : ""}
              onClick={() => setTab("attendance")}
            >
              Посещаемость
            </button>
            <button className={tab === "invoice" ? "active" : ""} onClick={() => setTab("invoice")}>
              Счета
            </button>
            <button className={tab === "debtors" ? "active" : ""} onClick={() => setTab("debtors")}>
              Должники
            </button>

            <div className="spacer" />

            {showMonthPicker && (
              <div className="monthpickers">
                <select value={month} onChange={(e) => setMonth(parseInt(e.target.value))}>
                  {months.map((m, i) => (
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
          </nav>

          {/* ---------------- Students ---------------- */}
          {tab === "students" && (
            <>
              <div className="controls">
                <button onClick={openAddStudent}>Добавить ученика</button>
                <input
                  placeholder="Поиск по имени / телефону / эл. почте…"
                  value={studentQ}
                  onChange={(e) => setStudentQ(e.target.value)}
                  style={{ width: 260 }}
                />
                <label className="inline">
                  <input
                    type="checkbox"
                    checked={includeInactive}
                    onChange={(e) => setIncludeInactive(e.target.checked)}
                  />
                  Показывать неактивных
                </label>
                <button onClick={loadStudents}>Обновить</button>
              </div>

              {studentLoading ? (
                <div>Загрузка…</div>
              ) : studentList.length === 0 ? (
                <div className="empty">Учеников пока нет.</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>Имя</th>
                      <th>Телефон</th>
                      <th>Эл. почта</th>
                      <th>Активен</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {studentList.map((s) => (
                      <tr key={s.id}>
                        <td>
                          <button
                            className="linkButton"
                            onClick={() => openStudentCard(s)}
                            title="Открыть карточку ученика"
                          >
                            {s.fullName}
                          </button>
                        </td>
                        <td>{s.phone}</td>
                        <td>{s.email}</td>
                        <td>{s.isActive ? "да" : "нет"}</td>
                        <td>
                          <button onClick={() => openStudentCard(s)}>Карточка</button>
                          <button onClick={() => openEditStudent(s)}>Редактировать</button>
                          <button onClick={() => toggleStudentActive(s)}>
                            {s.isActive ? "Деактивировать" : "Активировать"}
                          </button>
                          {!s.isActive && (
                            <button onClick={() => removeStudent(s.id)}>Удалить</button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {studentModalOpen && (
                <div className="modal">
                  <div className="modalBody">
                    <h3>{editingStudent ? "Редактировать ученика" : "Добавить ученика"}</h3>

                    <div className="formRow">
                      <label>Полное имя</label>
                      <input value={sfName} onChange={(e) => setSfName(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>Персональный код</label>
                      <input
                        value={sfPersonalCode}
                        onChange={(e) => setSfPersonalCode(e.target.value)}
                      />
                    </div>
                    <div className="formRow">
                      <label>{sfIsMinor ? "Телефон родителя" : "Телефон"}</label>
                      <input value={sfPhone} onChange={(e) => setSfPhone(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>{sfIsMinor ? "Эл. почта родителя" : "Эл. почта"}</label>
                      <input value={sfEmail} onChange={(e) => setSfEmail(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>Заметка</label>
                      <input value={sfNote} onChange={(e) => setSfNote(e.target.value)} />
                    </div>
                    <div className="formRow">
                      <label>Несовершеннолетний ученик</label>
                      <label className="inline">
                        <input
                          type="checkbox"
                          checked={sfIsMinor}
                          onChange={(e) => setSfIsMinor(e.target.checked)}
                        />
                        Да
                      </label>
                    </div>
                    {sfIsMinor && (
                      <>
                        <div className="formRow">
                          <label>Имя плательщика</label>
                          <input
                            value={sfPayerName}
                            onChange={(e) => setSfPayerName(e.target.value)}
                          />
                        </div>
                        <div className="formRow">
                          <label>Кем приходится плательщик</label>
                          <select
                            value={sfPayerRole}
                            onChange={(e) => setSfPayerRole(e.target.value)}
                          >
                            <option value="">Выберите роль…</option>
                            {payerRoleOptions.map((role) => (
                              <option key={role} value={role}>
                                {payerRoleLabel(role)}
                              </option>
                            ))}
                          </select>
                        </div>
                      </>
                    )}

                    <div className="modalActions">
                      <button onClick={saveStudent}>Сохранить</button>
                      <button onClick={() => setStudentModalOpen(false)}>Отмена</button>
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
                <button onClick={openAddCourse}>Добавить курс</button>
                <input
                  placeholder="Поиск по курсу или учителю…"
                  value={courseQ}
                  onChange={(e) => setCourseQ(e.target.value)}
                  style={{ width: 260 }}
                />
                <button onClick={loadCourses}>Обновить</button>
              </div>

              {courseLoading ? (
                <div>Загрузка…</div>
              ) : courseList.length === 0 ? (
                <div className="empty">Курсов пока нет.</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>Название</th>
                      <th>Учитель</th>
                      <th>Тип</th>
                      <th style={{ textAlign: "right" }}>За занятие (EUR)</th>
                      <th style={{ textAlign: "right" }}>Абонемент (EUR)</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {courseList.map((c) => (
                      <tr key={c.id}>
                        <td>{c.name}</td>
                        <td>{c.teacherName || "—"}</td>
                        <td>{courseTypeLabel(c.type)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(c.lessonPrice)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(c.subscriptionPrice)}</td>
                        <td>
                          <button onClick={() => openEditCourse(c)}>Редактировать</button>
                          <button onClick={() => removeCourse(c.id)}>Удалить</button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {courseModalOpen && (
                <div className="modal">
                  <div className="modalBody">
                    <h3>{editingCourse ? "Редактировать курс" : "Добавить курс"}</h3>

                    <div className="formRow">
                      <label>Название</label>
                      <input value={cfName} onChange={(e) => setCfName(e.target.value)} />
                    </div>

                    <div className="formRow">
                      <label>Учитель</label>
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
                          placeholder="Выберите или добавьте учителя…"
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
                                    ? "Учитель добавляется..."
                                    : `Добавить учителя «${cfTeacherSearch.trim()}»`}
                                </span>
                                <span className="comboBoxMeta">
                                  Сохранить нового учителя и выбрать его для этого курса.
                                </span>
                              </button>
                            )}
                            {filteredTeachers.length === 0 && !cfTeacherSearch.trim() && (
                              <div className="comboBoxEmpty">Учителей пока нет.</div>
                            )}
                          </div>
                        )}
                      </div>
                    </div>

                    <div className="formRow">
                      <label>Тип</label>
                      <select value={cfType} onChange={(e) => setCfType(e.target.value as any)}>
                        <option value="group">группа</option>
                        <option value="individual">индивидуально</option>
                      </select>
                    </div>

                    <div className="formRow">
                      <label>Цена за занятие (EUR)</label>
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
                      <label>Цена абонемента (EUR)</label>
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
                      <button onClick={saveCourse}>Сохранить</button>
                      <button onClick={() => setCourseModalOpen(false)}>Отмена</button>
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
                <button onClick={openAddEnrollment}>Добавить зачисление</button>

                <select
                  value={enrStudentFilter ?? ""}
                  onChange={(e) => setEnrStudentFilter(intOrUndef(e.target.value))}
                >
                  <option value="">Все ученики</option>
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
                  <option value="">Все курсы</option>
                  {allCourses.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>

                <button onClick={loadEnrollments}>Обновить</button>
              </div>

              {enrLoading ? (
                <div>Загрузка…</div>
              ) : enrollments.length === 0 ? (
                <div className="empty">Зачислений пока нет.</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>Ученик</th>
                      <th>Курс</th>
                      <th>Учитель</th>
                      <th>Оплата</th>
                      <th style={{ textAlign: "right" }}>Скидка</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {enrollments.map((e) => (
                      <tr key={e.id}>
                        <td>
                          <button className="linkButton" onClick={() => void openStudentCardById(e.studentId)}>
                            {e.studentName}
                          </button>
                        </td>
                        <td>{e.courseName}</td>
                        <td>{e.teacherName || "—"}</td>
                        <td>{billingModeLabel(e.billingMode)}</td>
                        <td style={{ textAlign: "right" }}>{e.discountPct.toFixed(1)}%</td>
                        <td>
                          <button onClick={() => openEditEnrollment(e)}>Редактировать</button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {enrModalOpen && (
                <div className="modal">
                  <div className="modalBody">
                    <h3>{editingEnr ? "Редактировать зачисление" : "Добавить зачисление"}</h3>

                    <div className="formRow">
                      <label>Ученик</label>
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
                            placeholder="Поиск ученика по имени, телефону или эл. почте…"
                          />
                          {efStudentPickerOpen && (
                            <div className="comboBoxMenu">
                              {filteredEnrollmentStudents.length === 0 ? (
                                <div className="comboBoxEmpty">Ученики не найдены.</div>
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
                      <label>Курс</label>
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
                      <label>Оплата</label>
                      <select value={efMode} onChange={(e) => setEfMode(e.target.value as any)}>
                        <option value="per_lesson">по занятиям</option>
                        <option value="subscription">абонемент</option>
                      </select>
                    </div>

                    <div className="formRow">
                      <label>Скидка %</label>
                      <input
                        type="number"
                        min={0}
                        max={100}
                        step="0.1"
                        value={efDiscount}
                        onChange={(e) => setEfDiscount(numOrZero(e.target.value))}
                      />
                    </div>

                    <div className="formRow">
                      <label>Заметка</label>
                      <input value={efNote} onChange={(e) => setEfNote(e.target.value)} />
                    </div>

                    <div className="modalActions">
                      <button onClick={saveEnrollment}>Сохранить</button>
                      <button onClick={() => setEnrModalOpen(false)}>Отмена</button>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}

          {/* ---------------- Attendance ---------------- */}
          {tab === "attendance" && (
            <>
              <div className="controls">
                <button onClick={onAddAll}>+1 всем</button>

                <select
                  value={courseFilter ?? ""}
                  onChange={(e) => setCourseFilter(intOrUndef(e.target.value))}
                >
                  <option value="">Все группы</option>
                  {allCourses.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>

                <input
                  placeholder="Поиск по ученику / телефону / группе…"
                  value={attQ}
                  onChange={(e) => setAttQ(e.target.value)}
                  style={{ width: 260 }}
                />

                <select
                  value={attFilter}
                  onChange={(e) => setAttFilter(e.target.value as typeof attFilter)}
                >
                  <option value="all">Показать: всё</option>
                  <option value="missing">Только не заполненные</option>
                  <option value="filled">Только заполненные</option>
                  <option value="zero">Ноль занятий</option>
                </select>

                <button onClick={loadAttendance}>Обновить</button>
              </div>

              {rows.length > 0 && (
                <div className="attSummary">
                  Заполнено: {attendanceSummary.filled} / {attendanceSummary.total}
                  &nbsp;·&nbsp;Не заполнено: {attendanceSummary.missing}
                  &nbsp;·&nbsp;Ноль занятий: {attendanceSummary.zero}
                </div>
              )}

              {loadingAtt ? (
                <div>Загрузка…</div>
              ) : filteredAttendanceRows.length === 0 ? (
                <div className="empty">
                  {attQ.trim() || attFilter !== "all"
                    ? "По вашему запросу ничего не найдено."
                    : "Нет строк с оплатой по занятиям. Сначала создайте зачисления."}
                </div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>Ученик</th>
                      <th>Курс</th>
                      <th style={{ textAlign: "right" }}>Цена занятия (EUR)</th>
                      <th style={{ textAlign: "right" }}>Кол-во</th>
                      <th style={{ textAlign: "right" }}>Итого (EUR)</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredAttendanceRows.map((r) => (
                      <tr key={r.enrollmentId}>
                        <td>
                          <button className="linkButton" onClick={() => void openStudentCardById(r.studentId)}>
                            {r.studentName}
                          </button>
                        </td>
                        <td>
                          {r.courseName} ({courseTypeLabel(r.courseType)})
                        </td>
                        <td style={{ textAlign: "right" }}>{formatEUR(r.lessonPrice)}</td>
                        <td style={{ textAlign: "right" }}>
                          {!r.hasRecord && (
                            <span className="attBadge attBadge--missing">Не заполнено</span>
                          )}
                          {r.hasRecord && r.count === 0 && (
                            <span className="attBadge attBadge--zero">0 занятий</span>
                          )}
                          <div className="attendanceStepper">
                            <button
                              type="button"
                              className="attendanceStepperButton"
                              onClick={() => onChangeCount(r, r.count - 1)}
                              disabled={attendanceSavingRows[r.enrollmentId] || r.count <= 0}
                              aria-label={`Уменьшить количество занятий для ${r.studentName}`}
                            >
                              −
                            </button>
                            <input
                              type="number"
                              min={0}
                              value={r.count}
                              disabled={attendanceSavingRows[r.enrollmentId]}
                              onChange={(e) => onChangeCount(r, Number(e.target.value))}
                              className="attendanceStepperInput"
                              aria-label={`Количество занятий для ${r.studentName}`}
                            />
                            <button
                              type="button"
                              className="attendanceStepperButton"
                              onClick={() => onChangeCount(r, r.count + 1)}
                              disabled={attendanceSavingRows[r.enrollmentId]}
                              aria-label={`Увеличить количество занятий для ${r.studentName}`}
                            >
                              +
                            </button>
                          </div>
                        </td>
                        <td style={{ textAlign: "right" }}>{formatEUR(r.count * r.lessonPrice)}</td>
                        <td>
                          {!r.hasRecord && (
                            <button
                              onClick={() => onChangeCount(r, 0)}
                              disabled={attendanceSavingRows[r.enrollmentId]}
                              style={{ marginRight: "0.5rem" }}
                            >
                              Отметить 0
                            </button>
                          )}
                          {r.canDelete ? (
                            <button onClick={() => onDeleteEnrollmentFromSheet(r.enrollmentId)}>
                              Удалить зачисление
                            </button>
                          ) : (
                            <span className="mutedInline">Использовано в истории счёта</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <td colSpan={4} style={{ textAlign: "right" }}>
                        Итого по занятиям (EUR):
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
              <div className="controls">
                <button onClick={onGenerateDrafts}>Сформировать черновики</button>
                <button onClick={onIssueAll}>Выставить все</button>

                <select value={invStatus} onChange={(e) => setInvStatus(e.target.value)}>
                  <option value="draft">черновик</option>
                  <option value="issued">выставлен</option>
                  <option value="paid">оплачен</option>
                  <option value="all">все</option>
                </select>

                <input
                  placeholder="Поиск по ученику / телефону / эл. почте / номеру счёта"
                  value={invQ}
                  onChange={(e) => setInvQ(e.target.value)}
                  style={{ width: "320px" }}
                />

                <button onClick={loadInvoices}>Обновить</button>
              </div>

              {loadingInv ? (
                <div>Загрузка…</div>
              ) : filteredInvItems.length === 0 ? (
                <div className="empty">Счета за выбранный период, статус или поиск не найдены.</div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>Ученик</th>
                      <th>Период</th>
                      <th style={{ textAlign: "right" }}>Сумма (EUR)</th>
                      <th>Статус</th>
                      <th>Номер</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredInvItems.map((it) => (
                      <tr key={it.id}>
                        <td>
                          <button className="linkButton" onClick={() => void openStudentCardById(it.studentId)}>
                            {it.studentName}
                          </button>
                        </td>
                        <td>
                          {months[it.month - 1]} {it.year}
                        </td>
                        <td style={{ textAlign: "right" }}>{formatEUR(it.total)}</td>
                        <td>{invoiceStatusLabel(it.status)}</td>
                        <td>{it.number ?? ""}</td>
                        <td>
                          <button onClick={() => onOpenInvoice(it.id)}>Открыть</button>
                          {it.status === "draft" && (
                            <button onClick={() => onIssueOne(it.id)}>Выставить</button>
                          )}
                          {it.status === "issued" && (
                            <button onClick={() => void onReopenToDraft(it.id)}>
                              Вернуть в черновик
                            </button>
                          )}
                          {it.status !== "draft" && (
                            <button onClick={() => onOpenPdf(it.id)}>PDF</button>
                          )}
                          {it.status !== "draft" && (
                            <button
                              onClick={async () => {
                                try {
                                  const iv = await getInvoice(it.id);
                                  setSelectedInv(iv);
                                  const summary = await invoiceSummary(it.id);
                                  setInvSummary(summary);
                                  openPaymentModal(iv, summary);
                                } catch (e: any) {
                                  showMessage(`Ошибка: ${String(e?.message ?? e)}`, "error");
                                }
                              }}
                            >
                              Записать оплату
                            </button>
                          )}
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
              <div className="controls">
                <button onClick={loadDebtors}>Обновить</button>
              </div>

              {debtorsLoading ? (
                <div>Загрузка…</div>
              ) : debtors.length === 0 ? (
                <div className="empty">
                  Должников не найдено. Все ученики оплатили вовремя.
                </div>
              ) : (
                <table>
                  <thead>
                    <tr>
                      <th>Имя ученика</th>
                      <th style={{ textAlign: "right" }}>Долг (EUR)</th>
                      <th style={{ textAlign: "right" }}>Выставлено (EUR)</th>
                      <th style={{ textAlign: "right" }}>Оплачено (EUR)</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {debtors.map((d) => (
                      <tr key={d.studentId}>
                        <td>
                          <button className="linkButton" onClick={() => void openStudentCardById(d.studentId)}>
                            {d.studentName}
                          </button>
                        </td>
                        <td style={{ textAlign: "right", fontWeight: "bold", color: "#d32f2f" }}>
                          {formatEUR(d.debt)}
                        </td>
                        <td style={{ textAlign: "right" }}>{formatEUR(d.totalInvoiced)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(d.totalPaid)}</td>
                        <td>
                          <button onClick={() => openDebtorPaymentModal(d)}>Записать оплату</button>
                          <button onClick={() => openDebtDetails(d)}>Расшифровка долга</button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot>
                    <tr>
                      <td style={{ fontWeight: "bold" }}>Общий долг (EUR):</td>
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
        </section>
      </div>

      {/* Global Payment Modal */}
      {invoiceModalOpen && selectedInv && (
        <div className="modal" onClick={closeInvoiceModal}>
          <div className="modalBody modalBodyWide" onClick={(e) => e.stopPropagation()}>
            <h3>
              Счёт {selectedInv.number ? `#${selectedInv.number}` : ""} —{" "}
              <button
                className="linkButton"
                onClick={() => void openStudentCardById(selectedInv.studentId)}
              >
                {selectedInv.studentName}
              </button>{" "}
              — {months[selectedInv.month - 1]} {selectedInv.year}
            </h3>

            {invSummary && selectedInv.status !== "draft" && (
              <div className="invSummary">
                <div className="invSummaryRow">
                  <span>Получатель:</span>
                  <span>{selectedInv.recipientName || selectedInv.studentName}</span>
                </div>
                {selectedInv.studentPersonalCode && (
                  <div className="invSummaryRow">
                    <span>{selectedInv.isMinor ? "Персональный код ребёнка:" : "Персональный код:"}</span>
                    <span>{selectedInv.studentPersonalCode}</span>
                  </div>
                )}
                {selectedInv.isMinor && (
                  <div className="invSummaryRow">
                    <span>За ребёнка:</span>
                    <span>{selectedInv.childName}</span>
                  </div>
                )}
                <div className="invSummaryRow">
                  <span>Сумма:</span>
                  <span className="money">{formatEUR(invSummary.total)}</span>
                </div>
                <div className="invSummaryRow">
                  <span>Оплачено:</span>
                  <span className="money good">{formatEUR(invSummary.paid)}</span>
                </div>
                <div className="invSummaryRow">
                  <span>Осталось:</span>
                  <span className={`money ${invSummary.remaining > 0 ? "bad" : "good"}`}>
                    {formatEUR(invSummary.remaining)}
                  </span>
                </div>
                <div className="invSummaryRow">
                  <span>Статус:</span>
                  <span className="money">{invoiceStatusLabel(invSummary.status)}</span>
                </div>
              </div>
            )}

            <div style={{ overflowX: "auto" }}>
              <table>
                <thead>
                  <tr>
                    <th>Описание</th>
                    <th style={{ textAlign: "right" }}>Кол-во</th>
                    <th style={{ textAlign: "right" }}>Цена (EUR)</th>
                    <th style={{ textAlign: "right" }}>Сумма (EUR)</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedInv.lines.map((l, idx) => (
                    <tr key={idx}>
                      <td>{l.description}</td>
                      <td style={{ textAlign: "right" }}>{l.qty}</td>
                      <td style={{ textAlign: "right" }}>{formatEUR(l.unitPrice)}</td>
                      <td style={{ textAlign: "right" }}>{formatEUR(l.amount)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={3} style={{ textAlign: "right" }}>
                      Итого (EUR):
                    </td>
                    <td style={{ textAlign: "right" }}>{formatEUR(selectedInv.total)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>

            <div className="modalActions">
              {selectedInv.status === "draft" && (
                <button
                  onClick={async () => {
                    await onIssueOne(selectedInv.id);
                  }}
                >
                  Выставить
                </button>
              )}
              {selectedInv.status === "issued" && (
                <button
                  onClick={async () => {
                    await onReopenToDraft(selectedInv.id);
                  }}
                >
                  Вернуть в черновик
                </button>
              )}
              {selectedInv.status !== "draft" && (
                <>
                  <button onClick={() => onOpenPdf(selectedInv.id)}>PDF</button>
                  <button onClick={() => openPaymentModal()}>Записать оплату</button>
                </>
              )}
              <button onClick={closeInvoiceModal}>Закрыть</button>
            </div>
          </div>
        </div>
      )}

      {paymentModalOpen && paymentStudentId > 0 && (
        <div className="modal" onClick={() => setPaymentModalOpen(false)}>
          <div className="modalBody" onClick={(e) => e.stopPropagation()}>
            <h3>Записать оплату</h3>
            <div className="formRow">
              <label>Ученик</label>
              <input value={paymentStudentName} disabled />
            </div>
            {paymentInvoiceId && (
              <div className="formRow">
                <label>Применить к</label>
                <input value={`Счёт #${paymentInvoiceId}`} disabled />
              </div>
            )}
            <div className="formRow">
              <label>Сумма (EUR):</label>
              <input
                type="number"
                step="0.01"
                value={paymentAmount}
                onChange={(e) => setPaymentAmount(e.target.value)}
                autoFocus
              />
            </div>
            <div className="formRow">
              <label>Способ:</label>
              <select
                value={paymentMethod}
                onChange={(e) => setPaymentMethod(e.target.value as "cash" | "bank")}
              >
                <option value="cash">Наличные</option>
                <option value="bank">Банк</option>
              </select>
            </div>
            <div className="formRow">
              <label>Заметка (необязательно):</label>
              <input
                type="text"
                value={paymentNote}
                onChange={(e) => setPaymentNote(e.target.value)}
                placeholder="Заметка к оплате..."
              />
            </div>
            <div className="modalActions">
              <button onClick={closePaymentModal}>Отмена</button>
              <button onClick={handleCreatePayment}>Записать оплату</button>
            </div>
          </div>
        </div>
      )}

      {debtDetailsOpen && selectedDebtor && (
        <div className="modal" onClick={() => setDebtDetailsOpen(false)}>
          <div className="modalBody" onClick={(e) => e.stopPropagation()}>
            <h3>Расшифровка долга</h3>

            <div className="invSummary">
              <div className="invSummaryRow">
                <span>Ученик</span>
                <button
                  className="linkButton"
                  onClick={() => void openStudentCardById(selectedDebtor.studentId)}
                >
                  {selectedDebtor.studentName}
                </button>
              </div>
              <div className="invSummaryRow">
                <span>Общий долг</span>
                <strong className="money bad">{formatEUR(selectedDebtor.debt)}</strong>
              </div>
            </div>

            {debtDetailsLoading ? (
              <div>Загрузка...</div>
            ) : debtDetails.length === 0 ? (
              <div className="empty">Нет открытых счетов с остатком.</div>
            ) : (
              <div style={{ overflowX: "auto" }}>
                <table>
                  <thead>
                    <tr>
                      <th>Месяц</th>
                      <th>Счёт</th>
                      <th style={{ textAlign: "right" }}>Сумма</th>
                      <th style={{ textAlign: "right" }}>Оплачено</th>
                      <th style={{ textAlign: "right" }}>Осталось</th>
                      <th>Статус</th>
                    </tr>
                  </thead>
                  <tbody>
                    {debtDetails.map((x) => (
                      <tr key={x.invoiceId}>
                        <td>
                          {months[x.month - 1]} {x.year}
                        </td>
                        <td>{x.number ?? "Без номера"}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(x.total)}</td>
                        <td style={{ textAlign: "right" }}>{formatEUR(x.paid)}</td>
                        <td style={{ textAlign: "right" }}>
                          <strong className="money bad">{formatEUR(x.remaining)}</strong>
                        </td>
                        <td>{invoiceStatusLabel(x.status)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <div className="modalActions">
              {!debtDetailsLoading && debtDetails.length > 0 && (
                <>
                  <button onClick={openPaymentFromDebtDetails}>Записать оплату</button>
                  <button onClick={() => void copyDebtMessage("ru")}>Скопировать по-русски</button>
                  <button onClick={() => void copyDebtMessage("lv")}>Скопировать по-латышски</button>
                </>
              )}
              <button onClick={() => setDebtDetailsOpen(false)}>Закрыть</button>
            </div>
          </div>
        </div>
      )}

      {/* Student Card Modal */}
      {studentCardOpen && selectedStudentCard && (
        <div className="modal" onClick={() => setStudentCardOpen(false)}>
          <div className="modalBody modalBodyWide" onClick={(e) => e.stopPropagation()}>
            <h3>Карточка ученика</h3>

            {studentCardLoading ? (
              <div style={{ padding: "24px 0", textAlign: "center" }}>Загрузка…</div>
            ) : (
              <>
                {/* Basic info */}
                <div className="cardSection">
                  <div className="cardSectionTitle">Основная информация</div>
                  <div className="invSummary">
                    <div className="invSummaryRow">
                      <span>Имя</span>
                      <span style={{ fontWeight: 700 }}>{selectedStudentCard.fullName}</span>
                    </div>
                    {selectedStudentCard.phone && (
                      <div className="invSummaryRow">
                        <span>{selectedStudentCard.isMinor ? "Телефон родителя" : "Телефон"}</span>
                        <span>{selectedStudentCard.phone}</span>
                      </div>
                    )}
                    {selectedStudentCard.personalCode && (
                      <div className="invSummaryRow">
                        <span>Персональный код</span>
                        <span>{selectedStudentCard.personalCode}</span>
                      </div>
                    )}
                    {selectedStudentCard.email && (
                      <div className="invSummaryRow">
                        <span>{selectedStudentCard.isMinor ? "Эл. почта родителя" : "Эл. почта"}</span>
                        <span>{selectedStudentCard.email}</span>
                      </div>
                    )}
                    {selectedStudentCard.note && (
                      <div className="invSummaryRow">
                        <span>Заметка</span>
                        <span>{selectedStudentCard.note}</span>
                      </div>
                    )}
                    <div className="invSummaryRow">
                      <span>Статус</span>
                      <span className={`money ${selectedStudentCard.isActive ? "good" : ""}`}>
                        {selectedStudentCard.isActive ? "Активен" : "Неактивен"}
                      </span>
                    </div>
                    <div className="invSummaryRow">
                      <span>Тип ученика</span>
                      <span>{selectedStudentCard.isMinor ? "Ребёнок / несовершеннолетний" : "Взрослый"}</span>
                    </div>
                  </div>
                  <div style={{ display: "flex", gap: 10, marginTop: 10 }}>
                    <button
                      onClick={() => {
                        setStudentCardOpen(false);
                        openEditStudent(selectedStudentCard);
                      }}
                    >
                      Редактировать ученика
                    </button>
                    <button onClick={openStudentCardPaymentModal}>Записать оплату</button>
                  </div>
                </div>

                {selectedStudentCard.isMinor && (
                  <div className="cardSection">
                    <div className="cardSectionTitle">Плательщик</div>
                    <div className="invSummary">
                      <div className="invSummaryRow">
                        <span>Имя плательщика</span>
                        <span>{selectedStudentCard.payerName || "—"}</span>
                      </div>
                      <div className="invSummaryRow">
                        <span>Кем приходится</span>
                        <span>
                          {selectedStudentCard.payerRole
                            ? payerRoleLabel(selectedStudentCard.payerRole)
                            : "—"}
                        </span>
                      </div>
                      <div className="invSummaryRow">
                        <span>Телефон</span>
                        <span>{selectedStudentCard.phone || "—"}</span>
                      </div>
                      <div className="invSummaryRow">
                        <span>Эл. почта</span>
                        <span>{selectedStudentCard.email || "—"}</span>
                      </div>
                    </div>
                  </div>
                )}

                {/* Enrollments */}
                <div className="cardSection">
                  <div className="cardSectionTitle">Курсы</div>
                  {studentCardEnrollments.length === 0 ? (
                    <div className="empty">Зачислений нет.</div>
                  ) : (
                    <table>
                      <thead>
                        <tr>
                          <th>Курс</th>
                          <th>Учитель</th>
                          <th>Оплата</th>
                          <th style={{ textAlign: "right" }}>Скидка</th>
                          <th>Заметка</th>
                        </tr>
                      </thead>
                      <tbody>
                        {studentCardEnrollments.map((e) => (
                          <tr key={e.id}>
                            <td>{e.courseName}</td>
                            <td>{e.teacherName || "—"}</td>
                            <td>{billingModeLabel(e.billingMode)}</td>
                            <td style={{ textAlign: "right" }}>{e.discountPct.toFixed(1)}%</td>
                            <td>{e.note}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>

                {/* Balance */}
                <div className="cardSection">
                  <div className="cardSectionTitle">Баланс</div>
                  {studentCardBalance ? (
                    <div className="invSummary">
                      <div className="invSummaryRow">
                        <span>Выставлено</span>
                        <span className="money">{formatEUR(studentCardBalance.totalInvoiced)}</span>
                      </div>
                      <div className="invSummaryRow">
                        <span>Оплачено</span>
                        <span className="money good">{formatEUR(studentCardBalance.totalPaid)}</span>
                      </div>
                      <div className="invSummaryRow">
                        <span>Текущий долг</span>
                        <span className={`money ${studentCardBalance.debt > 0 ? "bad" : "good"}`}>
                          {studentCardBalance.debt > 0
                            ? formatEUR(studentCardBalance.debt)
                            : "Долга нет"}
                        </span>
                      </div>
                    </div>
                  ) : (
                    <div className="empty">Баланс недоступен.</div>
                  )}
                </div>

                {/* Open debts */}
                <div className="cardSection">
                  <div className="cardSectionTitle">Открытые долги</div>
                  {studentCardDebts.length === 0 ? (
                    <div className="empty">Всё оплачено — открытых долгов нет.</div>
                  ) : (
                    <div style={{ overflowX: "auto" }}>
                      <table>
                        <thead>
                          <tr>
                            <th>Месяц</th>
                            <th>Счёт</th>
                            <th style={{ textAlign: "right" }}>Сумма</th>
                            <th style={{ textAlign: "right" }}>Оплачено</th>
                            <th style={{ textAlign: "right" }}>Осталось</th>
                            <th>Статус</th>
                          </tr>
                        </thead>
                        <tbody>
                          {studentCardDebts.map((x) => (
                            <tr key={x.invoiceId}>
                              <td>
                                {months[x.month - 1]} {x.year}
                              </td>
                              <td>{x.number ?? "Без номера"}</td>
                              <td style={{ textAlign: "right" }}>{formatEUR(x.total)}</td>
                              <td style={{ textAlign: "right" }}>{formatEUR(x.paid)}</td>
                              <td style={{ textAlign: "right" }}>
                                <strong className="money bad">{formatEUR(x.remaining)}</strong>
                              </td>
                              <td>{invoiceStatusLabel(x.status)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>

                {/* Recent payments */}
                <div className="cardSection">
                  <div className="cardSectionTitle">Последние оплаты</div>
                  {studentCardPayments.length === 0 ? (
                    <div className="empty">Оплат пока нет.</div>
                  ) : (
                    <>
                      <table>
                        <thead>
                          <tr>
                            <th>Дата</th>
                            <th style={{ textAlign: "right" }}>Сумма</th>
                            <th>Способ</th>
                            <th>Заметка</th>
                            <th>Действия</th>
                          </tr>
                        </thead>
                        <tbody>
                          {studentCardPayments.slice(0, 10).map((p) => (
                            <tr key={p.id}>
                              <td>{p.paidAt.slice(0, 10)}</td>
                              <td style={{ textAlign: "right" }}>
                                <span className="money good">{formatEUR(p.amount)}</span>
                              </td>
                              <td>{paymentMethodLabel(p.method)}</td>
                              <td>{p.note}</td>
                              <td>
                                <button
                                  onClick={() => deleteStudentPayment(p)}
                                  disabled={studentCardDeletingPaymentId === p.id}
                                >
                                  {studentCardDeletingPaymentId === p.id ? "Удаление..." : "Удалить"}
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                      {studentCardPayments.length > 10 && (
                        <p className="mutedInline" style={{ marginTop: 8, textAlign: "right" }}>
                          Показаны 10 последних оплат из {studentCardPayments.length}.
                        </p>
                      )}
                    </>
                  )}
                </div>
              </>
            )}

            <div className="modalActions">
              {!studentCardLoading && studentCardDebts.length > 0 && (
                <>
                  <button onClick={openStudentCardPaymentModal}>Записать оплату</button>
                  <button onClick={() => void copyStudentCardDebtMessage("ru")}>Скопировать по-русски</button>
                  <button onClick={() => void copyStudentCardDebtMessage("lv")}>Скопировать по-латышски</button>
                </>
              )}
              <button onClick={() => setStudentCardOpen(false)}>Закрыть</button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
