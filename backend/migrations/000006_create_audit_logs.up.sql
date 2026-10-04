-- Audit logging (section 29, infrastructure added in M6).
--
-- M6 is the first milestone with real administrative mutations
-- (STAFF_CREATED, STAFF_UPDATED, USER_CREATED ...), so the table lands here
-- rather than M19 having to retrofit audit writes into every handler built
-- in M6–M13. M19 adds the authorized read/report endpoints (audit.view).
--
-- Shape per section 29: actor, action, resource type/id, timestamp,
-- metadata. Design notes:
--   * actor_user_id is NULLABLE: system/background actions have no human
--     actor. ON DELETE SET NULL — if a user account were ever removed, the
--     historical log entry must survive (section 7); users are never
--     hard-deleted in practice anyway.
--   * resource_id is NULLABLE: some actions target a collection or a
--     non-UUID resource (e.g. a filter config), not a single row.
--   * metadata is JSONB, defaulting to '{}': action-specific context (old
--     -> new status, affected role name, ...) without one column per case.
--   * No updated_at: audit rows are append-only. Nothing in this codebase
--     UPDATEs or DELETEs them; a database-level guard trigger can be added
--     in M20 (security hardening) if wanted — not blocking here (section 39).

CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   UUID,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes serve the audit.view screens/reports planned for M19: filter by
-- actor, by action type, by resource, and time-ordered browsing.
CREATE INDEX idx_audit_logs_actor    ON audit_logs (actor_user_id);
CREATE INDEX idx_audit_logs_action   ON audit_logs (action);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
