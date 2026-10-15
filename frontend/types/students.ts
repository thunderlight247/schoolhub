// Students & guardians domain types (M25). Mirrors backend/internal/students DTOs.

export interface GuardianLink {
  guardian_id: string;
  relationship: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
  occupation?: string | null;
  phone?: string | null;
  email?: string | null;
}

export interface StudentLink {
  student_id: string;
  student_number: string;
  status: string;
  relationship: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
}

export interface StudentDetail {
  id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
  gender?: string | null;
  date_of_birth?: string | null;
  phone?: string | null;
  email?: string | null;
  address?: string | null;
  admission_date?: string | null;
  status: "ACTIVE" | "ARCHIVED";
  guardians: GuardianLink[];
  created_at: string;
  updated_at: string;
}

export interface StudentListItem {
  id: string;
  student_number: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
  status: "ACTIVE" | "ARCHIVED";
  admission_date?: string | null;
  created_at: string;
}

export interface StudentListResponse {
  students: StudentListItem[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
}

/** Parent-access summary attached to a guardian profile (nil until provisioned). */
export interface GuardianAccount {
  id: string;
  email: string;
  status: string;
  roles: string[];
}

export interface GuardianDetail {
  id: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
  gender?: string | null;
  date_of_birth?: string | null;
  phone?: string | null;
  email?: string | null;
  address?: string | null;
  occupation?: string | null;
  students: StudentLink[];
  account?: GuardianAccount | null;
  created_at: string;
  updated_at: string;
}

export interface GuardianListItem {
  id: string;
  first_name: string;
  last_name: string;
  other_names?: string | null;
  phone?: string | null;
  email?: string | null;
  occupation?: string | null;
  student_count: number;
  created_at: string;
}

export interface GuardianListResponse {
  guardians: GuardianListItem[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
}

export const STUDENT_STATUSES = ["ACTIVE", "ARCHIVED"] as const;

export const GUARDIAN_RELATIONSHIPS = [
  "MOTHER",
  "FATHER",
  "GUARDIAN",
  "GRANDPARENT",
  "SIBLING",
  "AUNT",
  "UNCLE",
  "OTHER",
] as const;
