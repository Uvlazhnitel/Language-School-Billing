import { CourseFormModal } from "../components/modals/CourseFormModal";
import type { CourseDTO } from "../lib/courses";
import type { TeacherDTO } from "../lib/teachers";
import type { TranslateFn } from "../lib/i18n";
import type { Ref, RefObject } from "react";
import { EmptyState } from "../components/EmptyState";

type CoursesScreenProps = {
  loading: boolean;
  courses: CourseDTO[];
  query: string;
  canDeleteCourses: boolean;
  courseTypeLabel: (type: string) => string;
  formatEUR: (value: number) => string;
  onQueryChange: (value: string) => void;
  onAddCourse: () => void;
  onEditCourse: (course: CourseDTO) => void;
  onDeleteCourse: (courseId: number) => void | Promise<void>;
  courseModalOpen: boolean;
  editingCourse: boolean;
  cfName: string;
  cfTeacherSearch: string;
  cfTeacherId?: number;
  cfTeacherPickerOpen: boolean;
  selectedCourseTeacher?: TeacherDTO | null;
  filteredTeachers: TeacherDTO[];
  exactTeacherMatch: TeacherDTO | null;
  cfTeacherCreating: boolean;
  cfType: "group" | "individual";
  cfLessonPrice: string;
  cfSubscriptionPrice: string;
  cfTeacherComboRef: RefObject<HTMLDivElement | null>;
  onCfNameChange: (value: string) => void;
  onCfTeacherSearchChange: (value: string) => void;
  onCfTeacherIdChange: (value: number | undefined) => void;
  onCfTeacherPickerOpenChange: (value: boolean) => void;
  onAddTeacherFromCourseForm: () => void | Promise<void>;
  onCfTypeChange: (value: "group" | "individual") => void;
  onCfLessonPriceChange: (value: string) => void;
  onCfSubscriptionPriceChange: (value: string) => void;
  onSaveCourse: () => void;
  onCloseCourseModal: () => void;
  t: TranslateFn;
};

export function CoursesScreen({
  loading,
  courses,
  query,
  canDeleteCourses,
  courseTypeLabel,
  formatEUR,
  onQueryChange,
  onAddCourse,
  onEditCourse,
  onDeleteCourse,
  courseModalOpen,
  editingCourse,
  cfName,
  cfTeacherSearch,
  cfTeacherId,
  cfTeacherPickerOpen,
  selectedCourseTeacher,
  filteredTeachers,
  exactTeacherMatch,
  cfTeacherCreating,
  cfType,
  cfLessonPrice,
  cfSubscriptionPrice,
  cfTeacherComboRef,
  onCfNameChange,
  onCfTeacherSearchChange,
  onCfTeacherIdChange,
  onCfTeacherPickerOpenChange,
  onAddTeacherFromCourseForm,
  onCfTypeChange,
  onCfLessonPriceChange,
  onCfSubscriptionPriceChange,
  onSaveCourse,
  onCloseCourseModal,
  t,
}: CoursesScreenProps) {
  return (
    <>
      <div className="controls">
        <button onClick={onAddCourse}>{t("button.addCourse")}</button>
        <input
          className="searchField"
          placeholder={t("msg.searchPlaceholderCourse")}
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
        />
      </div>

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : courses.length === 0 ? (
        query.trim() ? (
          <EmptyState
            title={t("msg.noCoursesSearchTitle")}
            description={t("msg.noCoursesSearchDescription")}
            actionLabel={t("button.clearSearch")}
            onAction={() => onQueryChange("")}
          />
        ) : (
          <EmptyState
            title={t("msg.noCoursesTitle")}
            description={t("msg.noCoursesDescription")}
            actionLabel={t("button.addCourse")}
            onAction={onAddCourse}
          />
        )
      ) : (
        <table>
          <thead>
            <tr>
              <th>{t("field.name")}</th>
              <th>{t("field.teacher")}</th>
              <th>{t("field.type")}</th>
              <th style={{ textAlign: "right" }}>{t("field.lessonPrice")} (EUR)</th>
              <th style={{ textAlign: "right" }}>{t("field.subscriptionPrice")} (EUR)</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {courses.map((course) => (
              <tr key={course.id}>
                <td>{course.name}</td>
                <td>{course.teacherName || "—"}</td>
                <td>{courseTypeLabel(course.type)}</td>
                <td style={{ textAlign: "right" }}>{formatEUR(course.lessonPrice)}</td>
                <td style={{ textAlign: "right" }}>{formatEUR(course.subscriptionPrice)}</td>
                <td>
                  <button onClick={() => onEditCourse(course)}>{t("button.edit")}</button>
                  {canDeleteCourses && (
                    <button onClick={() => void onDeleteCourse(course.id)}>{t("button.delete")}</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {courseModalOpen && (
        <CourseFormModal
          editing={editingCourse}
          name={cfName}
          teacherSearch={cfTeacherSearch}
          teacherId={cfTeacherId}
          teacherPickerOpen={cfTeacherPickerOpen}
          selectedTeacher={selectedCourseTeacher}
          filteredTeachers={filteredTeachers}
          exactTeacherMatch={exactTeacherMatch}
          teacherCreating={cfTeacherCreating}
          type={cfType}
          lessonPrice={cfLessonPrice}
          subscriptionPrice={cfSubscriptionPrice}
          teacherComboRef={cfTeacherComboRef as Ref<HTMLDivElement>}
          onNameChange={onCfNameChange}
          onTeacherSearchChange={onCfTeacherSearchChange}
          onTeacherIdChange={onCfTeacherIdChange}
          onTeacherPickerOpenChange={onCfTeacherPickerOpenChange}
          onCreateTeacher={onAddTeacherFromCourseForm}
          onTypeChange={onCfTypeChange}
          onLessonPriceChange={onCfLessonPriceChange}
          onSubscriptionPriceChange={onCfSubscriptionPriceChange}
          onSave={onSaveCourse}
          onCancel={onCloseCourseModal}
          t={t}
        />
      )}
    </>
  );
}
