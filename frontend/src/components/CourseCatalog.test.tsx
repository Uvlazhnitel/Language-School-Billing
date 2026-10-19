import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { CourseCatalog } from "./CourseCatalog";
import { groupCoursesBySubject } from "../lib/courseGroups";
import { createTranslator, type UiLocale } from "../lib/i18n";
import type { CourseDTO } from "../lib/courses";

const courses: CourseDTO[] = [
  {
    id: 1,
    version: 1,
    name: "English",
    teacherId: 1,
    teacherName: "Ilja",
    type: "individual",
    lessonPrice: 30,
    subscriptionPrice: 0,
  },
  {
    id: 2,
    version: 1,
    name: "English",
    teacherId: 1,
    teacherName: "Ilja",
    type: "group",
    lessonPrice: 15,
    subscriptionPrice: 12.5,
  },
  {
    id: 3,
    version: 1,
    name: "Biology",
    teacherName: "",
    type: "individual",
    lessonPrice: 0,
    subscriptionPrice: 0,
  },
];

function render(openKey: string | null, canDeleteCourses = true, locale: UiLocale = "en-US") {
  const t = createTranslator(locale);
  return renderToStaticMarkup(
    <CourseCatalog
      groups={groupCoursesBySubject(courses)}
      openKey={openKey}
      onToggle={vi.fn()}
      canDeleteCourses={canDeleteCourses}
      courseTypeLabel={(type) => t(`course.${type}`)}
      formatEUR={(value) => `€${value.toFixed(2)}`}
      onEditCourse={vi.fn()}
      onDeleteCourse={vi.fn()}
      t={t}
    />
  );
}

describe("course catalog rendering", () => {
  it("shows each subject once, with counts and accessible collapsed controls", () => {
    const html = render(null);
    expect(html.match(/class="courseSubject"/g)).toHaveLength(2);
    expect(html.match(/aria-expanded="false"/g)).toHaveLength(2);
    expect(html).toContain("Teachers: 1 · Variants: 2");
    expect(html).not.toContain("<table");
    expect(html).toContain("aria-controls=");
    expect(html.match(/hidden=""/g)).toHaveLength(2);
  });
  it("shows just the open subject's variants, both prices and mobile field labels", () => {
    const html = render("english");
    expect(html.match(/aria-expanded="true"/g)).toHaveLength(1);
    expect(html.match(/<table/g)).toHaveLength(1);
    expect(html.match(/>Edit<\/button>/g)).toHaveLength(2);
    expect(html.match(/>Delete<\/button>/g)).toHaveLength(2);
    expect(html).toContain('data-label="Hourly rate"');
    expect(html).toContain("€30.00");
    expect(html).toContain("€12.50");
    expect(html).toContain("€0.00");
  });
  it("keeps editing available without deletion permission", () => {
    const html = render("english", false);
    expect(html).toContain(">Edit</button>");
    expect(html).not.toContain(">Delete</button>");
  });
  it("labels unassigned teachers and retains zero prices", () => {
    const html = render("biology");
    expect(html).toContain(createTranslator("en-US")("filter.withoutTeacher"));
    expect(html.match(/€0.00/g)).toHaveLength(2);
  });
  it.each(["en-US", "ru-RU", "lv-LV"] as const)("translates catalog labels in %s", (locale) => {
    const html = render("english", true, locale);
    expect(html).not.toContain("course.subjectCounts");
    expect(html).not.toContain("course.actions");
    expect(html).not.toContain("{teachers}");
  });
});
