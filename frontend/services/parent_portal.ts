import { apiGet } from "@/lib/api";
import {
  ChildDetail,
  ParentOverviewData,
} from "@/types/parent_portal";

interface ApiResponse<T> {
  data: T;
}

export async function fetchParentOverview(): Promise<ParentOverviewData> {
  const response = await apiGet<ApiResponse<ParentOverviewData>>("/portal/parent/overview");
  return response.data;
}

export async function fetchChildDetail(studentId: string): Promise<ChildDetail> {
  const response = await apiGet<ApiResponse<ChildDetail>>(
    `/portal/parent/children/${studentId}`
  );
  return response.data;
}
