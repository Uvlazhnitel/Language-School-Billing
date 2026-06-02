import { getTransport, type TeacherDTO } from "./api";
export type { TeacherDTO } from "./api";

export async function listTeachers(q: string) {
  const transport = await getTransport();
  return transport.listTeachers(q);
}

export async function createTeacher(fullName: string) {
  const transport = await getTransport();
  return transport.createTeacher(fullName);
}
