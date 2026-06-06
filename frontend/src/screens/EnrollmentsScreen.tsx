import type { Ref, RefObject } from "react";
import { EnrollmentFormModal } from "../components/modals/EnrollmentFormModal";
import type { CourseDTO } from "../lib/courses";
import type { EnrollmentDTO } from "../lib/enrollments";
import type { StudentDTO } from "../lib/students";
import type { TranslateFn } from "../lib/i18n";

type EnrollmentsScreenProps = {
  loading: boolean;
  enrollments: EnrollmentDTO[];
  studentFilter?: number;
  courseFilter?: number;
  allStudents: StudentDTO[];
  allCourses: CourseDTO[];
  billingModeLabel: (mode: string) => string;
  onStudentFilterChange: (value: number | undefined) => void;
  onCourseFilterChange: (value: number | undefined) => void;
  onRefresh: () => void;
  onAddEnrollment: () => void;
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
  enrollmentDiscount: number;
  enrollmentSubscriptionDiscount: number;
  enrollmentNote: string;
  studentComboRef: RefObject<HTMLDivElement | null>;
  onStudentSearchChange: (value: string) => void;
  onStudentIdChange: (value: number | undefined) => void;
  onStudentPickerOpenChange: (value: boolean) => void;
  onEnrollmentCourseIdChange: (value: number) => void;
  onEnrollmentModeChange: (value: "per_lesson" | "subscription") => void;
  onEnrollmentDiscountChange: (value: number) => void;
  onEnrollmentSubscriptionDiscountChange: (value: number) => void;
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
  onStudentFilterChange,
  onCourseFilterChange,
  onRefresh,
  onAddEnrollment,
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
  enrollmentDiscount,
  enrollmentSubscriptionDiscount,
  enrollmentNote,
  studentComboRef,
  onStudentSearchChange,
  onStudentIdChange,
  onStudentPickerOpenChange,
  onEnrollmentCourseIdChange,
  onEnrollmentModeChange,
  onEnrollmentDiscountChange,
  onEnrollmentSubscriptionDiscountChange,
  onEnrollmentNoteChange,
  onSaveEnrollment,
  onCloseEnrollmentModal,
  t,
}: EnrollmentsScreenProps) {
  return (
    <>
      <div className="controls">
        <button onClick={onAddEnrollment}>{t("button.addEnrollment")}</button>
        <select
          value={studentFilter ?? ""}
          onChange={(e) => onStudentFilterChange(e.target.value ? parseInt(e.target.value, 10) : undefined)}
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
          onChange={(e) => onCourseFilterChange(e.target.value ? parseInt(e.target.value, 10) : undefined)}
        >
          <option value="">{t("filter.allCourses")}</option>
          {allCourses.map((course) => (
            <option key={course.id} value={course.id}>
              {course.teacherName ? `${course.name} — ${course.teacherName}` : course.name}
            </option>
          ))}
        </select>
        <button onClick={onRefresh}>{t("button.refresh")}</button>
      </div>

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : enrollments.length === 0 ? (
        <div className="empty">{t("msg.noEnrollmentsYet")}</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>{t("field.student")}</th>
              <th>{t("field.course")}</th>
              <th>{t("field.teacher")}</th>
              <th>{t("field.billing")}</th>
              <th style={{ textAlign: "right" }}>{t("field.discount")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {enrollments.map((enrollment) => (
              <tr key={enrollment.id}>
                <td>
                  <button className="linkButton" onClick={() => void onOpenStudent(enrollment.studentId)}>
                    {enrollment.studentName}
                  </button>
                </td>
                <td>{enrollment.courseName}</td>
                <td>{enrollment.teacherName || "—"}</td>
                <td>{billingModeLabel(enrollment.billingMode)}</td>
                <td style={{ textAlign: "right" }}>{enrollment.discountPct.toFixed(1)}%</td>
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
          discount={enrollmentDiscount}
          subscriptionDiscount={enrollmentSubscriptionDiscount}
          note={enrollmentNote}
          allCourses={allCourses}
          studentComboRef={studentComboRef as Ref<HTMLDivElement>}
          onStudentSearchChange={onStudentSearchChange}
          onStudentIdChange={onStudentIdChange}
          onStudentPickerOpenChange={onStudentPickerOpenChange}
          onCourseIdChange={onEnrollmentCourseIdChange}
          onModeChange={onEnrollmentModeChange}
          onDiscountChange={onEnrollmentDiscountChange}
          onSubscriptionDiscountChange={onEnrollmentSubscriptionDiscountChange}
          onNoteChange={onEnrollmentNoteChange}
          onSave={onSaveEnrollment}
          onCancel={onCloseEnrollmentModal}
          t={t}
        />
      )}
    </>
  );
}
