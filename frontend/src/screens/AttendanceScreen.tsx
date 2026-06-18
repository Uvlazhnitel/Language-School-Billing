import type { MutableRefObject } from "react";
import {
  BillingModePerLesson,
  BillingModeSubscription,
  InvoiceStatusCanceled,
  InvoiceStatusIssuedPendingPdf,
  InvoiceStatusIssued,
  InvoiceStatusPaidPendingPdf,
  InvoiceStatusPaid,
} from "../lib/constants";
import type { Row } from "../lib/attendance";
import type { CourseDTO } from "../lib/courses";
import type { TranslateFn } from "../lib/i18n";
import { EmptyState } from "../components/EmptyState";
import { FilterToolbar } from "../components/FilterToolbar";

type AttendanceFilter = "all" | "missing" | "filled" | "zero";

type AttendanceScreenProps = {
  attendanceSummary: { missing: number; total: number; filled: number; zero: number };
  courseFilter?: number;
  allCourses: CourseDTO[];
  query: string;
  filter: AttendanceFilter;
  rows: Row[];
  filteredRows: Row[];
  loading: boolean;
  attendanceSavingRows: Record<number, boolean>;
  attendancePendingSelectRef: MutableRefObject<number | null>;
  subscriptionLeadEnrollmentIds: Set<number>;
  subscriptionMonthLessons: Record<number, number>;
  subscriptionMonthSaving: Record<number, boolean>;
  year: number;
  month: number;
  perLessonTotal: number;
  courseTypeLabel: (type: string) => string;
  formatEUR: (value: number) => string;
  normalizeHoursDraftInput: (value: string) => string | null;
  getAttendanceStepBase: (row: Row) => number;
  getAttendanceInputValue: (row: Row) => string;
  getSubscriptionMonthLessonsValue: (courseId: number) => string;
  setAttendanceDraft: (enrollmentId: number, value: string) => void;
  clearAttendanceDraft: (enrollmentId: number) => void;
  commitAttendanceDraft: (row: Row) => void | Promise<void>;
  onChangeHours: (row: Row, nextBase: number) => void;
  setSubscriptionMonthLessonsDraft: (courseId: number, value: string) => void;
  clearSubscriptionMonthLessonsDraft: (courseId: number) => void;
  commitSubscriptionMonthLessonsDraft: (row: Row) => void | Promise<void>;
  onAdjustSubscriptionLessons: (courseId: number, nextValue: number) => void | Promise<void>;
  onRefresh: () => void;
  onOpenInvoices: () => void;
  onOpenEnrollments: () => void;
  onCourseFilterChange: (value: number | undefined) => void;
  onQueryChange: (value: string) => void;
  onFilterChange: (value: AttendanceFilter) => void;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onDeleteEnrollmentFromSheet: (
    enrollmentId: number,
    enrollmentVersion: number
  ) => void | Promise<void>;
  t: TranslateFn;
};

export function AttendanceScreen({
  attendanceSummary,
  courseFilter,
  allCourses,
  query,
  filter,
  rows,
  filteredRows,
  loading,
  attendanceSavingRows,
  attendancePendingSelectRef,
  subscriptionLeadEnrollmentIds,
  subscriptionMonthLessons,
  subscriptionMonthSaving,
  year: _year,
  month: _month,
  perLessonTotal,
  courseTypeLabel,
  formatEUR,
  normalizeHoursDraftInput,
  getAttendanceStepBase,
  getAttendanceInputValue,
  getSubscriptionMonthLessonsValue,
  setAttendanceDraft,
  clearAttendanceDraft,
  commitAttendanceDraft,
  onChangeHours,
  setSubscriptionMonthLessonsDraft,
  clearSubscriptionMonthLessonsDraft,
  commitSubscriptionMonthLessonsDraft,
  onAdjustSubscriptionLessons,
  onRefresh,
  onOpenInvoices,
  onOpenEnrollments,
  onCourseFilterChange,
  onQueryChange,
  onFilterChange,
  onOpenStudent,
  onDeleteEnrollmentFromSheet,
  t,
}: AttendanceScreenProps) {
  const hasActiveFilters = Boolean(query.trim() || filter !== "all" || courseFilter);

  function courseOptionLabel(course: CourseDTO): string {
    const typeLabel = courseTypeLabel(course.type);
    return course.teacherName
      ? `${course.name} — ${typeLabel} — ${course.teacherName}`
      : `${course.name} — ${typeLabel}`;
  }

  return (
    <>
      <div className="sectionBanner">
        <div>
          <div className="dashboardCardEyebrow">{t("msg.monthStatus")}</div>
          <strong>
            {attendanceSummary.missing > 0
              ? t("msg.monthStatusMissing", { count: attendanceSummary.missing })
              : attendanceSummary.total > 0
                ? t("msg.monthStatusDone")
                : t("msg.monthStatusEmpty")}
          </strong>
        </div>
      </div>

      <FilterToolbar
        search={
          <input
            className="searchField"
            placeholder={t("msg.searchPlaceholderAttendance")}
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
          />
        }
        filters={
          <>
            <select
              value={courseFilter ?? ""}
              onChange={(e) =>
                onCourseFilterChange(e.target.value ? parseInt(e.target.value, 10) : undefined)
              }
            >
              <option value="">{t("filter.allGroups")}</option>
              {allCourses.map((course) => (
                <option key={course.id} value={course.id}>
                  {courseOptionLabel(course)}
                </option>
              ))}
            </select>
            <select
              value={filter}
              onChange={(e) => onFilterChange(e.target.value as AttendanceFilter)}
            >
              <option value="all">{t("status.showAll")}</option>
              <option value="missing">{t("status.onlyMissing")}</option>
              <option value="filled">{t("status.onlyFilled")}</option>
              <option value="zero">{t("status.zeroLessons")}</option>
            </select>
          </>
        }
        hasActiveFilters={hasActiveFilters}
        onClearFilters={() => {
          onCourseFilterChange(undefined);
          onQueryChange("");
          onFilterChange("all");
        }}
        clearLabel={t("button.clearFilters")}
        secondaryActions={
          <>
            <button className="workspaceActionButton" onClick={onRefresh}>
              {t("msg.refreshSheet")}
            </button>
            <button
              className="workspaceActionButton workspaceActionButtonPrimary"
              onClick={onOpenInvoices}
              disabled={attendanceSummary.total === 0}
            >
              {t("msg.openMonthInvoices")}
            </button>
          </>
        }
      />

      {rows.length > 0 && (
        <div className="attSummary">
          {t("msg.attFilled")}: {attendanceSummary.filled} / {attendanceSummary.total}
          &nbsp;·&nbsp;{t("msg.attMissing")}: {attendanceSummary.missing}
          &nbsp;·&nbsp;{t("msg.attZero")}: {attendanceSummary.zero}
        </div>
      )}

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : filteredRows.length === 0 ? (
        hasActiveFilters ? (
          <EmptyState
            title={t("msg.noAttendanceSearchTitle")}
            description={t("msg.noAttendanceSearchDescription")}
            actionLabel={t("button.clearFilters")}
            onAction={() => {
              onCourseFilterChange(undefined);
              onQueryChange("");
              onFilterChange("all");
            }}
          />
        ) : (
          <EmptyState
            title={t("msg.noAttendanceTitle")}
            description={t("msg.noAttendanceDescription")}
            actionLabel={t("button.openEnrollments")}
            onAction={onOpenEnrollments}
          />
        )
      ) : (
        <table>
          <thead>
            <tr>
              <th>{t("field.student")}</th>
              <th>{t("field.course")}</th>
              <th style={{ textAlign: "right" }}>{t("field.lessonPrice")} (EUR)</th>
              <th style={{ textAlign: "right" }}>{t("field.quantity")}</th>
              <th style={{ textAlign: "right" }}>{t("field.totalEur")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {filteredRows.map((row) => (
              <tr key={row.enrollmentId}>
                <td>
                  <button className="linkButton" onClick={() => void onOpenStudent(row.studentId)}>
                    {row.studentName}
                  </button>
                </td>
                <td>
                  {row.courseName} ({courseTypeLabel(row.courseType)})
                  {row.billingMode === BillingModeSubscription && (
                    <>
                      {" "}
                      <span className="attBadge attBadge--subscription">
                        {t("billing.subscription")}
                      </span>
                    </>
                  )}
                </td>
                <td style={{ textAlign: "right" }}>{formatEUR(row.lessonPrice)}</td>
                <td style={{ textAlign: "right" }}>
                  {row.billingMode === BillingModePerLesson && !row.hasRecord && (
                    <span className="attBadge attBadge--missing">{t("msg.attMissing")}</span>
                  )}
                  {row.billingMode === BillingModePerLesson && row.hasRecord && row.hours === 0 && (
                    <span className="attBadge attBadge--zero">0h</span>
                  )}
                  {row.billingMode === BillingModePerLesson && !row.attendanceLocked ? (
                    <div className="attendanceStepper">
                      <button
                        type="button"
                        className="attendanceStepperButton"
                        onClick={() =>
                          onChangeHours(row, Math.max(0, getAttendanceStepBase(row) - 1))
                        }
                        disabled={
                          attendanceSavingRows[row.enrollmentId] || getAttendanceStepBase(row) <= 0
                        }
                        aria-label={`Decrease hours for ${row.studentName}`}
                      >
                        −
                      </button>
                      <input
                        type="text"
                        inputMode="decimal"
                        value={getAttendanceInputValue(row)}
                        disabled={attendanceSavingRows[row.enrollmentId]}
                        onChange={(e) => {
                          const nextValue = normalizeHoursDraftInput(e.target.value);
                          if (nextValue !== null) {
                            setAttendanceDraft(row.enrollmentId, nextValue);
                          }
                        }}
                        onPointerDown={() => {
                          attendancePendingSelectRef.current = row.enrollmentId;
                        }}
                        onFocus={(e) => {
                          if (attendancePendingSelectRef.current !== row.enrollmentId) {
                            e.currentTarget.select();
                          }
                        }}
                        onMouseUp={(e) => {
                          if (attendancePendingSelectRef.current === row.enrollmentId) {
                            e.preventDefault();
                            e.currentTarget.select();
                            attendancePendingSelectRef.current = null;
                          }
                        }}
                        onBlur={() => {
                          if (attendancePendingSelectRef.current === row.enrollmentId) {
                            attendancePendingSelectRef.current = null;
                          }
                          void commitAttendanceDraft(row);
                        }}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            void commitAttendanceDraft(row);
                          }
                          if (e.key === "Escape") {
                            e.preventDefault();
                            clearAttendanceDraft(row.enrollmentId);
                            e.currentTarget.blur();
                          }
                        }}
                        className="attendanceStepperInput"
                        aria-label={`Hours for ${row.studentName}`}
                      />
                      <button
                        type="button"
                        className="attendanceStepperButton"
                        onClick={() => onChangeHours(row, getAttendanceStepBase(row) + 1)}
                        disabled={attendanceSavingRows[row.enrollmentId]}
                        aria-label={`Increase hours for ${row.studentName}`}
                      >
                        +
                      </button>
                    </div>
                  ) : row.billingMode === BillingModeSubscription && !row.attendanceLocked ? (
                    subscriptionLeadEnrollmentIds.has(row.enrollmentId) ? (
                      <div className="attendanceStepper">
                        <button
                          type="button"
                          className="attendanceStepperButton"
                          onClick={() =>
                            void onAdjustSubscriptionLessons(
                              row.courseId,
                              Math.max(0, (subscriptionMonthLessons[row.courseId] ?? 0) - 1)
                            )
                          }
                          disabled={
                            subscriptionMonthSaving[row.courseId] ||
                            (subscriptionMonthLessons[row.courseId] ?? 0) <= 0
                          }
                          aria-label={`Decrease subscription lessons for ${row.courseName}`}
                        >
                          −
                        </button>
                        <input
                          type="text"
                          inputMode="decimal"
                          value={getSubscriptionMonthLessonsValue(row.courseId)}
                          disabled={subscriptionMonthSaving[row.courseId]}
                          onChange={(e) => {
                            const nextValue = normalizeHoursDraftInput(e.target.value);
                            if (nextValue !== null) {
                              setSubscriptionMonthLessonsDraft(row.courseId, nextValue);
                            }
                          }}
                          onFocus={(e) => e.currentTarget.select()}
                          onBlur={() => {
                            void commitSubscriptionMonthLessonsDraft(row);
                          }}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              e.preventDefault();
                              void commitSubscriptionMonthLessonsDraft(row);
                            }
                            if (e.key === "Escape") {
                              e.preventDefault();
                              clearSubscriptionMonthLessonsDraft(row.courseId);
                              e.currentTarget.blur();
                            }
                          }}
                          className="attendanceStepperInput"
                          aria-label={`Subscription lessons held for ${row.courseName}`}
                        />
                        <button
                          type="button"
                          className="attendanceStepperButton"
                          onClick={() =>
                            void onAdjustSubscriptionLessons(
                              row.courseId,
                              (subscriptionMonthLessons[row.courseId] ?? 0) + 1
                            )
                          }
                          disabled={subscriptionMonthSaving[row.courseId]}
                          aria-label={`Increase subscription lessons for ${row.courseName}`}
                        >
                          +
                        </button>
                      </div>
                    ) : (
                      <div className="attendanceReadOnly">
                        <span className="attBadge attBadge--subscription">{t("msg.readOnly")}</span>
                        <span className="mutedInline">{t("msg.subscriptionSharedValue")}</span>
                      </div>
                    )
                  ) : (
                    <div className="attendanceReadOnly">
                      <span className="attBadge attBadge--subscription">{t("msg.readOnly")}</span>
                      <span className="mutedInline">
                        {row.invoiceStatus === InvoiceStatusIssued ||
                        row.invoiceStatus === InvoiceStatusIssuedPendingPdf
                          ? t("msg.lockedIssuedInvoice")
                          : row.invoiceStatus === InvoiceStatusPaid ||
                              row.invoiceStatus === InvoiceStatusPaidPendingPdf
                            ? t("msg.lockedPaidInvoice")
                            : row.invoiceStatus === InvoiceStatusCanceled
                              ? t("msg.lockedCanceledInvoice")
                              : t("msg.lockedUntilDraft")}
                      </span>
                    </div>
                  )}
                </td>
                <td style={{ textAlign: "right" }}>
                  {row.billingMode === BillingModePerLesson
                    ? formatEUR(row.hours * row.lessonPrice)
                    : formatEUR(row.subscriptionLessonPrice * (subscriptionMonthLessons[row.courseId] ?? 0))}
                </td>
                <td>
                  {row.billingMode === BillingModePerLesson &&
                    !row.attendanceLocked &&
                    !row.hasRecord && (
                      <button
                        onClick={() => onChangeHours(row, 0)}
                        disabled={attendanceSavingRows[row.enrollmentId]}
                        style={{ marginRight: "0.5rem" }}
                      >
                        {t("msg.setZeroHours")}
                      </button>
                    )}
                  {row.canDelete ? (
                    <button
                      onClick={() =>
                        void onDeleteEnrollmentFromSheet(row.enrollmentId, row.enrollmentVersion)
                      }
                    >
                      {t("msg.deleteEnrollment")}
                    </button>
                  ) : (
                    <span className="mutedInline">{t("msg.deleteEnrollmentBlocked")}</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <td colSpan={4} style={{ textAlign: "right" }}>
                {t("msg.lessonsTotalEur")}:
              </td>
              <td style={{ textAlign: "right" }}>{formatEUR(perLessonTotal)}</td>
              <td></td>
            </tr>
          </tfoot>
        </table>
      )}
    </>
  );
}
