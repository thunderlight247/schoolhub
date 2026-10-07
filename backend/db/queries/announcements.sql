-- name: CreateAnnouncement :one
INSERT INTO announcements (
    title,
    content,
    author_id,
    status,
    target_type,
    target_role,
    target_class_id,
    priority,
    published_at,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetAnnouncementByID :one
SELECT 
    a.id,
    a.title,
    a.content,
    a.author_id,
    u.email AS author_email,
    p.first_name AS author_first_name,
    p.last_name AS author_last_name,
    a.status,
    a.target_type,
    a.target_role,
    a.target_class_id,
    c.name AS target_class_name,
    a.priority,
    a.published_at,
    a.expires_at,
    a.created_at,
    a.updated_at
FROM announcements a
JOIN users u ON u.id = a.author_id
JOIN people p ON p.id = u.person_id
LEFT JOIN classes c ON c.id = a.target_class_id
WHERE a.id = $1;

-- name: ListAnnouncements :many
SELECT 
    a.id,
    a.title,
    a.content,
    a.author_id,
    u.email AS author_email,
    p.first_name AS author_first_name,
    p.last_name AS author_last_name,
    a.status,
    a.target_type,
    a.target_role,
    a.target_class_id,
    c.name AS target_class_name,
    a.priority,
    a.published_at,
    a.expires_at,
    a.created_at,
    a.updated_at
FROM announcements a
JOIN users u ON u.id = a.author_id
JOIN people p ON p.id = u.person_id
LEFT JOIN classes c ON c.id = a.target_class_id
WHERE (sqlc.narg('status')::text IS NULL OR a.status = sqlc.narg('status'))
  AND (sqlc.narg('target_type')::text IS NULL OR a.target_type = sqlc.narg('target_type'))
  AND (sqlc.narg('target_role')::text IS NULL OR a.target_role = sqlc.narg('target_role'))
  AND (sqlc.narg('target_class_id')::uuid IS NULL OR a.target_class_id = sqlc.narg('target_class_id'))
  AND (sqlc.narg('author_id')::uuid IS NULL OR a.author_id = sqlc.narg('author_id'))
  AND (
      sqlc.narg('search')::text IS NULL
      OR a.title ILIKE '%' || sqlc.narg('search') || '%'
      OR a.content ILIKE '%' || sqlc.narg('search') || '%'
  )
ORDER BY COALESCE(a.published_at, a.created_at) DESC
LIMIT $1 OFFSET $2;

-- name: CountAnnouncements :one
SELECT count(*)
FROM announcements a
WHERE (sqlc.narg('status')::text IS NULL OR a.status = sqlc.narg('status'))
  AND (sqlc.narg('target_type')::text IS NULL OR a.target_type = sqlc.narg('target_type'))
  AND (sqlc.narg('target_role')::text IS NULL OR a.target_role = sqlc.narg('target_role'))
  AND (sqlc.narg('target_class_id')::uuid IS NULL OR a.target_class_id = sqlc.narg('target_class_id'))
  AND (sqlc.narg('author_id')::uuid IS NULL OR a.author_id = sqlc.narg('author_id'))
  AND (
      sqlc.narg('search')::text IS NULL
      OR a.title ILIKE '%' || sqlc.narg('search') || '%'
      OR a.content ILIKE '%' || sqlc.narg('search') || '%'
  );

-- name: ListFeedAnnouncements :many
SELECT 
    a.id,
    a.title,
    a.content,
    a.author_id,
    u.email AS author_email,
    p.first_name AS author_first_name,
    p.last_name AS author_last_name,
    a.status,
    a.target_type,
    a.target_role,
    a.target_class_id,
    c.name AS target_class_name,
    a.priority,
    a.published_at,
    a.expires_at,
    a.created_at,
    a.updated_at
FROM announcements a
JOIN users u ON u.id = a.author_id
JOIN people p ON p.id = u.person_id
LEFT JOIN classes c ON c.id = a.target_class_id
WHERE (
    (a.status = 'PUBLISHED' AND (a.expires_at IS NULL OR a.expires_at > now()))
    OR a.author_id = $1
)
AND (
    a.author_id = $1
    OR a.target_type = 'ALL'
    OR (a.target_type = 'ROLE' AND a.target_role IN (
        SELECT r.name FROM user_roles ur
        JOIN roles r ON r.id = ur.role_id
        WHERE ur.user_id = $1
    ))
    OR (a.target_type = 'CLASS' AND a.target_class_id IN (
        SELECT ta.class_id FROM teacher_assignments ta
        JOIN staff st ON st.id = ta.teacher_id
        JOIN users u2 ON u2.person_id = st.person_id
        WHERE u2.id = $1
        UNION
        SELECT e.class_id FROM enrollments e
        JOIN students s ON s.id = e.student_id
        JOIN users u2 ON u2.person_id = s.person_id
        WHERE u2.id = $1 AND e.status = 'ENROLLED'
        UNION
        SELECT e.class_id FROM enrollments e
        JOIN student_guardians sg ON sg.student_id = e.student_id
        JOIN guardians g ON g.id = sg.guardian_id
        JOIN users u2 ON u2.person_id = g.person_id
        WHERE u2.id = $1 AND e.status = 'ENROLLED'
    ))
)
ORDER BY COALESCE(a.published_at, a.created_at) DESC
LIMIT $2 OFFSET $3;

-- name: CountFeedAnnouncements :one
SELECT count(*)
FROM announcements a
WHERE (
    (a.status = 'PUBLISHED' AND (a.expires_at IS NULL OR a.expires_at > now()))
    OR a.author_id = $1
)
AND (
    a.author_id = $1
    OR a.target_type = 'ALL'
    OR (a.target_type = 'ROLE' AND a.target_role IN (
        SELECT r.name FROM user_roles ur
        JOIN roles r ON r.id = ur.role_id
        WHERE ur.user_id = $1
    ))
    OR (a.target_type = 'CLASS' AND a.target_class_id IN (
        SELECT ta.class_id FROM teacher_assignments ta
        JOIN staff st ON st.id = ta.teacher_id
        JOIN users u2 ON u2.person_id = st.person_id
        WHERE u2.id = $1
        UNION
        SELECT e.class_id FROM enrollments e
        JOIN students s ON s.id = e.student_id
        JOIN users u2 ON u2.person_id = s.person_id
        WHERE u2.id = $1 AND e.status = 'ENROLLED'
        UNION
        SELECT e.class_id FROM enrollments e
        JOIN student_guardians sg ON sg.student_id = e.student_id
        JOIN guardians g ON g.id = sg.guardian_id
        JOIN users u2 ON u2.person_id = g.person_id
        WHERE u2.id = $1 AND e.status = 'ENROLLED'
    ))
);

-- name: UpdateAnnouncement :one
UPDATE announcements
SET
    title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    target_type = COALESCE(sqlc.narg('target_type'), target_type),
    target_role = sqlc.narg('target_role'),
    target_class_id = sqlc.narg('target_class_id'),
    priority = COALESCE(sqlc.narg('priority'), priority),
    expires_at = sqlc.narg('expires_at')
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateAnnouncementStatus :one
UPDATE announcements
SET
    status = $2,
    published_at = $3
WHERE id = $1
RETURNING *;

-- name: DeleteAnnouncement :exec
DELETE FROM announcements
WHERE id = $1;
