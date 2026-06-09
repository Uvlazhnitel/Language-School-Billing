import { CourseType } from "./constants";
import { getTransport } from "./api";
export type { CourseDTO } from "./api";

export async function listCourses(q: string) {
  const transport = await getTransport();
  return transport.listCourses(q);
}

export async function getCourse(id: number) {
  const transport = await getTransport();
  return transport.getCourse(id);
}

export async function createCourse(
  name: string,
  teacherId: number | undefined,
  courseType: CourseType,
  lessonPrice: number,
  subscriptionPrice: number
) {
  const transport = await getTransport();
  return transport.createCourse(name, teacherId, courseType, lessonPrice, subscriptionPrice);
}

export async function updateCourse(
  id: number,
  version: number,
  name: string,
  teacherId: number | undefined,
  courseType: CourseType,
  lessonPrice: number,
  subscriptionPrice: number
) {
  const transport = await getTransport();
  return transport.updateCourse(
    id,
    version,
    name,
    teacherId,
    courseType,
    lessonPrice,
    subscriptionPrice
  );
}

export async function deleteCourse(id: number, version: number) {
  const transport = await getTransport();
  return transport.deleteCourse(id, version);
}
