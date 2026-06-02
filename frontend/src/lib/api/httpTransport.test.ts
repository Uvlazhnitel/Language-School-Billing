import { afterEach, describe, expect, it, vi } from "vitest";

import { httpTransport } from "./httpTransport";

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
      if (url.endsWith("/api/meta")) {
        return jsonResponse({
          ready: true,
          locale: "en-US",
          capabilities: { pdfDownload: true },
        });
      }
      throw new Error(`unexpected url ${url}`);
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await httpTransport.bootstrap();

    expect(result.ready).toBe(true);
    expect(result.capabilities.isDesktop).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("uses VITE_API_BASE_URL override", async () => {
    vi.stubEnv("VITE_API_BASE_URL", "http://localhost:9999/api");
    const fetchMock = vi.fn(async () => jsonResponse([]));
    vi.stubGlobal("fetch", fetchMock);

    await httpTransport.listStudents("", true);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:9999/api/students?q=&includeInactive=true",
      expect.any(Object)
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

  it("creates backups and returns safe metadata", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ filename: "app-20260602.sqlite" })));

    await expect(httpTransport.createBackup()).resolves.toEqual({
      filename: "app-20260602.sqlite",
    });
  });

  it("throws readable error messages", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "boom" }, 400)));

    await expect(httpTransport.getStudent(1)).rejects.toThrow("boom");
  });
});
