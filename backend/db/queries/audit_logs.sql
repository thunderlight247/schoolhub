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
