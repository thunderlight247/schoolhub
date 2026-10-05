-- name: CreateEnrollment :one
INSERT INTO enrollments (
    student_id,
    academic_year_id,
    class_id,
    status,
    enrollment_date,
    notes
) VALUES (
    sqlc.arg(student_id),
    sqlc.arg(academic_year_id),
    sqlc.arg(class_id),
    sqlc.arg(status),
    sqlc.arg(enrollment_date),
    sqlc.narg(notes)
)
RETURNING *;

-- name: GetEnrollmentByID :one
SELECT
    e.id,
    e.student_id,
    e.academic_year_id,
    e.class_id,
    e.status,
    e.enrollment_date,
    e.exit_date,
    e.notes,
    e.created_at,
    e.updated_at,
    s.student_number,
    p.first_name || ' ' || p.last_name AS student_name,
    ay.name AS academic_year_name,
    c.name AS class_name
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN people p ON p.id = s.person_id
JOIN academic_years ay ON ay.id = e.academic_year_id
JOIN classes c ON c.id = e.class_id
WHERE e.id = $1;

-- name: ListEnrollments :many
SELECT
    e.id,
    e.student_id,
    e.academic_year_id,
    e.class_id,
    e.status,
    e.enrollment_date,
    e.exit_date,
    e.notes,
    e.created_at,
    e.updated_at,
    s.student_number,
    p.first_name || ' ' || p.last_name AS student_name,
    ay.name AS academic_year_name,
    c.name AS class_name
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN people p ON p.id = s.person_id
JOIN academic_years ay ON ay.id = e.academic_year_id
JOIN classes c ON c.id = e.class_id
WHERE
    (sqlc.narg('student_id')::uuid IS NULL OR e.student_id = sqlc.narg('student_id'))
    AND (sqlc.narg('academic_year_id')::uuid IS NULL OR e.academic_year_id = sqlc.narg('academic_year_id'))
    AND (sqlc.narg('class_id')::uuid IS NULL OR e.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('status')::text IS NULL OR e.status = sqlc.narg('status'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' ' || s.student_number || ' ' || c.name
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY e.enrollment_date DESC, e.created_at DESC
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountEnrollments :one
SELECT count(*)
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN people p ON p.id = s.person_id
JOIN academic_years ay ON ay.id = e.academic_year_id
JOIN classes c ON c.id = e.class_id
WHERE
    (sqlc.narg('student_id')::uuid IS NULL OR e.student_id = sqlc.narg('student_id'))
    AND (sqlc.narg('academic_year_id')::uuid IS NULL OR e.academic_year_id = sqlc.narg('academic_year_id'))
    AND (sqlc.narg('class_id')::uuid IS NULL OR e.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('status')::text IS NULL OR e.status = sqlc.narg('status'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' ' || s.student_number || ' ' || c.name
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    );

-- name: UpdateEnrollmentStatus :one
UPDATE enrollments
SET
    status = sqlc.arg(status),
    exit_date = CASE
        WHEN sqlc.arg(status) IN ('WITHDRAWN', 'COMPLETED', 'TRANSFERRED') THEN COALESCE(exit_date, CURRENT_DATE)
        ELSE NULL
    END
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: HasActiveEnrollmentForStudentYear :one
SELECT EXISTS (
    SELECT 1
    FROM enrollments e
    WHERE e.student_id = sqlc.arg(student_id)
      AND e.academic_year_id = sqlc.arg(academic_year_id)
      AND e.status IN ('ENROLLED', 'COMPLETED', 'TRANSFERRED')
);
