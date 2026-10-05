-- name: CreateAcademicYear :one
INSERT INTO academic_years (name, start_date, end_date, is_current)
VALUES (sqlc.arg(name), sqlc.arg(start_date), sqlc.arg(end_date), sqlc.arg(is_current))
RETURNING *;

-- name: GetAcademicYearByID :one
SELECT * FROM academic_years
WHERE id = $1;

-- name: GetCurrentAcademicYear :one
SELECT * FROM academic_years
WHERE is_current = true
LIMIT 1;

-- name: ListAcademicYears :many
SELECT * FROM academic_years
ORDER BY start_date DESC;

-- name: UpdateAcademicYear :one
UPDATE academic_years
SET
    name = COALESCE(sqlc.narg(name), name),
    start_date = COALESCE(sqlc.narg(start_date), start_date),
    end_date = COALESCE(sqlc.narg(end_date), end_date)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UnsetCurrentAcademicYears :exec
UPDATE academic_years
SET is_current = false
WHERE is_current = true;

-- name: SetAcademicYearCurrent :one
UPDATE academic_years
SET is_current = true
WHERE id = sqlc.arg(id)
RETURNING *;
