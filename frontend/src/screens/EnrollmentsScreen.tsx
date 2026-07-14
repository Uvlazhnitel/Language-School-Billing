import { useEffect, useState, type Ref, type RefObject } from "react";
import { EnrollmentFormModal } from "../components/modals/EnrollmentFormModal";
import { EmptyState } from "../components/EmptyState";
import type { CourseDTO } from "../lib/courses";
import type { EnrollmentDTO } from "../lib/enrollments";
import type { StudentDTO } from "../lib/students";
import type { TranslateFn } from "../lib/i18n";
import { FilterToolbar } from "../components/FilterToolbar";
import {
  ENROLLMENT_SELECTION_STORAGE_KEY,
  groupEnrollmentsByTeacher,
  normalizeEnrollmentSelectionPreferences,
  parseEnrollmentSelectionPreferences,
  resolveEnrollmentSelection,
  type EnrollmentSelectionPreferences,
} from "../lib/enrollmentTeacherGroups";

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

type AutomaticSelection = {
  context: string;
  teacherKey: string | null;
  courseId?: number;
};

function readSelectionPreferences(): EnrollmentSelectionPreferences {
  if (typeof window === "undefined") return { courseByTeacher: {} };
  try {
    return parseEnrollmentSelectionPreferences(
      window.localStorage.getItem(ENROLLMENT_SELECTION_STORAGE_KEY)
    );
  } catch {
    return { courseByTeacher: {} };
  }
}

function saveSelectionPreferences(preferences: EnrollmentSelectionPreferences) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(ENROLLMENT_SELECTION_STORAGE_KEY, JSON.stringify(preferences));
  } catch {
    // Persistence is an enhancement; selection still works for the current session.
  }
}

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
  const [selectionPreferences, setSelectionPreferences] =
    useState<EnrollmentSelectionPreferences>(readSelectionPreferences);
  const [manualOpenTeacherKey, setManualOpenTeacherKey] = useState<string | null>();
  const [automaticSelection, setAutomaticSelection] = useState<AutomaticSelection>();
  const normalizedQuery = query.trim().toLocaleLowerCase();
  const visibleEnrollments = normalizedQuery
    ? enrollments.filter((enrollment) =>
        enrollment.studentName.toLocaleLowerCase().includes(normalizedQuery)
      )
    : enrollments;
  const hasActiveFilters =
    normalizedQuery !== "" || studentFilter !== undefined || courseFilter !== undefined;
  const automaticMode = hasActiveFilters;
  const selectionContext = `${normalizedQuery}|${studentFilter ?? ""}|${courseFilter ?? ""}`;
  const teacherGroups = groupEnrollmentsByTeacher(visibleEnrollments, t("filter.withoutTeacher"));
  const normalizedPreferences = normalizeEnrollmentSelectionPreferences(
    teacherGroups,
    selectionPreferences
  );
  const manualSelection = resolveEnrollmentSelection(teacherGroups, normalizedPreferences);
  const defaultAutomaticTeacher = teacherGroups[0];
  const currentAutomaticSelection =
    automaticSelection?.context === selectionContext ? automaticSelection : undefined;
  const automaticTeacherKey =
    currentAutomaticSelection?.teacherKey === null
      ? null
      : teacherGroups.some((teacher) => teacher.key === currentAutomaticSelection?.teacherKey)
        ? currentAutomaticSelection?.teacherKey
        : defaultAutomaticTeacher?.key;
  const openTeacherKey = automaticMode
    ? automaticTeacherKey
    : manualOpenTeacherKey === null
      ? null
      : teacherGroups.some((teacher) => teacher.key === manualOpenTeacherKey)
        ? manualOpenTeacherKey
        : manualSelection.teacherKey;
  const openTeacher = teacherGroups.find((teacher) => teacher.key === openTeacherKey);
  const preferredCourseId = automaticMode
    ? currentAutomaticSelection?.courseId
    : openTeacher
      ? normalizedPreferences.courseByTeacher[openTeacher.key]
      : undefined;
  const selectedCourse =
    openTeacher?.courses.find((course) => course.courseId === preferredCourseId) ??
    openTeacher?.courses[0];

  useEffect(() => {
    if (automaticMode || teacherGroups.length === 0) return;
    if (JSON.stringify(normalizedPreferences) === JSON.stringify(selectionPreferences)) return;

    setSelectionPreferences(normalizedPreferences);
    saveSelectionPreferences(normalizedPreferences);
  }, [automaticMode, normalizedPreferences, selectionPreferences, teacherGroups.length]);

  function updateManualSelection(teacherKey: string, courseId?: number) {
    const teacher = teacherGroups.find((candidate) => candidate.key === teacherKey);
    if (!teacher) return;
    const savedCourseId = selectionPreferences.courseByTeacher[teacherKey];
    const nextCourseId =
      teacher.courses.find((course) => course.courseId === courseId)?.courseId ??
      teacher.courses.find((course) => course.courseId === savedCourseId)?.courseId ??
      teacher.courses[0].courseId;
    const next = {
      activeTeacherKey: teacherKey,
      courseByTeacher: {
        ...selectionPreferences.courseByTeacher,
        [teacherKey]: nextCourseId,
      },
    };
    setSelectionPreferences(next);
    saveSelectionPreferences(next);
  }

  function toggleTeacher(teacherKey: string) {
    const isOpen = openTeacherKey === teacherKey;
    if (automaticMode) {
      setAutomaticSelection({
        context: selectionContext,
        teacherKey: isOpen ? null : teacherKey,
      });
      return;
    }

    if (isOpen) {
      setManualOpenTeacherKey(null);
      return;
    }
    setManualOpenTeacherKey(teacherKey);
    updateManualSelection(teacherKey);
  }

  function selectCourse(teacherKey: string, courseId: number) {
    if (automaticMode) {
      setAutomaticSelection({ context: selectionContext, teacherKey, courseId });
      return;
    }
    setManualOpenTeacherKey(teacherKey);
    updateManualSelection(teacherKey, courseId);
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
        <div className="enrollmentTeacherList">
          {teacherGroups.map((teacher) => {
            const expanded = openTeacherKey === teacher.key;
            return (
              <section
                key={teacher.key}
                className={`enrollmentTeacherGroup ${
                  expanded ? "enrollmentTeacherGroup--expanded" : ""
                }`}
              >
                <button
                  type="button"
                  className="enrollmentTeacherHeader"
                  aria-expanded={expanded}
                  onClick={() => toggleTeacher(teacher.key)}
                >
                  <span className="enrollmentTeacherChevron" aria-hidden="true">
                    {expanded ? "▾" : "▸"}
                  </span>
                  <span className="enrollmentTeacherTitle">
                    <strong>{teacher.teacherName}</strong>
                    <span>
                      {t("msg.teacherEnrollmentSummary", {
                        courses: teacher.courses.length,
                        students: teacher.studentCount,
                      })}
                    </span>
                  </span>
                </button>

                {expanded && (
                  <div className="enrollmentTeacherContent">
                    <div className="enrollmentCourseChips" role="tablist">
                      {teacher.courses.map((course) => {
                        const selected = selectedCourse?.courseId === course.courseId;
                        return (
                          <button
                            key={course.courseId}
                            type="button"
                            role="tab"
                            aria-selected={selected}
                            className={`enrollmentCourseChip ${
                              selected ? "enrollmentCourseChip--active" : ""
                            }`}
                            onClick={() => selectCourse(teacher.key, course.courseId)}
                          >
                            <strong>{course.courseName}</strong>
                            <span>
                              {courseTypeLabel(course.courseType)} ·{" "}
                              {t("msg.studentsCount", {
                                count: course.enrollments.length,
                              })}
                            </span>
                          </button>
                        );
                      })}
                    </div>

                    {selectedCourse && (
                      <div className="tableWrap enrollmentGroupTableWrap">
                        <table className="enrollmentCompactTable">
                          <thead>
                            <tr>
                              <th>{t("field.student")}</th>
                              <th>{t("field.billing")}</th>
                              <th style={{ textAlign: "right" }}>
                                {t("field.lessonPriceOverride")}
                              </th>
                              <th>{t("field.materials")}</th>
                              <th>{t("field.note")}</th>
                              <th aria-label={t("field.actions")}></th>
                            </tr>
                          </thead>
                          <tbody>
                            {selectedCourse.enrollments.map((enrollment) => (
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
