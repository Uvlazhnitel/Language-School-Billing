import type { Ref } from "react";
import type { TeacherDTO } from "../../lib/teachers";
import type { TranslateFn } from "../../lib/i18n";

type CourseFormModalProps = {
  editing: boolean;
  name: string;
  teacherSearch: string;
  teacherId?: number;
  teacherPickerOpen: boolean;
  selectedTeacher?: TeacherDTO | null;
  filteredTeachers: TeacherDTO[];
  exactTeacherMatch: TeacherDTO | null;
  teacherCreating: boolean;
  type: "group" | "individual";
  lessonPrice: string;
  subscriptionPrice: string;
  teacherComboRef: Ref<HTMLDivElement>;
  onNameChange: (value: string) => void;
  onTeacherSearchChange: (value: string) => void;
  onTeacherIdChange: (value: number | undefined) => void;
  onTeacherPickerOpenChange: (value: boolean) => void;
  onCreateTeacher: () => void | Promise<void>;
  onTypeChange: (value: "group" | "individual") => void;
  onLessonPriceChange: (value: string) => void;
  onSubscriptionPriceChange: (value: string) => void;
  onSave: () => void;
  onCancel: () => void;
  t: TranslateFn;
};

export function CourseFormModal({
  editing,
  name,
  teacherSearch,
  teacherId,
  teacherPickerOpen,
  selectedTeacher,
  filteredTeachers,
  exactTeacherMatch,
  teacherCreating,
  type,
  lessonPrice,
  subscriptionPrice,
  teacherComboRef,
  onNameChange,
  onTeacherSearchChange,
  onTeacherIdChange,
  onTeacherPickerOpenChange,
  onCreateTeacher,
  onTypeChange,
  onLessonPriceChange,
  onSubscriptionPriceChange,
  onSave,
  onCancel,
  t,
}: CourseFormModalProps) {
  return (
    <div className="modal">
      <div className="modalBody">
        <h3>{editing ? t("modal.editCourse") : t("modal.addCourse")}</h3>

        <div className="formRow">
          <label>{t("field.name")}</label>
          <input value={name} onChange={(e) => onNameChange(e.target.value)} />
        </div>

        <div className="formRow">
          <label>{t("field.teacher")}</label>
          <div className="comboBox" ref={teacherComboRef}>
            <input
              value={selectedTeacher?.fullName ?? teacherSearch}
              onChange={(e) => {
                onTeacherSearchChange(e.target.value);
                onTeacherIdChange(undefined);
                onTeacherPickerOpenChange(true);
              }}
              onFocus={() => onTeacherPickerOpenChange(true)}
              onKeyDown={(e) => {
                if (e.key === "Escape") {
                  onTeacherPickerOpenChange(false);
                }
              }}
              placeholder={t("filter.selectTeacher")}
            />
            {teacherPickerOpen && (
              <div className="comboBoxMenu">
                {filteredTeachers.map((teacher) => (
                  <button
                    key={teacher.id}
                    type="button"
                    className={`comboBoxOption ${teacher.id === teacherId ? "active" : ""}`}
                    onClick={() => {
                      onTeacherIdChange(teacher.id);
                      onTeacherSearchChange(teacher.fullName);
                      onTeacherPickerOpenChange(false);
                    }}
                  >
                    <span className="comboBoxPrimary">{teacher.fullName}</span>
                  </button>
                ))}
                {!exactTeacherMatch && teacherSearch.trim() && (
                  <button
                    type="button"
                    className="comboBoxOption"
                    onClick={() => void onCreateTeacher()}
                    disabled={teacherCreating}
                  >
                    <span className="comboBoxPrimary">
                      {teacherCreating
                        ? `${t("field.teacher")}...`
                        : `${t("button.addCourse")}: ${teacherSearch.trim()}`}
                    </span>
                    <span className="comboBoxMeta">{t("field.teacher")}</span>
                  </button>
                )}
                {filteredTeachers.length === 0 && !teacherSearch.trim() && (
                  <div className="comboBoxEmpty">{t("msg.noTeachers")}</div>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="formRow">
          <label>{t("field.type")}</label>
          <select value={type} onChange={(e) => onTypeChange(e.target.value as "group" | "individual")}>
            <option value="group">{t("course.group")}</option>
            <option value="individual">{t("course.individual")}</option>
          </select>
        </div>

        <div className="formRow">
          <label>{t("field.lessonPrice")} (EUR)</label>
          <input
            type="text"
            inputMode="decimal"
            min={0}
            step="0.01"
            value={lessonPrice}
            onChange={(e) => onLessonPriceChange(e.target.value)}
          />
        </div>

        <div className="formRow">
          <label>{t("field.subscriptionPrice")} (EUR)</label>
          <input
            type="text"
            inputMode="decimal"
            min={0}
            step="0.01"
            value={subscriptionPrice}
            onChange={(e) => onSubscriptionPriceChange(e.target.value)}
          />
        </div>

        <div className="modalActions">
          <button onClick={onSave}>{t("button.save")}</button>
          <button onClick={onCancel}>{t("button.cancel")}</button>
        </div>
      </div>
    </div>
  );
}
