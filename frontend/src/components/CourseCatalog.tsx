import { useId } from "react";
import type { CourseGroup } from "../lib/courseGroups";
import type { CourseDTO } from "../lib/courses";
import type { TranslateFn } from "../lib/i18n";

type CourseCatalogProps = {
  groups: CourseGroup[];
  openKey: string | null;
  onToggle: (key: string) => void;
  canDeleteCourses: boolean;
  courseTypeLabel: (type: string) => string;
  formatEUR: (value: number) => string;
  onEditCourse: (course: CourseDTO) => void;
  onDeleteCourse: (id: number) => void | Promise<void>;
  t: TranslateFn;
};

export function CourseCatalog({
  groups,
  openKey,
  onToggle,
  canDeleteCourses,
  courseTypeLabel,
  formatEUR,
  onEditCourse,
  onDeleteCourse,
  t,
}: CourseCatalogProps) {
  const catalogId = useId();
  return (
    <div className="courseCatalog">
      {groups.map((group, index) => {
        const expanded = openKey === group.key;
        const panelId = `${catalogId}-subject-${index}`;
        return (
          <section className="courseSubject" key={group.key}>
            <button
              type="button"
              className="courseSubjectHeader"
              aria-expanded={expanded}
              aria-controls={panelId}
              id={`${panelId}-heading`}
              onClick={() => onToggle(group.key)}
            >
              <span className="courseSubjectChevron" aria-hidden="true">
                {expanded ? "▾" : "▸"}
              </span>
              <strong>{group.name}</strong>
              <span className="courseSubjectCounts">
                {t("course.subjectCounts", {
                  teachers: group.teacherCount,
                  variants: group.courses.length,
                })}
              </span>
            </button>
            <div id={panelId} hidden={!expanded} aria-labelledby={`${panelId}-heading`}>
              {expanded && (
                <table className="courseVariants">
                  <thead>
                    <tr>
                      <th>{t("field.teacher")}</th>
                      <th>{t("field.type")}</th>
                      <th className="coursePrice">{t("field.lessonPrice")} (EUR)</th>
                      <th className="coursePrice">{t("field.subscriptionPrice")} (EUR)</th>
                      <th>
                        <span className="courseVisuallyHidden">{t("course.actions")}</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {group.courses.map((course) => (
                      <tr key={course.id}>
                        <td data-label={t("field.teacher")}>
                          {course.teacherName || t("filter.withoutTeacher")}
                        </td>
                        <td data-label={t("field.type")}>{courseTypeLabel(course.type)}</td>
                        <td className="coursePrice" data-label={t("field.lessonPrice")}>
                          {formatEUR(course.lessonPrice)}
                        </td>
                        <td className="coursePrice" data-label={t("field.subscriptionPrice")}>
                          {formatEUR(course.subscriptionPrice)}
                        </td>
                        <td className="courseVariantActions">
                          <button type="button" onClick={() => onEditCourse(course)}>
                            {t("button.edit")}
                          </button>
                          {canDeleteCourses && (
                            <button type="button" onClick={() => void onDeleteCourse(course.id)}>
                              {t("button.delete")}
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </section>
        );
      })}
    </div>
  );
}
