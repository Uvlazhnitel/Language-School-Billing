import { afterEach, describe, expect, it, vi } from "vitest";

import { getTransport, isWailsRuntime, setTransportForTests } from "./shared";

describe("transport selection", () => {
  afterEach(() => {
    setTransportForTests(null);
    vi.unstubAllGlobals();
    vi.resetModules();
  });

  it("detects Wails runtime when bridge exists", () => {
    vi.stubGlobal("window", {
      go: { main: { App: {} } },
    });

    expect(isWailsRuntime()).toBe(true);
  });

  it("detects non-Wails runtime when bridge is absent", () => {
    vi.stubGlobal("window", {});

    expect(isWailsRuntime()).toBe(false);
  });

  it("returns injected transport in tests", async () => {
    const transport = {
      bootstrap: vi.fn(),
    } as any;
    setTransportForTests(transport);

    await expect(getTransport()).resolves.toBe(transport);
  });
});
