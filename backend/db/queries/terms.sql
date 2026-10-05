-- name: CreateTerm :one
INSERT INTO terms (academic_year_id, name, start_date, end_date, is_current)
VALUES (sqlc.arg(academic_year_id), sqlc.arg(name), sqlc.arg(start_date), sqlc.arg(end_date), sqlc.arg(is_current))
RETURNING *;

-- name: GetTermByID :one
SELECT
    t.*,
    ay.name AS academic_year_name
FROM terms t
JOIN academic_years ay ON ay.id = t.academic_year_id
WHERE t.id = $1;

-- name: GetCurrentTerm :one
SELECT
    t.*,
    ay.name AS academic_year_name
FROM terms t
JOIN academic_years ay ON ay.id = t.academic_year_id
WHERE t.is_current = true
LIMIT 1;

-- name: ListTermsForAcademicYear :many
SELECT
    t.*,
    ay.name AS academic_year_name
FROM terms t
JOIN academic_years ay ON ay.id = t.academic_year_id
WHERE t.academic_year_id = $1
ORDER BY t.start_date ASC;

-- name: ListAllTerms :many
SELECT
    t.*,
    ay.name AS academic_year_name
FROM terms t
JOIN academic_years ay ON ay.id = t.academic_year_id
ORDER BY ay.start_date DESC, t.start_date ASC;

-- name: UpdateTerm :one
UPDATE terms
SET
    name = COALESCE(sqlc.narg(name), name),
    start_date = COALESCE(sqlc.narg(start_date), start_date),
    end_date = COALESCE(sqlc.narg(end_date), end_date)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UnsetCurrentTerms :exec
UPDATE terms
SET is_current = false
WHERE is_current = true;

-- name: SetTermCurrent :one
UPDATE terms
SET is_current = true
WHERE id = sqlc.arg(id)
RETURNING *;
