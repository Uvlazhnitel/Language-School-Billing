import type { Ref } from "react";
import type { CourseDTO } from "../../lib/courses";
import type { EnrollmentDTO } from "../../lib/enrollments";
import type { StudentDTO } from "../../lib/students";
import type { TranslateFn } from "../../lib/i18n";
import { courseTypeLabel } from "../../lib/appUi";

type EnrollmentFormModalProps = {
  editing: EnrollmentDTO | null;
  studentSearch: string;
  studentId?: number;
  studentPickerOpen: boolean;
  filteredStudents: StudentDTO[];
  selectedStudent?: StudentDTO | null;
  courseId: number;
  mode: "per_lesson" | "subscription";
  chargeMaterials: boolean;
  lessonPriceOverride: string;
  subscriptionLessonPrice: string;
  note: string;
  allCourses: CourseDTO[];
  studentComboRef: Ref<HTMLDivElement>;
  onStudentSearchChange: (value: string) => void;
  onStudentIdChange: (value: number | undefined) => void;
  onStudentPickerOpenChange: (value: boolean) => void;
  onCourseIdChange: (value: number) => void;
  onModeChange: (value: "per_lesson" | "subscription") => void;
  onChargeMaterialsChange: (value: boolean) => void;
  onLessonPriceOverrideChange: (value: string) => void;
  onSubscriptionLessonPriceChange: (value: string) => void;
  onNoteChange: (value: string) => void;
  onSave: () => void;
  onCancel: () => void;
  t: TranslateFn;
};

export function EnrollmentFormModal({
  editing,
  studentSearch,
  studentId,
  studentPickerOpen,
  filteredStudents,
  selectedStudent,
  courseId,
  mode,
  chargeMaterials,
  lessonPriceOverride,
  subscriptionLessonPrice,
  note,
  allCourses,
  studentComboRef,
  onStudentSearchChange,
  onStudentIdChange,
  onStudentPickerOpenChange,
  onCourseIdChange,
  onModeChange,
  onChargeMaterialsChange,
  onLessonPriceOverrideChange,
  onSubscriptionLessonPriceChange,
  onNoteChange,
  onSave,
  onCancel,
  t,
}: EnrollmentFormModalProps) {
  function courseOptionLabel(course: CourseDTO): string {
    const typeLabel = courseTypeLabel(course.type, t);
    return course.teacherName
      ? `${course.name} — ${typeLabel} — ${course.teacherName}`
      : `${course.name} — ${typeLabel}`;
  }

  return (
    <div className="modal">
      <div className="modalBody">
        <h3>{editing ? t("modal.editEnrollment") : t("modal.addEnrollment")}</h3>

        <div className="formRow">
          <label>{t("field.student")}</label>
          {editing ? (
            <input value={selectedStudent?.fullName ?? studentSearch} disabled />
          ) : (
            <div className="comboBox" ref={studentComboRef}>
              <input
                value={studentSearch}
                onChange={(e) => {
                  onStudentSearchChange(e.target.value);
                  onStudentPickerOpenChange(true);
                }}
                onFocus={() => onStudentPickerOpenChange(true)}
                onKeyDown={(e) => {
                  if (e.key === "Escape") {
                    onStudentPickerOpenChange(false);
                  }
                }}
                placeholder={t("msg.searchPlaceholderStudent")}
              />
              {studentPickerOpen && (
                <div className="comboBoxMenu">
                  {filteredStudents.length === 0 ? (
                    <div className="comboBoxEmpty">{t("msg.noStudentsFound")}</div>
                  ) : (
                    filteredStudents.map((student) => (
                      <button
                        key={student.id}
                        type="button"
                        className={`comboBoxOption ${student.id === studentId ? "active" : ""}`}
                        onClick={() => {
                          onStudentIdChange(student.id);
                          onStudentSearchChange(student.fullName);
                          onStudentPickerOpenChange(false);
                        }}
                      >
                        <span className="comboBoxPrimary">{student.fullName}</span>
                        <span className="comboBoxMeta">
                          {[student.phone, student.email].filter(Boolean).join(" · ")}
                        </span>
                      </button>
                    ))
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        <div className="formRow">
          <label>{t("field.course")}</label>
          <select
            value={courseId}
            disabled={Boolean(editing)}
            onChange={(e) => onCourseIdChange(parseInt(e.target.value, 10))}
          >
            {allCourses.map((course) => (
              <option key={course.id} value={course.id}>
                {courseOptionLabel(course)}
              </option>
            ))}
          </select>
        </div>

        <div className="formRow">
          <label>{t("field.billing")}</label>
          <select value={mode} onChange={(e) => onModeChange(e.target.value as "per_lesson" | "subscription")}>
            <option value="per_lesson">{t("billing.perLesson")}</option>
            <option value="subscription">{t("billing.subscription")}</option>
          </select>
        </div>

        <div className="formRow">
          <label>{t("field.chargeMaterials")}</label>
          <input
            className="formCheckbox"
            type="checkbox"
            checked={chargeMaterials}
            onChange={(e) => onChargeMaterialsChange(e.target.checked)}
          />
        </div>

        {mode === "per_lesson" && (
          <div className="formRow">
            <label>{t("field.lessonPriceOverride")} (EUR)</label>
            <input
              type="number"
              min={0}
              step="0.1"
              value={lessonPriceOverride}
              onFocus={(e) => e.currentTarget.select()}
              onChange={(e) => onLessonPriceOverrideChange(e.target.value)}
            />
          </div>
        )}

        {mode === "subscription" && (
          <div className="formRow">
            <label>{t("field.subscriptionLessonPrice")} (EUR)</label>
            <input
              type="number"
              min={0}
              step="0.1"
              value={subscriptionLessonPrice}
              onFocus={(e) => e.currentTarget.select()}
              onChange={(e) => onSubscriptionLessonPriceChange(e.target.value)}
            />
          </div>
        )}

        <div className="formRow">
          <label>{t("field.note")}</label>
          <input value={note} onChange={(e) => onNoteChange(e.target.value)} />
        </div>

        <div className="modalActions">
          <button onClick={onSave}>{t("button.save")}</button>
          <button onClick={onCancel}>{t("button.cancel")}</button>
        </div>
      </div>
    </div>
  );
}
