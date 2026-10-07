import { apiGet } from "@/lib/api";
import { AssignedClass, StudentRosterItem, TeacherOverviewData } from "@/types/teacher_portal";

interface ApiResponse<T> {
  data: T;
}

export async function fetchTeacherOverview(): Promise<TeacherOverviewData> {
  const response = await apiGet<ApiResponse<TeacherOverviewData>>("/portal/teacher/overview");
  return response.data;
}

export async function fetchTeacherClasses(): Promise<AssignedClass[]> {
  const response = await apiGet<ApiResponse<AssignedClass[]>>("/portal/teacher/classes");
  return response.data;
}

export async function fetchClassRoster(classId: string): Promise<StudentRosterItem[]> {
  const response = await apiGet<ApiResponse<StudentRosterItem[]>>(`/portal/teacher/classes/${classId}/roster`);
  return response.data;
}
