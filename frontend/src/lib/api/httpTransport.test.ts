import { afterEach, describe, expect, it, vi } from "vitest";

import { httpTransport } from "./httpTransport";
import { AuthRequiredError, AuthorizationError } from "./shared";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

describe("httpTransport", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
  });

  it("bootstraps using same-origin api by default", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/healthz")) return jsonResponse({ ready: true });
      if (url.endsWith("/api/auth/session")) {
        return jsonResponse({
          authenticated: false,
          ready: true,
          locale: "en-US",
          capabilities: { pdfDownload: true, emailSend: true, invoiceArchive: true },
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await httpTransport.bootstrap();

    expect(result.ready).toBe(true);
    expect(result.capabilities.canDownloadPdf).toBe(true);
    expect(result.capabilities.canSendEmail).toBe(true);
    expect(result.capabilities.canViewInvoiceArchive).toBe(true);
    expect(result.authRequired).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("bootstraps authenticated users with persisted per-user locale", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/healthz")) return jsonResponse({ ready: true });
      if (url.endsWith("/api/auth/session")) {
        return jsonResponse({
          authenticated: true,
          ready: true,
          locale: "lv-LV",
          capabilities: { pdfDownload: true, emailSend: true, invoiceArchive: true },
          user: { id: 1, username: "tester", role: "staff" },
        });
      }
      if (url.endsWith("/api/me/locale")) {
        return jsonResponse({ locale: "ru-RU" });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await httpTransport.bootstrap();

    expect(result.locale).toBe("ru-RU");
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("falls back to session locale when per-user locale fetch fails", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/healthz")) return jsonResponse({ ready: true });
      if (url.endsWith("/api/auth/session")) {
        return jsonResponse({
          authenticated: true,
          ready: true,
          locale: "lv-LV",
          capabilities: { pdfDownload: true, emailSend: true, invoiceArchive: true },
          user: { id: 1, username: "tester", role: "staff" },
        });
      }
      if (url.endsWith("/api/me/locale")) {
        return jsonResponse({ error: "boom" }, 500);
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await httpTransport.bootstrap();

    expect(result.locale).toBe("lv-LV");
  });

  it("uses VITE_API_BASE_URL override", async () => {
    vi.stubEnv("VITE_API_BASE_URL", "http://localhost:9999/api");
    const fetchMock = vi.fn(async () => jsonResponse([]));
    vi.stubGlobal("fetch", fetchMock);

    await httpTransport.listStudents("", true);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:9999/api/students?q=&includeInactive=true",
      expect.objectContaining({ credentials: "include" })
    );
  });

  it("maps student duplicate-check endpoint", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/students/duplicate-check")) {
        return jsonResponse({
          exactMatch: {
            id: 7,
            version: 1,
            fullName: "Anna Student",
            personalCode: "020202-23456",
            phone: "123",
            email: "anna@example.com",
            note: "",
            isMinor: false,
            payerName: "",
            payerRole: "",
            isActive: true,
            balance: 0,
            debt: 0,
          },
          possibleMatches: [],
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      httpTransport.checkStudentDuplicates("Anna Student", "020202-23456", "123", "anna@example.com")
    ).resolves.toEqual({
      exactMatch: expect.objectContaining({
        id: 7,
        fullName: "Anna Student",
        personalCode: "020202-23456",
      }),
      possibleMatches: [],
    });
  });

  it("maps atomic student onboarding endpoint", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (!url.endsWith("/api/students/onboard")) {
        throw new Error(`unexpected url ${url}`);
      }
      expect(JSON.parse(String(init?.body))).toEqual({
        student: expect.objectContaining({ fullName: "Anna Student" }),
        enrollment: expect.objectContaining({ courseId: 4, billingMode: "per_lesson" }),
      });
      return jsonResponse({
        student: { id: 7, fullName: "Anna Student" },
        enrollments: [{ id: 9, studentId: 7, courseId: 4 }],
        enrollment: { id: 9, studentId: 7, courseId: 4 },
      }, 201);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      httpTransport.createStudentWithEnrollment(
        {
          fullName: "Anna Student",
          personalCode: "",
          phone: "",
          email: "",
          note: "",
          isMinor: false,
          payerName: "",
          payerRole: "",
        },
        {
          courseId: 4,
          billingMode: "per_lesson",
          chargeMaterials: true,
          lessonPriceOverride: 15,
          subscriptionLessonPrice: 0,
          note: "",
        }
      )
    ).resolves.toEqual({
      student: expect.objectContaining({ id: 7 }),
      enrollments: [expect.objectContaining({ id: 9 })],
      enrollment: expect.objectContaining({ id: 9 }),
    });
  });

  it("maps multi-course student onboarding and bulk enrollment endpoints", async () => {
    const enrollments = [
      {
        courseId: 4,
        billingMode: "per_lesson" as const,
        chargeMaterials: true,
        lessonPriceOverride: 15,
        subscriptionLessonPrice: 0,
        note: "",
      },
      {
        courseId: 5,
        billingMode: "per_lesson" as const,
        chargeMaterials: false,
        lessonPriceOverride: 25,
        subscriptionLessonPrice: 0,
        note: "",
      },
    ];
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const payload = JSON.parse(String(init?.body));
      if (url.endsWith("/api/students/onboard")) {
        expect(payload.enrollments).toEqual(enrollments);
        return jsonResponse({
          student: { id: 7, fullName: "Anna Student" },
          enrollments: enrollments.map((item, index) => ({ id: index + 10, ...item })),
          enrollment: { id: 10, ...enrollments[0] },
        }, 201);
      }
      if (url.endsWith("/api/enrollments/bulk")) {
        expect(payload).toEqual({ studentId: 7, enrollments });
        return jsonResponse({
          enrollments: [{ id: 11, ...enrollments[1] }],
          skippedCourseIds: [4],
        }, 201);
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    const student = {
      fullName: "Anna Student",
      personalCode: "",
      phone: "",
      email: "",
      note: "",
      isMinor: false,
      payerName: "",
      payerRole: "",
    };
    await expect(httpTransport.createStudentWithEnrollments(student, enrollments)).resolves.toEqual(
      expect.objectContaining({ enrollments: expect.arrayContaining([expect.objectContaining({ id: 10 })]) })
    );
    await expect(httpTransport.createEnrollmentsBulk(7, enrollments)).resolves.toEqual({
      enrollments: [expect.objectContaining({ id: 11 })],
      skippedCourseIds: [4],
    });
  });

  it("maps invoice pdf endpoints", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/invoices/12/pdf-status")) return jsonResponse({ ready: true });
      if (url.endsWith("/api/invoices/12/pdf")) {
        return jsonResponse({ filename: "invoice-12.pdf", downloadUrl: "/api/invoices/12/pdf" });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(httpTransport.hasPdf(12)).resolves.toBe(true);
    await expect(httpTransport.ensurePdf(12)).resolves.toEqual({
      filename: "invoice-12.pdf",
      downloadUrl: "/api/invoices/12/pdf",
    });
  });

  it("maps invoice issue endpoints with pdf status", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/invoices/12/issue")) {
        return jsonResponse({ number: "LS-202606-001", pdfReady: false, pdfStatus: "pending" });
      }
      if (url.endsWith("/api/invoices/issue-all")) {
        return jsonResponse({
          count: 2,
          pdfPaths: ["/tmp/LS-202606-001.pdf"],
          generatedCount: 1,
          pendingCount: 1,
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(httpTransport.issueInvoice(12, 3)).resolves.toEqual({
      number: "LS-202606-001",
      pdfReady: false,
      pdfStatus: "pending",
    });
    await expect(httpTransport.issueAllInvoices(2026, 6)).resolves.toEqual({
      count: 2,
      pdfPaths: ["/tmp/LS-202606-001.pdf"],
      generatedCount: 1,
      pendingCount: 1,
    });
  });

  it("maps ensure-all-pdfs endpoint", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/invoices/ensure-pdf-all")) {
        return jsonResponse({
          year: 2026,
          month: 6,
          processed: 2,
          generatedCount: 1,
          alreadyReadyCount: 1,
          failedCount: 0,
          items: [
            {
              invoiceId: 1,
              number: "LS-202606-001",
              studentName: "Alice",
              status: "issued",
              result: "generated",
            },
            {
              invoiceId: 2,
              number: "LS-202606-002",
              studentName: "Bob",
              status: "paid",
              result: "already_ready",
            },
          ],
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(httpTransport.ensureAllPdfs(2026, 6)).resolves.toEqual({
      year: 2026,
      month: 6,
      processed: 2,
      generatedCount: 1,
      alreadyReadyCount: 1,
      failedCount: 0,
      items: [
        {
          invoiceId: 1,
          number: "LS-202606-001",
          studentName: "Alice",
          status: "issued",
          result: "generated",
        },
        {
          invoiceId: 2,
          number: "LS-202606-002",
          studentName: "Bob",
          status: "paid",
          result: "already_ready",
        },
      ],
    });
  });

  it("maps invoice email endpoints", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/invoices/12/email-preview")) {
        return jsonResponse({
          to: "alice@example.com",
          subject: "Rēķins LS-202606-001",
          body: "Labdien!",
          attachmentFilename: "LS-202606-001.pdf",
        });
      }
      if (url.endsWith("/api/invoices/12/send-email")) {
        return jsonResponse({
          to: "alice@example.com",
          subject: "Rēķins LS-202606-001",
          attachmentFilename: "LS-202606-001.pdf",
          sentAt: "2026-06-15T12:00:00Z",
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(httpTransport.previewInvoiceEmail(12)).resolves.toEqual({
      to: "alice@example.com",
      subject: "Rēķins LS-202606-001",
      body: "Labdien!",
      attachmentFilename: "LS-202606-001.pdf",
    });
    await expect(
      httpTransport.sendInvoiceEmail(12, {
        to: "alice@example.com",
        subject: "Rēķins LS-202606-001",
        body: "Labdien!",
      })
    ).resolves.toEqual({
      to: "alice@example.com",
      subject: "Rēķins LS-202606-001",
      attachmentFilename: "LS-202606-001.pdf",
      sentAt: "2026-06-15T12:00:00Z",
    });
  });

  it("maps invoice email settings endpoints", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/settings/invoice-email")) {
        return jsonResponse({
          subjectTemplate: "Rēķins {invoice_number}",
          bodyTemplate: "Labdien!",
          replyTo: "hello@example.com",
          availablePlaceholders: ["{invoice_number}"],
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(httpTransport.getInvoiceEmailSettings()).resolves.toEqual({
      subjectTemplate: "Rēķins {invoice_number}",
      bodyTemplate: "Labdien!",
      replyTo: "hello@example.com",
      availablePlaceholders: ["{invoice_number}"],
    });
    await expect(
      httpTransport.saveInvoiceEmailSettings({
        subjectTemplate: "",
        bodyTemplate: "",
        replyTo: "",
      })
    ).resolves.toEqual({
      subjectTemplate: "Rēķins {invoice_number}",
      bodyTemplate: "Labdien!",
      replyTo: "hello@example.com",
      availablePlaceholders: ["{invoice_number}"],
    });
  });

  it("maps invoice archive endpoint", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/invoice-archive")) {
        return jsonResponse({
          years: [
            {
              year: 2026,
              count: 4,
              expandedByDefault: true,
              months: [
                {
                  month: 6,
                  count: 4,
                  expandedByDefault: true,
                  invoices: [
                    {
                      invoiceId: 1001,
                      year: 2026,
                      month: 6,
                      number: "LS-202606-001",
                      studentName: "Archive Student",
                      recipientName: "Archive Parent",
                      total: 30,
                      status: "issued",
                      pdfStatus: "ready",
                      pdfUpdatedAt: "2026-06-15T12:00:00Z",
                      openUrl: "/api/invoice-archive/2026/06/LS-202606-001.pdf/open",
                      downloadUrl: "/api/invoice-archive/2026/06/LS-202606-001.pdf/download",
                    },
                    {
                      invoiceId: 1002,
                      year: 2026,
                      month: 6,
                      number: "LS-202606-002",
                      studentName: "Missing Student",
                      recipientName: "Missing Parent",
                      total: 40,
                      status: "issued",
                      pdfStatus: "missing",
                    },
                    {
                      invoiceId: 1003,
                      year: 2026,
                      month: 6,
                      number: "LS-202606-003",
                      studentName: "Outdated Student",
                      recipientName: "Outdated Parent",
                      total: 50,
                      status: "paid",
                      pdfStatus: "outdated",
                      pdfUpdatedAt: "2026-06-14T12:00:00Z",
                    },
                    {
                      invoiceId: 1004,
                      year: 2026,
                      month: 6,
                      number: "LS-202606-004",
                      studentName: "Error Student",
                      recipientName: "Error Parent",
                      total: 60,
                      status: "issued",
                      pdfStatus: "error",
                    },
                  ],
                },
              ],
            },
          ],
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(httpTransport.listInvoiceArchive()).resolves.toEqual({
      years: [
        {
          year: 2026,
          count: 4,
          expandedByDefault: true,
          months: [
            {
              month: 6,
              count: 4,
              expandedByDefault: true,
              invoices: [
                {
                  invoiceId: 1001,
                  year: 2026,
                  month: 6,
                  number: "LS-202606-001",
                  studentName: "Archive Student",
                  recipientName: "Archive Parent",
                  total: 30,
                  status: "issued",
                  pdfStatus: "ready",
                  pdfUpdatedAt: "2026-06-15T12:00:00Z",
                  openUrl: "/api/invoice-archive/2026/06/LS-202606-001.pdf/open",
                  downloadUrl: "/api/invoice-archive/2026/06/LS-202606-001.pdf/download",
                },
                {
                  invoiceId: 1002,
                  year: 2026,
                  month: 6,
                  number: "LS-202606-002",
                  studentName: "Missing Student",
                  recipientName: "Missing Parent",
                  total: 40,
                  status: "issued",
                  pdfStatus: "missing",
                },
                {
                  invoiceId: 1003,
                  year: 2026,
                  month: 6,
                  number: "LS-202606-003",
                  studentName: "Outdated Student",
                  recipientName: "Outdated Parent",
                  total: 50,
                  status: "paid",
                  pdfStatus: "outdated",
                  pdfUpdatedAt: "2026-06-14T12:00:00Z",
                },
                {
                  invoiceId: 1004,
                  year: 2026,
                  month: 6,
                  number: "LS-202606-004",
                  studentName: "Error Student",
                  recipientName: "Error Parent",
                  total: 60,
                  status: "issued",
                  pdfStatus: "error",
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("creates backups and returns safe metadata", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ filename: "app-20260602.sqlite" })));

    await expect(httpTransport.createBackup()).resolves.toEqual({
      filename: "app-20260602.sqlite",
    });
  });

  it("uses per-user locale endpoints", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("/api/me/locale")) {
        return jsonResponse({ locale: "ru-RU" });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(httpTransport.getLocale()).resolves.toBe("ru-RU");
    await expect(httpTransport.setLocale("ru-RU")).resolves.toBeUndefined();
  });

  it("throws readable error messages", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "boom" }, 400)));

    await expect(httpTransport.getStudent(1)).rejects.toThrow("boom");
  });

  it("surfaces 401 as auth errors", async () => {
    vi.stubGlobal("window", { dispatchEvent: vi.fn() } as unknown as Window);
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "auth" }, 401)));

    await expect(httpTransport.getStudent(1)).rejects.toBeInstanceOf(AuthRequiredError);
  });

  it("surfaces 403 as authorization errors", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "forbidden" }, 403)));

    await expect(httpTransport.createBackup()).rejects.toBeInstanceOf(AuthorizationError);
  });
});
