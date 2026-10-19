import type { Row } from "./attendance";

export type AttendanceEntry = { draft: string; saving?: boolean; error?: string };
export type AttendanceSnapshot = {
  rows: Row[];
  entries: Record<number, AttendanceEntry>;
  loading: boolean;
  error?: string;
};
type Save = (
  student: number,
  course: number,
  year: number,
  month: number,
  hours: number
) => Promise<void>;

// A session belongs to one month. Old responses can only update their own session.
export class AttendanceEditor {
  snapshot: AttendanceSnapshot = { rows: [], entries: {}, loading: false };
  private listeners = new Set<() => void>();
  private pending = new Map<number, Promise<boolean>>();
  private generation = 0;
  constructor(
    readonly year: number,
    readonly month: number,
    private save: Save,
    private invalidMessage: () => string,
    private lockedMessage: () => string = () => "Attendance is locked by the invoice"
  ) {}
  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };
  getSnapshot = () => this.snapshot;
  private publish(next: Partial<AttendanceSnapshot>) {
    this.snapshot = { ...this.snapshot, ...next };
    this.listeners.forEach((listener) => listener());
  }
  async load(fetch: (year: number, month: number) => Promise<Row[]>) {
    const generation = ++this.generation;
    this.publish({ loading: true, error: undefined });
    try {
      const rows = await fetch(this.year, this.month);
      if (generation === this.generation) this.publish({ rows });
    } catch (error) {
      if (generation === this.generation)
        this.publish({ error: String(error instanceof Error ? error.message : error) });
    } finally {
      if (generation === this.generation) this.publish({ loading: false });
    }
  }
  value(row: Row) {
    return (
      this.snapshot.entries[row.enrollmentId]?.draft ?? (row.hasRecord ? String(row.hours) : "")
    );
  }
  edit(row: Row, draft: string) {
    if (row.attendanceLocked || this.pending.has(row.enrollmentId)) return;
    this.publish({ entries: { ...this.snapshot.entries, [row.enrollmentId]: { draft } } });
  }
  discard(row: Row) {
    if (this.pending.has(row.enrollmentId)) return;
    const entries = { ...this.snapshot.entries };
    delete entries[row.enrollmentId];
    this.publish({ entries });
  }
  commit(row: Row): Promise<boolean> {
    const id = row.enrollmentId;
    const existing = this.pending.get(id);
    if (existing) return existing;
    const entry = this.snapshot.entries[id];
    if (!entry) return Promise.resolve(true);
    const saved = this.snapshot.rows.find((item) => item.enrollmentId === id) ?? row;
    if (saved.attendanceLocked) {
      this.publish({
        entries: { ...this.snapshot.entries, [id]: { ...entry, error: this.lockedMessage() } },
      });
      return Promise.resolve(false);
    }
    const text = entry.draft.trim().replace(",", ".");
    if (!text) {
      this.discard(row);
      return Promise.resolve(true);
    }
    const hours = Number(text);
    if (
      !/^(?:\d+(?:\.\d*)?|\.\d+)$/.test(text) ||
      !Number.isFinite(hours) ||
      hours < 0 ||
      Math.abs(hours * 4 - Math.round(hours * 4)) > 1e-8
    ) {
      this.publish({
        entries: { ...this.snapshot.entries, [id]: { ...entry, error: this.invalidMessage() } },
      });
      return Promise.resolve(false);
    }
    if (saved.hasRecord && hours === saved.hours) {
      this.discard(row);
      return Promise.resolve(true);
    }
    // Register synchronously, before starting the request or notifying React.
    const operation = Promise.resolve().then(async () => {
      try {
        await this.save(row.studentId, row.courseId, this.year, this.month, hours);
        ++this.generation; // An earlier load must not overwrite this confirmed save.
        const entries = { ...this.snapshot.entries };
        delete entries[id];
        this.publish({
          rows: this.snapshot.rows.map((item) =>
            item.enrollmentId === id ? { ...item, hours, hasRecord: true } : item
          ),
          entries,
          loading: false,
        });
        return true;
      } catch (error) {
        this.publish({
          entries: {
            ...this.snapshot.entries,
            [id]: {
              draft: entry.draft,
              error: String(error instanceof Error ? error.message : error),
            },
          },
        });
        return false;
      } finally {
        this.pending.delete(id);
      }
    });
    this.pending.set(id, operation);
    this.publish({
      entries: { ...this.snapshot.entries, [id]: { draft: entry.draft, saving: true } },
    });
    return operation;
  }
  step(row: Row, delta: number) {
    const text = this.value(row).trim().replace(",", ".");
    const value = text ? Number(text) : 0;
    if (!Number.isFinite(value) || value < 0 || Math.abs(value * 4 - Math.round(value * 4)) > 1e-8)
      return this.commit(row);
    this.edit(row, String(Math.max(0, value + delta)));
    return this.commit(row);
  }
  async flush() {
    const results = await Promise.all(this.snapshot.rows.map((row) => this.commit(row)));
    return results.every(Boolean);
  }
  get dirty() {
    return Object.keys(this.snapshot.entries).length > 0 || this.pending.size > 0;
  }
  async guard(action: () => unknown) {
    if (await this.flush()) await action();
  }
}
