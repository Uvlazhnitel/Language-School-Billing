import { useEffect, useMemo, useRef, useState } from "react";
import type { AttendanceEditor, AttendanceSnapshot } from "../lib/attendanceEditor";
import {
  courseKey,
  emptySheetFilters,
  filterSheet,
  groupSheet,
  initialSheetGroup,
  monthSummary,
  rowPrice,
  rowTotal,
  teacherKey,
  type SheetFilters,
} from "../lib/attendanceGroups";
import type { TranslateFn } from "../lib/i18n";
import {
  findAttendanceFocusEnrollmentId,
  type AttendanceFocusTarget,
} from "../lib/attendanceFocus";
export type { AttendanceFocusTarget } from "../lib/attendanceFocus";

type Props = {
  editor: AttendanceEditor;
  snapshot: AttendanceSnapshot;
  phones: Map<number, string>;
  onRefresh: () => void;
  onOpenInvoices: () => void;
  onOpenEnrollments: () => void;
  onOpenStudent: (id: number) => unknown;
  focusTarget?: AttendanceFocusTarget | null;
  onFocusTargetHandled?: () => void;
  t: TranslateFn;
  formatEUR: (value: number) => string;
  courseTypeLabel: (type: string) => string;
};

export function AttendanceScreen({
  editor,
  snapshot,
  phones,
  onRefresh,
  onOpenInvoices,
  onOpenEnrollments,
  onOpenStudent,
  focusTarget,
  onFocusTargetHandled,
  t,
  formatEUR,
  courseTypeLabel,
}: Props) {
  const { rows, entries, loading, error } = snapshot;
  const [filters, setFilters] = useState<SheetFilters>(emptySheetFilters);
  const [selection, setSelection] = useState<{ teacher: string; course: string } | null>(null);
  const refs = useRef(new Map<number, HTMLInputElement>());
  const filtered = useMemo(() => filterSheet(rows, filters, phones), [rows, filters, phones]);
  const groups = useMemo(() => groupSheet(filtered), [filtered]);
  const allTeachers = useMemo(() => groupSheet(rows), [rows]);
  const courses = useMemo(
    () =>
      [...new Map(rows.map((row) => [row.courseId, row])).values()].sort((a, b) =>
        a.courseName.localeCompare(b.courseName)
      ),
    [rows]
  );
  const summary = monthSummary(rows);
  const selectedTeacher = groups.find((group) => group.key === selection?.teacher);
  const open =
    !selection || (selection.teacher && !selectedTeacher)
      ? initialSheetGroup(groups)
      : selectedTeacher &&
          selection.course &&
          !selectedTeacher.courses.some((course) => course.key === selection.course)
        ? initialSheetGroup([selectedTeacher])
        : selection;
  useEffect(() => {
    if (selection === null && groups.length > 0) setSelection(initialSheetGroup(groups));
  }, [selection, groups]);
  const change = (patch: Partial<SheetFilters>) =>
    void editor.guard(() => {
      setFilters((prev) => ({ ...prev, ...patch }));
      setSelection(null);
    });

  useEffect(() => {
    if (!focusTarget) return;
    const enrollmentId = findAttendanceFocusEnrollmentId(rows, focusTarget);
    const row = rows.find((item) => item.enrollmentId === enrollmentId);
    if (!row) return;
    const target = { teacher: teacherKey(row), course: courseKey(row) };
    if (
      open.teacher !== target.teacher ||
      open.course !== target.course ||
      !filtered.includes(row)
    ) {
      setFilters(emptySheetFilters);
      setSelection(target);
      return;
    }
    const input = refs.current.get(row.enrollmentId);
    if (!input && !row.attendanceLocked) return;
    input?.focus();
    input?.select();
    input?.scrollIntoView({ block: "center" });
    onFocusTargetHandled?.();
  }, [focusTarget, rows, open.teacher, open.course, filtered, onFocusTargetHandled]);

  const states = ["all", "missing", "filled", "zero", "locked"] as const;
  return (
    <div className="attendanceSheet" aria-busy={loading}>
      <div className="attendanceSummary" aria-label={t("attendance.monthSummary")}>
        {(["missing", "filled", "locked"] as const).map((key) => (
          <div key={key}>
            <span>{t(`attendance.${key}`)}</span>
            <strong>{summary[key]}</strong>
          </div>
        ))}
        <p>{t("attendance.autosave")}</p>
      </div>
      <div className="attendanceToolbar">
        <div className="attendanceToolbarTop">
          <input
            type="search"
            aria-label={t("attendance.search")}
            placeholder={t("attendance.search")}
            value={filters.query}
            onChange={(event) => change({ query: event.target.value })}
          />
          <button className="btnPrimary" onClick={() => void editor.guard(onOpenInvoices)}>
            {t("attendance.invoices")}
          </button>
          <button onClick={() => void editor.guard(onRefresh)}>{t("button.refresh")}</button>
          <button onClick={() => void editor.guard(onOpenEnrollments)}>
            {t("tabs.enrollments")}
          </button>
        </div>
        <div className="attendanceToolbarFilters">
          <div className="attendanceStates" aria-label={t("attendance.status")}>
            {states.map((status) => (
              <button
                key={status}
                type="button"
                aria-pressed={filters.status === status}
                onClick={() => change({ status })}
              >
                {t(`attendance.filter.${status}`)}
              </button>
            ))}
          </div>
          <select
            aria-label={t("attendance.teacher")}
            value={filters.teacher}
            onChange={(event) => change({ teacher: event.target.value })}
          >
            <option value="">{t("attendance.allTeachers")}</option>
            {allTeachers.map((teacher) => (
              <option key={teacher.key} value={teacher.key}>
                {teacher.name || t("attendance.unassigned")}
              </option>
            ))}
          </select>
          <select
            aria-label={t("attendance.course")}
            value={filters.course}
            onChange={(event) => change({ course: event.target.value })}
          >
            <option value="">{t("attendance.allCourses")}</option>
            {courses.map((row) => (
              <option key={row.courseId} value={row.courseId}>
                {row.courseName} · {courseTypeLabel(row.courseType)}
              </option>
            ))}
          </select>
          <select
            aria-label={t("attendance.billing")}
            value={filters.billing}
            onChange={(event) => change({ billing: event.target.value })}
          >
            <option value="">{t("attendance.allBilling")}</option>
            <option value="per_lesson">{t("attendance.hourly")}</option>
            <option value="subscription">{t("attendance.subscription")}</option>
          </select>
        </div>
      </div>
      {loading && <p role="status">{t("label.loading")}</p>}
      {error && (
        <p role="alert">
          {error} <button onClick={onRefresh}>{t("attendance.retry")}</button>
        </p>
      )}
      <div className="attendanceResults" role="status">
        <span>{t("attendance.found", { count: filtered.length })}</span>
        <span>
          {t("attendance.total")}:{" "}
          <strong>
            {formatEUR(
              filtered.reduce((sum, row) => sum + Math.round(rowTotal(row) * 100), 0) / 100
            )}
          </strong>
        </span>
      </div>
      {!loading && !error && groups.length === 0 && (
        <div className="attendanceEmpty">
          <p>{t(rows.length ? "attendance.noResults" : "attendance.empty")}</p>
          {rows.length > 0 && (
            <button onClick={() => change(emptySheetFilters)}>{t("attendance.reset")}</button>
          )}
        </div>
      )}
      {groups.map((teacher) => (
        <section className="attendanceTeacher" key={teacher.key}>
          <h2>
            <button
              className="attendanceTeacherToggle"
              aria-expanded={open.teacher === teacher.key}
              aria-controls={`teacher-${teacher.key}`}
              onClick={() =>
                void editor.guard(() =>
                  setSelection(
                    open.teacher === teacher.key
                      ? { teacher: "", course: "" }
                      : { teacher: teacher.key, course: initialSheetGroup([teacher]).course }
                  )
                )
              }
            >
              <span aria-hidden="true">{open.teacher === teacher.key ? "▾" : "▸"}</span>
              <span>{teacher.name || t("attendance.unassigned")}</span>
              <small>{t("attendance.toFill", { count: teacher.missing })}</small>
            </button>
          </h2>
          {open.teacher === teacher.key && (
            <div id={`teacher-${teacher.key}`} className="attendanceCourses">
              {teacher.courses.map((course) => {
                const first = course.rows[0];
                const subscription = first.billingMode === "subscription";
                const quantityLabel = t(subscription ? "attendance.lessons" : "attendance.hours");
                return (
                  <section className="attendanceCourse" key={course.key}>
                    <h3>
                      <button
                        className="attendanceCourseToggle"
                        aria-expanded={open.course === course.key}
                        aria-controls={`course-${course.key}`}
                        onClick={() =>
                          void editor.guard(() =>
                            setSelection({
                              teacher: teacher.key,
                              course: open.course === course.key ? "" : course.key,
                            })
                          )
                        }
                      >
                        <span aria-hidden="true">{open.course === course.key ? "▾" : "▸"}</span>
                        <span>
                          {first.courseName}{" "}
                          <small>
                            {courseTypeLabel(first.courseType)} ·{" "}
                            {t(subscription ? "attendance.subscription" : "attendance.hourly")}
                          </small>
                        </span>
                        <small>{t("attendance.toFill", { count: course.missing })}</small>
                      </button>
                    </h3>
                    {open.course === course.key && (
                      <div
                        id={`course-${course.key}`}
                        className="attendanceGrid"
                        role="table"
                        aria-label={first.courseName}
                      >
                        <div className="attendanceGridHead" role="row">
                          <span role="columnheader">{t("attendance.student")}</span>
                          <span role="columnheader">
                            {t(subscription ? "attendance.lessonPrice" : "attendance.hourPrice")}
                          </span>
                          <span role="columnheader">{quantityLabel}</span>
                          <span role="columnheader">{t("attendance.total")}</span>
                        </div>
                        {course.rows.map((row) => {
                          const entry = entries[row.enrollmentId];
                          const errorId = `attendance-error-${row.enrollmentId}`;
                          return (
                            <div className="attendanceRow" key={row.enrollmentId} role="row">
                              <div role="cell">
                                <button
                                  className="attendanceStudent"
                                  onClick={() =>
                                    void editor.guard(() => onOpenStudent(row.studentId))
                                  }
                                >
                                  {row.studentName}
                                </button>
                              </div>
                              <div className="attendancePrice" role="cell">
                                <span className="attendanceMobileLabel">
                                  {t(
                                    subscription ? "attendance.lessonPrice" : "attendance.hourPrice"
                                  )}
                                </span>
                                {formatEUR(rowPrice(row))}
                              </div>
                              <div role="cell" className="attendanceQuantity">
                                <span className="attendanceMobileLabel">{quantityLabel}</span>
                                {row.attendanceLocked ? (
                                  <div className="attendanceLocked">
                                    <strong>{row.hasRecord ? row.hours : "—"}</strong>
                                    <small>
                                      {t(
                                        row.invoiceStatus === "canceled"
                                          ? "attendance.lockCanceled"
                                          : row.invoiceStatus?.startsWith("paid")
                                            ? "attendance.lockPaid"
                                            : "attendance.lockIssued"
                                      )}
                                    </small>
                                  </div>
                                ) : (
                                  <>
                                    <div className="attendanceStepper">
                                      <button
                                        aria-label={`${t("attendance.decrease")} · ${row.studentName}`}
                                        disabled={entry?.saving}
                                        onMouseDown={(event) => event.preventDefault()}
                                        onClick={() => void editor.step(row, -0.25)}
                                      >
                                        −
                                      </button>
                                      <input
                                        ref={(element) => {
                                          if (element) refs.current.set(row.enrollmentId, element);
                                          else refs.current.delete(row.enrollmentId);
                                        }}
                                        type="text"
                                        inputMode="decimal"
                                        aria-label={`${quantityLabel} · ${row.studentName}`}
                                        aria-invalid={Boolean(entry?.error)}
                                        aria-describedby={entry?.error ? errorId : undefined}
                                        placeholder={t("attendance.placeholder")}
                                        value={editor.value(row)}
                                        disabled={entry?.saving}
                                        onChange={(event) => editor.edit(row, event.target.value)}
                                        onBlur={() => {
                                          if (
                                            !editor.getSnapshot().entries[row.enrollmentId]?.error
                                          )
                                            void editor.commit(row);
                                        }}
                                        onFocus={(event) => event.target.select()}
                                        onKeyDown={(event) => {
                                          if (event.key === "Enter") {
                                            event.preventDefault();
                                            void editor.commit(row);
                                          }
                                          if (event.key === "Escape") {
                                            event.preventDefault();
                                            editor.discard(row);
                                            event.currentTarget.blur();
                                          }
                                        }}
                                      />
                                      <button
                                        aria-label={`${t("attendance.increase")} · ${row.studentName}`}
                                        disabled={entry?.saving}
                                        onMouseDown={(event) => event.preventDefault()}
                                        onClick={() => void editor.step(row, 0.25)}
                                      >
                                        +
                                      </button>
                                    </div>
                                    <div className="attendanceSaveState" aria-live="polite">
                                      {entry?.saving ? (
                                        t("attendance.saving")
                                      ) : entry?.error ? null : entry ? (
                                        t("attendance.unsaved")
                                      ) : row.hasRecord ? (
                                        t("attendance.saved")
                                      ) : (
                                        <button
                                          aria-label={`${t("attendance.noLessons")} · ${row.studentName}`}
                                          onMouseDown={(event) => event.preventDefault()}
                                          onClick={() => {
                                            editor.edit(row, "0");
                                            void editor.commit(row);
                                          }}
                                        >
                                          {t("attendance.noLessons")}
                                        </button>
                                      )}
                                    </div>
                                  </>
                                )}
                                {entry?.error && (
                                  <div className="attendanceError" id={errorId} role="alert">
                                    <span>{entry.error}</span>
                                    <button onClick={() => void editor.commit(row)}>
                                      {t("attendance.retry")}
                                    </button>
                                    <button onClick={() => editor.discard(row)}>
                                      {t("attendance.discard")}
                                    </button>
                                  </div>
                                )}
                              </div>
                              <div className="attendanceAmount" role="cell">
                                <span className="attendanceMobileLabel">
                                  {t("attendance.total")}
                                </span>
                                {formatEUR(rowTotal(row))}
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </section>
                );
              })}
            </div>
          )}
        </section>
      ))}
    </div>
  );
}
