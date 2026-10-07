-- name: GetAdminDashboardCounts :one
SELECT
    (SELECT count(*) FROM students WHERE status = 'ACTIVE')::bigint AS active_students,
    (SELECT count(*) FROM staff WHERE employment_status = 'ACTIVE')::bigint AS active_staff,
    (SELECT count(DISTINCT s.id) FROM staff s LEFT JOIN staff_positions sp ON sp.id = s.position_id WHERE (sp.category = 'TEACHING' OR s.id IN (SELECT teacher_id FROM teacher_assignments)) AND s.employment_status = 'ACTIVE')::bigint AS active_teachers,
    (SELECT count(*) FROM classes)::bigint AS total_classes,
    (SELECT count(*) FROM enrollments WHERE status = 'ENROLLED')::bigint AS active_enrollments;

-- name: GetCurrentAcademicYearAndTerm :one
SELECT 
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    ay.start_date AS academic_year_start,
    ay.end_date AS academic_year_end,
    t.id AS term_id,
    t.name AS term_name,
    t.start_date AS term_start,
    t.end_date AS term_end
FROM academic_years ay
LEFT JOIN terms t ON t.academic_year_id = ay.id AND t.is_current = true
WHERE ay.is_current = true
LIMIT 1;

-- name: GetLatestAttendanceSummary :one
WITH latest_session AS (
    SELECT session_date
    FROM attendance_sessions
    ORDER BY session_date DESC
    LIMIT 1
)
SELECT
    COALESCE((SELECT session_date FROM latest_session), CURRENT_DATE)::date AS session_date,
    COALESCE((SELECT count(*) FROM attendance_sessions WHERE session_date = (SELECT session_date FROM latest_session)), 0)::bigint AS total_sessions,
    COALESCE(SUM(CASE WHEN ar.status = 'PRESENT' THEN 1 ELSE 0 END), 0)::bigint AS present_count,
    COALESCE(SUM(CASE WHEN ar.status = 'ABSENT' THEN 1 ELSE 0 END), 0)::bigint AS absent_count,
    COALESCE(SUM(CASE WHEN ar.status = 'LATE' THEN 1 ELSE 0 END), 0)::bigint AS late_count,
    COALESCE(SUM(CASE WHEN ar.status = 'EXCUSED' THEN 1 ELSE 0 END), 0)::bigint AS excused_count,
    COALESCE(count(ar.id), 0)::bigint AS total_marks
FROM attendance_records ar
JOIN attendance_sessions s ON s.id = ar.session_id
WHERE s.session_date = (SELECT session_date FROM latest_session);

-- name: GetRecentAnnouncementsForDashboard :many
SELECT
    a.id,
    a.title,
    a.status,
    a.target_type,
    a.target_role,
    a.target_class_id,
    c.name AS target_class_name,
    a.priority,
    a.published_at,
    a.created_at,
    p.first_name AS author_first_name,
    p.last_name AS author_last_name
FROM announcements a
JOIN users u ON u.id = a.author_id
JOIN people p ON p.id = u.person_id
LEFT JOIN classes c ON c.id = a.target_class_id
ORDER BY a.created_at DESC
LIMIT $1;

-- name: GetRecentAuditLogsForDashboard :many
SELECT
    al.id,
    al.actor_user_id,
    COALESCE(u.email, 'SYSTEM')::text AS actor_email,
    COALESCE(p.first_name, '')::text AS actor_first_name,
    COALESCE(p.last_name, '')::text AS actor_last_name,
    al.action,
    al.resource_type,
    al.resource_id,
    al.metadata,
    al.created_at
FROM audit_logs al
LEFT JOIN users u ON u.id = al.actor_user_id
LEFT JOIN people p ON p.id = u.person_id
ORDER BY al.created_at DESC
LIMIT $1;
