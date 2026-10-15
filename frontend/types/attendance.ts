export type AttendanceStatus = "PRESENT" | "ABSENT" | "LATE" | "EXCUSED";

export const ATTENDANCE_STATUSES: readonly AttendanceStatus[] = [
  "PRESENT",
  "ABSENT",
  "LATE",
  "EXCUSED",
] as const;

export interface AttendanceSession {
  id: string;
  class_id: string;
  subject_id: string;
  term_id: string;
  session_date: string;
  class_name?: string;
  academic_year_id?: string;
  academic_year_name?: string;
  subject_code?: string;
  subject_name?: string;
  term_name?: string;
  created_at: string;
  updated_at: string;
}

export interface AttendanceRecord {
  id: string;
  session_id: string;
  student_id: string;
  student_number?: string;
  student_name?: string;
  status: AttendanceStatus;
  notes?: string | null;
  created_at: string;
  updated_at: string;
}

export interface AttendanceRecordDetail {
  id: string;
  session_id: string;
  student_id: string;
  student_number?: string;
  student_name?: string;
  status: AttendanceStatus;
  notes?: string | null;
  session_date: string;
  class_id: string;
  class_name?: string;
  subject_code?: string;
  subject_name?: string;
  term_id: string;
  term_name?: string;
  created_at: string;
  updated_at: string;
}

export interface AttendanceSessionListResponse {
  sessions: AttendanceSession[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
}

export interface AttendanceSessionDetailResponse {
  session: AttendanceSession;
  records: AttendanceRecord[];
}

export interface RecordInput {
  student_id: string;
  status: AttendanceStatus;
  notes?: string | null;
}

export interface CreateSessionInput {
  class_id: string;
  subject_id: string;
  term_id: string;
  session_date: string;
  records: RecordInput[];
}

export interface AttendanceSummary {
  student_id?: string;
  class_id?: string;
  subject_id?: string;
  term_id?: string;
  academic_year_id?: string;
  counts: Record<string, number>;
  total_records: number;
}
