-- Parent portal (milestone M16). Read-only aggregation over the guardian
-- domain (M7), enrollment history (M9), attendance (M11), results (M12), and
-- announcements (M13). No schema of its own — mirrors how M14 (admin
-- dashboard) and M15 (teacher portal) work.
--
-- Row-level scoping note (flagged in migration 000004's comments): RBAC only
-- knows permission codes, so "parents see their own children's records" is
-- enforced here by anchoring every query on guardian_id resolved from the
-- authenticated user, and by the IsGuardianLinkedToStudent check the service
-- runs before returning any per-child detail.

-- name: GetGuardianProfileByUserID :one
-- Resolve the guardian record behind a PARENT-role account, mirroring
-- GetTeacherProfileByUserID in teacher_portal.sql. A parent user is a person
-- with a guardians row; the INNER JOINs mean a non-guardian account yields
-- pgx.ErrNoRows, which the service maps to a friendly 404.
SELECT
    g.id AS guardian_id,
    g.occupation,
    p.first_name,
    p.last_name,
    p.email,
    p.phone
FROM users u
JOIN people p ON p.id = u.person_id
JOIN guardians g ON g.person_id = p.id
WHERE u.id = $1;

-- name: ListChildrenForGuardianUser :many
-- One row per child linked to this guardian, with their currently-active
-- enrollment (latest ENROLLED history record) and lifetime attendance /
-- results aggregates for the overview cards. The LATERAL join picks a single
-- enrollment so a student with several ENROLLED history rows (one per
-- academic year, allowed by migration 000009's unique index) never duplicates
-- a child; ORDER BY keeps the most recent year's membership.
--
-- sqlc cannot see through the LEFT LATERAL's nullability, so every column
-- that can legitimately be NULL for a child without an active enrollment is
-- COALESCEd to a service-recognizable empty value (uuid.Nil / empty string);
-- the pgtype-scanned columns (enrolled_at, class_level) tolerate NULL natively.
SELECT
    s.id AS student_id,
    s.student_number,
    p.first_name,
    p.last_name,
    p.gender,
    p.date_of_birth,
    sg.relationship,
    s.status AS student_status,
    COALESCE(cur.enrollment_id, '00000000-0000-0000-0000-000000000000'::uuid) AS enrollment_id,
    COALESCE(cur.enrollment_status, '') AS enrollment_status,
    cur.enrolled_at,
    COALESCE(cur.class_id, '00000000-0000-0000-0000-000000000000'::uuid) AS class_id,
    COALESCE(cur.class_name, '') AS class_name,
    cur.class_level,
    COALESCE(cur.academic_year_id, '00000000-0000-0000-0000-000000000000'::uuid) AS academic_year_id,
    COALESCE(cur.academic_year_name, '') AS academic_year_name,
    (
        SELECT count(*) FROM attendance_records ar
        WHERE ar.student_id = s.id AND ar.status = 'PRESENT'
    )::bigint AS present_count,
    (
        SELECT count(*) FROM attendance_records ar
        WHERE ar.student_id = s.id AND ar.status = 'ABSENT'
    )::bigint AS absent_count,
    (
        SELECT count(*) FROM attendance_records ar
        WHERE ar.student_id = s.id AND ar.status = 'LATE'
    )::bigint AS late_count,
    (
        SELECT count(*) FROM attendance_records ar
        WHERE ar.student_id = s.id AND ar.status = 'EXCUSED'
    )::bigint AS excused_count,
    (
        SELECT count(*) FROM results r WHERE r.student_id = s.id
    )::bigint AS results_count,
    -- avg() over zero rows is NULL; COALESCEd to 0 and only surfaced by the
    -- service when results_count > 0, so "no results" and "0 average" stay
    -- distinguishable. Cast to float8 so sqlc emits a concrete float64 instead
    -- of interface{} for the NUMERIC-typed avg().
    COALESCE(
        (SELECT avg(r.score) FROM results r WHERE r.student_id = s.id),
        0
    )::float8 AS average_score
FROM student_guardians sg
JOIN students s ON s.id = sg.student_id
JOIN people p ON p.id = s.person_id
LEFT JOIN LATERAL (
    SELECT
        e.id AS enrollment_id,
        e.status AS enrollment_status,
        e.enrollment_date,
        e.created_at AS enrolled_at,
        c.id AS class_id,
        c.name AS class_name,
        c.level AS class_level,
        ay.id AS academic_year_id,
        ay.name AS academic_year_name
    FROM enrollments e
    JOIN classes c ON c.id = e.class_id
    JOIN academic_years ay ON ay.id = e.academic_year_id
    WHERE e.student_id = s.id AND e.status = 'ENROLLED'
    ORDER BY e.created_at DESC
    LIMIT 1
) cur ON true
WHERE sg.guardian_id = $1
ORDER BY p.first_name, p.last_name;

-- name: IsGuardianLinkedToStudent :one
-- The per-child access gate: a parent may only read attendance/results for
-- students they are linked to via student_guardians. Returns a boolean, so
-- the service distinguishes "not linked" (403) from a lookup failure (500).
SELECT EXISTS (
    SELECT 1
    FROM student_guardians
    WHERE guardian_id = $1 AND student_id = $2
);

-- name: GetChildRecentAttendanceRecords :many
-- Recent attendance marks for one child with full session context (date,
-- class, subject, term) — the per-child register view in the portal. Keeps
-- history readable even after the child's enrollment ended (marks stay).
SELECT
    ar.id AS record_id,
    ar.status,
    ar.notes,
    ar.updated_at,
    sess.session_date,
    c.id AS class_id,
    c.name AS class_name,
    sub.id AS subject_id,
    sub.name AS subject_name,
    sub.code AS subject_code,
    t.name AS term_name
FROM attendance_records ar
JOIN attendance_sessions sess ON sess.id = ar.session_id
JOIN classes c ON c.id = sess.class_id
JOIN subjects sub ON sub.id = sess.subject_id
JOIN terms t ON t.id = sess.term_id
WHERE ar.student_id = $1
ORDER BY sess.session_date DESC, sess.created_at DESC
LIMIT $2;

-- name: GetStudentIdentityForGuardian :one
-- Fallback identity lookup for GetChildDetail: a child who is linked to the
-- guardian but currently has no ENROLLED history row (and thus no row in
-- ListChildrenForGuardianUser's LATERAL join) still gets a readable shell
-- instead of a 404. Scoped by the same guardian link as the access gate.
SELECT
    s.student_number,
    p.first_name,
    p.last_name,
    sg.relationship,
    s.status AS student_status
FROM student_guardians sg
JOIN students s ON s.id = sg.student_id
JOIN people p ON p.id = s.person_id
WHERE sg.guardian_id = $1 AND sg.student_id = $2;
