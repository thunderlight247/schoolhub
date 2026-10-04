-- name: CreateStaff :one
INSERT INTO staff (
    person_id, staff_number, department_id, position_id,
    employment_status, hired_date
) VALUES (
    sqlc.arg(person_id), sqlc.arg(staff_number),
    sqlc.narg(department_id), sqlc.narg(position_id),
    sqlc.arg(employment_status), sqlc.narg(hired_date)
)
RETURNING *;

-- name: GetStaffByID :one
-- The staff detail row: employment record + person data + the lookup names,
-- in one round trip. The account columns stay out of this join on purpose —
-- the service resolves the (optionally absent) user account separately via
-- GetUserByPersonID, which sidesteps nullable-uuid edge cases in the scan
-- struct for a LEFT JOINed users row.
SELECT
    s.id,
    s.person_id,
    s.staff_number,
    s.department_id,
    s.position_id,
    s.employment_status,
    s.hired_date,
    s.created_at,
    s.updated_at,
    p.first_name,
    p.last_name,
    p.other_names,
    p.date_of_birth,
    p.gender,
    p.phone,
    p.email,
    p.address,
    d.name  AS department_name,
    sp.title AS position_title,
    sp.category AS position_category
FROM staff s
JOIN people p ON p.id = s.person_id
LEFT JOIN departments d ON d.id = s.department_id
LEFT JOIN staff_positions sp ON sp.id = s.position_id
WHERE s.id = sqlc.arg(id);

-- name: ListStaff :many
-- Paginated, filtered staff list (section 25). Filtering happens here at
-- the database layer, never in application code. Every filter is optional:
-- a NULL narg disables that condition. Keep this WHERE clause and
-- CountStaff's in sync — they must filter identically for total_pages to
-- be correct.
SELECT
    s.id,
    s.staff_number,
    s.employment_status,
    s.created_at,
    p.first_name,
    p.last_name,
    p.other_names,
    d.name AS department_name,
    sp.title AS position_title,
    sp.category AS position_category,
    EXISTS (SELECT 1 FROM users u WHERE u.person_id = s.person_id) AS has_account
FROM staff s
JOIN people p ON p.id = s.person_id
LEFT JOIN departments d ON d.id = s.department_id
LEFT JOIN staff_positions sp ON sp.id = s.position_id
WHERE
    (sqlc.narg('department_id')::uuid IS NULL
        OR s.department_id = sqlc.narg('department_id'))
    AND (sqlc.narg('employment_status')::text IS NULL
        OR s.employment_status = sqlc.narg('employment_status'))
    AND (sqlc.narg('position_category')::text IS NULL
        OR sp.category = sqlc.narg('position_category'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' '
            || COALESCE(p.other_names, '') || ' ' || s.staff_number
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    )
ORDER BY p.last_name, p.first_name, s.staff_number
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountStaff :one
-- Must stay in sync with ListStaff's WHERE clause (see comment there).
SELECT count(*)
FROM staff s
JOIN people p ON p.id = s.person_id
LEFT JOIN departments d ON d.id = s.department_id
LEFT JOIN staff_positions sp ON sp.id = s.position_id
WHERE
    (sqlc.narg('department_id')::uuid IS NULL
        OR s.department_id = sqlc.narg('department_id'))
    AND (sqlc.narg('employment_status')::text IS NULL
        OR s.employment_status = sqlc.narg('employment_status'))
    AND (sqlc.narg('position_category')::text IS NULL
        OR sp.category = sqlc.narg('position_category'))
    AND (
        sqlc.narg('search')::text IS NULL
        OR (
            p.first_name || ' ' || p.last_name || ' '
            || COALESCE(p.other_names, '') || ' ' || s.staff_number
        ) ILIKE '%' || sqlc.narg('search')::text || '%'
    );

-- name: UpdateStaffAssignment :one
-- The staff.update path: department, position, and hired date. Employment
-- status is deliberately NOT set here — status changes go through
-- SetStaffEmploymentStatus so they can carry the staff.deactivate
-- permission and their own audit event.
UPDATE staff
SET
    department_id = sqlc.narg(department_id),
    position_id   = sqlc.narg(position_id),
    hired_date    = sqlc.narg(hired_date)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetStaffEmploymentStatus :one
UPDATE staff
SET employment_status = sqlc.arg(employment_status)
WHERE id = sqlc.arg(id)
RETURNING *;
