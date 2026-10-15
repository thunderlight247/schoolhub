import { apiGet, apiPost, apiPatch, apiDelete } from "@/lib/api";
import type {
  Assessment,
  AssessmentDetailResponse,
  AssessmentListResponse,
  AssessmentResultsResponse,
  AssessmentType,
  CreateAssessmentInput,
  RecordResultItemInput,
  ResultDetail,
  ResultItem,
  StudentResult,
  UpdateAssessmentInput,
  UpdateResultInput,
} from "@/types/assessments";

export interface AssessmentListParams {
  class_id?: string;
  subject_id?: string;
  term_id?: string;
  type?: AssessmentType | string;
  page?: number;
  page_size?: number;
}

function toQuery(params: AssessmentListParams): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== "") {
      q.set(k, String(v));
    }
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export function listAssessments(
  params: AssessmentListParams = {},
): Promise<AssessmentListResponse> {
  return apiGet<AssessmentListResponse>(`/assessments${toQuery(params)}`);
}

export function getAssessment(id: string): Promise<Assessment> {
  return apiGet<AssessmentDetailResponse>(`/assessments/${id}`).then((r) => r.assessment);
}

export function createAssessment(input: CreateAssessmentInput): Promise<Assessment> {
  return apiPost<AssessmentDetailResponse>("/assessments", input).then((r) => r.assessment);
}

export function updateAssessment(
  id: string,
  input: UpdateAssessmentInput,
): Promise<Assessment> {
  return apiPatch<AssessmentDetailResponse>(`/assessments/${id}`, input).then((r) => r.assessment);
}

export function deleteAssessment(id: string): Promise<{ message: string }> {
  return apiDelete<{ message: string }>(`/assessments/${id}`);
}

export function listAssessmentResults(assessmentId: string): Promise<ResultItem[]> {
  return apiGet<AssessmentResultsResponse>(`/assessments/${assessmentId}/results`).then(
    (r) => r.results,
  );
}

export function recordResults(
  assessmentId: string,
  results: RecordResultItemInput[],
): Promise<ResultItem[]> {
  return apiPost<AssessmentResultsResponse>(`/assessments/${assessmentId}/results`, {
    results,
  }).then((r) => r.results);
}

export function getResult(id: string): Promise<ResultDetail> {
  return apiGet<{ result: ResultDetail }>(`/results/${id}`).then((r) => r.result);
}

export function updateResult(id: string, input: UpdateResultInput): Promise<ResultDetail> {
  return apiPatch<{ result: ResultDetail }>(`/results/${id}`, input).then((r) => r.result);
}

export function listStudentResults(
  studentId: string,
  params?: { term_id?: string; academic_year_id?: string },
): Promise<StudentResult[]> {
  const q = new URLSearchParams();
  if (params?.term_id) q.set("term_id", params.term_id);
  if (params?.academic_year_id) q.set("academic_year_id", params.academic_year_id);
  const qs = q.toString() ? `?${q.toString()}` : "";
  return apiGet<{ results: StudentResult[] }>(`/students/${studentId}/results${qs}`).then(
    (r) => r.results,
  );
}
