-- Student portal (milestone M17). Read-only, self-scoped aggregation over the
-- students domain (M7), enrollment history (M9), attendance (M11), results
-- (M12), and announcements (M13). No schema of its own — mirrors M14 (admin
-- dashboard), M15 (teacher portal), and M16 (parent portal).
--
-- Row-level scoping note (flagged in migration 000004's comments): a STUDENT
-- account may only read its own records, so every query here is anchored on
-- the student row resolved from the authenticated user's person_id — there is
-- no endpoint that takes a student id at all.

-- name: GetStudentProfileByUserID :one
-- Resolve the student record behind a STUDENT-role account, mirroring
-- GetGuardianProfileByUserID in parent_portal.sql. A student user is a person
-- with a students row; the INNER JOINs mean a non-student account yields
-- pgx.ErrNoRows, which the service maps to a friendly 404.
SELECT
    s.id AS student_id,
    s.student_number,
    s.admission_date,
    s.status AS student_status,
    p.first_name,
    p.last_name,
    p.other_names,
    p.gender,
    p.date_of_birth,
    p.email,
    p.phone,
    p.address
FROM users u
JOIN people p ON p.id = u.person_id
JOIN students s ON s.person_id = p.id
WHERE u.id = $1;

-- name: GetCurrentEnrollmentForStudent :one
-- The student's currently-active enrollment (latest ENROLLED history record)
-- with class, programme, and academic year context for the overview header.
-- Same LATERAL pattern as ListChildrenForGuardianUser in parent_portal.sql so
-- multiple ENROLLED history rows (one per year, allowed by migration 000009)
-- never duplicate; a student without an active enrollment yields pgx.ErrNoRows
-- and the service omits the enrollment block.
SELECT
    e.id AS enrollment_id,
    e.status AS enrollment_status,
    e.enrollment_date,
    e.created_at AS enrolled_at,
    c.id AS class_id,
    c.name AS class_name,
    c.level AS class_level,
    c.capacity AS class_capacity,
    pr.id AS programme_id,
    pr.code AS programme_code,
    pr.name AS programme_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name
FROM enrollments e
JOIN classes c ON c.id = e.class_id
LEFT JOIN programmes pr ON pr.id = c.programme_id
JOIN academic_years ay ON ay.id = e.academic_year_id
WHERE e.student_id = $1 AND e.status = 'ENROLLED'
ORDER BY e.created_at DESC
LIMIT 1;

-- name: ListStudentEnrollments :many
-- The student's full enrollment history across academic years and classes,
-- ordered newest to oldest.
SELECT
    e.id AS enrollment_id,
    e.status AS enrollment_status,
    e.enrollment_date,
    e.exit_date,
    e.created_at AS enrolled_at,
    c.id AS class_id,
    c.name AS class_name,
    c.level AS class_level,
    pr.id AS programme_id,
    pr.code AS programme_code,
    pr.name AS programme_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name
FROM enrollments e
JOIN classes c ON c.id = e.class_id
LEFT JOIN programmes pr ON pr.id = c.programme_id
JOIN academic_years ay ON ay.id = e.academic_year_id
WHERE e.student_id = $1
ORDER BY e.enrollment_date DESC, e.created_at DESC;

-- name: GetStudentAttendanceCounts :one
-- Lifetime per-status mark breakdown for the overview attendance card.
SELECT
    (SELECT count(*) FROM attendance_records ar WHERE ar.student_id = $1 AND ar.status = 'PRESENT')::bigint AS present_count,
    (SELECT count(*) FROM attendance_records ar WHERE ar.student_id = $1 AND ar.status = 'ABSENT')::bigint AS absent_count,
    (SELECT count(*) FROM attendance_records ar WHERE ar.student_id = $1 AND ar.status = 'LATE')::bigint AS late_count,
    (SELECT count(*) FROM attendance_records ar WHERE ar.student_id = $1 AND ar.status = 'EXCUSED')::bigint AS excused_count;

-- name: GetStudentResultsSummary :one
-- Results headline for the overview card. avg() over zero rows is NULL and is
-- COALESCEd to 0; the service only surfaces the average when results_count
-- > 0, keeping "no results" and "0 average" distinguishable (same approach as
-- ListChildrenForGuardianUser in parent_portal.sql).
SELECT
    (SELECT count(*) FROM results r WHERE r.student_id = $1)::bigint AS results_count,
    COALESCE((SELECT avg(r.score) FROM results r WHERE r.student_id = $1), 0)::float8 AS average_score;

-- name: GetStudentRecentAttendanceRecords :many
-- The student's own recent attendance marks with full session context (date,
-- class, subject, term) — mirrors GetChildRecentAttendanceRecords in
-- parent_portal.sql, named for this domain since the row shape is identical
-- but the anchor (self, not a linked child) is not.
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
