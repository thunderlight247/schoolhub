-- name: ListPermissions :many
SELECT * FROM permissions
ORDER BY code;

-- name: GetPermissionByCode :one
SELECT * FROM permissions
WHERE code = $1;
