export interface TeacherProfile {
  staff_id: string;
  staff_number: string;
  first_name: string;
  last_name: string;
  email: string;
  phone?: string;
  department_name?: string;
  position_title?: string;
  employment_status: string;
}

export interface TeacherCounts {
  total_classes: number;
  total_subjects: number;
  total_students: number;
  total_attendance_sessions: number;
  total_assessments: number;
}

export interface AcademicContext {
  academic_year_id?: string;
  academic_year_name?: string;
  term_id?: string;
  term_name?: string;
}

export interface AssignedClass {
  assignment_id: string;
  class_id: string;
  class_name: string;
  class_level?: string;
  class_capacity: number;
  academic_year_id: string;
  academic_year_name: string;
  subject_id: string;
  subject_name: string;
  subject_code: string;
  subject_type: string;
  enrolled_students_count: number;
}

export interface RecentAttendanceSession {
  session_id: string;
  session_date: string;
  class_id: string;
  class_name: string;
  subject_id: string;
  subject_name: string;
  subject_code: string;
  term_name: string;
  present_count: number;
  absent_count: number;
  late_count: number;
  excused_count: number;
  total_marks: number;
}

export interface RecentAssessment {
  assessment_id: string;
  title: string;
  assessment_type: string;
  max_score: number;
  weight?: number;
  assessment_date: string;
  class_id: string;
  class_name: string;
  subject_id: string;
  subject_name: string;
  subject_code: string;
  term_name: string;
  results_count: number;
  total_students_count: number;
}

export interface TeacherAnnouncement {
  id: string;
  title: string;
  content: string;
  priority: string;
  status: string;
  target_type: string;
  author_name: string;
  published_at?: string;
  created_at: string;
}

export interface StudentRosterItem {
  student_id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  gender?: string;
  date_of_birth?: string;
  enrollment_status: string;
  enrolled_at: string;
  guardian_name?: string;
  guardian_phone?: string;
  guardian_email?: string;
  guardian_relationship?: string;
}

export interface TeacherOverviewData {
  profile: TeacherProfile;
  academic_context: AcademicContext;
  counts: TeacherCounts;
  assigned_classes: AssignedClass[];
  recent_attendance: RecentAttendanceSession[];
  recent_assessments: RecentAssessment[];
  announcements: TeacherAnnouncement[];
}
