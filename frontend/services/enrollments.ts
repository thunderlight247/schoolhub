import { apiGet, apiPost, apiPatch, apiDelete } from "@/lib/api";
import type {
  Enrollment,
  EnrollmentListResponse,
  EnrollmentStatus,
  TeacherAssignmentListResponse,
} from "@/types/enrollments";

// Enrollment endpoints wrap results in {"enrollments": ...} / {"enrollment": ...};
// teacher assignments in {"assignments": ...}.

interface EnrollmentEnvelope {
  enrollment: Enrollment;
}

export interface EnrollmentListParams {
  student_id?: string;
  academic_year_id?: string;
  class_id?: string;
  status?: string;
  search?: string;
  page?: number;
  page_size?: number;
}

function toQuery(params: EnrollmentListParams): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v) q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export function listEnrollments(params: EnrollmentListParams): Promise<EnrollmentListResponse> {
  return apiGet<EnrollmentListResponse>(`/enrollments${toQuery(params)}`);
}

export function listStudentEnrollments(
  studentId: string,
  params: Omit<EnrollmentListParams, "student_id"> = {},
): Promise<EnrollmentListResponse> {
  return apiGet<EnrollmentListResponse>(`/students/${studentId}/enrollments${toQuery(params)}`);
}

export interface CreateEnrollmentBody {
  student_id: string;
  academic_year_id: string;
  class_id: string;
  status?: EnrollmentStatus;
  enrollment_date?: string;
  notes?: string;
}

export function createEnrollment(body: CreateEnrollmentBody): Promise<Enrollment> {
  return apiPost<EnrollmentEnvelope>("/enrollments", body).then((r) => r.enrollment);
}

export function setEnrollmentStatus(id: string, status: EnrollmentStatus): Promise<Enrollment> {
  return apiPatch<EnrollmentEnvelope>(`/enrollments/${id}/status`, { status }).then(
    (r) => r.enrollment,
  );
}

// --- teacher assignments (M10) ---

export interface TeacherAssignmentListParams {
  teacher_id?: string;
  class_id?: string;
  subject_id?: string;
  academic_year_id?: string;
  search?: string;
  page?: number;
  page_size?: number;
}

export function listTeacherAssignments(
  params: TeacherAssignmentListParams = {},
): Promise<TeacherAssignmentListResponse> {
  return apiGet<TeacherAssignmentListResponse>(`/teacher-assignments${toQuery(params)}`);
}

export function createTeacherAssignment(body: {
  teacher_id: string;
  class_id: string;
  subject_id: string;
}): Promise<void> {
  return apiPost("/teacher-assignments", body);
}

export function deleteTeacherAssignment(id: string): Promise<void> {
  return apiDelete(`/teacher-assignments/${id}`);
}
