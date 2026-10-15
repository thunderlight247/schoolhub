import { apiGet } from "@/lib/api";
import {
  ClassAttendanceReport,
  ClassPerformanceReport,
  EnrollmentReport,
  StudentReportCard,
} from "@/types/reports";

interface ApiResponse<T> {
  data: T;
}

export async function fetchEnrollmentReport(
  academicYearId?: string
): Promise<EnrollmentReport> {
  const query = academicYearId ? `?academic_year_id=${academicYearId}` : "";
  const res = await apiGet<ApiResponse<EnrollmentReport>>(
    `/reports/enrollment${query}`
  );
  return res.data;
}

export async function fetchClassAttendanceReport(
  classId: string,
  params?: {
    term_id?: string;
    start_date?: string;
    end_date?: string;
  }
): Promise<ClassAttendanceReport> {
  const q = new URLSearchParams();
  if (params?.term_id) q.append("term_id", params.term_id);
  if (params?.start_date) q.append("start_date", params.start_date);
  if (params?.end_date) q.append("end_date", params.end_date);
  const qs = q.toString() ? `?${q.toString()}` : "";

  const res = await apiGet<ApiResponse<ClassAttendanceReport>>(
    `/reports/attendance/classes/${classId}${qs}`
  );
  return res.data;
}

export async function fetchClassPerformanceReport(
  classId: string,
  params?: {
    term_id?: string;
    subject_id?: string;
  }
): Promise<ClassPerformanceReport> {
  const q = new URLSearchParams();
  if (params?.term_id) q.append("term_id", params.term_id);
  if (params?.subject_id) q.append("subject_id", params.subject_id);
  const qs = q.toString() ? `?${q.toString()}` : "";

  const res = await apiGet<ApiResponse<ClassPerformanceReport>>(
    `/reports/performance/classes/${classId}${qs}`
  );
  return res.data;
}

export async function fetchStudentReportCard(
  studentId: string,
  termId: string
): Promise<StudentReportCard> {
  const res = await apiGet<ApiResponse<StudentReportCard>>(
    `/reports/students/${studentId}/report-card?term_id=${termId}`
  );
  return res.data;
}
