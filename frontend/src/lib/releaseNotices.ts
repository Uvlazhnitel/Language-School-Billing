import { getTransport } from "./api";
import type { ReleaseNotice, ReleaseNoticeResult } from "./api/types";
export type NoticeSnapshot = { notice: ReleaseNotice | null; open: boolean; available: boolean };
type NoticeAPI = {
  visit: (signal?: AbortSignal) => Promise<ReleaseNoticeResult>;
  dismiss: (id: string, signal?: AbortSignal) => Promise<void>;
};
type NoticeStorage = Pick<Storage, "getItem" | "setItem" | "removeItem">;
export const releaseNoticeAPI: NoticeAPI = {
  visit: async (signal) => {
    const transport = await getTransport();
    signal?.throwIfAborted();
    return transport.visitReleaseNotice(signal);
  },
  dismiss: async (id, signal) => {
    const transport = await getTransport();
    signal?.throwIfAborted();
    return transport.dismissReleaseNotice(id, signal);
  },
};

// One controller per signed-in account. An existing tab never polls for new
// releases: reconnect only retries a failed initial load or queued dismissals.
export class ReleaseNoticeController {
  snapshot: NoticeSnapshot = { notice: null, open: false, available: false };
  private listeners = new Set<() => void>();
  private generation = 0;
  private requests = new Set<AbortController>();
  private active = false;
  private loaded = false;
  private loading = false;
  private deadline = 0;
  private wallDeadline = 0;
  private closed = new Set<string>();
  private retryQueue = new Set<string>();
  private pending = new Map<string, Promise<void>>();
  private storageKey: string;
  constructor(
    readonly userId: number,
    private api: NoticeAPI,
    private storage: NoticeStorage | null,
    private clock: () => number = () => performance.now(),
    private wallClock: () => number = () => Date.now()
  ) {
    this.storageKey = `release-notices:pending:${userId}`;
    for (const id of this.readPending()) {
      this.closed.add(id);
      this.retryQueue.add(id);
    }
  }
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };
  getSnapshot = () => this.snapshot;
  private publish(next: Partial<NoticeSnapshot>) {
    if (!this.active) return;
    this.snapshot = { ...this.snapshot, ...next };
    this.listeners.forEach((listener) => listener());
  }
  private readPending(): string[] {
    try {
      const value: unknown = JSON.parse(this.storage?.getItem(this.storageKey) ?? "[]");
      return Array.isArray(value) ? value.filter((id): id is string => typeof id === "string") : [];
    } catch {
      return [];
    }
  }
  private writePending(id: string, add: boolean) {
    try {
      const ids = new Set(this.readPending());
      if (add) ids.add(id);
      else ids.delete(id);
      if (ids.size) this.storage?.setItem(this.storageKey, JSON.stringify([...ids]));
      else this.storage?.removeItem(this.storageKey);
    } catch {
      /* Blocked storage must not trap the user in a modal. */
    }
  }
  async start() {
    this.active = true;
    this.loaded = false;
    this.loading = false;
    await this.load();
  }
  stop() {
    this.active = false;
    this.generation++;
    for (const request of this.requests) request.abort();
  }
  private async request<T>(work: (signal: AbortSignal) => Promise<T>): Promise<T> {
    const controller = new AbortController();
    this.requests.add(controller);
    const timeout = setTimeout(() => controller.abort(), 10000);
    try {
      return await work(controller.signal);
    } finally {
      clearTimeout(timeout);
      this.requests.delete(controller);
    }
  }
  private async load() {
    if (this.loading || !this.active) return;
    this.loading = true;
    const generation = ++this.generation;
    try {
      const result = await this.request((signal) => this.api.visit(signal));
      if (!this.active || generation !== this.generation) return;
      this.loaded = true;
      const notice = result.notice;
      const remaining = notice ? Date.parse(notice.expiresAt) - Date.parse(result.serverNow) : 0;
      const ttl = Math.max(0, Number.isFinite(remaining) ? remaining : 0);
      this.deadline = this.clock() + ttl;
      this.wallDeadline = this.wallClock() + ttl;
      const available = Boolean(notice && remaining > 0);
      this.publish({
        notice,
        available,
        open: Boolean(
          available &&
          notice &&
          !notice.dismissedAt &&
          !this.closed.has(notice.releaseId) &&
          !this.readPending().includes(notice.releaseId)
        ),
      });
      await this.retryDismissals();
    } catch {
      /* Announcements never block the application. Reconnect can retry. */
    } finally {
      if (generation === this.generation) this.loading = false;
    }
  }
  tick() {
    if (this.snapshot.available && this.remaining() === 0) this.publish({ available: false });
  }
  remaining() {
    // The wall clock also accounts for device sleep on platforms where the
    // monotonic clock pauses. Moving the wall clock back cannot extend a visit.
    return Math.max(
      0,
      Math.min(this.deadline - this.clock(), this.wallDeadline - this.wallClock())
    );
  }
  open() {
    this.tick();
    if (this.snapshot.available) this.publish({ open: true });
  }
  close() {
    const notice = this.snapshot.notice;
    if (!notice) return;
    this.closed.add(notice.releaseId);
    this.retryQueue.add(notice.releaseId);
    this.writePending(notice.releaseId, true);
    this.publish({ open: false });
    void this.dismiss(notice.releaseId);
  }
  private dismiss(id: string) {
    const existing = this.pending.get(id);
    if (existing) return existing;
    const operation = Promise.resolve().then(async () => {
      try {
        if (!this.active) return;
        await this.request((signal) => this.api.dismiss(id, signal));
        this.writePending(id, false);
        this.retryQueue.delete(id);
      } catch {
        /* Keep the account-scoped queue for the next connection. */
      } finally {
        this.pending.delete(id);
      }
    });
    this.pending.set(id, operation);
    return operation;
  }
  private async retryDismissals() {
    const ids = new Set([...this.readPending(), ...this.retryQueue]);
    await Promise.all([...ids].map((id) => this.dismiss(id)));
  }
  async reconnect() {
    if (!this.active) return;
    this.tick();
    if (!this.loaded) await this.load();
    else await this.retryDismissals();
  }
}
