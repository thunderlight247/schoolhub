-- name: GetTeacherProfileByUserID :one
SELECT 
    s.id AS staff_id,
    s.staff_number,
    s.employment_status,
    p.id AS person_id,
    p.first_name,
    p.last_name,
    p.email,
    p.phone,
    d.id AS department_id,
    d.name AS department_name,
    sp.id AS position_id,
    sp.title AS position_title
FROM users u
JOIN people p ON p.id = u.person_id
JOIN staff s ON s.person_id = p.id
LEFT JOIN departments d ON d.id = s.department_id
LEFT JOIN staff_positions sp ON sp.id = s.position_id
WHERE u.id = $1;

-- name: GetTeacherDashboardCounts :one
SELECT
    (SELECT count(DISTINCT ta.class_id) FROM teacher_assignments ta WHERE ta.teacher_id = $1)::bigint AS total_classes,
    (SELECT count(DISTINCT ta.subject_id) FROM teacher_assignments ta WHERE ta.teacher_id = $1)::bigint AS total_subjects,
    (
        SELECT count(DISTINCT e.student_id)
        FROM enrollments e
        JOIN teacher_assignments ta ON ta.class_id = e.class_id
        WHERE ta.teacher_id = $1 AND e.status = 'ENROLLED'
    )::bigint AS total_students,
    (
        SELECT count(*)
        FROM attendance_sessions ats
        WHERE EXISTS (
            SELECT 1 FROM teacher_assignments ta
            WHERE ta.teacher_id = $1 AND ta.class_id = ats.class_id AND ta.subject_id = ats.subject_id
        )
    )::bigint AS total_attendance_sessions,
    (
        SELECT count(*)
        FROM assessments a
        WHERE EXISTS (
            SELECT 1 FROM teacher_assignments ta
            WHERE ta.teacher_id = $1 AND ta.class_id = a.class_id AND ta.subject_id = a.subject_id
        )
    )::bigint AS total_assessments;

-- name: GetTeacherClassesWithStats :many
SELECT
    ta.id AS assignment_id,
    c.id AS class_id,
    c.name AS class_name,
    c.level AS class_level,
    c.capacity AS class_capacity,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    ay.is_current AS academic_year_is_current,
    sub.id AS subject_id,
    sub.name AS subject_name,
    sub.code AS subject_code,
    sub.type AS subject_type,
    (SELECT count(*) FROM enrollments e WHERE e.class_id = c.id AND e.status = 'ENROLLED')::bigint AS enrolled_students_count
FROM teacher_assignments ta
JOIN classes c ON c.id = ta.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = ta.subject_id
WHERE ta.teacher_id = $1
ORDER BY c.name, sub.name;

-- name: IsTeacherAssignedToClass :one
SELECT EXISTS (
    SELECT 1 FROM teacher_assignments
    WHERE teacher_id = $1 AND class_id = $2
);

-- name: GetClassRosterForTeacher :many
SELECT
    s.id AS student_id,
    s.student_number,
    p.first_name,
    p.last_name,
    p.gender,
    p.date_of_birth,
    e.status AS enrollment_status,
    e.created_at AS enrolled_at,
    gp.first_name AS guardian_first_name,
    gp.last_name AS guardian_last_name,
    gp.phone AS guardian_phone,
    gp.email AS guardian_email,
    sg.relationship AS guardian_relationship
FROM enrollments e
JOIN students s ON s.id = e.student_id
JOIN people p ON p.id = s.person_id
LEFT JOIN student_guardians sg ON sg.student_id = s.id
LEFT JOIN guardians g ON g.id = sg.guardian_id
LEFT JOIN people gp ON gp.id = g.person_id
WHERE e.class_id = $1 AND e.status = 'ENROLLED'
ORDER BY p.last_name, p.first_name;

-- name: GetTeacherAssessmentsSummary :many
SELECT
    a.id AS assessment_id,
    a.title,
    a.type AS assessment_type,
    a.max_score,
    a.weight,
    a.assessment_date,
    c.id AS class_id,
    c.name AS class_name,
    sub.id AS subject_id,
    sub.name AS subject_name,
    sub.code AS subject_code,
    t.name AS term_name,
    (SELECT count(*) FROM results r WHERE r.assessment_id = a.id)::bigint AS results_count,
    (SELECT count(*) FROM enrollments e WHERE e.class_id = c.id AND e.status = 'ENROLLED')::bigint AS total_students_count
FROM assessments a
JOIN classes c ON c.id = a.class_id
JOIN subjects sub ON sub.id = a.subject_id
JOIN terms t ON t.id = a.term_id
WHERE EXISTS (
    SELECT 1 FROM teacher_assignments ta
    WHERE ta.teacher_id = $1 AND ta.class_id = a.class_id AND ta.subject_id = a.subject_id
)
ORDER BY a.assessment_date DESC, a.created_at DESC
LIMIT $2;

-- name: GetTeacherRecentAttendanceSessions :many
SELECT
    ats.id AS session_id,
    ats.session_date,
    c.id AS class_id,
    c.name AS class_name,
    sub.id AS subject_id,
    sub.name AS subject_name,
    sub.code AS subject_code,
    t.name AS term_name,
    (SELECT count(*) FROM attendance_records ar WHERE ar.session_id = ats.id AND ar.status = 'PRESENT')::bigint AS present_count,
    (SELECT count(*) FROM attendance_records ar WHERE ar.session_id = ats.id AND ar.status = 'ABSENT')::bigint AS absent_count,
    (SELECT count(*) FROM attendance_records ar WHERE ar.session_id = ats.id AND ar.status = 'LATE')::bigint AS late_count,
    (SELECT count(*) FROM attendance_records ar WHERE ar.session_id = ats.id AND ar.status = 'EXCUSED')::bigint AS excused_count,
    (SELECT count(*) FROM attendance_records ar WHERE ar.session_id = ats.id)::bigint AS total_marks
FROM attendance_sessions ats
JOIN classes c ON c.id = ats.class_id
JOIN subjects sub ON sub.id = ats.subject_id
JOIN terms t ON t.id = ats.term_id
WHERE EXISTS (
    SELECT 1 FROM teacher_assignments ta
    WHERE ta.teacher_id = $1 AND ta.class_id = ats.class_id AND ta.subject_id = ats.subject_id
)
ORDER BY ats.session_date DESC, ats.created_at DESC
LIMIT $2;
