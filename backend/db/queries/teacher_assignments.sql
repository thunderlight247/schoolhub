-- name: CreateTeacherAssignment :one
INSERT INTO teacher_assignments (teacher_id, class_id, subject_id)
VALUES (sqlc.arg(teacher_id), sqlc.arg(class_id), sqlc.arg(subject_id))
RETURNING *;

-- name: GetTeacherAssignmentByID :one
-- The assignment detail row: mapping + the human-readable names the UI
-- shows (teacher person data via the staff row, class, its academic year,
-- subject), in one round trip — same shape as GetEnrollmentByID.
SELECT
    ta.id,
    ta.teacher_id,
    ta.class_id,
    ta.subject_id,
    ta.created_at,
    ta.updated_at,
    s.staff_number,
    p.first_name || ' ' || p.last_name AS teacher_name,
    c.name AS class_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    sub.code AS subject_code,
    sub.name AS subject_name
FROM teacher_assignments ta
JOIN staff s ON s.id = ta.teacher_id
JOIN people p ON p.id = s.person_id
JOIN classes c ON c.id = ta.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = ta.subject_id
WHERE ta.id = sqlc.arg(id);

-- name: ListTeacherAssignments :many
-- Paginated, filtered assignment list. Every filter is optional: a NULL
-- narg disables that condition. academic_year_id filters through the
-- class (classes carry the year, 000008). Keep this WHERE clause and
-- CountTeacherAssignments' in sync — see the same note on ListStaff.
SELECT
    ta.id,
    ta.teacher_id,
    ta.class_id,
    ta.subject_id,
    ta.created_at,
    ta.updated_at,
    s.staff_number,
    p.first_name || ' ' || p.last_name AS teacher_name,
    c.name AS class_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    sub.code AS subject_code,
    sub.name AS subject_name
FROM teacher_assignments ta
JOIN staff s ON s.id = ta.teacher_id
JOIN people p ON p.id = s.person_id
JOIN classes c ON c.id = ta.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = ta.subject_id
WHERE
    (sqlc.narg('teacher_id')::uuid IS NULL OR ta.teacher_id = sqlc.narg('teacher_id'))
    AND (sqlc.narg('class_id')::uuid IS NULL OR ta.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('subject_id')::uuid IS NULL OR ta.subject_id = sqlc.narg('subject_id'))
    AND (sqlc.narg('academic_year_id')::uuid IS NULL OR c.academic_year_id = sqlc.narg('academic_year_id'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' ' || s.staff_number || ' '
            || c.name || ' ' || sub.name
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY c.name, sub.name, p.last_name, p.first_name
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountTeacherAssignments :one
-- Must stay in sync with ListTeacherAssignments' WHERE clause (see the
-- comment there).
SELECT count(*)
FROM teacher_assignments ta
JOIN staff s ON s.id = ta.teacher_id
JOIN people p ON p.id = s.person_id
JOIN classes c ON c.id = ta.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = ta.subject_id
WHERE
    (sqlc.narg('teacher_id')::uuid IS NULL OR ta.teacher_id = sqlc.narg('teacher_id'))
    AND (sqlc.narg('class_id')::uuid IS NULL OR ta.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('subject_id')::uuid IS NULL OR ta.subject_id = sqlc.narg('subject_id'))
    AND (sqlc.narg('academic_year_id')::uuid IS NULL OR c.academic_year_id = sqlc.narg('academic_year_id'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' ' || s.staff_number || ' '
            || c.name || ' ' || sub.name
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    );

-- name: DeleteTeacherAssignment :exec
DELETE FROM teacher_assignments
WHERE id = sqlc.arg(id);

-- name: HasClassSubject :one
-- An assignment may only cover a subject the class actually teaches
-- (class_subjects, 000008).
SELECT EXISTS (
    SELECT 1
    FROM class_subjects cs
    WHERE cs.class_id = sqlc.arg(class_id)
      AND cs.subject_id = sqlc.arg(subject_id)
);
