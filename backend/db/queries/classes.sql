-- name: CreateClass :one
INSERT INTO classes (academic_year_id, programme_id, name, level, capacity)
VALUES (sqlc.arg(academic_year_id), sqlc.narg(programme_id), sqlc.arg(name), sqlc.narg(level), sqlc.narg(capacity))
RETURNING *;

-- name: GetClassByID :one
SELECT
    c.*,
    ay.name AS academic_year_name,
    p.name AS programme_name,
    p.code AS programme_code
FROM classes c
JOIN academic_years ay ON ay.id = c.academic_year_id
LEFT JOIN programmes p ON p.id = c.programme_id
WHERE c.id = $1;

-- name: ListClasses :many
SELECT
    c.*,
    ay.name AS academic_year_name,
    p.name AS programme_name,
    p.code AS programme_code
FROM classes c
JOIN academic_years ay ON ay.id = c.academic_year_id
LEFT JOIN programmes p ON p.id = c.programme_id
WHERE
    (sqlc.narg('academic_year_id')::uuid IS NULL OR c.academic_year_id = sqlc.narg('academic_year_id'))
    AND (sqlc.narg('programme_id')::uuid IS NULL OR c.programme_id = sqlc.narg('programme_id'))
    AND (sqlc.narg('level')::text IS NULL OR c.level = sqlc.narg('level'))
ORDER BY ay.start_date DESC, c.name ASC;

-- name: UpdateClass :one
UPDATE classes
SET
    programme_id = COALESCE(sqlc.narg(programme_id), programme_id),
    name = COALESCE(sqlc.narg(name), name),
    level = COALESCE(sqlc.narg(level), level),
    capacity = COALESCE(sqlc.narg(capacity), capacity)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: AssignSubjectToClass :one
INSERT INTO class_subjects (class_id, subject_id)
VALUES (sqlc.arg(class_id), sqlc.arg(subject_id))
RETURNING *;

-- name: RemoveSubjectFromClass :execrows
DELETE FROM class_subjects
WHERE class_id = sqlc.arg(class_id) AND subject_id = sqlc.arg(subject_id);

-- name: ListSubjectsForClass :many
SELECT
    s.id,
    s.code,
    s.name,
    s.description,
    s.type,
    s.department_id,
    cs.created_at AS assigned_at
FROM class_subjects cs
JOIN subjects s ON s.id = cs.subject_id
WHERE cs.class_id = $1
ORDER BY s.name ASC;
