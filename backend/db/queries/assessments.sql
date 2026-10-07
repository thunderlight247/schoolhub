-- Assessments (milestone M12). Header table for graded events; results
-- hang off it. See migrations/000012_create_assessments.up.sql.

-- name: CreateAssessment :one
INSERT INTO assessments (class_id, subject_id, term_id, title, type, max_score, weight, assessment_date)
VALUES (
    sqlc.arg(class_id),
    sqlc.arg(subject_id),
    sqlc.arg(term_id),
    sqlc.arg(title),
    sqlc.arg(type),
    sqlc.arg(max_score),
    sqlc.narg(weight),
    sqlc.arg(assessment_date)
)
RETURNING *;

-- name: GetAssessmentByID :one
-- The assessment header with everything the UI shows in one round trip:
-- class + its academic year, subject, term.
SELECT
    a.id,
    a.class_id,
    a.subject_id,
    a.term_id,
    a.title,
    a.type,
    a.max_score,
    a.weight,
    a.assessment_date,
    a.created_at,
    a.updated_at,
    c.name AS class_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    sub.code AS subject_code,
    sub.name AS subject_name,
    t.name AS term_name
FROM assessments a
JOIN classes c ON c.id = a.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = a.subject_id
JOIN terms t ON t.id = a.term_id
WHERE a.id = sqlc.arg(id);

-- name: ListAssessments :many
-- Paginated, filtered assessment list. Every filter is optional: a NULL
-- narg disables that condition. Keep this WHERE clause and
-- CountAssessments' in sync — see the same note on ListTeacherAssignments.
SELECT
    a.id,
    a.class_id,
    a.subject_id,
    a.term_id,
    a.title,
    a.type,
    a.max_score,
    a.weight,
    a.assessment_date,
    a.created_at,
    a.updated_at,
    c.name AS class_name,
    ay.id AS academic_year_id,
    ay.name AS academic_year_name,
    sub.code AS subject_code,
    sub.name AS subject_name,
    t.name AS term_name
FROM assessments a
JOIN classes c ON c.id = a.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = a.subject_id
JOIN terms t ON t.id = a.term_id
WHERE
    (sqlc.narg('class_id')::uuid IS NULL OR a.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('subject_id')::uuid IS NULL OR a.subject_id = sqlc.narg('subject_id'))
    AND (sqlc.narg('term_id')::uuid IS NULL OR a.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('type')::text IS NULL OR a.type = sqlc.narg('type'))
ORDER BY a.assessment_date DESC, c.name, sub.name
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountAssessments :one
-- Must stay in sync with ListAssessments' WHERE clause (see the comment
-- there).
SELECT count(*)
FROM assessments a
JOIN classes c ON c.id = a.class_id
JOIN academic_years ay ON ay.id = c.academic_year_id
JOIN subjects sub ON sub.id = a.subject_id
JOIN terms t ON t.id = a.term_id
WHERE
    (sqlc.narg('class_id')::uuid IS NULL OR a.class_id = sqlc.narg('class_id'))
    AND (sqlc.narg('subject_id')::uuid IS NULL OR a.subject_id = sqlc.narg('subject_id'))
    AND (sqlc.narg('term_id')::uuid IS NULL OR a.term_id = sqlc.narg('term_id'))
    AND (sqlc.narg('type')::text IS NULL OR a.type = sqlc.narg('type'));

-- name: UpdateAssessment :one
-- Partial edit: every field is optional (NULL narg = keep the current
-- value). The service validates type/max_score/weight/assessment_date
-- before calling this.
UPDATE assessments
SET
    title           = COALESCE(sqlc.narg(title), title),
    type            = COALESCE(sqlc.narg(type), type),
    max_score       = COALESCE(sqlc.narg(max_score), max_score),
    weight          = COALESCE(sqlc.narg(weight), weight),
    assessment_date = COALESCE(sqlc.narg(assessment_date), assessment_date)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteAssessment :exec
DELETE FROM assessments
WHERE id = sqlc.arg(id);
