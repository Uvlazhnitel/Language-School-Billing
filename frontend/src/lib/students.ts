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

export async function listStudents(q: string, includeInactive: boolean): Promise<StudentDTO[]> {
  return (await StudentList(q, includeInactive)) as StudentDTO[];
}

export async function getStudent(id: number): Promise<StudentDTO> {
  return (await StudentGet(id)) as StudentDTO;
}

export function createStudent(
  fullName: string,
  phone: string,
  email: string,
  note: string
): Promise<StudentDTO> {
  return StudentCreate(fullName, phone, email, note) as Promise<StudentDTO>;
}

export function updateStudent(
  id: number,
  fullName: string,
  phone: string,
  email: string,
  note: string
): Promise<StudentDTO> {
  return StudentUpdate(id, fullName, phone, email, note) as Promise<StudentDTO>;
}

export function setStudentActive(id: number, active: boolean): Promise<void> {
  return StudentSetActive(id, active);
}

export function deleteStudent(id: number): Promise<void> {
  return StudentDelete(id);
}
