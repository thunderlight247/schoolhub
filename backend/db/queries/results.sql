-- Results (milestone M12). One row per student per assessment: score,
-- optional letter grade, optional remarks. Portal reads live here too.

-- name: CreateResult :one
INSERT INTO results (assessment_id, student_id, score, grade, remarks)
VALUES (sqlc.arg(assessment_id), sqlc.arg(student_id), sqlc.narg(score), sqlc.narg(grade), sqlc.narg(remarks))
RETURNING *;

-- name: ListAssessmentResults :many
-- The score sheet: every student's result for one assessment, roster order
-- matching the attendance register (last name, then first name).
SELECT
    r.id,
    r.assessment_id,
    r.student_id,
    r.score,
    r.grade,
    r.remarks,
    r.created_at,
    r.updated_at,
    st.student_number,
    p.first_name || ' ' || p.last_name AS student_name
FROM results r
JOIN students st ON st.id = r.student_id
JOIN people p ON p.id = st.person_id
WHERE r.assessment_id = sqlc.arg(assessment_id)
ORDER BY p.last_name, p.first_name;

-- name: GetResultByID :one
-- A single result with its full context (student, assessment, class,
-- subject, term) — the shape the PATCH response and portals render.
SELECT
    r.id,
    r.assessment_id,
    r.student_id,
    r.score,
    r.grade,
    r.remarks,
    r.created_at,
    r.updated_at,
    st.student_number,
    p.first_name || ' ' || p.last_name AS student_name,
    a.title AS assessment_title,
    a.type AS assessment_type,
    a.max_score,
    a.assessment_date,
    a.class_id,
    c.name AS class_name,
    sub.code AS subject_code,
    sub.name AS subject_name,
    t.id AS term_id,
    t.name AS term_name
FROM results r
JOIN students st ON st.id = r.student_id
JOIN people p ON p.id = st.person_id
JOIN assessments a ON a.id = r.assessment_id
JOIN classes c ON c.id = a.class_id
JOIN subjects sub ON sub.id = a.subject_id
JOIN terms t ON t.id = a.term_id
WHERE r.id = sqlc.arg(id);

-- name: UpdateResult :one
-- Corrections flow: score, grade, and remarks are each optional (NULL narg
-- = keep the current value). The service validates score bounds against
-- the assessment's max_score before calling this.
UPDATE results
SET
    score   = COALESCE(sqlc.narg(score), score),
    grade   = COALESCE(sqlc.narg(grade), grade),
    remarks = COALESCE(sqlc.narg(remarks), remarks)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListStudentResults :many
-- One student's results across assessments — the read groundwork the
-- parent (M16) and student (M17) portals render. Optional term/year
-- filters scope the window; without them this is the full history.
SELECT
    r.id,
    r.assessment_id,
    r.student_id,
    r.score,
    r.grade,
    r.remarks,
    r.created_at,
    r.updated_at,
    a.title AS assessment_title,
    a.type AS assessment_type,
    a.max_score,
    a.assessment_date,
    sub.code AS subject_code,
    sub.name AS subject_name,
    c.name AS class_name,
    t.id AS term_id,
    t.name AS term_name
FROM results r
JOIN assessments a ON a.id = r.assessment_id
JOIN subjects sub ON sub.id = a.subject_id
JOIN classes c ON c.id = a.class_id
JOIN terms t ON t.id = a.term_id
WHERE r.student_id = sqlc.arg(student_id)
    AND (sqlc.narg('term_id')::uuid IS NULL OR a.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('academic_year_id')::uuid IS NULL OR t.academic_year_id = sqlc.narg('academic_year_id'))
ORDER BY a.assessment_date DESC, r.created_at DESC;
