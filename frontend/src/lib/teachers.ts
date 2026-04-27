import { TeacherCreate, TeacherList } from "../../wailsjs/go/main/App";

export type TeacherDTO = {
  id: number;
  fullName: string;
  isActive: boolean;
};

export async function listTeachers(q: string) {
  return (await TeacherList(q)) as TeacherDTO[];
}

export async function createTeacher(fullName: string) {
  return (await TeacherCreate(fullName)) as TeacherDTO;
}
