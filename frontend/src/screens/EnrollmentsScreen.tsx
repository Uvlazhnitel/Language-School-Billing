import { useEffect, useRef, useState, type Ref, type RefObject } from "react";
import { EnrollmentFormModal } from "../components/modals/EnrollmentFormModal";
import { EmptyState } from "../components/EmptyState";
import type { CourseDTO } from "../lib/courses";
import type { EnrollmentDTO } from "../lib/enrollments";
import type { StudentDTO } from "../lib/students";
import type { TranslateFn } from "../lib/i18n";
import { FilterToolbar } from "../components/FilterToolbar";

type EnrollmentsScreenProps = {
  loading: boolean;
  enrollments: EnrollmentDTO[];
  studentFilter?: number;
  courseFilter?: number;
  allStudents: StudentDTO[];
  allCourses: CourseDTO[];
  enrollmentCourseOptions: CourseDTO[];
  billingModeLabel: (mode: string) => string;
  courseTypeLabel: (type: string) => string;
  onStudentFilterChange: (value: number | undefined) => void;
  onCourseFilterChange: (value: number | undefined) => void;
  onAddEnrollment: () => void;
  onOpenStudents: () => void;
  onOpenCourses: () => void;
  onOpenStudent: (studentId: number) => void | Promise<void>;
  onEditEnrollment: (enrollment: EnrollmentDTO) => void;
  enrollmentModalOpen: boolean;
  editingEnrollment: EnrollmentDTO | null;
  studentSearch: string;
  studentId?: number;
  studentPickerOpen: boolean;
  filteredStudents: StudentDTO[];
  selectedStudent?: StudentDTO | null;
  enrollmentStudentLocked: boolean;
  enrollmentSaveLabel?: string;
  enrollmentCourseId: number;
  enrollmentMode: "per_lesson" | "subscription";
  enrollmentChargeMaterials: boolean;
  enrollmentLessonPriceOverride: string;
  enrollmentSubscriptionLessonPrice: string;
  enrollmentNote: string;
  studentComboRef: RefObject<HTMLDivElement | null>;
  onStudentSearchChange: (value: string) => void;
  onStudentIdChange: (value: number | undefined) => void;
  onStudentPickerOpenChange: (value: boolean) => void;
  onEnrollmentCourseIdChange: (value: number) => void;
  onEnrollmentModeChange: (value: "per_lesson" | "subscription") => void;
  onEnrollmentChargeMaterialsChange: (value: boolean) => void;
  onEnrollmentLessonPriceOverrideChange: (value: string) => void;
  onEnrollmentSubscriptionLessonPriceChange: (value: string) => void;
  onEnrollmentNoteChange: (value: string) => void;
  onSaveEnrollment: () => void;
  onCloseEnrollmentModal: () => void;
  t: TranslateFn;
};

type EnrollmentGroup = {
  courseId: number;
  courseName: string;
  courseType: string;
  teacherName: string;
  enrollments: EnrollmentDTO[];
};

export function EnrollmentsScreen({
  loading,
  enrollments,
  studentFilter,
  courseFilter,
  allStudents,
  allCourses,
  enrollmentCourseOptions,
  billingModeLabel,
  courseTypeLabel,
  onStudentFilterChange,
  onCourseFilterChange,
  onAddEnrollment,
  onOpenStudents,
  onOpenCourses,
  onOpenStudent,
  onEditEnrollment,
  enrollmentModalOpen,
  editingEnrollment,
  studentSearch,
  studentId,
  studentPickerOpen,
  filteredStudents,
  selectedStudent,
  enrollmentStudentLocked,
  enrollmentSaveLabel,
  enrollmentCourseId,
  enrollmentMode,
  enrollmentChargeMaterials,
  enrollmentLessonPriceOverride,
  enrollmentSubscriptionLessonPrice,
  enrollmentNote,
  studentComboRef,
  onStudentSearchChange,
  onStudentIdChange,
  onStudentPickerOpenChange,
  onEnrollmentCourseIdChange,
  onEnrollmentModeChange,
  onEnrollmentChargeMaterialsChange,
  onEnrollmentLessonPriceOverrideChange,
  onEnrollmentSubscriptionLessonPriceChange,
  onEnrollmentNoteChange,
  onSaveEnrollment,
  onCloseEnrollmentModal,
  t,
}: EnrollmentsScreenProps) {
  const [query, setQuery] = useState("");
  const normalizedQuery = query.trim().toLocaleLowerCase();
  const visibleEnrollments = normalizedQuery
    ? enrollments.filter((enrollment) =>
        enrollment.studentName.toLocaleLowerCase().includes(normalizedQuery)
      )
    : enrollments;
  const hasActiveFilters =
    normalizedQuery !== "" || studentFilter !== undefined || courseFilter !== undefined;
  const groupedEnrollments = new Map<number, EnrollmentGroup>();
  for (const enrollment of visibleEnrollments) {
    const existing = groupedEnrollments.get(enrollment.courseId);
    if (existing) {
      existing.enrollments.push(enrollment);
      continue;
    }
    groupedEnrollments.set(enrollment.courseId, {
      courseId: enrollment.courseId,
      courseName: enrollment.courseName,
      courseType: enrollment.courseType,
      teacherName: enrollment.teacherName,
      enrollments: [enrollment],
    });
  }
  const enrollmentGroups = Array.from(groupedEnrollments.values()).sort((left, right) =>
    left.courseName.localeCompare(right.courseName)
  );
  for (const group of enrollmentGroups) {
    group.enrollments.sort((left, right) => left.studentName.localeCompare(right.studentName));
  }
  const visibleCourseIdsKey = enrollmentGroups.map((group) => group.courseId).join(",");
  const [expandedCourseIds, setExpandedCourseIds] = useState<Set<number>>(
    () =>
      new Set(
        courseFilter !== undefined || studentFilter !== undefined
          ? enrollmentGroups.map((group) => group.courseId)
          : enrollmentGroups[0]
            ? [enrollmentGroups[0].courseId]
            : []
      )
  );
  const initializedExpansion = useRef(visibleEnrollments.length > 0);

  useEffect(() => {
    const visibleCourseIds = visibleCourseIdsKey.split(",").filter(Boolean).map(Number);
    if (visibleCourseIds.length === 0) return;

    if (normalizedQuery !== "" || courseFilter !== undefined || studentFilter !== undefined) {
      setExpandedCourseIds((current) => {
        const next = new Set(current);
        for (const id of visibleCourseIds) next.add(id);
        return next;
      });
      initializedExpansion.current = true;
      return;
    }

    if (!initializedExpansion.current) {
      setExpandedCourseIds(new Set([visibleCourseIds[0]]));
      initializedExpansion.current = true;
    }
  }, [courseFilter, normalizedQuery, studentFilter, visibleCourseIdsKey]);

  const allVisibleGroupsExpanded =
    enrollmentGroups.length > 0 &&
    enrollmentGroups.every((group) => expandedCourseIds.has(group.courseId));
  const hasVisibleExpandedGroup = enrollmentGroups.some((group) =>
    expandedCourseIds.has(group.courseId)
  );

  function toggleGroup(courseId: number) {
    setExpandedCourseIds((current) => {
      const next = new Set(current);
      if (next.has(courseId)) next.delete(courseId);
      else next.add(courseId);
      return next;
    });
  }

  function courseOptionLabel(course: CourseDTO): string {
    const typeLabel = courseTypeLabel(course.type);
    return course.teacherName
      ? `${course.name} — ${typeLabel} — ${course.teacherName}`
      : `${course.name} — ${typeLabel}`;
  }

  return (
    <>
      <FilterToolbar
        primaryAction={<button onClick={onAddEnrollment}>{t("button.addEnrollment")}</button>}
        search={
          <input
            className="searchField"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("msg.searchPlaceholderEnrollment")}
            aria-label={t("msg.searchPlaceholderEnrollment")}
          />
        }
        filters={
          <>
            <select
              value={studentFilter ?? ""}
              onChange={(e) =>
                onStudentFilterChange(e.target.value ? parseInt(e.target.value, 10) : undefined)
              }
            >
              <option value="">{t("filter.allStudents")}</option>
              {allStudents.map((student) => (
                <option key={student.id} value={student.id}>
                  {student.fullName}
                </option>
              ))}
            </select>
            <select
              value={courseFilter ?? ""}
              onChange={(e) =>
                onCourseFilterChange(e.target.value ? parseInt(e.target.value, 10) : undefined)
              }
            >
              <option value="">{t("filter.allCourses")}</option>
              {allCourses.map((course) => (
                <option key={course.id} value={course.id}>
                  {courseOptionLabel(course)}
                </option>
              ))}
            </select>
          </>
        }
        hasActiveFilters={hasActiveFilters}
        onClearFilters={() => {
          setQuery("");
          onStudentFilterChange(undefined);
          onCourseFilterChange(undefined);
        }}
        clearLabel={t("button.clearFilters")}
        secondaryActions={
          enrollmentGroups.length > 1 ? (
            <div className="enrollmentGroupControls">
              <button
                type="button"
                className="secondaryActionButton"
                disabled={allVisibleGroupsExpanded}
                onClick={() =>
                  setExpandedCourseIds(new Set(enrollmentGroups.map((group) => group.courseId)))
                }
              >
                {t("button.expandAll")}
              </button>
              <button
                type="button"
                className="secondaryActionButton"
                disabled={!hasVisibleExpandedGroup}
                onClick={() => setExpandedCourseIds(new Set())}
              >
                {t("button.collapseAll")}
              </button>
            </div>
          ) : undefined
        }
      />

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : visibleEnrollments.length === 0 ? (
        hasActiveFilters ? (
          <EmptyState
            title={t("msg.noEnrollmentsSearchTitle")}
            description={t("msg.noEnrollmentsSearchDescription")}
            actionLabel={t("button.clearFilters")}
            onAction={() => {
              setQuery("");
              onStudentFilterChange(undefined);
              onCourseFilterChange(undefined);
            }}
          />
        ) : allStudents.length === 0 ? (
          <EmptyState
            title={t("msg.noEnrollmentsNeedStudentsTitle")}
            description={t("msg.noEnrollmentsNeedStudentsDescription")}
            actionLabel={t("button.openStudents")}
            onAction={onOpenStudents}
          />
        ) : allCourses.length === 0 ? (
          <EmptyState
            title={t("msg.noEnrollmentsNeedCoursesTitle")}
            description={t("msg.noEnrollmentsNeedCoursesDescription")}
            actionLabel={t("button.openCourses")}
            onAction={onOpenCourses}
          />
        ) : (
          <EmptyState
            title={t("msg.noEnrollmentsTitle")}
            description={t("msg.noEnrollmentsDescription")}
            actionLabel={t("button.addEnrollment")}
            onAction={onAddEnrollment}
          />
        )
      ) : (
        <div className="enrollmentGroupList">
          {enrollmentGroups.map((group) => {
            const expanded = expandedCourseIds.has(group.courseId);
            return (
              <section
                key={group.courseId}
                className={`enrollmentGroup ${expanded ? "enrollmentGroup--expanded" : ""}`}
              >
                <button
                  type="button"
                  className="enrollmentGroupHeader"
                  aria-expanded={expanded}
                  onClick={() => toggleGroup(group.courseId)}
                >
                  <span className="enrollmentGroupChevron" aria-hidden="true">
                    {expanded ? "▾" : "▸"}
                  </span>
                  <span className="enrollmentGroupTitle">
                    <strong>{group.courseName}</strong>
                    <span>
                      {courseTypeLabel(group.courseType)} · {group.teacherName || "—"}
                    </span>
                  </span>
                  <span className="enrollmentGroupCount">
                    {t("msg.studentsCount", { count: group.enrollments.length })}
                  </span>
                </button>

                {expanded && (
                  <div className="tableWrap enrollmentGroupTableWrap">
                    <table className="enrollmentCompactTable">
                      <thead>
                        <tr>
                          <th>{t("field.student")}</th>
                          <th>{t("field.billing")}</th>
                          <th style={{ textAlign: "right" }}>{t("field.lessonPriceOverride")}</th>
                          <th>{t("field.materials")}</th>
                          <th>{t("field.note")}</th>
                          <th aria-label={t("field.actions")}></th>
                        </tr>
                      </thead>
                      <tbody>
                        {group.enrollments.map((enrollment) => (
                          <tr key={enrollment.id}>
                            <td>
                              <button
                                className="linkButton enrollmentStudentLink"
                                onClick={() => void onOpenStudent(enrollment.studentId)}
                              >
                                {enrollment.studentName}
                              </button>
                            </td>
                            <td>{billingModeLabel(enrollment.billingMode)}</td>
                            <td className="enrollmentCompactPrice">
                              {enrollment.billingMode === "per_lesson"
                                ? `€${enrollment.lessonPriceOverride.toFixed(2)}`
                                : `€${enrollment.subscriptionLessonPrice.toFixed(2)}`}
                            </td>
                            <td>
                              <span
                                className={`enrollmentMaterialsFlag ${
                                  enrollment.chargeMaterials
                                    ? "enrollmentMaterialsFlag--charged"
                                    : ""
                                }`}
                              >
                                {enrollment.chargeMaterials ? t("label.yes") : t("label.no")}
                              </span>
                            </td>
                            <td>
                              <span className="enrollmentCompactNote" title={enrollment.note}>
                                {enrollment.note || "—"}
                              </span>
                            </td>
                            <td className="enrollmentCompactActions">
                              <button
                                type="button"
                                className="enrollmentRowAction"
                                aria-label={`${t("button.edit")}: ${enrollment.studentName}`}
                                title={t("button.edit")}
                                onClick={() => onEditEnrollment(enrollment)}
                              >
                                ...
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </section>
            );
          })}
        </div>
      )}

      {enrollmentModalOpen && (
        <EnrollmentFormModal
          editing={editingEnrollment}
          studentSearch={studentSearch}
          studentId={studentId}
          studentPickerOpen={studentPickerOpen}
          filteredStudents={filteredStudents}
          selectedStudent={selectedStudent}
          studentLocked={enrollmentStudentLocked}
          courseId={enrollmentCourseId}
          mode={enrollmentMode}
          chargeMaterials={enrollmentChargeMaterials}
          lessonPriceOverride={enrollmentLessonPriceOverride}
          subscriptionLessonPrice={enrollmentSubscriptionLessonPrice}
          note={enrollmentNote}
          allCourses={enrollmentCourseOptions}
          studentComboRef={studentComboRef as Ref<HTMLDivElement>}
          onStudentSearchChange={onStudentSearchChange}
          onStudentIdChange={onStudentIdChange}
          onStudentPickerOpenChange={onStudentPickerOpenChange}
          onCourseIdChange={onEnrollmentCourseIdChange}
          onModeChange={onEnrollmentModeChange}
          onChargeMaterialsChange={onEnrollmentChargeMaterialsChange}
          onLessonPriceOverrideChange={onEnrollmentLessonPriceOverrideChange}
          onSubscriptionLessonPriceChange={onEnrollmentSubscriptionLessonPriceChange}
          onNoteChange={onEnrollmentNoteChange}
          onSave={onSaveEnrollment}
          saveLabel={enrollmentSaveLabel}
          onCancel={onCloseEnrollmentModal}
          t={t}
        />
      )}
    </>
  );
}
