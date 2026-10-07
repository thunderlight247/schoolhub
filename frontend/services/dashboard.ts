import { apiGet } from "@/lib/api";
import { AdminDashboardData } from "@/types/dashboard";

interface ApiResponse<T> {
  data: T;
}

export async function fetchAdminDashboard(): Promise<AdminDashboardData> {
  const response = await apiGet<ApiResponse<AdminDashboardData>>("/dashboard/admin");
  return response.data;
}
