-- name: CreateSession :one
INSERT INTO sessions (
    user_id, token_hash, user_agent, ip_address, expires_at
) VALUES (
    sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.narg(user_agent), sqlc.narg(ip_address), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT * FROM sessions
WHERE token_hash = $1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteAllSessionsForUser :exec
-- Used on password change (and by the admin "suspend user" action in M5) to
-- force re-login everywhere, not just the current device.
DELETE FROM sessions
WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
-- No cron/scheduler exists yet to call this periodically (deliberately out
-- of scope for M4 — see section 39, not adding infrastructure speculatively).
-- Expired sessions are already rejected by ValidateSession's expiry check
-- regardless; this just reclaims the rows.
DELETE FROM sessions
WHERE expires_at < now();
