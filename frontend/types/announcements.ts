// Announcements domain types (M30).
// Mirrors backend/internal/announcements DTOs (M13 handlers).

export type AnnouncementTargetType = "ALL" | "ROLE" | "CLASS";
export type AnnouncementStatus = "DRAFT" | "PUBLISHED" | "ARCHIVED";
export type AnnouncementPriority = "LOW" | "NORMAL" | "HIGH" | "URGENT";
export type AnnouncementTargetRole = "ADMIN" | "TEACHER" | "STAFF" | "PARENT" | "STUDENT";

export const ANNOUNCEMENT_TARGET_TYPES: readonly AnnouncementTargetType[] = [
  "ALL",
  "ROLE",
  "CLASS",
] as const;

export const ANNOUNCEMENT_STATUSES: readonly AnnouncementStatus[] = [
  "DRAFT",
  "PUBLISHED",
  "ARCHIVED",
] as const;

export const ANNOUNCEMENT_PRIORITIES: readonly AnnouncementPriority[] = [
  "LOW",
  "NORMAL",
  "HIGH",
  "URGENT",
] as const;

export const ANNOUNCEMENT_TARGET_ROLES: readonly AnnouncementTargetRole[] = [
  "TEACHER",
  "STAFF",
  "PARENT",
  "STUDENT",
] as const;

export interface Announcement {
  id: string;
  title: string;
  content: string;
  author_id: string;
  author_email: string;
  author_name: string;
  status: AnnouncementStatus | string;
  target_type: AnnouncementTargetType | string;
  target_role?: string | null;
  target_class_id?: string | null;
  target_class_name?: string | null;
  priority: AnnouncementPriority | string;
  published_at?: string | null;
  expires_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface AnnouncementListResponse {
  data: Announcement[];
  meta: {
    page: number;
    page_size: number;
    total: number;
  };
}

export interface CreateAnnouncementInput {
  title: string;
  content: string;
  target_type: AnnouncementTargetType | string;
  target_role?: string;
  target_class_id?: string;
  priority?: AnnouncementPriority | string;
  /** DRAFT or PUBLISHED — the composer can save or publish directly. */
  status?: "DRAFT" | "PUBLISHED";
  /** RFC3339 or YYYY-MM-DD; empty/omitted = never expires. */
  expires_at?: string;
}

export interface UpdateAnnouncementInput {
  title?: string;
  content?: string;
  target_type?: AnnouncementTargetType | string;
  target_role?: string;
  /** Empty string clears the class; UUID switches it. */
  target_class_id?: string;
  priority?: AnnouncementPriority | string;
  /** Empty string clears the expiry. */
  expires_at?: string;
}
