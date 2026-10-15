-- name: CreateAuditLog :exec
-- Append-only: no UPDATE/DELETE queries exist for this table by design
-- (see migration 000006). Callers pass a tx-bound Querier so the audit
-- row commits atomically with the mutation it describes.
INSERT INTO audit_logs (
    actor_user_id, action, resource_type, resource_id, metadata
) VALUES (
    sqlc.narg(actor_user_id), sqlc.arg(action), sqlc.arg(resource_type),
    sqlc.narg(resource_id), sqlc.arg(metadata)
);

-- M19: authorized admin audit review (section 19/29). Read-only queries
-- against the same append-only table CreateAuditLog writes to.

-- name: ListAuditLogs :many
-- Paginated, filtered audit log (section 25's pagination discipline
-- applies here too). Actor is resolved via a join to users/people so the
-- frontend doesn't need a second round trip per row to show "who did
-- this" — LEFT JOIN because actor_user_id is nullable (migration 000006:
-- system actions have no actor, and a removed user's historical entries
-- survive via ON DELETE SET NULL).
SELECT
    a.id,
    a.actor_user_id,
    a.action,
    a.resource_type,
    a.resource_id,
    a.metadata,
    a.created_at,
    u.email AS actor_email,
    p.first_name AS actor_first_name,
    p.last_name AS actor_last_name
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_user_id
LEFT JOIN people p ON p.id = u.person_id
WHERE
    (sqlc.narg('actor_user_id')::uuid IS NULL OR a.actor_user_id = sqlc.narg('actor_user_id'))
    AND (sqlc.narg('action')::text IS NULL OR a.action = sqlc.narg('action'))
    AND (sqlc.narg('resource_type')::text IS NULL OR a.resource_type = sqlc.narg('resource_type'))
    AND (sqlc.narg('resource_id')::uuid IS NULL OR a.resource_id = sqlc.narg('resource_id'))
    AND (sqlc.narg('date_from')::timestamptz IS NULL OR a.created_at >= sqlc.narg('date_from'))
    AND (sqlc.narg('date_to')::timestamptz IS NULL OR a.created_at <= sqlc.narg('date_to'))
ORDER BY a.created_at DESC
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountAuditLogs :one
-- Must stay in sync with ListAuditLogs' WHERE clause.
SELECT count(*)
FROM audit_logs a
WHERE
    (sqlc.narg('actor_user_id')::uuid IS NULL OR a.actor_user_id = sqlc.narg('actor_user_id'))
    AND (sqlc.narg('action')::text IS NULL OR a.action = sqlc.narg('action'))
    AND (sqlc.narg('resource_type')::text IS NULL OR a.resource_type = sqlc.narg('resource_type'))
    AND (sqlc.narg('resource_id')::uuid IS NULL OR a.resource_id = sqlc.narg('resource_id'))
    AND (sqlc.narg('date_from')::timestamptz IS NULL OR a.created_at >= sqlc.narg('date_from'))
    AND (sqlc.narg('date_to')::timestamptz IS NULL OR a.created_at <= sqlc.narg('date_to'));

-- name: GetAuditLogByID :one
SELECT
    a.id,
    a.actor_user_id,
    a.action,
    a.resource_type,
    a.resource_id,
    a.metadata,
    a.created_at,
    u.email AS actor_email,
    p.first_name AS actor_first_name,
    p.last_name AS actor_last_name
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_user_id
LEFT JOIN people p ON p.id = u.person_id
WHERE a.id = sqlc.arg(id);

-- name: ListDistinctAuditActions :many
-- Backs a frontend filter dropdown with the action codes that actually
-- appear in the log, rather than hardcoding the full set client-side —
-- that set has grown with every milestone since M6 and will keep growing.
SELECT DISTINCT action FROM audit_logs ORDER BY action;
