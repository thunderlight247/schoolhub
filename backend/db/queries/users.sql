-- name: CreateUser :one
INSERT INTO users (
    person_id, email, password_hash
) VALUES (
    sqlc.arg(person_id), sqlc.arg(email), sqlc.arg(password_hash)
)
RETURNING *;

-- name: SetUserInvitedBy :exec
-- M34: inviter tracking — every provisioned login (admin/staff/parent
-- account) stores who minted it, powering "see your invitees" and
-- inviter-block rights without a separate table. Admins re-provisioning
-- themselves? services refuse self-provisioning, so no self-cycle.
UPDATE users
SET invited_by = sqlc.arg(invited_by)
WHERE id = sqlc.arg(id);

-- name: ListUsersInvitedBy :many
-- GET /my-invitees: the accounts the caller personally provisioned,
-- together with their current role + status for the /access section.
SELECT
    u.id,
    u.email,
    u.status,
    u.created_at,
    p.first_name,
    p.last_name,
    COALESCE(
        (SELECT string_agg(r.name, ',' ORDER BY r.name)
         FROM user_roles ur
         JOIN roles r ON r.id = ur.role_id
         WHERE ur.user_id = u.id),
        ''
    ) AS roles
FROM users u
JOIN people p ON p.id = u.person_id
WHERE u.invited_by = sqlc.arg(invited_by)
ORDER BY u.created_at DESC;

-- name: CountActiveMasterAdminsExcept :one
-- Last-master guard for the blocking rules (M34): how many OTHER users are
-- ACTIVE masters — zero means the candidate is the only one and must not
-- be suspended.
SELECT count(*)::bigint
FROM users u
WHERE u.id <> sqlc.arg(exclude_user_id)
  AND u.status = 'ACTIVE'
  AND EXISTS (
      SELECT 1 FROM user_roles ur
      JOIN roles r ON r.id = ur.role_id
      WHERE ur.user_id = u.id AND r.name = 'MASTER_ADMIN'
  );

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

-- name: SetUserMustChangePassword :exec
-- M34: bootstrap accounts (the master admin) are provisioned with a
-- temporary password the developer knows; the flag forces a change at the
-- first login (the auth middleware rejects everything but change/logout/me
-- while it is set) so the bootstrap credential cannot be reused afterwards.
UPDATE users
SET must_change_password = sqlc.arg(must_change_password)
WHERE id = sqlc.arg(id);

-- name: RecordUserLogin :exec
UPDATE users
SET last_login_at = now()
WHERE id = $1;
