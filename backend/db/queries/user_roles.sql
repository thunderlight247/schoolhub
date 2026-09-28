-- name: ListRolesForUser :many
SELECT r.*
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.name;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id)
VALUES (sqlc.arg(user_id), sqlc.arg(role_id))
ON CONFLICT DO NOTHING;

-- name: RevokeRoleFromUser :exec
DELETE FROM user_roles
WHERE user_id = sqlc.arg(user_id) AND role_id = sqlc.arg(role_id);

-- name: ListPermissionsForUser :many
-- Every permission the user holds across all of their roles, deduplicated.
-- Section 11: "The backend must enforce authorization" — this is the query
-- the auth middleware will call to build a user's effective permission set.
SELECT DISTINCT p.*
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1
ORDER BY p.code;

-- name: UserHasPermission :one
-- A single boolean check, cheaper than fetching the full permission list
-- when a handler only needs to gate one action (e.g. "can this user do
-- student.archive?").
SELECT EXISTS (
    SELECT 1
    FROM role_permissions rp
    JOIN user_roles ur ON ur.role_id = rp.role_id
    JOIN permissions p ON p.id = rp.permission_id
    WHERE ur.user_id = sqlc.arg(user_id)
      AND p.code = sqlc.arg(permission_code)
) AS has_permission;
