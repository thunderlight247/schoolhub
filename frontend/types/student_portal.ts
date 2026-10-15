export interface StudentProfile {
  student_id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  other_names?: string;
  gender?: string;
  date_of_birth?: string;
  email?: string;
  phone?: string;
  address?: string;
  admission_date?: string;
  student_status: string;
}

export interface AcademicContext {
  academic_year_id?: string;
  academic_year_name?: string;
  term_id?: string;
  term_name?: string;
}

export interface StudentEnrollmentContext {
  enrollment_id: string;
  enrollment_status: string;
  enrollment_date: string;
  class_id: string;
  className?: string;
  class_name: string;
  class_level?: string;
  class_capacity?: number;
  programme_id?: string;
  programme_code?: string;
  programme_name?: string;
  academic_year_id: string;
  academic_year_name: string;
}

export interface StudentEnrollmentHistoryItem {
  enrollment_id: string;
  enrollment_status: string;
  enrollment_date: string;
  exit_date?: string;
  enrolled_at: string;
  class_id: string;
  class_name: string;
  class_level?: string;
  programme_id?: string;
  programme_code?: string;
  programme_name?: string;
  academic_year_id: string;
  academic_year_name: string;
}

export interface AttendanceCounts {
  present: number;
  absent: number;
  late: number;
  excused: number;
}

export interface ResultsSummary {
  results_count: number;
  average_score?: number;
}

export interface StudentAttendanceRecord {
  record_id: string;
  session_date: string;
  status: string;
  notes?: string;
  class_id: string;
  class_name: string;
  subject_id: string;
  subject_name: string;
  subject_code: string;
  term_name: string;
}

export interface StudentResult {
  result_id: string;
  assessment_id: string;
  assessment_title: string;
  assessment_type: string;
  subject_name: string;
  subject_code: string;
  class_name: string;
  term_name: string;
  assessment_date: string;
  max_score: number;
  score?: number;
  grade?: string;
  remarks?: string;
}

export interface StudentAnnouncement {
  id: string;
  title: string;
  content: string;
  priority: string;
  target_type: string;
  target_class?: string;
  author_name: string;
  published_at?: string;
  created_at: string;
}

export interface StudentOverviewData {
  profile: StudentProfile;
  academic_context: AcademicContext;
  enrollment?: StudentEnrollmentContext;
  enrollment_history: StudentEnrollmentHistoryItem[];
  attendance_counts: AttendanceCounts;
  results_summary: ResultsSummary;
  recent_attendance: StudentAttendanceRecord[];
  recent_results: StudentResult[];
  announcements: StudentAnnouncement[];
}
