import type { TranslateFn } from "../../lib/i18n";
import type { CourseDTO } from "../../lib/courses";
import type { EnrollmentDTO } from "../../lib/enrollments";
import type { StudentDTO, StudentDuplicateCheckResult } from "../../lib/students";
import { courseTypeLabel } from "../../lib/appUi";
import type {
  StudentOnboardingEnrollmentRow,
  StudentOnboardingEnrollmentRowPatch,
} from "../../lib/studentOnboarding";

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
  enrollmentRows: StudentOnboardingEnrollmentRow[];
  formatEUR: (value: number) => string;
  onNameChange: (value: string) => void;
  onPersonalCodeChange: (value: string) => void;
  onPhoneChange: (value: string) => void;
  onEmailChange: (value: string) => void;
  onNoteChange: (value: string) => void;
  onIsMinorChange: (value: boolean) => void;
  onPayerNameChange: (value: string) => void;
  onPayerRoleChange: (value: string) => void;
  onAddEnrollmentRow: () => void;
  onRemoveEnrollmentRow: (rowId: number) => void;
  onEnrollmentCourseChange: (rowId: number, courseId: number) => void;
  onEnrollmentModeChange: (rowId: number, value: EnrollmentDTO["billingMode"]) => void;
  onEnrollmentRowChange: (
    rowId: number,
    patch: StudentOnboardingEnrollmentRowPatch
  ) => void;
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
  enrollmentRows,
  formatEUR,
  onNameChange,
  onPersonalCodeChange,
  onPhoneChange,
  onEmailChange,
  onNoteChange,
  onIsMinorChange,
  onPayerNameChange,
  onPayerRoleChange,
  onAddEnrollmentRow,
  onRemoveEnrollmentRow,
  onEnrollmentCourseChange,
  onEnrollmentModeChange,
  onEnrollmentRowChange,
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
  const selectedCourseIds = enrollmentRows
    .map((row) => row.courseId)
    .filter((courseId) => courseId > 0);
  const hasSelectedCourses = selectedCourseIds.length > 0;
  const canAddEnrollmentRow =
    enrollmentRows.length < allCourses.length && enrollmentRows.every((row) => row.courseId > 0);

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
            <div className="studentOnboardingRows">
              {enrollmentRows.map((row, index) => {
                const selectedCourse = allCourses.find((course) => course.id === row.courseId);
                const effectivePrice = Number(
                  row.billingMode === "per_lesson" ? row.lessonPrice : row.subscriptionPrice
                );
                return (
                  <div className="studentOnboardingRow" key={row.id}>
                    <div className="studentOnboardingRowHeader">
                      <strong>{t("label.courseNumber", { number: index + 1 })}</strong>
                      {index > 0 && (
                        <button
                          type="button"
                          className="secondaryActionButton studentOnboardingRemove"
                          onClick={() => onRemoveEnrollmentRow(row.id)}
                        >
                          {t("button.removeCourse")}
                        </button>
                      )}
                    </div>
                    <div className="formRow">
                      <label>{t("field.course")}</label>
                      <select
                        value={row.courseId || ""}
                        onChange={(e) =>
                          onEnrollmentCourseChange(row.id, Number(e.target.value))
                        }
                      >
                        <option value="">{t("filter.noCourseYet")}</option>
                        {allCourses.map((course) => (
                          <option
                            key={course.id}
                            value={course.id}
                            disabled={
                              course.id !== row.courseId && selectedCourseIds.includes(course.id)
                            }
                          >
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
                              {row.billingMode === "per_lesson"
                                ? t("billing.perLesson")
                                : t("billing.subscription")}
                            </strong>
                          </div>
                          <div>
                            <span>{t("field.price")}</span>
                            <strong>
                              {formatEUR(Number.isFinite(effectivePrice) ? effectivePrice : 0)}
                            </strong>
                          </div>
                          <div>
                            <span>{t("field.chargeMaterials")}</span>
                            <strong>{row.chargeMaterials ? t("label.yes") : t("label.no")}</strong>
                          </div>
                        </div>
                        <button
                          type="button"
                          className="secondaryActionButton studentOnboardingToggle"
                          onClick={() =>
                            onEnrollmentRowChange(row.id, {
                              settingsOpen: !row.settingsOpen,
                            })
                          }
                        >
                          {row.settingsOpen
                            ? t("button.hideEnrollmentSettings")
                            : t("button.changeEnrollmentSettings")}
                        </button>
                        {row.settingsOpen && (
                          <div className="studentOnboardingAdvanced">
                            <div className="formRow">
                              <label>{t("field.billing")}</label>
                              <select
                                value={row.billingMode}
                                onChange={(e) =>
                                  onEnrollmentModeChange(
                                    row.id,
                                    e.target.value as EnrollmentDTO["billingMode"]
                                  )
                                }
                              >
                                <option value="per_lesson">{t("billing.perLesson")}</option>
                                <option value="subscription">{t("billing.subscription")}</option>
                              </select>
                            </div>
                            <div className="formRow">
                              <label>
                                {row.billingMode === "per_lesson"
                                  ? t("field.lessonPriceOverride")
                                  : t("field.subscriptionLessonPrice")} (EUR)
                              </label>
                              <input
                                type="number"
                                min={0}
                                step="0.1"
                                value={
                                  row.billingMode === "per_lesson"
                                    ? row.lessonPrice
                                    : row.subscriptionPrice
                                }
                                onFocus={(e) => e.currentTarget.select()}
                                onChange={(e) =>
                                  onEnrollmentRowChange(
                                    row.id,
                                    row.billingMode === "per_lesson"
                                      ? { lessonPrice: e.target.value }
                                      : { subscriptionPrice: e.target.value }
                                  )
                                }
                              />
                            </div>
                            <div className="formRow">
                              <label>{t("field.chargeMaterials")}</label>
                              <input
                                className="formCheckbox"
                                type="checkbox"
                                checked={row.chargeMaterials}
                                onChange={(e) =>
                                  onEnrollmentRowChange(row.id, {
                                    chargeMaterials: e.target.checked,
                                  })
                                }
                              />
                            </div>
                            <div className="formRow">
                              <label>{t("field.enrollmentNote")}</label>
                              <input
                                value={row.note}
                                onChange={(e) =>
                                  onEnrollmentRowChange(row.id, { note: e.target.value })
                                }
                              />
                            </div>
                          </div>
                        )}
                      </>
                    )}
                  </div>
                );
              })}
            </div>
            {canAddEnrollmentRow && (
              <button
                type="button"
                className="secondaryActionButton studentOnboardingAdd"
                onClick={onAddEnrollmentRow}
              >
                {t("button.addAnotherCourse")}
              </button>
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
                        hasSelectedCourses && onEnrollExistingStudent
                          ? onEnrollExistingStudent(student)
                          : onOpenExistingStudent(student.id)
                      }
                    >
                      {hasSelectedCourses && onEnrollExistingStudent
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
            {!editing && hasSelectedCourses
              ? t("button.createAndOpenAttendance")
              : t("button.save")}
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
