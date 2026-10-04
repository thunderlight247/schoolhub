-- name: ListDepartments :many
SELECT * FROM departments
ORDER BY name;

-- name: GetDepartmentByID :one
SELECT * FROM departments
WHERE id = $1;

-- name: CreateDepartment :one
INSERT INTO departments (name, description)
VALUES (sqlc.arg(name), sqlc.narg(description))
RETURNING *;
