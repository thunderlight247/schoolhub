// Academic structure domain types (M26). Mirrors backend/internal/academics DTOs.

export interface AcademicYear {
  id: string;
  name: string;
  start_date: string;
  end_date: string;
  is_current: boolean;
  created_at: string;
  updated_at: string;
}

export interface Term {
  id: string;
  academic_year_id: string;
  academic_year_name?: string;
  name: string;
  start_date: string;
  end_date: string;
  is_current: boolean;
  created_at: string;
  updated_at: string;
}

export interface Programme {
  id: string;
  code: string;
  name: string;
  description?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Subject {
  id: string;
  code: string;
  name: string;
  description?: string | null;
  type: "CORE" | "ELECTIVE";
  department_id?: string | null;
  department_name?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Class {
  id: string;
  academic_year_id: string;
  academic_year_name?: string;
  programme_id?: string | null;
  programme_name?: string | null;
  programme_code?: string | null;
  name: string;
  level?: string | null;
  capacity?: number | null;
  created_at: string;
  updated_at: string;
}

/** A subject assigned to a class (GET /classes/:id/subjects). */
export interface ClassSubject {
  id: string;
  code: string;
  name: string;
  description?: string | null;
  type: "CORE" | "ELECTIVE";
  department_id?: string | null;
  assigned_at: string;
}

export const SUBJECT_TYPES = ["CORE", "ELECTIVE"] as const;
