import {
    EnrollmentList,
    EnrollmentCreate,
    EnrollmentUpdate,
} from "../../wailsjs/go/main/App";
  
  export type EnrollmentDTO = {
    id: number;
    studentId: number;
    studentName: string;
    courseId: number;
    courseName: string;
    billingMode: "subscription" | "per_lesson";
    discountPct: number;
    note: string;
  };
  
  export async function listEnrollments(
    studentId?: number,
    courseId?: number,
    activeOnly: boolean = true
  ) {
    // Wails maps nil pointers as null.
    return (await EnrollmentList(studentId ?? null, courseId ?? null, activeOnly)) as EnrollmentDTO[];
  }
  
  export async function createEnrollment(
    studentId: number,
    courseId: number,
    billingMode: "subscription" | "per_lesson",
    discountPct: number,
    note: string
  ) {
    return (await EnrollmentCreate(
      studentId,
      courseId,
      billingMode,
      discountPct,
      note
    )) as EnrollmentDTO;
  }
  
  export async function updateEnrollment(
    enrollmentId: number,
    billingMode: "subscription" | "per_lesson",
    discountPct: number,
    note: string
  ) {
    return (await EnrollmentUpdate(
      enrollmentId,
      billingMode,
      discountPct,
      note
    )) as EnrollmentDTO;
  }
  