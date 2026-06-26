import { describe, expect, it } from "vitest";

import { buildIssueFeedback } from "./invoiceIssueFeedback";

describe("buildIssueFeedback", () => {
  const t = (key: string, vars?: Record<string, string | number>) => `${key}:${vars?.number ?? ""}`;

  it("returns success text when pdf is ready", () => {
    expect(buildIssueFeedback({ number: "42", pdfReady: true, pdfStatus: "ready" }, t)).toEqual({
      text: "msg.invoiceIssuedPdfReady:42",
      type: "success",
    });
  });

  it("returns warning text when pdf is pending", () => {
    expect(buildIssueFeedback({ number: "42", pdfReady: false, pdfStatus: "pending" }, t)).toEqual({
      text: "msg.invoiceIssuedPdfPending:42",
      type: "warning",
    });
  });
});
