import { getTransport, type StudentDTO } from "./api";
export type { StudentDTO } from "./api";

export async function listStudents(q: string, includeInactive: boolean): Promise<StudentDTO[]> {
  const transport = await getTransport();
  return transport.listStudents(q, includeInactive);
}

export async function getStudent(id: number): Promise<StudentDTO> {
  const transport = await getTransport();
  return transport.getStudent(id);
}

export async function createStudent(
  fullName: string,
  personalCode: string,
  phone: string,
  email: string,
  note: string,
  isMinor: boolean,
  payerName: string,
  payerRole: string
): Promise<StudentDTO> {
  const transport = await getTransport();
  return transport.createStudent(fullName, personalCode, phone, email, note, isMinor, payerName, payerRole);
}

export async function updateStudent(
  id: number,
  fullName: string,
  personalCode: string,
  phone: string,
  email: string,
  note: string,
  isMinor: boolean,
  payerName: string,
  payerRole: string
): Promise<StudentDTO> {
  const transport = await getTransport();
  return transport.updateStudent(id, fullName, personalCode, phone, email, note, isMinor, payerName, payerRole);
}

export async function setStudentActive(id: number, active: boolean): Promise<void> {
  const transport = await getTransport();
  return transport.setStudentActive(id, active);
}

export async function deleteStudent(id: number): Promise<void> {
  const transport = await getTransport();
  return transport.deleteStudent(id);
}
