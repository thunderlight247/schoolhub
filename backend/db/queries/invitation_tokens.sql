-- name: CreateInvitationToken :one
INSERT INTO invitation_tokens (
    person_id, email, role_id, invited_by, token_hash, expires_at
) VALUES (
    sqlc.arg(person_id), sqlc.arg(email), sqlc.arg(role_id), sqlc.arg(invited_by), sqlc.arg(token_hash), sqlc.arg(expires_at)
)
RETURNING *;

-- name: GetInvitationTokenByHash :one
SELECT * FROM invitation_tokens
WHERE token_hash = $1;

-- name: MarkInvitationTokenAccepted :exec
UPDATE invitation_tokens
SET accepted_at = now()
WHERE id = $1;
