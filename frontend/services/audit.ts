// Audit log service (M32). Paths are BARE — lib/api.ts already prefixes
// /api/v1 and handles credentials + error shaping. All endpoints are
// read-only; the backend gates them behind auth + audit.view (mounted in
// cmd/server/main.go).

import { apiGet } from "@/lib/api";
import type {
  AuditActionsResponse,
  AuditListParams,
  AuditListResponse,
  AuditEntry,
} from "@/types/audit";

function toQuery(params: AuditListParams): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== null && v !== "") {
      q.set(k, String(v));
    }
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

/** Filtered, paginated audit trail (GET /audit-logs). */
export function listAuditLogs(
  params: AuditListParams = {},
): Promise<AuditListResponse> {
  return apiGet<AuditListResponse>(`/audit-logs${toQuery(params)}`);
}

/** Single entry detail (GET /audit-logs/:id). */
export function getAuditLog(id: string): Promise<AuditEntry> {
  return apiGet<{ audit_log: AuditEntry }>(`/audit-logs/${id}`).then(
    (r) => r.audit_log,
  );
}

/** Distinct action codes for the filter dropdown (GET /audit-logs/actions). */
export function listAuditActions(): Promise<string[]> {
  return apiGet<AuditActionsResponse>("/audit-logs/actions").then(
    (r) => r.actions,
  );
}
