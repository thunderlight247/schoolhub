-- name: CreateStudent :one
INSERT INTO students (
    person_id, student_number, admission_date, status
) VALUES (
    sqlc.arg(person_id), sqlc.arg(student_number), sqlc.narg(admission_date), sqlc.arg(status)
)
RETURNING *;

-- name: GetStudentByID :one
SELECT
    s.id,
    s.person_id,
    s.student_number,
    s.admission_date,
    s.status,
    s.created_at,
    s.updated_at,
    p.first_name,
    p.last_name,
    p.other_names,
    p.date_of_birth,
    p.gender,
    p.phone,
    p.email,
    p.address
FROM students s
JOIN people p ON p.id = s.person_id
WHERE s.id = sqlc.arg(id);

-- name: ListStudents :many
-- Paginated, filtered student list (section 25). Keep this WHERE clause
-- and CountStudents' in sync — see the same note on ListStaff in staff.sql.
SELECT
    s.id,
    s.student_number,
    s.status,
    s.admission_date,
    s.created_at,
    p.first_name,
    p.last_name,
    p.other_names
FROM students s
JOIN people p ON p.id = s.person_id
WHERE
    (sqlc.narg('status')::text IS NULL OR s.status = sqlc.narg('status'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' '
            || COALESCE(p.other_names, '') || ' ' || s.student_number
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY p.last_name, p.first_name, s.student_number
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountStudents :one
-- Must stay in sync with ListStudents' WHERE clause.
SELECT count(*)
FROM students s
JOIN people p ON p.id = s.person_id
WHERE
    (sqlc.narg('status')::text IS NULL OR s.status = sqlc.narg('status'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' '
            || COALESCE(p.other_names, '') || ' ' || s.student_number
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    );

-- name: SetStudentStatus :one
UPDATE students
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(id)
RETURNING *;
