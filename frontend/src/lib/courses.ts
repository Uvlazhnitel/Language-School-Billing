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
  teacherName: string,
  courseType: CourseType,
  lessonPrice: number,
  subscriptionPrice: number
) {
  return (await CourseCreate(
    name,
    teacherName,
    courseType,
    lessonPrice,
    subscriptionPrice
  )) as CourseDTO;
}

export async function updateCourse(
  id: number,
  name: string,
  teacherName: string,
  courseType: CourseType,
  lessonPrice: number,
  subscriptionPrice: number
) {
  return (await CourseUpdate(
    id,
    name,
    teacherName,
    courseType,
    lessonPrice,
    subscriptionPrice
  )) as CourseDTO;
}

export async function deleteCourse(id: number) {
  return await CourseDelete(id);
}
