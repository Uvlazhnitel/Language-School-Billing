import { useCallback, useEffect, useMemo, useRef, useSyncExternalStore } from "react";
import { fetchRows, saveHours } from "../../lib/attendance";
import { AttendanceEditor } from "../../lib/attendanceEditor";
import type { TranslateFn } from "../../lib/i18n";

export function useAttendanceController(
  year: number,
  month: number,
  active: boolean,
  t: TranslateFn
) {
  const translation = useRef(t);
  translation.current = t;
  const editor = useMemo(
    () =>
      new AttendanceEditor(
        year,
        month,
        saveHours,
        () => translation.current("attendance.invalid"),
        () => translation.current("attendance.lockedChange")
      ),
    [year, month]
  );
  const snapshot = useSyncExternalStore(editor.subscribe, editor.getSnapshot, editor.getSnapshot);
  const reload = useCallback(async () => {
    await editor.guard(() => editor.load(fetchRows));
  }, [editor]);
  useEffect(() => {
    if (active) void editor.load(fetchRows);
  }, [active, editor]);
  useEffect(() => {
    const beforeUnload = (event: BeforeUnloadEvent) => {
      if (editor.dirty) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", beforeUnload);
    return () => window.removeEventListener("beforeunload", beforeUnload);
  }, [editor]);
  return { editor, snapshot, reload };
}
