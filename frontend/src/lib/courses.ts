import {
    CourseList,
    CourseGet,
    CourseCreate,
    CourseUpdate,
    CourseDelete,
} from "../../wailsjs/go/main/App";
  
  export type CourseDTO = {
    id: number;
    name: string;
    type: "group" | "individual";
    lessonPrice: number;
    subscriptionPrice: number;
    scheduleDays: number[]; // time.Weekday: Sunday=0, Monday=1, ...
  };
  
  export async function listCourses(q: string) {
    return (await CourseList(q)) as CourseDTO[];
  }
  
  export async function getCourse(id: number) {
    return (await CourseGet(id)) as CourseDTO;
  }
  
  export async function createCourse(
    name: string,
    courseType: "group" | "individual",
    lessonPrice: number,
    subscriptionPrice: number,
    scheduleDays: number[]
  ) {
    return (await CourseCreate(name, courseType, lessonPrice, subscriptionPrice, scheduleDays)) as CourseDTO;
  }
  
  export async function updateCourse(
    id: number,
    name: string,
    courseType: "group" | "individual",
    lessonPrice: number,
    subscriptionPrice: number,
    scheduleDays: number[]
  ) {
    return (await CourseUpdate(id, name, courseType, lessonPrice, subscriptionPrice, scheduleDays)) as CourseDTO;
  }
  
  export async function deleteCourse(id: number) {
    return await CourseDelete(id);
  }
  