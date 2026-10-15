import { apiGet, apiPost, apiPatch, apiPut } from "@/lib/api";
import type {
  Department,
  Role,
  StaffDetail,
  StaffListResponse,
  StaffPosition,
} from "@/types/staff";

// Staff endpoints wrap results in {"staff": ...}, unlike the report/portal
// endpoints which use {"data": ...}.
interface StaffEnvelope {
  staff: StaffDetail;
}

interface DepartmentsResponse {
  departments: Department[];
}

interface StaffPositionsResponse {
  staff_positions: StaffPosition[];
}

// --- staff (M6 endpoints, staff.view / staff.create / staff.update / staff.deactivate) ---

export interface StaffListParams {
  search?: string;
  department_id?: string;
  employment_status?: string;
  position_category?: string;
  page?: number;
  page_size?: number;
}

function toQuery(params: StaffListParams): string {
  const q = new URLSearchParams();
  if (params.search) q.set("search", params.search);
  if (params.department_id) q.set("department_id", params.department_id);
  if (params.employment_status) q.set("employment_status", params.employment_status);
  if (params.position_category) q.set("position_category", params.position_category);
  if (params.page) q.set("page", String(params.page));
  if (params.page_size) q.set("page_size", String(params.page_size));
  const s = q.toString();
  return s ? `?${s}` : "";
}

export function listStaff(params: StaffListParams): Promise<StaffListResponse> {
  return apiGet<StaffListResponse>(`/staff${toQuery(params)}`);
}

export function getStaff(id: string): Promise<StaffDetail> {
  return apiGet<StaffEnvelope>(`/staff/${id}`).then((r) => r.staff);
}

export interface CreateStaffBody {
  staff_number: string;
  first_name: string;
  last_name: string;
  other_names?: string;
  gender?: string;
  date_of_birth?: string;
  phone?: string;
  email?: string;
  address?: string;
  department_id?: string;
  position_id?: string;
  hired_date?: string;
}

export function createStaff(body: CreateStaffBody): Promise<StaffDetail> {
  return apiPost<StaffEnvelope>("/staff", body).then((r) => r.staff);
}

export interface UpdateStaffBody {
  first_name?: string;
  last_name?: string;
  other_names?: string | null;
  gender?: string;
  date_of_birth?: string;
  phone?: string | null;
  email?: string | null;
  address?: string | null;
}

export function updateStaff(id: string, body: UpdateStaffBody): Promise<StaffDetail> {
  return apiPatch<StaffEnvelope>(`/staff/${id}`, body).then((r) => r.staff);
}

export interface SetAssignmentBody {
  department_id?: string | null;
  position_id?: string | null;
  hired_date?: string | null;
}

export function setStaffAssignment(id: string, body: SetAssignmentBody): Promise<StaffDetail> {
  return apiPut<StaffEnvelope>(`/staff/${id}/assignment`, body).then((r) => r.staff);
}

export function setEmploymentStatus(id: string, employment_status: string): Promise<StaffDetail> {
  return apiPut<StaffEnvelope>(`/staff/${id}/status`, { employment_status }).then(
    (r) => r.staff,
  );
}

export interface CreateAccountBody {
  email: string;
  password: string;
  roles: string[];
}

export function createStaffAccount(id: string, body: CreateAccountBody): Promise<void> {
  return apiPost(`/staff/${id}/account`, body);
}

export function setAccountRoles(id: string, roles: string[]): Promise<void> {
  return apiPut(`/staff/${id}/account/roles`, { roles });
}

// --- reference data (departments / positions) ---

export function listDepartments(): Promise<Department[]> {
  return apiGet<DepartmentsResponse>("/departments").then((r) => r.departments);
}

export function createDepartment(name: string, description?: string): Promise<Department> {
  return apiPost<{ department: Department }>("/departments", { name, description }).then(
    (r) => r.department,
  );
}

export function listPositions(): Promise<StaffPosition[]> {
  return apiGet<StaffPositionsResponse>("/staff-positions").then((r) => r.staff_positions);
}

export function createPosition(
  title: string,
  category: string,
  description?: string,
): Promise<StaffPosition> {
  return apiPost<{ staff_position: StaffPosition }>("/staff-positions", {
    title,
    category,
    description,
  }).then((r) => r.staff_position);
}

// --- roles (rbac routes are ADMIN-gated) ---

export function listRoles(): Promise<Role[]> {
  return apiGet<{ roles: Role[] }>("/roles").then((r) => r.roles);
}
