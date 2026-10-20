import { describe, expect, it, vi } from "vitest";
import { ReleaseNoticeController } from "./releaseNotices";
import type { ReleaseNoticeResult } from "./api/types";
function response(
  id = "release-1",
  dismissedAt: string | null = null,
  remaining = 86400000
): ReleaseNoticeResult {
  const now = Date.parse("2026-10-20T10:00:00Z");
  return {
    serverNow: new Date(now).toISOString(),
    notice: {
      releaseId: id,
      firstSeenAt: new Date(now).toISOString(),
      expiresAt: new Date(now + remaining).toISOString(),
      dismissedAt,
      content: { "en-US": { title: "Update", intro: "Intro", sections: [] } },
    },
  };
}
function storage() {
  const values = new Map<string, string>();
  return {
    getItem: (k: string) => values.get(k) ?? null,
    setItem: (k: string, v: string) => {
      values.set(k, v);
    },
    removeItem: (k: string) => {
      values.delete(k);
    },
  };
}
function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}
function setup(user = 1, value = response(), saved = storage()) {
  const api = { visit: vi.fn(async () => value), dismiss: vi.fn(async (_id: string) => {}) };
  let now = 0;
  const controller = new ReleaseNoticeController(user, api, saved, () => now);
  return {
    controller,
    api,
    saved,
    setTime: (time: number) => {
      now = time;
    },
  };
}
describe("release notices", () => {
  it("opens once, permits manual reopening, and does not check for new releases on focus", async () => {
    const { controller, api } = setup();
    await controller.start();
    expect(controller.snapshot.open).toBe(true);
    expect(controller.snapshot.available).toBe(true);
    controller.close();
    expect(controller.snapshot.open).toBe(false);
    await controller.reconnect();
    expect(controller.snapshot.open).toBe(false);
    expect(api.visit).toHaveBeenCalledTimes(1);
    expect(api.dismiss).toHaveBeenCalledWith("release-1", expect.any(AbortSignal));
    controller.open();
    expect(controller.snapshot.open).toBe(true);
  });
  it("does not auto-open a dismissed announcement from another device", async () => {
    const { controller } = setup(1, response("release-1", "2026-10-20T10:01:00Z"));
    await controller.start();
    expect(controller.snapshot.open).toBe(false);
    expect(controller.snapshot.available).toBe(true);
    controller.open();
    expect(controller.snapshot.open).toBe(true);
  });
  it("hides at the exact deadline but lets an already-open dialog finish", async () => {
    const { controller, setTime } = setup();
    await controller.start();
    setTime(86399999);
    controller.tick();
    expect(controller.snapshot.available).toBe(true);
    setTime(86400000);
    controller.tick();
    expect(controller.snapshot.available).toBe(false);
    expect(controller.snapshot.open).toBe(true);
    controller.close();
    controller.open();
    expect(controller.snapshot.open).toBe(false);
  });
  it("rechecks expiry on return from background without requesting a new announcement", async () => {
    const { controller, api, setTime } = setup();
    await controller.start();
    controller.close();
    setTime(86400001);
    await controller.reconnect();
    expect(controller.snapshot.available).toBe(false);
    expect(api.visit).toHaveBeenCalledTimes(1);
  });
  it("uses server time even if the device wall clock is incorrect", async () => {
    const { controller, setTime } = setup(1, response("release-1", null, 60000));
    const spy = vi.spyOn(Date, "now").mockReturnValue(0);
    try {
      await controller.start();
      setTime(60000);
      controller.tick();
      expect(controller.snapshot.available).toBe(false);
    } finally {
      spy.mockRestore();
    }
  });
  it("closes during an outage and retries account-scoped dismissal after reload", async () => {
    const { controller, api, saved } = setup();
    api.dismiss.mockRejectedValue(new Error("offline"));
    await controller.start();
    controller.close();
    await controller.reconnect();
    expect(controller.snapshot.open).toBe(false);
    expect(saved.getItem("release-notices:pending:1")).toContain("release-1");
    controller.stop();
    const next = setup(1, response(), saved);
    await next.controller.start();
    expect(next.controller.snapshot.open).toBe(false);
    expect(next.api.dismiss).toHaveBeenCalledWith("release-1", expect.any(AbortSignal));
    expect(saved.getItem("release-notices:pending:1")).toBeNull();
  });
  it("keeps queued dismissals separate for different accounts and allows the next release", async () => {
    const shared = storage();
    shared.setItem("release-notices:pending:1", JSON.stringify(["release-1"]));
    const other = setup(2, response(), shared);
    await other.controller.start();
    expect(other.controller.snapshot.open).toBe(true);
    expect(other.api.dismiss).not.toHaveBeenCalled();
    const next = setup(1, response("release-2"), shared);
    await next.controller.start();
    expect(next.controller.snapshot.open).toBe(true);
    expect(next.api.dismiss).toHaveBeenCalledWith("release-1", expect.any(AbortSignal));
  });
  it("ignores a response after logout/account change", async () => {
    const pending = deferred<ReleaseNoticeResult>();
    const old = setup();
    old.api.visit.mockImplementation(() => pending.promise);
    const start = old.controller.start();
    old.controller.stop();
    const next = setup(2);
    await next.controller.start();
    pending.resolve(response());
    await start;
    expect(old.controller.snapshot.notice).toBeNull();
    expect(next.controller.snapshot.notice?.releaseId).toBe("release-1");
  });
  it("does not block the app on load failure and retries on reconnection", async () => {
    const { controller, api } = setup();
    api.visit.mockRejectedValueOnce(new Error("offline"));
    await controller.start();
    expect(controller.snapshot.available).toBe(false);
    await controller.reconnect();
    expect(controller.snapshot.open).toBe(true);
  });
  it("does not display expired server results", async () => {
    const { controller } = setup(1, { serverNow: "2026-10-21T10:00:00Z", notice: null });
    await controller.start();
    expect(controller.snapshot).toEqual({ notice: null, available: false, open: false });
  });
  it("allows closing and retrying when browser storage is unavailable", async () => {
    const bad = {
      getItem: () => {
        throw Error("blocked");
      },
      setItem: () => {
        throw Error("blocked");
      },
      removeItem: () => {
        throw Error("blocked");
      },
    };
    const { controller, api } = setup(1, response(), bad);
    await controller.start();
    controller.close();
    await controller.reconnect();
    expect(controller.snapshot.open).toBe(false);
    expect(api.dismiss).toHaveBeenCalled();
  });
  it("coalesces duplicate close actions", async () => {
    const { controller, api } = setup();
    const request = deferred<void>();
    api.dismiss.mockImplementation(() => request.promise);
    await controller.start();
    controller.close();
    controller.close();
    await Promise.resolve();
    expect(api.dismiss).toHaveBeenCalledTimes(1);
    request.resolve();
  });
  it("expires after device sleep even when the monotonic clock paused", async () => {
    let wall = 1000;
    const api = { visit: vi.fn(async () => response()), dismiss: vi.fn(async () => {}) };
    const controller = new ReleaseNoticeController(
      1,
      api,
      storage(),
      () => 0,
      () => wall
    );
    await controller.start();
    wall += 86400000;
    await controller.reconnect();
    expect(controller.snapshot.available).toBe(false);
    expect(controller.snapshot.open).toBe(true);
  });
  it("aborts the previous account's pending requests on logout", async () => {
    const pending = deferred<ReleaseNoticeResult>();
    let signal: AbortSignal | undefined;
    const api = {
      visit: vi.fn((value?: AbortSignal) => {
        signal = value;
        return pending.promise;
      }),
      dismiss: vi.fn(async () => {}),
    };
    const controller = new ReleaseNoticeController(1, api, storage());
    const start = controller.start();
    controller.stop();
    expect(signal?.aborted).toBe(true);
    pending.resolve(response());
    await start;
    expect(controller.snapshot.notice).toBeNull();
  });
});
