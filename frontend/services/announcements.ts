// Announcements service (M30). Paths are BARE — lib/api.ts already prefixes
// /api/v1 and handles credentials + error shaping.

import { apiGet, apiPost, apiPatch, apiDelete } from "@/lib/api";
import type {
  Announcement,
  AnnouncementListResponse,
  CreateAnnouncementInput,
  UpdateAnnouncementInput,
} from "@/types/announcements";

export interface AnnouncementListParams {
  status?: string;
  target_type?: string;
  target_role?: string;
  class_id?: string;
  author_id?: string;
  search?: string;
  page?: number;
  page_size?: number;
  /** true = audience feed (what I can see), false/omitted = admin directory. */
  feed?: boolean;
}

function toQuery(params: AnnouncementListParams): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== "") {
      q.set(k, String(v));
    }
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

/** Admin directory listing with filters + pagination. */
export function listAnnouncements(
  params: AnnouncementListParams = {},
): Promise<AnnouncementListResponse> {
  return apiGet<AnnouncementListResponse>(`/announcements${toQuery(params)}`);
}

/** Audience feed (`GET /announcements?feed=true`). */
export function listAnnouncementFeed(
  params: { page?: number; page_size?: number } = {},
): Promise<AnnouncementListResponse> {
  return apiGet<AnnouncementListResponse>(
    `/announcements${toQuery({ feed: true, ...params })}`,
  );
}

export function getAnnouncement(id: string): Promise<Announcement> {
  return apiGet<{ data: Announcement }>(`/announcements/${id}`).then((r) => r.data);
}

export function createAnnouncement(input: CreateAnnouncementInput): Promise<Announcement> {
  return apiPost<{ data: Announcement }>("/announcements", input).then((r) => r.data);
}

export function updateAnnouncement(
  id: string,
  input: UpdateAnnouncementInput,
): Promise<Announcement> {
  return apiPatch<{ data: Announcement }>(`/announcements/${id}`, input).then((r) => r.data);
}

export function publishAnnouncement(id: string): Promise<Announcement> {
  return apiPost<{ data: Announcement }>(`/announcements/${id}/publish`).then((r) => r.data);
}

export function archiveAnnouncement(id: string): Promise<Announcement> {
  return apiPost<{ data: Announcement }>(`/announcements/${id}/archive`).then((r) => r.data);
}

export function deleteAnnouncement(id: string): Promise<void> {
  return apiDelete<void>(`/announcements/${id}`);
}
