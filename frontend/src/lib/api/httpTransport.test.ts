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
          capabilities: { pdfDownload: true, emailSend: true },
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await httpTransport.bootstrap();

    expect(result.ready).toBe(true);
    expect(result.capabilities.canDownloadPdf).toBe(true);
    expect(result.capabilities.canSendEmail).toBe(true);
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
          capabilities: { pdfDownload: true, emailSend: true },
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
          capabilities: { pdfDownload: true, emailSend: true },
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
