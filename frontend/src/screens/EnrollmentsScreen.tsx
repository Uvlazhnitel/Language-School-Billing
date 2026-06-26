import type { Ref, RefObject } from "react";
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

export function EnrollmentsScreen({
  loading,
  enrollments,
  studentFilter,
  courseFilter,
  allStudents,
  allCourses,
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
  const hasActiveFilters = studentFilter !== undefined || courseFilter !== undefined;

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
          onStudentFilterChange(undefined);
          onCourseFilterChange(undefined);
        }}
        clearLabel={t("button.clearFilters")}
      />

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : enrollments.length === 0 ? (
        hasActiveFilters ? (
          <EmptyState
            title={t("msg.noEnrollmentsSearchTitle")}
            description={t("msg.noEnrollmentsSearchDescription")}
            actionLabel={t("button.clearFilters")}
            onAction={() => {
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
        <table>
          <thead>
            <tr>
              <th>{t("field.student")}</th>
              <th>{t("field.course")}</th>
              <th>{t("field.type")}</th>
              <th>{t("field.teacher")}</th>
              <th>{t("field.billing")}</th>
              <th style={{ textAlign: "right" }}>{t("field.lessonPriceOverride")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {enrollments.map((enrollment) => (
              <tr key={enrollment.id}>
                <td>
                  <button
                    className="linkButton"
                    onClick={() => void onOpenStudent(enrollment.studentId)}
                  >
                    {enrollment.studentName}
                  </button>
                </td>
                <td>{enrollment.courseName}</td>
                <td>{courseTypeLabel(enrollment.courseType)}</td>
                <td>{enrollment.teacherName || "—"}</td>
                <td>{billingModeLabel(enrollment.billingMode)}</td>
                <td style={{ textAlign: "right" }}>
                  {enrollment.billingMode === "per_lesson"
                    ? `€${enrollment.lessonPriceOverride.toFixed(2)}`
                    : `€${enrollment.subscriptionLessonPrice.toFixed(2)}`}
                </td>
                <td>
                  <button onClick={() => onEditEnrollment(enrollment)}>{t("button.edit")}</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {enrollmentModalOpen && (
        <EnrollmentFormModal
          editing={editingEnrollment}
          studentSearch={studentSearch}
          studentId={studentId}
          studentPickerOpen={studentPickerOpen}
          filteredStudents={filteredStudents}
          selectedStudent={selectedStudent}
          courseId={enrollmentCourseId}
          mode={enrollmentMode}
          chargeMaterials={enrollmentChargeMaterials}
          lessonPriceOverride={enrollmentLessonPriceOverride}
          subscriptionLessonPrice={enrollmentSubscriptionLessonPrice}
          note={enrollmentNote}
          allCourses={allCourses}
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
          onCancel={onCloseEnrollmentModal}
          t={t}
        />
      )}
    </>
  );
}
