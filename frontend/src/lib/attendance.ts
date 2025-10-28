import {
  AttendanceListPerLesson,
  AttendanceUpsert,
  AttendanceAddOne,
  AttendanceEstimate,
  AttendanceSetLocked,
  DevSeed
} from "../../wailsjs/go/main/App";


export type Row = {
  studentId: number;
  studentName: string;
  courseId: number;
  courseName: string;
  courseType: "group" | "individual";
  lessonPrice: number;
  count: number;
  locked: boolean;
};

export async function seedDemo() {
  return await DevSeed();
}

export async function fetchRows(year: number, month: number, courseId?: number) {
  const cid: number | undefined = courseId && courseId > 0 ? courseId : undefined;
  return (await AttendanceListPerLesson(year, month, cid)) as Row[];
}

export async function saveCount(studentId: number, courseId: number, year: number, month: number, count: number) {
  await AttendanceUpsert(studentId, courseId, year, month, count);
}

export async function addOneMass(year: number, month: number, courseId?: number) {
  const cid: number | undefined = courseId && courseId > 0 ? courseId : undefined;
  return await AttendanceAddOne(year, month, cid);
}

export async function estimateBySchedule(year: number, month: number, courseId?: number) {
  const cid: number | undefined = courseId && courseId > 0 ? courseId : undefined;
  return (await AttendanceEstimate(year, month, cid)) as Record<string, number>;
}

export async function setLocked(year: number, month: number, courseId: number | undefined, lock: boolean) {
  const cid: number | undefined = courseId && courseId > 0 ? courseId : undefined;
  return await AttendanceSetLocked(year, month, cid, lock);
}
