-- name: CreatePerson :one
INSERT INTO people (
    first_name, last_name, other_names, date_of_birth, gender, phone, email, address
) VALUES (
    sqlc.arg(first_name), sqlc.arg(last_name), sqlc.narg(other_names), sqlc.narg(date_of_birth),
    sqlc.narg(gender), sqlc.narg(phone), sqlc.narg(email), sqlc.narg(address)
)
RETURNING *;

-- name: GetPersonByID :one
SELECT * FROM people
WHERE id = $1;

-- name: UpdatePerson :one
UPDATE people
SET
    first_name = sqlc.arg(first_name),
    last_name = sqlc.arg(last_name),
    other_names = sqlc.narg(other_names),
    date_of_birth = sqlc.narg(date_of_birth),
    gender = sqlc.narg(gender),
    phone = sqlc.narg(phone),
    email = sqlc.narg(email),
    address = sqlc.narg(address)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SearchPeopleByName :many
-- Backs the admin "search students/staff by name" screens (section 25).
-- Pagination happens here at the query level, not in application code.
SELECT * FROM people
WHERE last_name ILIKE '%' || sqlc.arg(search)::text || '%'
   OR first_name ILIKE '%' || sqlc.arg(search)::text || '%'
ORDER BY last_name, first_name
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);
