import { apiGet, apiPost, apiPatch } from "@/lib/api";
import type {
  AttendanceRecordDetail,
  AttendanceSessionDetailResponse,
  AttendanceSessionListResponse,
  AttendanceStatus,
  AttendanceSummary,
  CreateSessionInput,
} from "@/types/attendance";

export interface AttendanceSessionListParams {
  class_id?: string;
  subject_id?: string;
  term_id?: string;
  date_from?: string;
  date_to?: string;
  page?: number;
  page_size?: number;
}

function toQuery(params: AttendanceSessionListParams): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== "") {
      q.set(k, String(v));
    }
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export function listAttendanceSessions(
  params: AttendanceSessionListParams = {},
): Promise<AttendanceSessionListResponse> {
  return apiGet<AttendanceSessionListResponse>(`/attendance/sessions${toQuery(params)}`);
}

export function getAttendanceSession(id: string): Promise<AttendanceSessionDetailResponse> {
  return apiGet<AttendanceSessionDetailResponse>(`/attendance/sessions/${id}`);
}

export function createAttendanceSession(
  input: CreateSessionInput,
): Promise<AttendanceSessionDetailResponse> {
  return apiPost<AttendanceSessionDetailResponse>("/attendance/sessions", input);
}

export function updateAttendanceRecord(
  id: string,
  body: { status: AttendanceStatus; notes?: string | null },
): Promise<AttendanceRecordDetail> {
  return apiPatch<{ record: AttendanceRecordDetail }>(`/attendance/records/${id}`, body).then(
    (res) => res.record,
  );
}

export function getClassAttendanceSummary(
  classId: string,
  params?: { term_id?: string; subject_id?: string },
): Promise<AttendanceSummary> {
  const q = new URLSearchParams();
  if (params?.term_id) q.set("term_id", params.term_id);
  if (params?.subject_id) q.set("subject_id", params.subject_id);
  const qs = q.toString() ? `?${q.toString()}` : "";
  return apiGet<{ summary: AttendanceSummary }>(`/classes/${classId}/attendance/summary${qs}`).then(
    (res) => res.summary,
  );
}

export function getStudentAttendanceSummary(
  studentId: string,
  params?: { term_id?: string; academic_year_id?: string },
): Promise<AttendanceSummary> {
  const q = new URLSearchParams();
  if (params?.term_id) q.set("term_id", params.term_id);
  if (params?.academic_year_id) q.set("academic_year_id", params.academic_year_id);
  const qs = q.toString() ? `?${q.toString()}` : "";
  return apiGet<{ summary: AttendanceSummary }>(`/students/${studentId}/attendance/summary${qs}`).then(
    (res) => res.summary,
  );
}
