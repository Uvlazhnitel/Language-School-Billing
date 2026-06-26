import type {
  AppTransport,
  AuditLogListResult,
  BackupResult,
  BalanceDTO,
  BootstrapResult,
  CourseDTO,
  DebtInvoiceDTO,
  DebtorDTO,
  EnrollmentDTO,
  EnsureAllPDFsResult,
  EnsurePdfResult,
  GenerateResult,
  InvoiceArchiveResult,
  InvoiceDTO,
  InvoiceEmailPreviewResult,
  InvoiceEmailSendResult,
  InvoiceEmailSettingsDTO,
  InvoiceListItem,
  InvoiceSummaryDTO,
  IssueAllResult,
  IssueResult,
  MonthOverviewDTO,
  PaymentDTO,
  RecentPaymentDTO,
  Row,
  StudentDTO,
  TeacherDTO,
  SessionInfo,
  UserDTO,
} from "./types";
import { AUTH_REQUIRED_EVENT, AuthRequiredError, AuthorizationError, ConflictError } from "./shared";

type RequestOptions = RequestInit & {
  suppressAuthEvent?: boolean;
};

function healthBase(): string {
  const override = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim();
  if (override) {
    return override.replace(/\/api\/?$/, "");
  }
  return "";
}

function apiBase(): string {
  const override = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim();
  if (override) {
    return override.replace(/\/+$/, "");
  }
  return "/api";
}

async function requestAbsolute<T>(url: string, init?: RequestOptions): Promise<T> {
  const { suppressAuthEvent = false, ...requestInit } = init ?? {};
  const response = await fetch(url, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(requestInit.headers ?? {}),
    },
    ...requestInit,
  });

  if (response.status === 401) {
    if (!suppressAuthEvent && typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent(AUTH_REQUIRED_EVENT));
    }
    let message = "Authentication required";
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch (error) {
      void error;
    }
    throw new AuthRequiredError(message);
  }

  if (response.status === 403) {
    let message = "Forbidden";
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch (error) {
      void error;
    }
    throw new AuthorizationError(message);
  }

  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch (error) {
      void error;
    }
    if (response.status === 409) {
      throw new ConflictError(message, response.status);
    }
    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

async function request<T>(path: string, init?: RequestOptions): Promise<T> {
  return requestAbsolute(`${apiBase()}${path}`, init);
}

async function requestVoid(path: string, init?: RequestInit): Promise<void> {
  await request<undefined>(path, init);
}

function body(value: unknown): RequestInit {
  return {
    body: JSON.stringify(value),
  };
}

export const httpTransport: AppTransport = {
  async bootstrap(): Promise<BootstrapResult> {
    const [health, session] = await Promise.all([
      requestAbsolute<{ ready: boolean }>(`${healthBase()}/healthz`, { suppressAuthEvent: true }),
      request<SessionInfo>("/auth/session"),
    ]);

    let locale = session.locale || "lv-LV";
    if (session.authenticated) {
      try {
        locale = await this.getLocale();
      } catch (error) {
        void error;
      }
    }

    return {
      ready: health.ready && session.ready,
      locale,
      capabilities: {
        canDownloadPdf: Boolean(session.capabilities?.pdfDownload),
        canSendEmail: Boolean(session.capabilities?.emailSend),
        canViewInvoiceArchive: Boolean(session.capabilities?.invoiceArchive),
      },
      authRequired: true,
      session,
    };
  },

  getSession() {
    return request<SessionInfo>("/auth/session");
  },

  login(username, password, rememberMe) {
    return request<SessionInfo>("/auth/login", {
      method: "POST",
      suppressAuthEvent: true,
      ...body({ username, password, rememberMe }),
    });
  },

  async logout() {
    await requestVoid("/auth/logout", {
      method: "POST",
      ...body({}),
    });
  },

  async getLocale() {
    const res = await request<{ locale: string }>("/me/locale");
    return res.locale;
  },

  async setLocale(locale: string) {
    await request("/me/locale", {
      method: "POST",
      ...body({ locale }),
    });
  },

  async createBackup(): Promise<BackupResult> {
    return request<BackupResult>("/backups", { method: "POST", ...body({}) });
  },

  async listUsers() {
    return request<UserDTO[]>("/users");
  },
  async createUser(username, password, role) {
    return request<UserDTO>("/users", { method: "POST", ...body({ username, password, role }) });
  },
  async updateUser(id, username, role, isActive) {
    return request<UserDTO>(`/users/${id}`, { method: "PUT", ...body({ username, role, isActive }) });
  },
  async deleteUser(id) {
    await requestVoid(`/users/${id}`, { method: "DELETE" });
  },
  async setUserPassword(id, password) {
    await requestVoid(`/users/${id}/password`, { method: "POST", ...body({ password }) });
  },
  async setUserActive(id, active) {
    return request<UserDTO>(`/users/${id}/active`, { method: "POST", ...body({ active }) });
  },

  async listStudents(q, includeInactive) {
    return request<StudentDTO[]>(
      `/students?q=${encodeURIComponent(q)}&includeInactive=${String(includeInactive)}`
    );
  },
  async getStudent(id) {
    return request<StudentDTO>(`/students/${id}`);
  },
  async createStudent(fullName, personalCode, phone, email, note, isMinor, payerName, payerRole) {
    return request<StudentDTO>("/students", {
      method: "POST",
      ...body({ fullName, personalCode, phone, email, note, isMinor, payerName, payerRole }),
    });
  },
  async updateStudent(id, version, fullName, personalCode, phone, email, note, isMinor, payerName, payerRole) {
    return request<StudentDTO>(`/students/${id}`, {
      method: "PUT",
      ...body({ version, fullName, personalCode, phone, email, note, isMinor, payerName, payerRole }),
    });
  },
  async setStudentActive(id, version, active) {
    await requestVoid(`/students/${id}/active`, { method: "POST", ...body({ version, active }) });
  },
  async deleteStudent(id, version) {
    await requestVoid(`/students/${id}?version=${encodeURIComponent(String(version))}`, { method: "DELETE" });
  },

  async listTeachers(q) {
    return request<TeacherDTO[]>(`/teachers?q=${encodeURIComponent(q)}`);
  },
  async createTeacher(fullName) {
    return request<TeacherDTO>("/teachers", { method: "POST", ...body({ fullName }) });
  },

  async listCourses(q) {
    return request<CourseDTO[]>(`/courses?q=${encodeURIComponent(q)}`);
  },
  async getCourse(id) {
    return request<CourseDTO>(`/courses/${id}`);
  },
  async createCourse(name, teacherId, courseType, lessonPrice, subscriptionPrice) {
    return request<CourseDTO>("/courses", {
      method: "POST",
      ...body({ name, teacherId, type: courseType, lessonPrice, subscriptionPrice }),
    });
  },
  async updateCourse(id, version, name, teacherId, courseType, lessonPrice, subscriptionPrice) {
    return request<CourseDTO>(`/courses/${id}`, {
      method: "PUT",
      ...body({ version, name, teacherId, type: courseType, lessonPrice, subscriptionPrice }),
    });
  },
  async deleteCourse(id, version) {
    await requestVoid(`/courses/${id}?version=${encodeURIComponent(String(version))}`, { method: "DELETE" });
  },

  async listEnrollments(studentId, courseId) {
    const params = new URLSearchParams();
    if (typeof studentId === "number") params.set("studentId", String(studentId));
    if (typeof courseId === "number") params.set("courseId", String(courseId));
    const query = params.toString();
    return request<EnrollmentDTO[]>(`/enrollments${query ? `?${query}` : ""}`);
  },
  async createEnrollment(studentId, courseId, billingMode, chargeMaterials, lessonPriceOverride, subscriptionLessonPrice, note) {
    return request<EnrollmentDTO>("/enrollments", {
      method: "POST",
      ...body({
        studentId,
        courseId,
        billingMode,
        chargeMaterials,
        lessonPriceOverride,
        subscriptionLessonPrice,
        note,
      }),
    });
  },
  async updateEnrollment(enrollmentId, version, billingMode, chargeMaterials, lessonPriceOverride, subscriptionLessonPrice, note) {
    return request<EnrollmentDTO>(`/enrollments/${enrollmentId}`, {
      method: "PUT",
      ...body({ version, billingMode, chargeMaterials, lessonPriceOverride, subscriptionLessonPrice, note }),
    });
  },
  async deleteEnrollment(enrollmentId, version) {
    await requestVoid(`/enrollments/${enrollmentId}?version=${encodeURIComponent(String(version))}`, { method: "DELETE" });
  },

  async fetchAttendanceRows(year, month, courseId) {
    const params = new URLSearchParams({ year: String(year), month: String(month) });
    if (typeof courseId === "number") params.set("courseId", String(courseId));
    return request<Row[]>(`/attendance/per-lesson?${params.toString()}`);
  },
  async listCourseMonthSubscriptions(year, month, courseId) {
    const params = new URLSearchParams({ year: String(year), month: String(month) });
    if (typeof courseId === "number") params.set("courseId", String(courseId));
    return request(`/attendance/subscription-month?${params.toString()}`);
  },
  async saveCourseMonthSubscriptionLessons(courseId, year, month, lessonsHeld) {
    return request("/attendance/subscription-month", {
      method: "PUT",
      ...body({ courseId, year, month, lessonsHeld }),
    });
  },
  async saveAttendanceHours(studentId, courseId, year, month, hours) {
    await requestVoid("/attendance", {
      method: "PUT",
      ...body({ studentId, courseId, year, month, hours }),
    });
  },
  async addAttendanceHours(year, month, courseId) {
    const res = await request<{ count: number }>("/attendance/add-one", {
      method: "POST",
      ...body({ year, month, courseId }),
    });
    return res.count;
  },

  async listInvoices(year, month, status) {
    const params = new URLSearchParams({ year: String(year), month: String(month), status });
    return request<InvoiceListItem[]>(`/invoices?${params.toString()}`);
  },
  async getInvoice(id) {
    return request<InvoiceDTO>(`/invoices/${id}`);
  },
  async generateDrafts(year, month) {
    return request<GenerateResult>("/invoices/generate-drafts", {
      method: "POST",
      ...body({ year, month }),
    });
  },
  async deleteDraft(id, version) {
    await requestVoid(`/invoices/${id}/draft?version=${encodeURIComponent(String(version))}`, { method: "DELETE" });
  },
  async reopenToDraft(id, version) {
    await requestVoid(`/invoices/${id}/reopen-draft`, { method: "POST", ...body({ version }) });
  },
  async issueInvoice(id, version) {
    return request<IssueResult>(`/invoices/${id}/issue`, { method: "POST", ...body({ version }) });
  },
  async issueAllInvoices(year, month) {
    return request<IssueAllResult>("/invoices/issue-all", {
      method: "POST",
      ...body({ year, month }),
    });
  },
  async rebuildStudentDraft(studentId, year, month) {
    return request<GenerateResult>("/invoices/rebuild-student-draft", {
      method: "POST",
      ...body({ studentId, year, month }),
    });
  },
  async ensurePdf(invoiceId) {
    const res = await request<{ filename: string; downloadUrl: string }>(`/invoices/${invoiceId}/pdf`, {
      method: "POST",
      ...body({}),
    });
    return {
      filename: res.filename,
      downloadUrl: res.downloadUrl,
    } satisfies EnsurePdfResult;
  },
  async ensureAllPdfs(year, month) {
    return request<EnsureAllPDFsResult>("/invoices/ensure-pdf-all", {
      method: "POST",
      ...body({ year, month }),
    });
  },
  async hasPdf(invoiceId) {
    const res = await request<{ ready: boolean }>(`/invoices/${invoiceId}/pdf-status`);
    return res.ready;
  },
  async previewInvoiceEmail(invoiceId) {
    return request<InvoiceEmailPreviewResult>(`/invoices/${invoiceId}/email-preview`, {
      method: "POST",
      ...body({}),
    });
  },
  async sendInvoiceEmail(invoiceId, payload) {
    return request<InvoiceEmailSendResult>(`/invoices/${invoiceId}/send-email`, {
      method: "POST",
      ...body(payload),
    });
  },
  async listInvoiceArchive() {
    return request<InvoiceArchiveResult>("/invoice-archive");
  },
  async getInvoiceEmailSettings() {
    return request<InvoiceEmailSettingsDTO>("/settings/invoice-email");
  },
  async saveInvoiceEmailSettings(payload) {
    return request<InvoiceEmailSettingsDTO>("/settings/invoice-email", {
      method: "POST",
      ...body(payload),
    });
  },

  async createPayment(studentId, invoiceId, amount, method, paidAt, note) {
    return request<PaymentDTO>("/payments", {
      method: "POST",
      ...body({ studentId, invoiceId, amount, method, paidAt, note }),
    });
  },
  async deletePayment(paymentId) {
    await requestVoid(`/payments/${paymentId}`, { method: "DELETE" });
  },
  async listDebtors() {
    return request<DebtorDTO[]>("/debtors");
  },
  async invoiceSummary(invoiceId) {
    return request<InvoiceSummaryDTO>(`/invoices/${invoiceId}/payment-summary`);
  },
  async studentDebtDetails(studentId) {
    return request<DebtInvoiceDTO[]>(`/students/${studentId}/debt-details`);
  },
  async studentBalance(studentId) {
    return request<BalanceDTO>(`/payments/student/${studentId}/balance`);
  },
  async paymentListForStudent(studentId) {
    return request<PaymentDTO[]>(`/payments/student/${studentId}`);
  },
  async quickCash(studentId, amount, note) {
    return request<PaymentDTO>("/payments/quick-cash", {
      method: "POST",
      ...body({ studentId, amount, note }),
    });
  },

  async listAuditLogs(filters) {
    const params = new URLSearchParams();
    if (filters.q) params.set("q", filters.q);
    if (filters.actorLabel) params.set("actor", filters.actorLabel);
    if (filters.entityType) params.set("entityType", filters.entityType);
    if (filters.action) params.set("action", filters.action);
    if (filters.dateFrom) params.set("dateFrom", filters.dateFrom);
    if (filters.dateTo) params.set("dateTo", filters.dateTo);
    params.set("page", String(filters.page ?? 1));
    params.set("pageSize", String(filters.pageSize ?? 50));
    return request<AuditLogListResult>(`/audit-logs?${params.toString()}`);
  },

  async loadMonthOverview(year, month) {
    return request<MonthOverviewDTO>(`/dashboard/month-overview?year=${year}&month=${month}`);
  },
  async loadRecentPayments(limit = 8) {
    return request<RecentPaymentDTO[]>(`/dashboard/recent-payments?limit=${limit}`);
  },
};
