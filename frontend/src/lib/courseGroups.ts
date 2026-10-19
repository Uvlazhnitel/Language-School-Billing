import type { CourseDTO } from "./courses";

export type CourseGroup = {
  key: string;
  name: string;
  courses: CourseDTO[];
  teacherCount: number;
};

export function courseGroupKey(name: string): string {
  return name.trim().toLowerCase();
}

export function groupCoursesBySubject(courses: CourseDTO[]): CourseGroup[] {
  const groups = new Map<string, CourseGroup>();
  for (const course of courses) {
    const key = courseGroupKey(course.name);
    let group = groups.get(key);
    if (!group) {
      group = { key, name: course.name.trim(), courses: [], teacherCount: 0 };
      groups.set(key, group);
    }
    group.courses.push(course);
  }
  for (const group of groups.values()) {
    group.courses.sort(
      (a, b) =>
        a.teacherName.localeCompare(b.teacherName) || a.type.localeCompare(b.type) || a.id - b.id
    );
    group.teacherCount = new Set(
      group.courses.flatMap((course) => {
        if (course.teacherId) return [`id:${course.teacherId}`];
        const name = course.teacherName.trim().toLowerCase();
        return name ? [`name:${name}`] : [];
      })
    ).size;
  }
  return [...groups.values()].sort((a, b) => a.name.localeCompare(b.name));
}

export type CourseDisclosure = {
  filterKey: string;
  savedCourse: CourseDTO | null;
  openKey: string | null;
};

export function resolveCourseDisclosure(
  previous: CourseDisclosure,
  filterKey: string,
  savedCourse: CourseDTO | null,
  groups: CourseGroup[]
): CourseDisclosure {
  let openKey = previous.openKey;
  if (previous.filterKey !== filterKey) openKey = groups[0]?.key ?? null;
  if (previous.savedCourse !== savedCourse && savedCourse) {
    const savedGroup = groups.find((group) =>
      group.courses.some((course) => course.id === savedCourse.id)
    );
    if (savedGroup) openKey = savedGroup.key;
  }
  if (!groups.some((group) => group.key === openKey)) openKey = null;
  return { filterKey, savedCourse, openKey };
}
