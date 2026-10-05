-- name: CreateSubject :one
INSERT INTO subjects (code, name, description, type, department_id)
VALUES (sqlc.arg(code), sqlc.arg(name), sqlc.narg(description), sqlc.arg(type), sqlc.narg(department_id))
RETURNING *;

-- name: GetSubjectByID :one
SELECT
    s.*,
    d.name AS department_name
FROM subjects s
LEFT JOIN departments d ON d.id = s.department_id
WHERE s.id = $1;

-- name: GetSubjectByCode :one
SELECT * FROM subjects
WHERE code = $1;

-- name: ListSubjects :many
SELECT
    s.*,
    d.name AS department_name
FROM subjects s
LEFT JOIN departments d ON d.id = s.department_id
WHERE
    (sqlc.narg('type')::text IS NULL OR s.type = sqlc.narg('type'))
    AND (sqlc.narg('department_id')::uuid IS NULL OR s.department_id = sqlc.narg('department_id'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (s.name || ' ' || s.code) ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY s.name ASC;

-- name: UpdateSubject :one
UPDATE subjects
SET
    code = COALESCE(sqlc.narg(code), code),
    name = COALESCE(sqlc.narg(name), name),
    description = COALESCE(sqlc.narg(description), description),
    type = COALESCE(sqlc.narg(type), type),
    department_id = COALESCE(sqlc.narg(department_id), department_id)
WHERE id = sqlc.arg(id)
RETURNING *;
