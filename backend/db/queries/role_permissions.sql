-- name: ListPermissionsForRole :many
SELECT p.*
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = $1
ORDER BY p.code;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES (sqlc.arg(role_id), sqlc.arg(permission_id))
ON CONFLICT DO NOTHING;

-- name: RevokePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = sqlc.arg(role_id) AND permission_id = sqlc.arg(permission_id);
