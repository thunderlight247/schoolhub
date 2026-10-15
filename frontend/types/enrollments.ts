// Enrollment (M9) & teacher assignment (M10) domain types.

export type EnrollmentStatus = "ENROLLED" | "COMPLETED" | "TRANSFERRED" | "WITHDRAWN";

export const ENROLLMENT_STATUSES: EnrollmentStatus[] = [
  "ENROLLED",
  "COMPLETED",
  "TRANSFERRED",
  "WITHDRAWN",
];

export interface Enrollment {
  id: string;
  student_id: string;
  academic_year_id: string;
  class_id: string;
  status: EnrollmentStatus;
  enrollment_date: string;
  exit_date?: string | null;
  notes?: string | null;
  student_number?: string;
  student_name?: string;
  academic_year_name?: string;
  class_name?: string;
  created_at: string;
  updated_at: string;
}

export interface EnrollmentListResponse {
  enrollments: Enrollment[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
}

export interface TeacherAssignment {
  id: string;
  teacher_id: string;
  class_id: string;
  subject_id: string;
  staff_number?: string;
  teacher_name?: string;
  academic_year_id?: string;
  academic_year_name?: string;
  class_name?: string;
  subject_code?: string;
  subject_name?: string;
  created_at: string;
  updated_at: string;
}

export interface TeacherAssignmentListResponse {
  assignments: TeacherAssignment[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
}
