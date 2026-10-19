import { CourseFormModal } from "../components/modals/CourseFormModal";
import type { CourseDTO } from "../lib/courses";
import type { TeacherDTO } from "../lib/teachers";
import type { TranslateFn } from "../lib/i18n";
import { useState, type Ref, type RefObject } from "react";
import {
  groupCoursesBySubject,
  resolveCourseDisclosure,
  type CourseDisclosure,
} from "../lib/courseGroups";
import { CourseCatalog } from "../components/CourseCatalog";
import { EmptyState } from "../components/EmptyState";
import { FilterToolbar } from "../components/FilterToolbar";

type CoursesScreenProps = {
  loading: boolean;
  courses: CourseDTO[];
  savedCourse?: CourseDTO | null;
  query: string;
  typeFilter: "" | "group" | "individual";
  teacherFilter: string;
  pricingFilter: "all" | "lesson" | "subscription" | "both" | "lesson_only" | "subscription_only";
  teacherOptions: Array<{ value: string; label: string }>;
  hasActiveFilters: boolean;
  canDeleteCourses: boolean;
  courseTypeLabel: (type: string) => string;
  formatEUR: (value: number) => string;
  onQueryChange: (value: string) => void;
  onTypeFilterChange: (value: "" | "group" | "individual") => void;
  onTeacherFilterChange: (value: string) => void;
  onPricingFilterChange: (
    value: "all" | "lesson" | "subscription" | "both" | "lesson_only" | "subscription_only"
  ) => void;
  onClearFilters: () => void;
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
  savedCourse = null,
  query,
  typeFilter,
  teacherFilter,
  pricingFilter,
  teacherOptions,
  hasActiveFilters,
  canDeleteCourses,
  courseTypeLabel,
  formatEUR,
  onQueryChange,
  onTypeFilterChange,
  onTeacherFilterChange,
  onPricingFilterChange,
  onClearFilters,
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
  const groups = groupCoursesBySubject(courses);
  const filterKey = JSON.stringify([query, typeFilter, teacherFilter, pricingFilter]);
  const [disclosure, setDisclosure] = useState<CourseDisclosure>({
    filterKey,
    savedCourse,
    openKey: null,
  });
  const resolved = loading
    ? disclosure
    : resolveCourseDisclosure(disclosure, filterKey, savedCourse, groups);
  if (
    resolved.filterKey !== disclosure.filterKey ||
    resolved.savedCourse !== disclosure.savedCourse ||
    resolved.openKey !== disclosure.openKey
  ) {
    setDisclosure(resolved);
  }
  return (
    <>
      <FilterToolbar
        primaryAction={<button className="btnPrimary" onClick={onAddCourse}>{t("button.addCourse")}</button>}
        search={
          <input
            className="searchField"
            placeholder={t("msg.searchPlaceholderCourse")}
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
          />
        }
        filters={
          <>
            <select
              value={typeFilter}
              onChange={(e) => onTypeFilterChange(e.target.value as "" | "group" | "individual")}
            >
              <option value="">{t("filter.allTypes")}</option>
              <option value="group">{courseTypeLabel("group")}</option>
              <option value="individual">{courseTypeLabel("individual")}</option>
            </select>
            <select value={teacherFilter} onChange={(e) => onTeacherFilterChange(e.target.value)}>
              <option value="all">{t("filter.allTeachers")}</option>
              <option value="none">{t("filter.withoutTeacher")}</option>
              {teacherOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <select
              value={pricingFilter}
              onChange={(e) =>
                onPricingFilterChange(
                  e.target.value as
                    | "all"
                    | "lesson"
                    | "subscription"
                    | "both"
                    | "lesson_only"
                    | "subscription_only"
                )
              }
            >
              <option value="all">{t("filter.pricingAll")}</option>
              <option value="lesson">{t("filter.pricingLesson")}</option>
              <option value="subscription">{t("filter.pricingSubscription")}</option>
              <option value="both">{t("filter.pricingBoth")}</option>
              <option value="lesson_only">{t("filter.pricingLessonOnly")}</option>
              <option value="subscription_only">{t("filter.pricingSubscriptionOnly")}</option>
            </select>
          </>
        }
        hasActiveFilters={hasActiveFilters}
        onClearFilters={onClearFilters}
        clearLabel={t("button.clearFilters")}
      />

      {loading ? (
        <div>{t("label.loading")}</div>
      ) : courses.length === 0 ? (
        hasActiveFilters ? (
          <EmptyState
            title={t("msg.noCoursesSearchTitle")}
            description={t("msg.noCoursesSearchDescription")}
            actionLabel={t("button.clearFilters")}
            onAction={onClearFilters}
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
        <CourseCatalog
          groups={groups}
          openKey={resolved.openKey}
          onToggle={(key) =>
            setDisclosure({ ...resolved, openKey: resolved.openKey === key ? null : key })
          }
          canDeleteCourses={canDeleteCourses}
          courseTypeLabel={courseTypeLabel}
          formatEUR={formatEUR}
          onEditCourse={onEditCourse}
          onDeleteCourse={onDeleteCourse}
          t={t}
        />
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
