import {
  ContactList,
  ContactCreate,
  ContactUpdate,
  ContactSetActive,
  StudentContactsList,
  StudentContactCreate,
  StudentContactUpdate,
  StudentContactDelete,
} from "../../wailsjs/go/main/App";

export type ContactDTO = {
  id: number;
  fullName: string;
  phone: string;
  email: string;
  note: string;
  isActive: boolean;
  createdAt: string;
};

export type StudentContactDTO = {
  id: number;
  studentId: number;
  contactId: number;
  contactFullName: string;
  contactPhone: string;
  contactEmail: string;
  relation: string;
  isPrimary: boolean;
  isPayer: boolean;
  receivesMessages: boolean;
  note: string;
};

export const RELATIONS = ["mother", "father", "guardian", "self", "other"] as const;
export type Relation = (typeof RELATIONS)[number];

export async function listContacts(q: string, includeInactive: boolean): Promise<ContactDTO[]> {
  return (await ContactList(q, includeInactive)) as ContactDTO[];
}

export function createContact(
  fullName: string,
  phone: string,
  email: string,
  note: string
): Promise<ContactDTO> {
  return ContactCreate(fullName, phone, email, note) as Promise<ContactDTO>;
}

export function updateContact(
  id: number,
  fullName: string,
  phone: string,
  email: string,
  note: string
): Promise<ContactDTO> {
  return ContactUpdate(id, fullName, phone, email, note) as Promise<ContactDTO>;
}

export function setContactActive(id: number, active: boolean): Promise<void> {
  return ContactSetActive(id, active);
}

export async function listStudentContacts(studentID: number): Promise<StudentContactDTO[]> {
  return (await StudentContactsList(studentID)) as StudentContactDTO[];
}

export function createStudentContact(
  studentID: number,
  contactID: number,
  relation: string,
  isPrimary: boolean,
  isPayer: boolean,
  receivesMessages: boolean,
  note: string
): Promise<StudentContactDTO> {
  return StudentContactCreate(
    studentID,
    contactID,
    relation,
    isPrimary,
    isPayer,
    receivesMessages,
    note
  ) as Promise<StudentContactDTO>;
}

export function updateStudentContact(
  id: number,
  relation: string,
  isPrimary: boolean,
  isPayer: boolean,
  receivesMessages: boolean,
  note: string
): Promise<StudentContactDTO> {
  return StudentContactUpdate(
    id,
    relation,
    isPrimary,
    isPayer,
    receivesMessages,
    note
  ) as Promise<StudentContactDTO>;
}

export function deleteStudentContact(id: number): Promise<void> {
  return StudentContactDelete(id);
}
