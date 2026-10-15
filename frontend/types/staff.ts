// Staff domain types (M24). Mirrors backend/internal/staff DTOs.

export interface StaffAccount {
  id: string;
  email: string;
  status: string;
  roles: string[];
}

/** Full staff record — GET /staff/:id and mutation responses. */
export interface StaffDetail {
  id: string;
  staff_number: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
  gender?: string | null;
  date_of_birth?: string | null;
  phone?: string | null;
  email?: string | null;
  address?: string | null;
  department_id?: string | null;
  department_name?: string | null;
  position_id?: string | null;
  position_title?: string | null;
  position_category?: string | null;
  hired_date?: string | null;
  employment_status: string;
  account?: StaffAccount | null;
  created_at: string;
  updated_at: string;
}

/** List row — GET /staff. */
export interface StaffListItem {
  id: string;
  staff_number: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
  department_name?: string | null;
  position_title?: string | null;
  position_category?: string | null;
  employment_status: string;
  has_account: boolean;
  created_at: string;
}

export interface StaffListResponse {
  staff: StaffListItem[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
}

export interface Department {
  id: string;
  name: string;
  description?: string | null;
}

export interface StaffPosition {
  id: string;
  title: string;
  category: string;
  description?: string | null;
}

export interface Role {
  id: string;
  name: string;
  description?: string;
}

export const EMPLOYMENT_STATUSES = [
  "ACTIVE",
  "INACTIVE",
  "SUSPENDED",
  "ON_LEAVE",
] as const;

export const POSITION_CATEGORIES = [
  "TEACHING",
  "ADMINISTRATION",
  "ICT",
  "ACCOUNTS",
  "MANAGEMENT",
  "SUPPORT",
] as const;

export const GENDERS = ["FEMALE", "MALE", "OTHER"] as const;

export const SYSTEM_ROLES = ["ADMIN", "TEACHER", "STAFF", "PARENT", "STUDENT"] as const;
