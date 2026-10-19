import { describe, expect, it, vi } from "vitest";
import { AttendanceEditor } from "./attendanceEditor";
import { attendanceRow } from "./attendanceFixture";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
async function setup() {
  const save = vi.fn(async () => {});
  const row = attendanceRow();
  const editor = new AttendanceEditor(2026, 9, save, () => "step 0.25");
  await editor.load(async () => [row]);
  return { editor, row, save };
}
describe("attendance editor", () => {
  it("confirms an explicit zero but leaves blank missing and restores cleared saved values", async () => {
    const { editor, row, save } = await setup();
    expect(editor.value(row)).toBe("");
    editor.edit(row, "");
    await editor.commit(row);
    expect(save).not.toHaveBeenCalled();
    editor.edit(row, "0");
    expect(await editor.commit(row)).toBe(true);
    expect(save).toHaveBeenCalledWith(1, 10, 2026, 9, 0);
    const saved = editor.snapshot.rows[0];
    expect(saved.hasRecord).toBe(true);
    editor.edit(saved, "");
    await editor.commit(saved);
    expect(editor.value(saved)).toBe("0");
    expect(save).toHaveBeenCalledTimes(1);
  });
  it("coalesces Enter, blur, repeated clicks and navigation into one pending request", async () => {
    const { editor, row, save } = await setup();
    const pending = deferred<void>();
    save.mockImplementation(() => pending.promise);
    editor.edit(row, "1,25");
    const first = editor.commit(row);
    expect(editor.commit(row)).toBe(first);
    const navigate = vi.fn();
    const guard = editor.guard(navigate);
    editor.edit(row, "3");
    editor.discard(row);
    await Promise.resolve();
    expect(save).toHaveBeenCalledTimes(1);
    expect(navigate).not.toHaveBeenCalled();
    expect(editor.value(row)).toBe("1,25");
    pending.resolve();
    await guard;
    expect(navigate).toHaveBeenCalledTimes(1);
    expect(editor.snapshot.rows[0].hours).toBe(1.25);
    expect(editor.dirty).toBe(false);
  });
  it("retains failed input and stays on page; retry succeeds, discard cancels", async () => {
    const { editor, row, save } = await setup();
    save.mockRejectedValue(new Error("offline"));
    editor.edit(row, "4");
    const navigate = vi.fn();
    await editor.guard(navigate);
    expect(navigate).not.toHaveBeenCalled();
    expect(editor.value(row)).toBe("4");
    expect(editor.snapshot.entries[1].error).toBe("offline");
    expect(editor.dirty).toBe(true);
    save.mockResolvedValue();
    await editor.commit(row);
    expect(editor.snapshot.rows[0].hours).toBe(4);
    editor.edit(row, "6");
    editor.discard(row);
    await editor.commit(row);
    expect(save).toHaveBeenCalledTimes(2);
    expect(editor.dirty).toBe(false);
  });
  it("preserves invalid quarter-step input without rounding, including subscriptions", async () => {
    const { editor, row, save } = await setup();
    for (const value of ["1.1", "-1", "hello", "1e2", "Infinity"]) {
      editor.edit(row, value);
      expect(await editor.commit({ ...row, billingMode: "subscription" })).toBe(false);
      expect(editor.value(row)).toBe(value);
      expect(editor.snapshot.entries[1].error).toBe("step 0.25");
    }
    expect(save).not.toHaveBeenCalled();
    editor.edit(row, "0.25");
    await editor.commit({ ...row, billingMode: "subscription" });
    expect(save).toHaveBeenCalledWith(1, 10, 2026, 9, 0.25);
  });
  it("ignores out-of-order loads and loads that precede a confirmed save", async () => {
    const { editor, row } = await setup();
    const old = deferred<ReturnType<typeof attendanceRow>[]>();
    const load = editor.load(() => old.promise);
    await editor.load(async () => [{ ...row, studentName: "Latest" }]);
    old.resolve([{ ...row, studentName: "Old" }]);
    await load;
    expect(editor.snapshot.rows[0].studentName).toBe("Latest");
    const slow = deferred<ReturnType<typeof attendanceRow>[]>();
    const stale = editor.load(() => slow.promise);
    editor.edit(row, "2");
    await editor.commit(row);
    slow.resolve([row]);
    await stale;
    expect(editor.snapshot.rows[0].hours).toBe(2);
  });
  it("isolates previous month save responses from the current month", async () => {
    const { editor, row, save } = await setup();
    const pending = deferred<void>();
    save.mockImplementation(() => pending.promise);
    editor.edit(row, "5");
    const operation = editor.commit(row);
    const next = new AttendanceEditor(2026, 10, save, () => "invalid");
    await next.load(async () => [row]);
    pending.resolve();
    await operation;
    expect(next.snapshot.rows[0].hasRecord).toBe(false);
    expect(editor.snapshot.rows[0].hours).toBe(5);
  });
  it("blocks navigation when an invoice locks an edited row, until the edit is discarded", async () => {
    const { editor, row, save } = await setup();
    editor.edit(row, "3");
    await editor.load(async () => [{ ...row, attendanceLocked: true }]);
    const navigate = vi.fn();
    await editor.guard(navigate);
    expect(navigate).not.toHaveBeenCalled();
    expect(save).not.toHaveBeenCalled();
    expect(editor.snapshot.entries[1].error).toBeTruthy();
    editor.discard(row);
    await editor.guard(navigate);
    expect(navigate).toHaveBeenCalledOnce();
  });
});
