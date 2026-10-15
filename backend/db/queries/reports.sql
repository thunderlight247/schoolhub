-- Reports domain (milestone M18).
-- Institutional and academic reporting queries: enrollment summaries,
-- class attendance analytics, class performance distribution, and
-- terminal student report cards.
-- Access is gated under the report.view permission.

-- name: GetEnrollmentReport :many
-- Institutional enrollment report per class with capacity utilization and
-- gender breakdown across the school for an academic year.
SELECT
    c.id AS class_id,
    c.name AS class_name,
    c.level AS class_level,
    c.capacity,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    COALESCE(pr.name, '') AS programme_name,
    COUNT(e.id) FILTER (WHERE e.status = 'ENROLLED')::bigint AS enrolled_count,
    COUNT(e.id) FILTER (WHERE e.status = 'COMPLETED')::bigint AS completed_count,
    COUNT(e.id) FILTER (WHERE e.status = 'TRANSFERRED')::bigint AS transferred_count,
    COUNT(e.id) FILTER (WHERE e.status = 'WITHDRAWN')::bigint AS withdrawn_count,
    COUNT(e.id) FILTER (WHERE e.status = 'ENROLLED' AND p.gender = 'MALE')::bigint AS male_count,
    COUNT(e.id) FILTER (WHERE e.status = 'ENROLLED' AND p.gender = 'FEMALE')::bigint AS female_count,
    COUNT(e.id) FILTER (WHERE e.status = 'ENROLLED' AND (p.gender IS NULL OR p.gender NOT IN ('MALE', 'FEMALE')))::bigint AS other_gender_count
FROM classes c
JOIN academic_years ay ON ay.id = c.academic_year_id
LEFT JOIN programmes pr ON pr.id = c.programme_id
LEFT JOIN enrollments e ON e.class_id = c.id
LEFT JOIN students s ON s.id = e.student_id
LEFT JOIN people p ON p.id = s.person_id
WHERE (sqlc.narg('academic_year_id')::uuid IS NULL OR c.academic_year_id = sqlc.narg('academic_year_id'))
GROUP BY c.id, c.name, c.level, c.capacity, ay.id, ay.name, pr.name
ORDER BY c.name ASC;

-- name: GetClassAttendanceSummaryReport :many
-- Per-student attendance breakdown for a class in a given term or date window.
SELECT
    s.id AS student_id,
    s.student_number,
    p.first_name,
    p.last_name,
    COUNT(ar.id) FILTER (WHERE ar.status = 'PRESENT')::bigint AS present_count,
    COUNT(ar.id) FILTER (WHERE ar.status = 'ABSENT')::bigint AS absent_count,
    COUNT(ar.id) FILTER (WHERE ar.status = 'LATE')::bigint AS late_count,
    COUNT(ar.id) FILTER (WHERE ar.status = 'EXCUSED')::bigint AS excused_count,
    COUNT(ar.id)::bigint AS total_records
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN people p ON p.id = s.person_id
LEFT JOIN attendance_records ar ON ar.student_id = s.id
LEFT JOIN attendance_sessions sess ON sess.id = ar.session_id 
    AND sess.class_id = e.class_id
    AND (sqlc.narg('term_id')::uuid IS NULL OR sess.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('start_date')::date IS NULL OR sess.session_date >= sqlc.narg('start_date'))
    AND (sqlc.narg('end_date')::date IS NULL OR sess.session_date <= sqlc.narg('end_date'))
WHERE e.class_id = sqlc.arg('class_id')
  AND e.status = 'ENROLLED'
GROUP BY s.id, s.student_number, p.first_name, p.last_name
ORDER BY p.last_name ASC, p.first_name ASC;

-- name: GetClassAttendanceSessionCount :one
-- Total sessions held for a class within the given term or date window.
SELECT COUNT(*)::bigint AS total_sessions
FROM attendance_sessions sess
WHERE sess.class_id = sqlc.arg('class_id')
  AND (sqlc.narg('term_id')::uuid IS NULL OR sess.term_id = sqlc.narg('term_id'))
  AND (sqlc.narg('start_date')::date IS NULL OR sess.session_date >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date')::date IS NULL OR sess.session_date <= sqlc.narg('end_date'));

-- name: GetClassAssessmentPerformanceReport :many
-- Assessment performance metrics for a class and term (average, high, low, submissions).
SELECT
    a.id AS assessment_id,
    a.title AS assessment_title,
    a.type AS assessment_type,
    a.max_score,
    a.weight,
    a.assessment_date,
    sub.id AS subject_id,
    sub.code AS subject_code,
    sub.name AS subject_name,
    COUNT(r.id)::bigint AS results_count,
    COALESCE(AVG(r.score), 0)::float8 AS average_score,
    COALESCE(MAX(r.score), 0)::float8 AS highest_score,
    COALESCE(MIN(r.score), 0)::float8 AS lowest_score
FROM assessments a
JOIN subjects sub ON sub.id = a.subject_id
LEFT JOIN results r ON r.assessment_id = a.id
WHERE a.class_id = sqlc.arg('class_id')
  AND (sqlc.narg('term_id')::uuid IS NULL OR a.term_id = sqlc.narg('term_id'))
  AND (sqlc.narg('subject_id')::uuid IS NULL OR a.subject_id = sqlc.narg('subject_id'))
GROUP BY a.id, a.title, a.type, a.max_score, a.weight, a.assessment_date, sub.id, sub.code, sub.name
ORDER BY a.assessment_date DESC, a.created_at DESC;

-- name: GetStudentReportCardProfile :one
-- Header metadata for a student terminal report card.
SELECT
    s.id AS student_id,
    s.student_number,
    p.first_name,
    p.last_name,
    p.gender,
    p.date_of_birth,
    c.id AS class_id,
    c.name AS class_name,
    c.level AS class_level,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    t.id AS term_id,
    t.name AS term_name
FROM students s
JOIN people p ON p.id = s.person_id
JOIN enrollments e ON e.student_id = s.id AND e.status = 'ENROLLED'
JOIN classes c ON c.id = e.class_id
JOIN academic_years ay ON ay.id = e.academic_year_id
JOIN terms t ON t.id = sqlc.arg('term_id') AND t.academic_year_id = ay.id
WHERE s.id = sqlc.arg('student_id');

-- name: GetStudentSubjectResultsForReportCard :many
-- Subject-level aggregates for a student in a specific class and term.
SELECT
    sub.id AS subject_id,
    sub.code AS subject_code,
    sub.name AS subject_name,
    sub.type AS subject_type,
    COUNT(r.id)::bigint AS assessments_taken,
    COALESCE(SUM(r.score), 0)::float8 AS total_score,
    COALESCE(SUM(a.max_score), 0)::float8 AS total_max_score,
    COALESCE(AVG(CASE WHEN a.max_score > 0 THEN (r.score / a.max_score) * 100 ELSE 0 END), 0)::float8 AS average_percentage
FROM class_subjects cs
JOIN subjects sub ON sub.id = cs.subject_id
LEFT JOIN assessments a ON a.class_id = cs.class_id AND a.subject_id = sub.id AND a.term_id = sqlc.arg('term_id')
LEFT JOIN results r ON r.assessment_id = a.id AND r.student_id = sqlc.arg('student_id')
WHERE cs.class_id = sqlc.arg('class_id')
GROUP BY sub.id, sub.code, sub.name, sub.type
ORDER BY sub.type DESC, sub.name ASC;

-- name: GetStudentAttendanceForTerm :one
-- Attendance summary for a student within a specific term.
SELECT
    COUNT(ar.id) FILTER (WHERE ar.status = 'PRESENT')::bigint AS present_count,
    COUNT(ar.id) FILTER (WHERE ar.status = 'ABSENT')::bigint AS absent_count,
    COUNT(ar.id) FILTER (WHERE ar.status = 'LATE')::bigint AS late_count,
    COUNT(ar.id) FILTER (WHERE ar.status = 'EXCUSED')::bigint AS excused_count,
    COUNT(ar.id)::bigint AS total_sessions
FROM attendance_records ar
JOIN attendance_sessions sess ON sess.id = ar.session_id
WHERE ar.student_id = sqlc.arg('student_id')
  AND sess.term_id = sqlc.arg('term_id');
