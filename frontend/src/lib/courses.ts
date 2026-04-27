import {
  CourseList,
  CourseGet,
  CourseCreate,
  CourseUpdate,
  CourseDelete,
} from "../../wailsjs/go/main/App";
import { CourseType } from "./constants";

export type CourseDTO = {
  id: number;
  name: string;
  teacherId?: number;
  teacherName: string;
  type: CourseType;
  lessonPrice: number;
  subscriptionPrice: number;
};

export async function listCourses(q: string) {
  return (await CourseList(q)) as CourseDTO[];
}

export async function getCourse(id: number) {
  return (await CourseGet(id)) as CourseDTO;
}

export async function createCourse(
  name: string,
  teacherId: number | undefined,
  courseType: CourseType,
  lessonPrice: number,
  subscriptionPrice: number
) {
  const teacher = typeof teacherId === "number" && teacherId > 0 ? teacherId : undefined;
  return (await CourseCreate(
    name,
    teacher,
    courseType,
    lessonPrice,
    subscriptionPrice
  )) as CourseDTO;
}

export async function updateCourse(
  id: number,
  name: string,
  teacherId: number | undefined,
  courseType: CourseType,
  lessonPrice: number,
  subscriptionPrice: number
) {
  const teacher = typeof teacherId === "number" && teacherId > 0 ? teacherId : undefined;
  return (await CourseUpdate(
    id,
    name,
    teacher,
    courseType,
    lessonPrice,
    subscriptionPrice
  )) as CourseDTO;
}

export async function deleteCourse(id: number) {
  return await CourseDelete(id);
}
