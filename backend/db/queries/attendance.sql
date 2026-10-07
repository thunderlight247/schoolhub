-- Attendance (milestone M11). Session-level capture, record-level marks.
-- See migrations/000011_create_attendance.up.sql for the two-level model.

-- name: CreateAttendanceSession :one
INSERT INTO attendance_sessions (class_id, subject_id, term_id, session_date)
VALUES (sqlc.arg(class_id), sqlc.arg(subject_id), sqlc.arg(term_id), sqlc.arg(session_date))
RETURNING *;

-- name: GetAttendanceSessionByID :one
-- The register header with everything the UI shows in one round trip:
-- class + its academic year, subject, and the term it was captured in.
SELECT
    s.id,
    s.class_id,
    s.subject_id,
    s.term_id,
    s.session_date,
    s.created_at,
    s.updated_at,
    c.name AS class_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    sub.code AS subject_code,
    sub.name AS subject_name,
    t.name AS term_name
FROM attendance_sessions s
JOIN classes c ON c.id = s.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = s.subject_id
JOIN terms t ON t.id = s.term_id
WHERE s.id = sqlc.arg(id);

-- name: ListAttendanceSessions :many
-- Paginated, filtered register list. Every filter is optional: a NULL narg
-- disables that condition. date_from/date_to bound the session_date. Keep
-- this WHERE clause and CountAttendanceSessions' in sync — see the same
-- note on ListTeacherAssignments.
SELECT
    s.id,
    s.class_id,
    s.subject_id,
    s.term_id,
    s.session_date,
    s.created_at,
    s.updated_at,
    c.name AS class_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    sub.code AS subject_code,
    sub.name AS subject_name,
    t.name AS term_name
FROM attendance_sessions s
JOIN classes c ON c.id = s.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = s.subject_id
JOIN terms t ON t.id = s.term_id
WHERE
    (sqlc.narg('class_id')::uuid IS NULL OR s.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('subject_id')::uuid IS NULL OR s.subject_id = sqlc.narg('subject_id'))
    AND (sqlc.narg('term_id')::uuid IS NULL OR s.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('date_from')::date IS NULL OR s.session_date >= sqlc.narg('date_from'))
    AND (sqlc.narg('date_to')::date IS NULL OR s.session_date <= sqlc.narg('date_to'))
ORDER BY s.session_date DESC, c.name, sub.name
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountAttendanceSessions :one
-- Must stay in sync with ListAttendanceSessions' WHERE clause (see the
-- comment there).
SELECT count(*)
FROM attendance_sessions s
JOIN classes c ON c.id = s.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = s.subject_id
JOIN terms t ON t.id = s.term_id
WHERE
    (sqlc.narg('class_id')::uuid IS NULL OR s.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('subject_id')::uuid IS NULL OR s.subject_id = sqlc.narg('subject_id'))
    AND (sqlc.narg('term_id')::uuid IS NULL OR s.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('date_from')::date IS NULL OR s.session_date >= sqlc.narg('date_from'))
    AND (sqlc.narg('date_to')::date IS NULL OR s.session_date <= sqlc.narg('date_to'));

-- name: CreateAttendanceRecord :one
INSERT INTO attendance_records (session_id, student_id, status, notes)
VALUES (sqlc.arg(session_id), sqlc.arg(student_id), sqlc.arg(status), sqlc.narg(notes))
RETURNING *;

-- name: ListAttendanceSessionRecords :many
-- The full register: every student's mark for one session, roster order
-- matching the other person listings (last name, then first name).
SELECT
    ar.id,
    ar.session_id,
    ar.student_id,
    ar.status,
    ar.notes,
    ar.created_at,
    ar.updated_at,
    st.student_number,
    p.first_name || ' ' || p.last_name AS student_name
FROM attendance_records ar
JOIN students st ON st.id = ar.student_id
JOIN people p ON p.id = st.person_id
WHERE ar.session_id = sqlc.arg(session_id)
ORDER BY p.last_name, p.first_name;

-- name: GetAttendanceRecordByID :one
-- A single mark with its full context (student, session date, class,
-- subject, term) — the shape the PATCH response and portals render.
SELECT
    ar.id,
    ar.session_id,
    ar.student_id,
    ar.status,
    ar.notes,
    ar.created_at,
    ar.updated_at,
    st.student_number,
    p.first_name || ' ' || p.last_name AS student_name,
    sess.session_date,
    sess.class_id,
    c.name AS class_name,
    sub.code AS subject_code,
    sub.name AS subject_name,
    t.id AS term_id,
    t.name AS term_name
FROM attendance_records ar
JOIN students st ON st.id = ar.student_id
JOIN people p ON p.id = st.person_id
JOIN attendance_sessions sess ON sess.id = ar.session_id
JOIN classes c ON c.id = sess.class_id
JOIN subjects sub ON sub.id = sess.subject_id
JOIN terms t ON t.id = sess.term_id
WHERE ar.id = sqlc.arg(id);

-- name: UpdateAttendanceRecord :one
-- Corrections flow: the mark itself must always be given (a mark is never
-- "unset", it is changed); notes are COALESCEd so an omitted field leaves
-- the existing note untouched.
UPDATE attendance_records
SET
    status = sqlc.arg(status),
    notes = COALESCE(sqlc.narg(notes), notes)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SummarizeStudentAttendance :many
-- Per-status mark counts for one student — the groundwork the parent (M16)
-- and student (M17) portals render. Optional term/year filters scope the
-- window; without them this is the all-time record.
SELECT ar.status, count(*) AS total
FROM attendance_records ar
JOIN attendance_sessions sess ON sess.id = ar.session_id
JOIN terms t ON t.id = sess.term_id
WHERE ar.student_id = sqlc.arg(student_id)
    AND (sqlc.narg('term_id')::uuid IS NULL OR sess.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('academic_year_id')::uuid IS NULL OR t.academic_year_id = sqlc.narg('academic_year_id'))
GROUP BY ar.status;

-- name: SummarizeClassAttendance :many
-- Per-status mark counts for one class — the teacher's register overview.
-- subject_id optionally narrows to one teaching slot; term_id scopes to a
-- term.
SELECT ar.status, count(*) AS total
FROM attendance_records ar
JOIN attendance_sessions sess ON sess.id = ar.session_id
WHERE sess.class_id = sqlc.arg(class_id)
    AND (sqlc.narg('term_id')::uuid IS NULL OR sess.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('subject_id')::uuid IS NULL OR sess.subject_id = sqlc.narg('subject_id'))
GROUP BY ar.status;
