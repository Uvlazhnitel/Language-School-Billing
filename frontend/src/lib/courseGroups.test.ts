import { describe, expect, it } from "vitest";
import { groupCoursesBySubject, resolveCourseDisclosure } from "./courseGroups";
import type { CourseDTO } from "./courses";

const courseFixtures: CourseDTO[] = [
  {
    id: 1,
    version: 1,
    name: " English ",
    teacherId: 2,
    teacherName: "Milena",
    type: "individual",
    lessonPrice: 30,
    subscriptionPrice: 0,
  },
  {
    id: 2,
    version: 1,
    name: "english",
    teacherId: 1,
    teacherName: "Ilja",
    type: "individual",
    lessonPrice: 30,
    subscriptionPrice: 0,
  },
  {
    id: 3,
    version: 1,
    name: "English",
    teacherId: 1,
    teacherName: "Ilja",
    type: "group",
    lessonPrice: 15,
    subscriptionPrice: 12.5,
  },
  {
    id: 4,
    version: 1,
    name: "Biology",
    teacherName: "",
    type: "individual",
    lessonPrice: 0,
    subscriptionPrice: 0,
  },
];

describe("subject grouping", () => {
  it("combines case and edge-space variants, sorts, counts unique teachers and preserves IDs", () => {
    const groups = groupCoursesBySubject(courseFixtures);
    expect(groups.map((g) => g.name)).toEqual(["Biology", "English"]);
    expect(groups[1].courses.map((c) => c.id)).toEqual([3, 2, 1]);
    expect(groups[1].teacherCount).toBe(2);
    expect(groups[0].teacherCount).toBe(0);
    expect(courseFixtures.map((c) => c.id)).toEqual([1, 2, 3, 4]);
  });

  it("counts only filtered variants and handles an empty result", () => {
    const groups = groupCoursesBySubject(
      courseFixtures.filter(
        (c) => c.teacherId === 1 && c.type === "group" && c.subscriptionPrice > 0
      )
    );
    expect(groups).toHaveLength(1);
    expect(groups[0].courses.map((c) => c.id)).toEqual([3]);
    expect(groups[0].teacherCount).toBe(1);
    expect(groupCoursesBySubject([])).toEqual([]);
  });
});

describe("subject disclosure", () => {
  const groups = groupCoursesBySubject(courseFixtures);
  const initial = { filterKey: "all", savedCourse: null, openKey: null };
  it("starts collapsed, opens the first filtered subject, and respects manually closing it", () => {
    expect(resolveCourseDisclosure(initial, "all", null, groups).openKey).toBeNull();
    const filtered = resolveCourseDisclosure(initial, "filtered", null, groups);
    expect(filtered.openKey).toBe("biology");
    expect(
      resolveCourseDisclosure({ ...filtered, openKey: null }, "filtered", null, groups).openKey
    ).toBeNull();
  });
  it("opens the saved subject including after renaming, without reopening on ordinary refresh", () => {
    const renamed = { ...courseFixtures[0], name: "Zoology" };
    const updatedGroups = groupCoursesBySubject([renamed, courseFixtures[3]]);
    const saved = resolveCourseDisclosure(initial, "all", renamed, updatedGroups);
    expect(saved.openKey).toBe("zoology");
    expect(
      resolveCourseDisclosure({ ...saved, openKey: null }, "all", renamed, updatedGroups).openKey
    ).toBeNull();
  });
  it("does not reveal a saved variant excluded by filters and clears a deleted subject", () => {
    const previous = { ...initial, openKey: "biology" };
    expect(resolveCourseDisclosure(previous, "all", courseFixtures[0], [groups[0]]).openKey).toBe(
      "biology"
    );
    expect(resolveCourseDisclosure(previous, "all", null, [groups[1]]).openKey).toBeNull();
  });
});
