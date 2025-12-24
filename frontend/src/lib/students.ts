import {
    StudentList,
    StudentGet,
    StudentCreate,
    StudentUpdate,
    StudentSetActive,
    StudentDelete,
} from "../../wailsjs/go/main/App";
  
  export type StudentDTO = {
    id: number;
    fullName: string;
    phone: string;
    email: string;
    note: string;
    isActive: boolean;
  };
  
  export async function listStudents(q: string, includeInactive: boolean) {
    return (await StudentList(q, includeInactive)) as StudentDTO[];
  }
  
  export async function getStudent(id: number) {
    return (await StudentGet(id)) as StudentDTO;
  }
  
  export async function createStudent(fullName: string, phone: string, email: string, note: string) {
    return (await StudentCreate(fullName, phone, email, note)) as StudentDTO;
  }
  
  export async function updateStudent(id: number, fullName: string, phone: string, email: string, note: string) {
    return (await StudentUpdate(id, fullName, phone, email, note)) as StudentDTO;
  }
  
  export async function setStudentActive(id: number, active: boolean) {
    return await StudentSetActive(id, active);
  }

  export async function deleteStudent(id: number) {
    return await StudentDelete(id);
  }
  