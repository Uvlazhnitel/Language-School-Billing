import { CourseTypeIndividual, type CourseType } from "./constants";

export function defaultLessonPriceForCourseType(courseType: CourseType): string {
  return courseType === CourseTypeIndividual ? "25.00" : "15.00";
}
