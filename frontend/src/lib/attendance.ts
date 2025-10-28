// frontend/src/lib/attendance.ts
import { AttendanceListPerLesson, 
    AttendanceUpsert,
     AttendanceAddOne,
     AttendanceEstimate, 
     AttendanceSetLocked } from "../../wailsjs/go/main/App";

  export async function fetchRows(year:number, month:number, courseId?:number) {
    const cid = courseId && courseId>0 ? courseId : undefined;
    return await AttendanceListPerLesson(year, month, cid);
  }
  
  export async function saveCount(studentId:number, courseId:number, year:number, month:number, count:number) {
    await AttendanceUpsert(studentId, courseId, year, month, count);
  }
  
  export async function addOneMass(year:number, month:number, courseId?:number) {
    const cid = courseId && courseId>0 ? courseId : undefined;
    return await AttendanceAddOne(year, month, cid);
  }
  
  export async function estimateBySchedule(year:number, month:number, courseId?:number) {
    const cid = courseId && courseId>0 ? courseId : undefined;
    return await AttendanceEstimate(year, month, cid);
  }
  
  export async function setLocked(year:number, month:number, courseId:number|undefined, lock:boolean) {
    const cid = courseId && courseId>0 ? courseId : undefined;
    return await AttendanceSetLocked(year, month, cid, lock);
  }
  