import type { TranslateFn } from "../../lib/i18n";
import type { CourseDTO } from "../../lib/courses";
import type { EnrollmentDTO } from "../../lib/enrollments";
import type { StudentDTO, StudentDuplicateCheckResult } from "../../lib/students";
import { courseTypeLabel } from "../../lib/appUi";

type StudentFormModalProps = {
  editing: boolean;
  name: string;
  personalCode: string;
  phone: string;
  email: string;
  note: string;
  isMinor: boolean;
  payerName: string;
  payerRole: string;
  payerRoleOptions: readonly string[];
  payerRoleLabel: (role: string) => string;
  allCourses: CourseDTO[];
  courseId: number;
  enrollmentMode: EnrollmentDTO["billingMode"];
  enrollmentChargeMaterials: boolean;
  enrollmentLessonPrice: string;
  enrollmentSubscriptionPrice: string;
  enrollmentNote: string;
  enrollmentSettingsOpen: boolean;
  formatEUR: (value: number) => string;
  onNameChange: (value: string) => void;
  onPersonalCodeChange: (value: string) => void;
  onPhoneChange: (value: string) => void;
  onEmailChange: (value: string) => void;
  onNoteChange: (value: string) => void;
  onIsMinorChange: (value: boolean) => void;
  onPayerNameChange: (value: string) => void;
  onPayerRoleChange: (value: string) => void;
  onCourseIdChange: (value: number) => void;
  onEnrollmentModeChange: (value: EnrollmentDTO["billingMode"]) => void;
  onEnrollmentChargeMaterialsChange: (value: boolean) => void;
  onEnrollmentLessonPriceChange: (value: string) => void;
  onEnrollmentSubscriptionPriceChange: (value: string) => void;
  onEnrollmentNoteChange: (value: string) => void;
  onEnrollmentSettingsOpenChange: (value: boolean) => void;
  onSave: () => void;
  onSaveAndAddAnother?: () => void;
  onCancel: () => void;
  duplicateCheckResult?: StudentDuplicateCheckResult | null;
  onOpenExistingStudent: (studentId: number) => void;
  onEnrollExistingStudent?: (student: StudentDTO) => void;
  onCreateAnyway: () => void;
  t: TranslateFn;
};

export function StudentFormModal({
  editing,
  name,
  personalCode,
  phone,
  email,
  note,
  isMinor,
  payerName,
  payerRole,
  payerRoleOptions,
  payerRoleLabel,
  allCourses,
  courseId,
  enrollmentMode,
  enrollmentChargeMaterials,
  enrollmentLessonPrice,
  enrollmentSubscriptionPrice,
  enrollmentNote,
  enrollmentSettingsOpen,
  formatEUR,
  onNameChange,
  onPersonalCodeChange,
  onPhoneChange,
  onEmailChange,
  onNoteChange,
  onIsMinorChange,
  onPayerNameChange,
  onPayerRoleChange,
  onCourseIdChange,
  onEnrollmentModeChange,
  onEnrollmentChargeMaterialsChange,
  onEnrollmentLessonPriceChange,
  onEnrollmentSubscriptionPriceChange,
  onEnrollmentNoteChange,
  onEnrollmentSettingsOpenChange,
  onSave,
  onSaveAndAddAnother,
  onCancel,
  duplicateCheckResult,
  onOpenExistingStudent,
  onEnrollExistingStudent,
  onCreateAnyway,
  t,
}: StudentFormModalProps) {
  const exactMatch = duplicateCheckResult?.exactMatch;
  const possibleMatches = duplicateCheckResult?.possibleMatches ?? [];
  const selectedCourse = allCourses.find((course) => course.id === courseId);
  const effectivePrice = Number(
    enrollmentMode === "per_lesson"
      ? enrollmentLessonPrice
      : enrollmentSubscriptionPrice
  );

  function courseOptionLabel(course: CourseDTO): string {
    const typeLabel = courseTypeLabel(course.type, t);
    return course.teacherName
      ? `${course.name} — ${typeLabel} — ${course.teacherName}`
      : `${course.name} — ${typeLabel}`;
  }

  return (
    <div className="modal">
      <div className="modalBody studentFormModalBody">
        <h3>{editing ? t("modal.editStudent") : t("modal.addStudent")}</h3>
        <div className="studentFormGrid">
          <div className="formRow studentFormFieldWide">
            <label>{t("field.name")}</label>
            <input value={name} onChange={(e) => onNameChange(e.target.value)} />
          </div>
          <div className="formRow">
            <label>{t("field.personalCode")}</label>
            <input value={personalCode} onChange={(e) => onPersonalCodeChange(e.target.value)} />
          </div>
          <div className="formRow">
            <label>{isMinor ? t("student.parentPhone") : t("field.phone")}</label>
            <input value={phone} onChange={(e) => onPhoneChange(e.target.value)} />
          </div>
          <div className="formRow">
            <label>{isMinor ? t("student.parentEmail") : t("field.email")}</label>
            <input value={email} onChange={(e) => onEmailChange(e.target.value)} />
          </div>
          <div className="formRow">
            <label>{t("field.note")}</label>
            <input value={note} onChange={(e) => onNoteChange(e.target.value)} />
          </div>
        </div>
        <div className="formRow">
          <label>{t("field.studentType")}</label>
          <label className="inline">
            <input
              type="checkbox"
              checked={isMinor}
              onChange={(e) => onIsMinorChange(e.target.checked)}
            />
            {t("student.minor")}
          </label>
        </div>
        {isMinor && (
          <div className="studentFormGrid studentPayerGrid">
            <div className="formRow">
              <label>{t("field.payerName")}</label>
              <input value={payerName} onChange={(e) => onPayerNameChange(e.target.value)} />
            </div>
            <div className="formRow">
              <label>{t("field.payerRole")}</label>
              <select value={payerRole} onChange={(e) => onPayerRoleChange(e.target.value)}>
                <option value="">{t("filter.selectRole")}</option>
                {payerRoleOptions.map((role) => (
                  <option key={role} value={role}>
                    {payerRoleLabel(role)}
                  </option>
                ))}
              </select>
            </div>
          </div>
        )}

        {!editing && (
          <section className="studentOnboardingSection">
            <div className="studentOnboardingHeader">
              <div>
                <div className="duplicateAlertEyebrow">{t("label.quickOnboarding")}</div>
                <strong>{t("field.courseOrGroup")}</strong>
              </div>
              <span>{t("label.optional")}</span>
            </div>
            <div className="formRow">
              <label>{t("field.course")}</label>
              <select value={courseId || ""} onChange={(e) => onCourseIdChange(Number(e.target.value))}>
                <option value="">{t("filter.noCourseYet")}</option>
                {allCourses.map((course) => (
                  <option key={course.id} value={course.id}>
                    {courseOptionLabel(course)}
                  </option>
                ))}
              </select>
            </div>

            {selectedCourse && (
              <>
                <div className="studentOnboardingSummary">
                  <div>
                    <span>{t("field.billing")}</span>
                    <strong>
                      {enrollmentMode === "per_lesson"
                        ? t("billing.perLesson")
                        : t("billing.subscription")}
                    </strong>
                  </div>
                  <div>
                    <span>{t("field.price")}</span>
                    <strong>{formatEUR(Number.isFinite(effectivePrice) ? effectivePrice : 0)}</strong>
                  </div>
                  <div>
                    <span>{t("field.chargeMaterials")}</span>
                    <strong>{enrollmentChargeMaterials ? t("label.yes") : t("label.no")}</strong>
                  </div>
                </div>
                <button
                  type="button"
                  className="secondaryActionButton studentOnboardingToggle"
                  onClick={() => onEnrollmentSettingsOpenChange(!enrollmentSettingsOpen)}
                >
                  {enrollmentSettingsOpen
                    ? t("button.hideEnrollmentSettings")
                    : t("button.changeEnrollmentSettings")}
                </button>
                {enrollmentSettingsOpen && (
                  <div className="studentOnboardingAdvanced">
                    <div className="formRow">
                      <label>{t("field.billing")}</label>
                      <select
                        value={enrollmentMode}
                        onChange={(e) =>
                          onEnrollmentModeChange(e.target.value as EnrollmentDTO["billingMode"])
                        }
                      >
                        <option value="per_lesson">{t("billing.perLesson")}</option>
                        <option value="subscription">{t("billing.subscription")}</option>
                      </select>
                    </div>
                    <div className="formRow">
                      <label>
                        {enrollmentMode === "per_lesson"
                          ? t("field.lessonPriceOverride")
                          : t("field.subscriptionLessonPrice")} (EUR)
                      </label>
                      <input
                        type="number"
                        min={0}
                        step="0.1"
                        value={
                          enrollmentMode === "per_lesson"
                            ? enrollmentLessonPrice
                            : enrollmentSubscriptionPrice
                        }
                        onFocus={(e) => e.currentTarget.select()}
                        onChange={(e) =>
                          enrollmentMode === "per_lesson"
                            ? onEnrollmentLessonPriceChange(e.target.value)
                            : onEnrollmentSubscriptionPriceChange(e.target.value)
                        }
                      />
                    </div>
                    <div className="formRow">
                      <label>{t("field.chargeMaterials")}</label>
                      <input
                        className="formCheckbox"
                        type="checkbox"
                        checked={enrollmentChargeMaterials}
                        onChange={(e) => onEnrollmentChargeMaterialsChange(e.target.checked)}
                      />
                    </div>
                    <div className="formRow">
                      <label>{t("field.enrollmentNote")}</label>
                      <input value={enrollmentNote} onChange={(e) => onEnrollmentNoteChange(e.target.value)} />
                    </div>
                  </div>
                )}
              </>
            )}
          </section>
        )}

        {(exactMatch || possibleMatches.length > 0) && (
          <section className="duplicateAlert">
            <div className="duplicateAlertHeader">
              <div className="duplicateAlertEyebrow">{t("field.warning")}</div>
              <div className="duplicateAlertTitle">
                {exactMatch ? t("student.duplicateExactTitle") : t("student.duplicatePossibleTitle")}
              </div>
              <p className="duplicateAlertText">
                {exactMatch ? t("msg.studentDuplicateExact") : t("msg.studentDuplicatePossible")}
              </p>
            </div>

            <div className="duplicateMatchList">
              {(exactMatch ? [exactMatch] : possibleMatches).map((student) => (
                <article key={student.id} className="duplicateMatchCard">
                  <div className="duplicateMatchMain">
                    <div className="duplicateMatchName">{student.fullName}</div>
                    <div className="duplicateMatchMeta">
                      {[student.personalCode, student.phone, student.email].filter(Boolean).join(" · ")}
                    </div>
                    <div className="duplicateMatchStatus">
                      {student.isActive ? t("student.statusActive") : t("student.statusInactive")}
                    </div>
                  </div>
                  <div className="duplicateMatchActions">
                    <button
                      type="button"
                      onClick={() =>
                        courseId && onEnrollExistingStudent
                          ? onEnrollExistingStudent(student)
                          : onOpenExistingStudent(student.id)
                      }
                    >
                      {courseId && onEnrollExistingStudent
                        ? t("button.enrollExistingStudent")
                        : t("button.openStudent")}
                    </button>
                  </div>
                </article>
              ))}
            </div>

            {!exactMatch && possibleMatches.length > 0 && (
              <div className="duplicateAlertFooter">
                <button type="button" onClick={onCreateAnyway}>
                  {t("button.createAnyway")}
                </button>
              </div>
            )}
          </section>
        )}

        <div className="modalActions">
          <button onClick={onSave}>
            {!editing && courseId ? t("button.createAndOpenAttendance") : t("button.save")}
          </button>
          {!editing && onSaveAndAddAnother && (
            <button onClick={onSaveAndAddAnother}>{t("button.saveAndAddAnother")}</button>
          )}
          <button onClick={onCancel}>{t("button.cancel")}</button>
        </div>
      </div>
    </div>
  );
}
