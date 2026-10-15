import { apiGet, apiPost, apiPatch, apiPut, apiDelete } from "@/lib/api";
import type {
  GuardianDetail,
  GuardianLink,
  GuardianListResponse,
  StudentDetail,
  StudentListResponse,
} from "@/types/students";

// Students & guardians wrap results in {"students": ...} / {"student": ...} /
// {"guardians": ...} / {"guardian": ...} — same per-domain envelope convention
// as the staff endpoints.

interface StudentEnvelope {
  student: StudentDetail;
}
interface GuardiansEnvelope {
  guardians: GuardianLink[];
}
interface GuardianEnvelope {
  guardian: GuardianDetail;
}

export interface StudentListParams {
  search?: string;
  status?: string;
  page?: number;
  page_size?: number;
}

function toQuery(params: StudentListParams): string {
  const q = new URLSearchParams();
  if (params.search) q.set("search", params.search);
  if (params.status) q.set("status", params.status);
  if (params.page) q.set("page", String(params.page));
  if (params.page_size) q.set("page_size", String(params.page_size));
  const s = q.toString();
  return s ? `?${s}` : "";
}

// --- students ---

export function listStudents(params: StudentListParams): Promise<StudentListResponse> {
  return apiGet<StudentListResponse>(`/students${toQuery(params)}`);
}

export function getStudent(id: string): Promise<StudentDetail> {
  return apiGet<StudentEnvelope>(`/students/${id}`).then((r) => r.student);
}

export interface CreateStudentBody {
  student_number: string;
  first_name: string;
  last_name: string;
  other_names?: string;
  gender?: string;
  date_of_birth?: string;
  phone?: string;
  email?: string;
  address?: string;
  admission_date?: string;
}

export function createStudent(body: CreateStudentBody): Promise<StudentDetail> {
  return apiPost<StudentEnvelope>("/students", body).then((r) => r.student);
}

export interface UpdateStudentBody {
  first_name?: string;
  last_name?: string;
  other_names?: string | null;
  gender?: string;
  date_of_birth?: string;
  phone?: string | null;
  email?: string | null;
  address?: string | null;
}

export function updateStudent(id: string, body: UpdateStudentBody): Promise<StudentDetail> {
  return apiPatch<StudentEnvelope>(`/students/${id}`, body).then((r) => r.student);
}

export function setStudentStatus(id: string, status: "ACTIVE" | "ARCHIVED"): Promise<StudentDetail> {
  return apiPut<StudentEnvelope>(`/students/${id}/status`, { status }).then((r) => r.student);
}

export function linkGuardian(
  studentId: string,
  guardianId: string,
  relationship: string,
): Promise<GuardianLink[]> {
  return apiPost<GuardiansEnvelope>(`/students/${studentId}/guardians`, {
    guardian_id: guardianId,
    relationship,
  }).then((r) => r.guardians);
}

export function unlinkGuardian(studentId: string, guardianId: string): Promise<GuardianLink[]> {
  return apiDelete<GuardiansEnvelope>(`/students/${studentId}/guardians/${guardianId}`).then(
    (r) => r.guardians,
  );
}

// --- guardians ---

export function listGuardians(params: StudentListParams): Promise<GuardianListResponse> {
  return apiGet<GuardianListResponse>(`/guardians${toQuery(params)}`);
}

export function getGuardian(id: string): Promise<GuardianDetail> {
  return apiGet<GuardianEnvelope>(`/guardians/${id}`).then((r) => r.guardian);
}

export interface CreateGuardianBody {
  first_name: string;
  last_name: string;
  other_names?: string;
  gender?: string;
  date_of_birth?: string;
  phone?: string;
  email?: string;
  address?: string;
  occupation?: string;
}

export function createGuardian(body: CreateGuardianBody): Promise<GuardianDetail> {
  return apiPost<GuardianEnvelope>("/guardians", body).then((r) => r.guardian);
}

export interface UpdateGuardianBody {
  first_name?: string;
  last_name?: string;
  other_names?: string | null;
  gender?: string;
  date_of_birth?: string;
  phone?: string | null;
  email?: string | null;
  address?: string | null;
  occupation?: string | null;
}

export function updateGuardian(id: string, body: UpdateGuardianBody): Promise<GuardianDetail> {
  return apiPatch<GuardianEnvelope>(`/guardians/${id}`, body).then((r) => r.guardian);
}

// --- guardian login accounts (parent access provisioning) ---

export interface CreateGuardianAccountBody {
  /** The parent's username for sign-in — always their email. */
  email: string;
  /** Minimum 8 chars, hashed server-side, never emailed. */
  password: string;
}

/** POST /guardians/:id/account — "Create parent access" on the profile page. */
export function createGuardianAccount(
  guardianId: string,
  body: CreateGuardianAccountBody,
): Promise<{ account: { email: string; status: string; roles: string[] } }> {
  return apiPost(`/guardians/${guardianId}/account`, body);
}
