import type { EnrollmentDTO } from "./enrollments";

export const ENROLLMENT_SELECTION_STORAGE_KEY = "langschool.enrollment-selection.v1";
export const UNASSIGNED_TEACHER_KEY = "teacher:unassigned";

export type EnrollmentCourseGroup = {
  courseId: number;
  courseName: string;
  courseType: string;
  enrollments: EnrollmentDTO[];
};

export type EnrollmentTeacherGroup = {
  key: string;
  teacherId?: number;
  teacherName: string;
  courses: EnrollmentCourseGroup[];
  studentCount: number;
};

export type EnrollmentSelectionPreferences = {
  activeTeacherKey?: string;
  courseByTeacher: Record<string, number>;
};

export type ResolvedEnrollmentSelection = {
  teacherKey?: string;
  courseId?: number;
};

function teacherKey(enrollment: EnrollmentDTO): string {
  return enrollment.teacherId === undefined
    ? UNASSIGNED_TEACHER_KEY
    : `teacher:${enrollment.teacherId}`;
}

export function groupEnrollmentsByTeacher(
  enrollments: EnrollmentDTO[],
  unassignedTeacherLabel: string
): EnrollmentTeacherGroup[] {
  const teachers = new Map<string, EnrollmentTeacherGroup>();

  for (const enrollment of enrollments) {
    const key = teacherKey(enrollment);
    let teacher = teachers.get(key);
    if (!teacher) {
      teacher = {
        key,
        teacherId: enrollment.teacherId,
        teacherName:
          enrollment.teacherId === undefined
            ? unassignedTeacherLabel
            : enrollment.teacherName || unassignedTeacherLabel,
        courses: [],
        studentCount: 0,
      };
      teachers.set(key, teacher);
    }

    let course = teacher.courses.find((candidate) => candidate.courseId === enrollment.courseId);
    if (!course) {
      course = {
        courseId: enrollment.courseId,
        courseName: enrollment.courseName,
        courseType: enrollment.courseType,
        enrollments: [],
      };
      teacher.courses.push(course);
    }
    course.enrollments.push(enrollment);
  }

  const result = Array.from(teachers.values());
  for (const teacher of result) {
    teacher.courses.sort((left, right) => left.courseName.localeCompare(right.courseName));
    for (const course of teacher.courses) {
      course.enrollments.sort((left, right) => left.studentName.localeCompare(right.studentName));
    }
    teacher.studentCount = new Set(
      teacher.courses.flatMap((course) =>
        course.enrollments.map((enrollment) => enrollment.studentId)
      )
    ).size;
  }

  return result.sort((left, right) => {
    if (left.key === UNASSIGNED_TEACHER_KEY) return 1;
    if (right.key === UNASSIGNED_TEACHER_KEY) return -1;
    return left.teacherName.localeCompare(right.teacherName);
  });
}

export function parseEnrollmentSelectionPreferences(
  rawValue: string | null
): EnrollmentSelectionPreferences {
  const fallback: EnrollmentSelectionPreferences = { courseByTeacher: {} };
  if (!rawValue) return fallback;

  try {
    const parsed = JSON.parse(rawValue) as unknown;
    if (!parsed || typeof parsed !== "object") return fallback;

    const value = parsed as {
      activeTeacherKey?: unknown;
      courseByTeacher?: unknown;
    };
    const courseByTeacher: Record<string, number> = {};
    if (value.courseByTeacher && typeof value.courseByTeacher === "object") {
      for (const [key, courseId] of Object.entries(value.courseByTeacher)) {
        if (typeof courseId === "number" && Number.isInteger(courseId) && courseId > 0) {
          courseByTeacher[key] = courseId;
        }
      }
    }

    return {
      activeTeacherKey:
        typeof value.activeTeacherKey === "string" ? value.activeTeacherKey : undefined,
      courseByTeacher,
    };
  } catch {
    return fallback;
  }
}

export function normalizeEnrollmentSelectionPreferences(
  groups: EnrollmentTeacherGroup[],
  preferences: EnrollmentSelectionPreferences
): EnrollmentSelectionPreferences {
  if (groups.length === 0) return { courseByTeacher: {} };

  const activeTeacher =
    groups.find((group) => group.key === preferences.activeTeacherKey) ?? groups[0];
  const courseByTeacher = { ...preferences.courseByTeacher };

  for (const group of groups) {
    const savedCourseId = courseByTeacher[group.key];
    if (!group.courses.some((course) => course.courseId === savedCourseId)) {
      courseByTeacher[group.key] = group.courses[0].courseId;
    }
  }

  return {
    activeTeacherKey: activeTeacher.key,
    courseByTeacher,
  };
}

export function resolveEnrollmentSelection(
  groups: EnrollmentTeacherGroup[],
  preferences: EnrollmentSelectionPreferences
): ResolvedEnrollmentSelection {
  if (groups.length === 0) return {};

  const teacher = groups.find((group) => group.key === preferences.activeTeacherKey) ?? groups[0];
  const savedCourseId = preferences.courseByTeacher[teacher.key];
  const course =
    teacher.courses.find((candidate) => candidate.courseId === savedCourseId) ?? teacher.courses[0];

  return { teacherKey: teacher.key, courseId: course.courseId };
}
