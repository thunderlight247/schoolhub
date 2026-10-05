-- name: LinkGuardianToStudent :one
INSERT INTO student_guardians (student_id, guardian_id, relationship)
VALUES (sqlc.arg(student_id), sqlc.arg(guardian_id), sqlc.arg(relationship))
RETURNING *;

-- name: UnlinkGuardianFromStudent :execrows
-- :execrows (not :exec) so the service can tell "unlinked something" from
-- "that link didn't exist" and return a friendly 404 instead of a silent
-- no-op 200.
DELETE FROM student_guardians
WHERE student_id = sqlc.arg(student_id) AND guardian_id = sqlc.arg(guardian_id);

-- name: ListGuardiansForStudent :many
SELECT
    g.id,
    g.occupation,
    sg.relationship,
    p.first_name,
    p.last_name,
    p.other_names,
    p.phone,
    p.email
FROM student_guardians sg
JOIN guardians g ON g.id = sg.guardian_id
JOIN people p ON p.id = g.person_id
WHERE sg.student_id = sqlc.arg(student_id)
ORDER BY p.last_name, p.first_name;

-- name: ListStudentsForGuardian :many
SELECT
    s.id,
    s.student_number,
    s.status,
    sg.relationship,
    p.first_name,
    p.last_name,
    p.other_names
FROM student_guardians sg
JOIN students s ON s.id = sg.student_id
JOIN people p ON p.id = s.person_id
WHERE sg.guardian_id = sqlc.arg(guardian_id)
ORDER BY p.last_name, p.first_name;
