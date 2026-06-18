import { afterEach, describe, expect, it, vi } from "vitest";

import { genDrafts, listInvoices } from "./invoices";
import { setTransportForTests } from "./api";

describe("invoice transport helpers", () => {
  afterEach(() => {
    setTransportForTests(null);
  });

  it("calls generateDrafts only for explicit draft sync", async () => {
    const transport = {
      generateDrafts: vi.fn(async () => ({ created: 0, updated: 0, skippedHasInvoice: 0, skippedNoLines: 0 })),
      listInvoices: vi.fn(async () => []),
    } as any;

    setTransportForTests(transport);

    await listInvoices(2026, 6, "all");
    expect(transport.listInvoices).toHaveBeenCalledWith(2026, 6, "all");
    expect(transport.generateDrafts).not.toHaveBeenCalled();

    await genDrafts(2026, 6);
    expect(transport.generateDrafts).toHaveBeenCalledWith(2026, 6);
  });
});
