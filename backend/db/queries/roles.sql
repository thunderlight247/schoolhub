-- name: ListRoles :many
SELECT * FROM roles
ORDER BY name;

-- name: GetRoleByID :one
SELECT * FROM roles
WHERE id = $1;

-- name: GetRoleByName :one
SELECT * FROM roles
WHERE name = $1;
