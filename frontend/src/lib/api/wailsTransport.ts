import {
  AppDirs,
  AppReady,
  AttendanceAddOne,
  AttendanceListPerLesson,
  AttendanceUpsert,
  BackupNow,
  CourseCreate,
  CourseDelete,
  CourseGet,
  CourseList,
  CourseUpdate,
  DebtorsList,
  EnrollmentCreate,
  EnrollmentDelete,
  EnrollmentList,
  EnrollmentUpdate,
  InvoiceDeleteDraft,
  InvoiceEnsurePDF,
  InvoiceGenerateDrafts,
  InvoiceGet,
  InvoiceHasPDF,
  InvoiceIssue,
  InvoiceList,
  InvoicePaymentSummary,
  InvoiceRebuildStudentDraft,
  InvoiceReopenDraft,
  MonthOverview,
  OpenFile,
  PaymentCreate,
  PaymentDelete,
  PaymentListForStudent,
  PaymentQuickCash,
  RecentPayments,
  SettingsGetLocale,
  SettingsSetLocale,
  StudentBalance,
  StudentCreate,
  StudentDebtDetails,
  StudentDelete,
  StudentGet,
  StudentList,
  StudentSetActive,
  StudentUpdate,
  TeacherCreate,
  TeacherList,
} from "../../../wailsjs/go/main/App";
import type {
  AppTransport,
  BackupResult,
  BootstrapResult,
  CourseDTO,
  EnsurePdfResult,
  GenerateResult,
  InvoiceDTO,
  InvoiceListItem,
  IssueResult,
  PaymentDTO,
  Row,
  StudentDTO,
  TeacherDTO,
  SessionInfo,
} from "./types";

function pathToFilename(path: string): string {
  const parts = path.split(/[\\/]/);
  return parts[parts.length - 1] ?? path;
}

export const wailsTransport: AppTransport = {
  async bootstrap(): Promise<BootstrapResult> {
    for (let attempt = 0; attempt < 50; attempt += 1) {
      const ready = await AppReady().catch(() => false);
      if (ready) {
        const [appDirs, locale] = await Promise.all([
          AppDirs(),
          SettingsGetLocale().catch(() => "en-US"),
        ]);
        return {
          ready: true,
          locale,
          appDirs,
          capabilities: {
            isDesktop: true,
            canOpenLocalFiles: true,
            canOpenFolders: true,
            canDownloadPdf: true,
          },
          authRequired: false,
          session: {
            authenticated: true,
            user: {
              id: 0,
              email: "desktop@local",
              role: "admin",
            },
            locale,
            capabilities: {
              backups: true,
              pdfDownload: true,
              pdfGenerate: true,
              desktopPaths: true,
            },
            ready: true,
          },
        };
      }
      await new Promise((resolve) => window.setTimeout(resolve, 100));
    }

    throw new Error("backend startup timed out");
  },

  async getSession(): Promise<SessionInfo> {
    const locale = await SettingsGetLocale().catch(() => "en-US");
    return {
      authenticated: true,
      user: {
        id: 0,
        email: "desktop@local",
        role: "admin",
      },
      locale,
      capabilities: {
        backups: true,
        pdfDownload: true,
        pdfGenerate: true,
        desktopPaths: true,
      },
      ready: true,
    };
  },

  async login() {
    const locale = await SettingsGetLocale().catch(() => "en-US");
    return {
      authenticated: true,
      user: {
        id: 0,
        email: "desktop@local",
        role: "admin",
      },
      locale,
      capabilities: {
        backups: true,
        pdfDownload: true,
        pdfGenerate: true,
        desktopPaths: true,
      },
      ready: true,
    };
  },

  async logout() {},

  getLocale: SettingsGetLocale,
  setLocale: SettingsSetLocale,

  async createBackup(): Promise<BackupResult> {
    const path = await BackupNow();
    return {
      filename: pathToFilename(path),
      path,
    };
  },

  openLocalPath: OpenFile,

  async listStudents(q, includeInactive) {
    return (await StudentList(q, includeInactive)) as StudentDTO[];
  },
  async getStudent(id) {
    return (await StudentGet(id)) as StudentDTO;
  },
  async createStudent(fullName, personalCode, phone, email, note, isMinor, payerName, payerRole) {
    return (await StudentCreate(
      fullName,
      personalCode,
      phone,
      email,
      note,
      isMinor,
      payerName,
      payerRole
    )) as StudentDTO;
  },
  async updateStudent(id, fullName, personalCode, phone, email, note, isMinor, payerName, payerRole) {
    return (await StudentUpdate(
      id,
      fullName,
      personalCode,
      phone,
      email,
      note,
      isMinor,
      payerName,
      payerRole
    )) as StudentDTO;
  },
  setStudentActive: StudentSetActive,
  deleteStudent: StudentDelete,

  async listTeachers(q) {
    return (await TeacherList(q)) as TeacherDTO[];
  },
  async createTeacher(fullName) {
    return (await TeacherCreate(fullName)) as TeacherDTO;
  },

  async listCourses(q) {
    return (await CourseList(q)) as CourseDTO[];
  },
  async getCourse(id) {
    return (await CourseGet(id)) as CourseDTO;
  },
  async createCourse(name, teacherId, courseType, lessonPrice, subscriptionPrice) {
    const teacher = typeof teacherId === "number" && teacherId > 0 ? teacherId : undefined;
    return (await CourseCreate(
      name,
      teacher,
      courseType,
      lessonPrice,
      subscriptionPrice
    )) as CourseDTO;
  },
  async updateCourse(id, name, teacherId, courseType, lessonPrice, subscriptionPrice) {
    const teacher = typeof teacherId === "number" && teacherId > 0 ? teacherId : undefined;
    return (await CourseUpdate(
      id,
      name,
      teacher,
      courseType,
      lessonPrice,
      subscriptionPrice
    )) as CourseDTO;
  },
  deleteCourse: CourseDelete,

  async listEnrollments(studentId, courseId) {
    const sid = typeof studentId === "number" && studentId > 0 ? studentId : null;
    const cid = typeof courseId === "number" && courseId > 0 ? courseId : null;
    return (await EnrollmentList(sid, cid)) as any;
  },
  async createEnrollment(studentId, courseId, billingMode, discountPct, note) {
    return (await EnrollmentCreate(studentId, courseId, billingMode, discountPct, note)) as any;
  },
  async updateEnrollment(enrollmentId, billingMode, discountPct, note) {
    return (await EnrollmentUpdate(enrollmentId, billingMode, discountPct, note)) as any;
  },
  deleteEnrollment: EnrollmentDelete,

  async fetchAttendanceRows(year, month, courseId) {
    const cid = typeof courseId === "number" && courseId > 0 ? courseId : undefined;
    return (await AttendanceListPerLesson(year, month, cid)) as Row[];
  },
  saveAttendanceHours: AttendanceUpsert,
  async addAttendanceHours(year, month, courseId) {
    const cid = typeof courseId === "number" && courseId > 0 ? courseId : undefined;
    return AttendanceAddOne(year, month, cid);
  },

  async listInvoices(year, month, status) {
    return (await InvoiceList(year, month, status)) as InvoiceListItem[];
  },
  async getInvoice(id) {
    return (await InvoiceGet(id)) as InvoiceDTO;
  },
  async generateDrafts(year, month) {
    return (await InvoiceGenerateDrafts(year, month)) as GenerateResult;
  },
  deleteDraft: InvoiceDeleteDraft,
  reopenToDraft: InvoiceReopenDraft,
  async issueInvoice(id) {
    return (await InvoiceIssue(id)) as IssueResult;
  },
  async rebuildStudentDraft(studentId, year, month) {
    return (await InvoiceRebuildStudentDraft(studentId, year, month)) as GenerateResult;
  },
  async ensurePdf(invoiceId): Promise<EnsurePdfResult> {
    const path = await InvoiceEnsurePDF(invoiceId);
    return {
      filename: pathToFilename(path),
      localPath: path,
    };
  },
  hasPdf: InvoiceHasPDF,

  async createPayment(studentId, invoiceId, amount, method, paidAt, note) {
    const inv = invoiceId ? invoiceId : undefined;
    return (await PaymentCreate(studentId, inv, amount, method, paidAt, note)) as PaymentDTO;
  },
  deletePayment: PaymentDelete,
  async listDebtors() {
    return (await DebtorsList()) as any;
  },
  async invoiceSummary(invoiceId) {
    return (await InvoicePaymentSummary(invoiceId)) as any;
  },
  async studentDebtDetails(studentId) {
    return (await StudentDebtDetails(studentId)) as any;
  },
  async studentBalance(studentId) {
    return (await StudentBalance(studentId)) as any;
  },
  async paymentListForStudent(studentId) {
    return (await PaymentListForStudent(studentId)) as any;
  },
  async quickCash(studentId, amount, note) {
    return (await PaymentQuickCash(studentId, amount, note)) as any;
  },

  async loadMonthOverview(year, month) {
    return (await MonthOverview(year, month)) as any;
  },
  async loadRecentPayments(limit = 8) {
    return (await RecentPayments(limit)) as any;
  },
};
