import { apiGet } from "@/lib/api";
import {
  StudentAttendanceRecord,
  StudentEnrollmentHistoryItem,
  StudentOverviewData,
  StudentResult,
} from "@/types/student_portal";

interface ApiResponse<T> {
  data: T;
}

export async function fetchStudentOverview(): Promise<StudentOverviewData> {
  const response = await apiGet<ApiResponse<StudentOverviewData>>("/portal/student/overview");
  return response.data;
}

export async function fetchStudentAttendance(
  limit: number = 50
): Promise<StudentAttendanceRecord[]> {
  const response = await apiGet<ApiResponse<StudentAttendanceRecord[]>>(
    `/portal/student/attendance?limit=${limit}`
  );
  return response.data;
}

export async function fetchStudentResults(
  termId?: string,
  academicYearId?: string
): Promise<StudentResult[]> {
  const params = new URLSearchParams();
  if (termId) params.append("term_id", termId);
  if (academicYearId) params.append("academic_year_id", academicYearId);
  const qs = params.toString() ? `?${params.toString()}` : "";
  const response = await apiGet<ApiResponse<StudentResult[]>>(
    `/portal/student/results${qs}`
  );
  return response.data;
}

export async function fetchStudentEnrollments(): Promise<StudentEnrollmentHistoryItem[]> {
  const response = await apiGet<ApiResponse<StudentEnrollmentHistoryItem[]>>(
    "/portal/student/enrollments"
  );
  return response.data;
}
