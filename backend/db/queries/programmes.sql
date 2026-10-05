-- name: CreateProgramme :one
INSERT INTO programmes (code, name, description)
VALUES (sqlc.arg(code), sqlc.arg(name), sqlc.narg(description))
RETURNING *;

-- name: GetProgrammeByID :one
SELECT * FROM programmes
WHERE id = $1;

-- name: GetProgrammeByCode :one
SELECT * FROM programmes
WHERE code = $1;

-- name: ListProgrammes :many
SELECT * FROM programmes
ORDER BY name ASC;

-- name: UpdateProgramme :one
UPDATE programmes
SET
    code = COALESCE(sqlc.narg(code), code),
    name = COALESCE(sqlc.narg(name), name),
    description = COALESCE(sqlc.narg(description), description)
WHERE id = sqlc.arg(id)
RETURNING *;
