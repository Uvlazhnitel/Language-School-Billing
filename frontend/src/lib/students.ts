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
  version: number,
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
  return transport.updateStudent(
    id,
    version,
    fullName,
    personalCode,
    phone,
    email,
    note,
    isMinor,
    payerName,
    payerRole
  );
}

export async function setStudentActive(id: number, version: number, active: boolean): Promise<void> {
  const transport = await getTransport();
  return transport.setStudentActive(id, version, active);
}

export async function deleteStudent(id: number, version: number): Promise<void> {
  const transport = await getTransport();
  return transport.deleteStudent(id, version);
}
