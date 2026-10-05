-- name: CreateGuardian :one
INSERT INTO guardians (person_id, occupation)
VALUES (sqlc.arg(person_id), sqlc.narg(occupation))
RETURNING *;

-- name: GetGuardianByID :one
SELECT
    g.id,
    g.person_id,
    g.occupation,
    g.created_at,
    g.updated_at,
    p.first_name,
    p.last_name,
    p.other_names,
    p.date_of_birth,
    p.gender,
    p.phone,
    p.email,
    p.address
FROM guardians g
JOIN people p ON p.id = g.person_id
WHERE g.id = sqlc.arg(id);

-- name: ListGuardians :many
-- Backs the "search existing guardian to link" workflow as well as a
-- standalone guardians directory. No status filter (guardians, unlike
-- students/staff, have no archived/inactive state in this milestone).
SELECT
    g.id,
    g.occupation,
    g.created_at,
    p.first_name,
    p.last_name,
    p.other_names,
    p.phone,
    p.email
FROM guardians g
JOIN people p ON p.id = g.person_id
WHERE
    sqlc.narg('search')::text IS NULL
    OR (
        p.first_name || ' ' || p.last_name || ' ' || COALESCE(p.other_names, '')
    ) ILIKE '%' || sqlc.narg('search')::text || '%'
ORDER BY p.last_name, p.first_name
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountGuardians :one
SELECT count(*)
FROM guardians g
JOIN people p ON p.id = g.person_id
WHERE
    sqlc.narg('search')::text IS NULL
    OR (
        p.first_name || ' ' || p.last_name || ' ' || COALESCE(p.other_names, '')
    ) ILIKE '%' || sqlc.narg('search')::text || '%';

-- name: UpdateGuardianOccupation :one
UPDATE guardians
SET occupation = sqlc.narg(occupation)
WHERE id = sqlc.arg(id)
RETURNING *;
