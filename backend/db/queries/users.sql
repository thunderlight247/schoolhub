-- name: CreateUser :one
INSERT INTO users (
    person_id, email, password_hash
) VALUES (
    sqlc.arg(person_id), sqlc.arg(email), sqlc.arg(password_hash)
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
-- The login lookup. Returns SUSPENDED/INACTIVE users too — M4's auth
-- service is responsible for checking `status` before issuing a session,
-- not this query (see fix(auth): prevent suspended users from logging in
-- in section 35's example commit log).
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByPersonID :one
SELECT * FROM users
WHERE person_id = $1;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = sqlc.arg(password_hash)
WHERE id = sqlc.arg(id);

-- name: UpdateUserStatus :exec
UPDATE users
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(id);

-- name: RecordUserLogin :exec
UPDATE users
SET last_login_at = now()
WHERE id = $1;
