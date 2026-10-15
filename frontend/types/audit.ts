// Audit log domain types (M32).
// Mirrors backend/internal/audit DTOs (M19 handlers): entryDTO and the
// list response envelope `{ audit_logs, pagination }`.

export interface AuditEntry {
  id: string;
  actor_user_id?: string;
  actor_email?: string;
  actor_first_name?: string;
  actor_last_name?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  /** Decoded JSONB metadata — arbitrary key/value pairs Record wrote. */
  metadata: Record<string, unknown>;
  /** RFC3339 timestamp. */
  created_at: string;
}

export interface AuditListResponse {
  audit_logs: AuditEntry[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
}

/** GET /audit-logs/actions payload. */
export interface AuditActionsResponse {
  actions: string[];
}

/** GET /audit-logs query params (all optional). Note date_from/date_to are
 * RFC3339 timestamps — the backend rejects bare YYYY-MM-DD here. */
export interface AuditListParams {
  actor_user_id?: string;
  action?: string;
  resource_type?: string;
  resource_id?: string;
  date_from?: string;
  date_to?: string;
  page?: number;
  page_size?: number;
}
