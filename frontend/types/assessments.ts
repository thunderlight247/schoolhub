// Assessments & Results domain types (M29).
// Mirrors backend/internal/assessments DTOs.

export type AssessmentType =
  | "QUIZ"
  | "TEST"
  | "ASSIGNMENT"
  | "PROJECT"
  | "MIDTERM"
  | "FINAL";

export const ASSESSMENT_TYPES: readonly AssessmentType[] = [
  "QUIZ",
  "TEST",
  "ASSIGNMENT",
  "PROJECT",
  "MIDTERM",
  "FINAL",
] as const;

export interface Assessment {
  id: string;
  class_id: string;
  subject_id: string;
  term_id: string;
  title: string;
  type: AssessmentType;
  max_score: number;
  weight?: number | null;
  assessment_date: string;
  class_name?: string;
  academic_year_id?: string;
  academic_year_name?: string;
  subject_code?: string;
  subject_name?: string;
  term_name?: string;
  created_at: string;
  updated_at: string;
}

export interface ResultItem {
  id: string;
  assessment_id: string;
  student_id: string;
  student_number?: string;
  student_name?: string;
  score: number | null;
  grade: string | null;
  remarks: string | null;
  created_at: string;
  updated_at: string;
}

export interface ResultDetail {
  id: string;
  assessment_id: string;
  student_id: string;
  student_number?: string;
  student_name?: string;
  assessment_title: string;
  assessment_type: string;
  max_score: number;
  assessment_date: string;
  class_id: string;
  class_name?: string;
  subject_code?: string;
  subject_name?: string;
  term_id: string;
  term_name?: string;
  score: number | null;
  grade: string | null;
  remarks: string | null;
  created_at: string;
  updated_at: string;
}

export interface StudentResult {
  id: string;
  assessment_id: string;
  student_id: string;
  score: number | null;
  grade: string | null;
  remarks: string | null;
  assessment_title: string;
  assessment_type: string;
  max_score: number;
  assessment_date: string;
  subject_code?: string;
  subject_name?: string;
  class_name?: string;
  term_id?: string;
  term_name?: string;
  created_at: string;
  updated_at: string;
}

export interface AssessmentListResponse {
  assessments: Assessment[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
  };
}

export interface AssessmentDetailResponse {
  assessment: Assessment;
}

export interface AssessmentResultsResponse {
  results: ResultItem[];
}

export interface CreateAssessmentInput {
  class_id: string;
  subject_id: string;
  term_id: string;
  title: string;
  type: AssessmentType;
  max_score: number;
  weight?: number | null;
  assessment_date: string;
}

export interface UpdateAssessmentInput {
  title?: string;
  type?: AssessmentType;
  max_score?: number;
  weight?: number | null;
  assessment_date?: string;
}

export interface RecordResultItemInput {
  student_id: string;
  score?: number | null;
  grade?: string | null;
  remarks?: string | null;
}

export interface UpdateResultInput {
  score?: number | null;
  grade?: string | null;
  remarks?: string | null;
}
