export interface EnrollmentClassSummary {
  class_id: string;
  class_name: string;
  class_level?: string;
  capacity?: number;
  academic_year_id: string;
  academic_year_name: string;
  programme_name?: string;
  enrolled_count: number;
  completed_count: number;
  transferred_count: number;
  withdrawn_count: number;
  male_count: number;
  female_count: number;
  other_gender_count: number;
  utilization_rate?: number;
}

export interface EnrollmentReport {
  academic_year_id?: string;
  academic_year_name?: string;
  total_classes: number;
  total_enrolled: number;
  total_capacity: number;
  overall_utilization?: number;
  total_male: number;
  total_female: number;
  total_other_gender: number;
  classes: EnrollmentClassSummary[];
}

export interface ClassAttendanceStudentSummary {
  student_id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  present_count: number;
  absent_count: number;
  late_count: number;
  excused_count: number;
  total_records: number;
  attendance_rate: number;
}

export interface ClassAttendanceReport {
  class_id: string;
  class_name: string;
  term_id?: string;
  term_name?: string;
  start_date?: string;
  end_date?: string;
  total_sessions: number;
  total_students: number;
  average_attendance_rate: number;
  students: ClassAttendanceStudentSummary[];
}

export interface AssessmentPerformanceItem {
  assessment_id: string;
  assessment_title: string;
  assessment_type: string;
  max_score: number;
  weight: number;
  assessment_date: string;
  subject_id: string;
  subject_code: string;
  subject_name: string;
  results_count: number;
  average_score: number;
  highest_score: number;
  lowest_score: number;
  average_percentage?: number;
}

export interface ClassPerformanceReport {
  class_id: string;
  class_name: string;
  term_id?: string;
  term_name?: string;
  subject_id?: string;
  subject_name?: string;
  assessments: AssessmentPerformanceItem[];
}

export interface ReportCardAttendance {
  present_count: number;
  absent_count: number;
  late_count: number;
  excused_count: number;
  total_sessions: number;
  attendance_rate: number;
}

export interface ReportCardSubjectItem {
  subject_id: string;
  subject_code: string;
  subject_name: string;
  subject_type: string;
  assessments_taken: number;
  total_score: number;
  total_max_score: number;
  average_percentage: number;
  grade: string;
}

export interface StudentReportCard {
  student_id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  gender?: string;
  date_of_birth?: string;
  class_id: string;
  class_name: string;
  class_level?: string;
  academic_year_id: string;
  academic_year_name: string;
  term_id: string;
  term_name: string;
  attendance: ReportCardAttendance;
  overall_average_percentage?: number;
  overall_grade?: string;
  subjects: ReportCardSubjectItem[];
}
