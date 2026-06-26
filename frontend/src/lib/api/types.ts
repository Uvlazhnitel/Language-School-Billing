export type TransportCapabilities = {
  canDownloadPdf: boolean;
  canSendEmail: boolean;
  canViewInvoiceArchive: boolean;
};

export type BootstrapResult = {
  ready: boolean;
  locale: string;
  capabilities: TransportCapabilities;
  authRequired: boolean;
  session: SessionInfo;
};

export type SessionUser = {
  id: number;
  username: string;
  role: string;
};

export type UserDTO = {
  id: number;
  username: string;
  role: string;
  isActive: boolean;
};

export type SessionInfo = {
  authenticated: boolean;
  user?: SessionUser;
  locale: string;
  capabilities: Record<string, boolean>;
  ready: boolean;
};

export type BackupResult = {
  filename: string;
  path?: string;
};

export type EnsurePdfResult = {
  filename: string;
  localPath?: string;
  downloadUrl?: string;
};

export type EnsureAllPDFsItemResult = {
  invoiceId: number;
  number: string;
  studentName: string;
  status: InvoiceStatus;
  result: "generated" | "already_ready" | "failed";
  message?: string;
};

export type EnsureAllPDFsResult = {
  year: number;
  month: number;
  processed: number;
  generatedCount: number;
  alreadyReadyCount: number;
  failedCount: number;
  items: EnsureAllPDFsItemResult[];
};

export type InvoiceEmailPreviewResult = {
  to: string;
  subject: string;
  body: string;
  attachmentFilename: string;
};

export type InvoiceEmailSendResult = {
  to: string;
  subject: string;
  attachmentFilename: string;
  sentAt: string;
};

export type InvoiceEmailSettingsDTO = {
  subjectTemplate: string;
  bodyTemplate: string;
  replyTo: string;
  availablePlaceholders: string[];
};

export type InvoiceArchiveInvoiceDTO = {
  invoiceId: number;
  year: number;
  month: number;
  number: string;
  studentName: string;
  recipientName: string;
  total: number;
  status: string;
  pdfStatus: "ready" | "missing" | "outdated" | "error";
  pdfFilename?: string;
  pdfUpdatedAt?: string;
  openUrl?: string;
  downloadUrl?: string;
};

export type InvoiceArchiveMonthDTO = {
  month: number;
  count: number;
  readyPdfCount: number;
  missingPdfCount: number;
  zipDownloadUrl?: string;
  expandedByDefault: boolean;
  invoices: InvoiceArchiveInvoiceDTO[];
};

export type InvoiceArchiveYearDTO = {
  year: number;
  count: number;
  expandedByDefault: boolean;
  months: InvoiceArchiveMonthDTO[];
};

export type InvoiceArchiveResult = {
  years: InvoiceArchiveYearDTO[];
};

export type Row = {
  enrollmentId: number;
  enrollmentVersion: number;
  studentId: number;
  studentName: string;
  courseId: number;
  courseName: string;
  courseType: CourseType;
  billingMode: BillingMode;
  lessonPrice: number;
  discountPct: number;
  subscriptionLessonPrice: number;
  hours: number;
  hasRecord: boolean;
  canDelete: boolean;
  attendanceLocked: boolean;
  invoiceStatus?: InvoiceStatus;
};

export type StudentDTO = {
  id: number;
  version: number;
  fullName: string;
  personalCode: string;
  phone: string;
  email: string;
  note: string;
  isMinor: boolean;
  payerName: string;
  payerRole: string;
  isActive: boolean;
  balance: number;
  debt: number;
};

export type TeacherDTO = {
  id: number;
  fullName: string;
  isActive: boolean;
};

export type CourseDTO = {
  id: number;
  version: number;
  name: string;
  teacherId?: number;
  teacherName: string;
  type: CourseType;
  lessonPrice: number;
  subscriptionPrice: number;
};

export type EnrollmentDTO = {
  id: number;
  version: number;
  studentId: number;
  studentName: string;
  courseId: number;
  courseName: string;
  courseType: CourseType;
  teacherId?: number;
  teacherName: string;
  billingMode: BillingMode;
  chargeMaterials: boolean;
  discountPct: number;
  subscriptionLessonPrice: number;
  note: string;
  createdAt: string;
};

export type CourseMonthSubscriptionDTO = {
  courseId: number;
  year: number;
  month: number;
  lessonsHeld: number;
};

export type InvoiceListItem = {
  id: number;
  version: number;
  studentId: number;
  studentName: string;
  year: number;
  month: number;
  total: number;
  status: InvoiceStatus;
  pdfReady: boolean;
  linesCount: number;
  number?: string;
  eventDate: string;
  lastEmailedAt?: string;
  lastEmailedTo?: string;
};

export type InvoiceListItemView = InvoiceListItem;

export type InvoiceLine = {
  enrollmentId: number;
  description: string;
  qty: number;
  unitPrice: number;
  amount: number;
};

export type InvoiceDTO = {
  id: number;
  version: number;
  studentId: number;
  studentName: string;
  recipientName: string;
  recipientPhone: string;
  recipientEmail: string;
  childName: string;
  studentPersonalCode: string;
  isMinor: boolean;
  year: number;
  month: number;
  total: number;
  status: InvoiceStatus;
  pdfReady: boolean;
  number?: string;
  lastEmailedAt?: string;
  lastEmailedTo?: string;
  lines: InvoiceLine[];
};

export type GenerateResult = {
  created: number;
  updated: number;
  skippedHasInvoice: number;
  skippedNoLines: number;
};

export type IssueResult = {
  number: string;
  pdfReady: boolean;
  pdfStatus: "ready" | "pending";
};
export type IssueAllResult = {
  count: number;
  pdfPaths: string[];
  generatedCount: number;
  pendingCount: number;
};

export type PaymentDTO = {
  id: number;
  studentId: number;
  invoiceId?: number;
  paidAt: string;
  amount: number;
  method: PaymentMethod;
  note: string;
  createdAt: string;
};

export type DebtorDTO = {
  studentId: number;
  studentName: string;
  debt: number;
  totalInvoiced: number;
  totalPaid: number;
};

export type InvoiceSummaryDTO = {
  invoiceId: number;
  total: number;
  paid: number;
  remaining: number;
  status: string;
  number?: string;
};

export type DebtInvoiceDTO = {
  invoiceId: number;
  year: number;
  month: number;
  number?: string;
  total: number;
  paid: number;
  remaining: number;
  status: string;
};

export type BalanceDTO = {
  studentId: number;
  studentName: string;
  totalInvoiced: number;
  totalPaid: number;
  balance: number;
  debt: number;
};

export type MonthOverviewDTO = {
  year: number;
  month: number;
  activeStudents: number;
  activeCourses: number;
  enrollments: number;
  perLessonEnrollments: number;
  attendanceFilled: number;
  attendanceMissing: number;
  subscriptionCoursesTracked: number;
  subscriptionFilled: number;
  subscriptionMissing: number;
  monthControlTotal: number;
  monthControlFilled: number;
  monthControlMissing: number;
  draftInvoices: number;
  issuedInvoices: number;
  paidInvoices: number;
  pendingPdfInvoices: number;
  readyPdfInvoices: number;
  monthInvoicesTotal: number;
  emailedInvoices: number;
  notEmailedInvoices: number;
  overdueInvoicesCount: number;
  requiredStepsTotal: number;
  requiredStepsDone: number;
  monthClosingProgressPct: number;
  monthClosingStage:
    | "collecting_data"
    | "ready_to_issue"
    | "ready_to_generate_pdf"
    | "ready_to_send"
    | "ready_to_close";
  monthReadyToClose: boolean;
  totalIssued: number;
  totalPaid: number;
  paymentsMonthTotal: number;
  paymentsMonthCashTotal: number;
  paymentsMonthBankTotal: number;
  unlinkedCreditTotal: number;
  monthDebtTotal: number;
  historicalDebtTotal: number;
  actionQueueCount: number;
  debtorsCount: number;
  totalDebt: number;
};

export type RecentPaymentDTO = {
  id: number;
  studentId: number;
  studentName: string;
  invoiceId?: number;
  amount: number;
  method: string;
  paidAt: string;
  note: string;
};

export type AuditLogItem = {
  id: number;
  actorUserId?: number;
  actorLabel: string;
  entityType: string;
  entityId?: number;
  action: string;
  summary: string;
  beforeJson: string;
  afterJson: string;
  studentId?: number;
  invoiceId?: number;
  createdAt: string;
};

export type AuditLogListResult = {
  items: AuditLogItem[];
  total: number;
  page: number;
  pageSize: number;
};

export type CourseType = "group" | "individual";
export type BillingMode = "subscription" | "per_lesson";
export type InvoiceStatus =
  | "draft"
  | "issued_pending_pdf"
  | "issued"
  | "paid_pending_pdf"
  | "paid"
  | "canceled";
export type PaymentMethod = "cash" | "bank";

export interface AppTransport {
  bootstrap(): Promise<BootstrapResult>;
  getSession(): Promise<SessionInfo>;
  login(username: string, password: string, rememberMe: boolean): Promise<SessionInfo>;
  logout(): Promise<void>;
  getLocale(): Promise<string>;
  setLocale(locale: string): Promise<void>;
  createBackup(): Promise<BackupResult>;
  listUsers(): Promise<UserDTO[]>;
  createUser(username: string, password: string, role: string): Promise<UserDTO>;
  updateUser(id: number, username: string, role: string, isActive: boolean): Promise<UserDTO>;
  deleteUser(id: number): Promise<void>;
  setUserPassword(id: number, password: string): Promise<void>;
  setUserActive(id: number, active: boolean): Promise<UserDTO>;

  listStudents(q: string, includeInactive: boolean): Promise<StudentDTO[]>;
  getStudent(id: number): Promise<StudentDTO>;
  createStudent(
    fullName: string,
    personalCode: string,
    phone: string,
    email: string,
    note: string,
    isMinor: boolean,
    payerName: string,
    payerRole: string
  ): Promise<StudentDTO>;
  updateStudent(
    id: number,
    version: number,
    fullName: string,
    personalCode: string,
    phone: string,
    email: string,
    note: string,
    isMinor: boolean,
    payerName: string,
    payerRole: string
  ): Promise<StudentDTO>;
  setStudentActive(id: number, version: number, active: boolean): Promise<void>;
  deleteStudent(id: number, version: number): Promise<void>;

  listTeachers(q: string): Promise<TeacherDTO[]>;
  createTeacher(fullName: string): Promise<TeacherDTO>;

  listCourses(q: string): Promise<CourseDTO[]>;
  getCourse(id: number): Promise<CourseDTO>;
  createCourse(
    name: string,
    teacherId: number | undefined,
    courseType: CourseType,
    lessonPrice: number,
    subscriptionPrice: number
  ): Promise<CourseDTO>;
  updateCourse(
    id: number,
    version: number,
    name: string,
    teacherId: number | undefined,
    courseType: CourseType,
    lessonPrice: number,
    subscriptionPrice: number
  ): Promise<CourseDTO>;
  deleteCourse(id: number, version: number): Promise<void>;

  listEnrollments(studentId?: number, courseId?: number): Promise<EnrollmentDTO[]>;
  createEnrollment(
    studentId: number,
    courseId: number,
    billingMode: EnrollmentDTO["billingMode"],
    chargeMaterials: boolean,
    discountPct: number,
    subscriptionLessonPrice: number,
    note: string
  ): Promise<EnrollmentDTO>;
  updateEnrollment(
    enrollmentId: number,
    version: number,
    billingMode: EnrollmentDTO["billingMode"],
    chargeMaterials: boolean,
    discountPct: number,
    subscriptionLessonPrice: number,
    note: string
  ): Promise<EnrollmentDTO>;
  deleteEnrollment(enrollmentId: number, version: number): Promise<void>;

  fetchAttendanceRows(year: number, month: number, courseId?: number): Promise<Row[]>;
  listCourseMonthSubscriptions(
    year: number,
    month: number,
    courseId?: number
  ): Promise<CourseMonthSubscriptionDTO[]>;
  saveCourseMonthSubscriptionLessons(
    courseId: number,
    year: number,
    month: number,
    lessonsHeld: number
  ): Promise<CourseMonthSubscriptionDTO>;
  saveAttendanceHours(
    studentId: number,
    courseId: number,
    year: number,
    month: number,
    hours: number
  ): Promise<void>;
  addAttendanceHours(year: number, month: number, courseId?: number): Promise<number>;

  listInvoices(year: number, month: number, status: string): Promise<InvoiceListItem[]>;
  getInvoice(id: number): Promise<InvoiceDTO>;
  generateDrafts(year: number, month: number): Promise<GenerateResult>;
  deleteDraft(id: number, version: number): Promise<void>;
  reopenToDraft(id: number, version: number): Promise<void>;
  issueInvoice(id: number, version: number): Promise<IssueResult>;
  issueAllInvoices(year: number, month: number): Promise<IssueAllResult>;
  rebuildStudentDraft(studentId: number, year: number, month: number): Promise<GenerateResult>;
  ensurePdf(invoiceId: number): Promise<EnsurePdfResult>;
  ensureAllPdfs(year: number, month: number): Promise<EnsureAllPDFsResult>;
  hasPdf(invoiceId: number): Promise<boolean>;
  previewInvoiceEmail(invoiceId: number): Promise<InvoiceEmailPreviewResult>;
  sendInvoiceEmail(
    invoiceId: number,
    payload: Pick<InvoiceEmailPreviewResult, "to" | "subject" | "body">
  ): Promise<InvoiceEmailSendResult>;
  listInvoiceArchive(): Promise<InvoiceArchiveResult>;
  getInvoiceEmailSettings(): Promise<InvoiceEmailSettingsDTO>;
  saveInvoiceEmailSettings(
    payload: Pick<InvoiceEmailSettingsDTO, "subjectTemplate" | "bodyTemplate" | "replyTo">
  ): Promise<InvoiceEmailSettingsDTO>;

  createPayment(
    studentId: number,
    invoiceId: number | undefined,
    amount: number,
    method: PaymentMethod,
    paidAt: string,
    note: string
  ): Promise<PaymentDTO>;
  deletePayment(paymentId: number): Promise<void>;
  listDebtors(): Promise<DebtorDTO[]>;
  invoiceSummary(invoiceId: number): Promise<InvoiceSummaryDTO>;
  studentDebtDetails(studentId: number): Promise<DebtInvoiceDTO[]>;
  studentBalance(studentId: number): Promise<BalanceDTO>;
  paymentListForStudent(studentId: number): Promise<PaymentDTO[]>;
  quickCash(studentId: number, amount: number, note: string): Promise<PaymentDTO>;
  listAuditLogs(filters: {
    q?: string;
    actorLabel?: string;
    entityType?: string;
    action?: string;
    dateFrom?: string;
    dateTo?: string;
    page?: number;
    pageSize?: number;
  }): Promise<AuditLogListResult>;

  loadMonthOverview(year: number, month: number): Promise<MonthOverviewDTO>;
  loadRecentPayments(limit?: number): Promise<RecentPaymentDTO[]>;
}
