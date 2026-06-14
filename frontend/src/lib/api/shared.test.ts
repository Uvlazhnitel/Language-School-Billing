import { afterEach, describe, expect, it, vi } from "vitest";

import { getTransport, setTransportForTests } from "./shared";

describe("transport selection", () => {
  afterEach(() => {
    setTransportForTests(null);
    vi.resetModules();
  });

  it("returns injected transport in tests", async () => {
    const transport = {
      bootstrap: vi.fn(),
    } as any;
    setTransportForTests(transport);

    await expect(getTransport()).resolves.toBe(transport);
  });
});
