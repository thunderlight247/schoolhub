export interface GuardianProfile {
  guardian_id: string;
  first_name: string;
  last_name: string;
  email?: string;
  phone?: string;
  occupation?: string;
}

export interface ChildEnrollmentContext {
  enrollment_id?: string;
  class_id?: string;
  class_name?: string;
  class_level?: string;
  academic_year_id?: string;
  academic_year_name?: string;
}

export interface ChildAttendanceCounts {
  present: number;
  absent: number;
  late: number;
  excused: number;
}

export interface ChildSummary {
  student_id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  gender?: string;
  date_of_birth?: string;
  relationship: string;
  student_status: string;
  enrollment: ChildEnrollmentContext;
  attendance_counts: ChildAttendanceCounts;
  results_count: number;
  average_score?: number;
}

export interface ParentAnnouncement {
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

export interface ParentOverviewData {
  profile: GuardianProfile;
  children: ChildSummary[];
  announcements: ParentAnnouncement[];
}

export interface ChildAttendanceRecord {
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

export interface ChildResult {
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

export interface ChildDetail {
  student: ChildSummary;
  attendance_counts: ChildAttendanceCounts;
  recent_attendance: ChildAttendanceRecord[];
  results: ChildResult[];
}
