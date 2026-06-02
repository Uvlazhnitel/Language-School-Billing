import type {
  AppTransport,
  BackupResult,
  BalanceDTO,
  BootstrapResult,
  CourseDTO,
  DebtInvoiceDTO,
  DebtorDTO,
  EnrollmentDTO,
  EnsurePdfResult,
  GenerateResult,
  InvoiceDTO,
  InvoiceListItem,
  InvoiceSummaryDTO,
  IssueResult,
  MonthOverviewDTO,
  PaymentDTO,
  PaymentMethod,
  RecentPaymentDTO,
  Row,
  StudentDTO,
  TeacherDTO,
} from "./types";

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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase()}${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {}
    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
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
    const [health, meta] = await Promise.all([
      request<{ ready: boolean }>(`${healthBase()}/healthz`),
      request<{
        ready: boolean;
        locale: string;
        capabilities: Record<string, boolean>;
      }>("/meta"),
    ]);

    return {
      ready: health.ready && meta.ready,
      locale: meta.locale || "en-US",
      appDirs: null,
      capabilities: {
        isDesktop: false,
        canOpenLocalFiles: false,
        canOpenFolders: false,
        canDownloadPdf: Boolean(meta.capabilities?.pdfDownload),
      },
    };
  },

  async getLocale() {
    const res = await request<{ locale: string }>("/settings/locale");
    return res.locale;
  },

  async setLocale(locale: string) {
    await request("/settings/locale", {
      method: "POST",
      ...body({ locale }),
    });
  },

  async createBackup(): Promise<BackupResult> {
    return request<BackupResult>("/backups", { method: "POST", ...body({}) });
  },

  async openLocalPath() {
    throw new Error("Local file access is unavailable in web mode");
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
  async updateStudent(id, fullName, personalCode, phone, email, note, isMinor, payerName, payerRole) {
    return request<StudentDTO>(`/students/${id}`, {
      method: "PUT",
      ...body({ fullName, personalCode, phone, email, note, isMinor, payerName, payerRole }),
    });
  },
  async setStudentActive(id, active) {
    await requestVoid(`/students/${id}/active`, { method: "POST", ...body({ active }) });
  },
  async deleteStudent(id) {
    await requestVoid(`/students/${id}`, { method: "DELETE" });
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
  async updateCourse(id, name, teacherId, courseType, lessonPrice, subscriptionPrice) {
    return request<CourseDTO>(`/courses/${id}`, {
      method: "PUT",
      ...body({ name, teacherId, type: courseType, lessonPrice, subscriptionPrice }),
    });
  },
  async deleteCourse(id) {
    await requestVoid(`/courses/${id}`, { method: "DELETE" });
  },

  async listEnrollments(studentId, courseId) {
    const params = new URLSearchParams();
    if (typeof studentId === "number") params.set("studentId", String(studentId));
    if (typeof courseId === "number") params.set("courseId", String(courseId));
    const query = params.toString();
    return request<EnrollmentDTO[]>(`/enrollments${query ? `?${query}` : ""}`);
  },
  async createEnrollment(studentId, courseId, billingMode, discountPct, note) {
    return request<EnrollmentDTO>("/enrollments", {
      method: "POST",
      ...body({ studentId, courseId, billingMode, discountPct, note }),
    });
  },
  async updateEnrollment(enrollmentId, billingMode, discountPct, note) {
    return request<EnrollmentDTO>(`/enrollments/${enrollmentId}`, {
      method: "PUT",
      ...body({ billingMode, discountPct, note }),
    });
  },
  async deleteEnrollment(enrollmentId) {
    await requestVoid(`/enrollments/${enrollmentId}`, { method: "DELETE" });
  },

  async fetchAttendanceRows(year, month, courseId) {
    const params = new URLSearchParams({ year: String(year), month: String(month) });
    if (typeof courseId === "number") params.set("courseId", String(courseId));
    return request<Row[]>(`/attendance/per-lesson?${params.toString()}`);
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
  async deleteDraft(id) {
    await requestVoid(`/invoices/${id}/draft`, { method: "DELETE" });
  },
  async reopenToDraft(id) {
    await requestVoid(`/invoices/${id}/reopen-draft`, { method: "POST", ...body({}) });
  },
  async issueInvoice(id) {
    return request<IssueResult>(`/invoices/${id}/issue`, { method: "POST", ...body({}) });
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
  async hasPdf(invoiceId) {
    const res = await request<{ ready: boolean }>(`/invoices/${invoiceId}/pdf-status`);
    return res.ready;
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

  async loadMonthOverview(year, month) {
    return request<MonthOverviewDTO>(`/dashboard/month-overview?year=${year}&month=${month}`);
  },
  async loadRecentPayments(limit = 8) {
    return request<RecentPaymentDTO[]>(`/dashboard/recent-payments?limit=${limit}`);
  },
};
