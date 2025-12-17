import {
    EnrollmentList,
    EnrollmentCreate,
    EnrollmentUpdate,
    EnrollmentEnd,
} from "../../wailsjs/go/main/App";
  
  export type EnrollmentDTO = {
    id: number;
    studentId: number;
    studentName: string;
    courseId: number;
    courseName: string;
    billingMode: "subscription" | "per_lesson";
    startDate: string; // YYYY-MM-DD
    endDate?: string;
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
    startDate: string,
    endDate: string | undefined,
    discountPct: number,
    note: string
  ) {
    return (await EnrollmentCreate(
      studentId,
      courseId,
      billingMode,
      startDate,
      endDate ? endDate : null,
      discountPct,
      note
    )) as EnrollmentDTO;
  }
  
  export async function updateEnrollment(
    enrollmentId: number,
    billingMode: "subscription" | "per_lesson",
    startDate: string,
    endDate: string | undefined,
    discountPct: number,
    note: string
  ) {
    return (await EnrollmentUpdate(
      enrollmentId,
      billingMode,
      startDate,
      endDate ? endDate : null,
      discountPct,
      note
    )) as EnrollmentDTO;
  }
  
  export async function endEnrollment(enrollmentId: number, endDate: string) {
    return await EnrollmentEnd(enrollmentId, endDate);
  }
  