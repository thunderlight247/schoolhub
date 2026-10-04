-- name: ListStaffPositions :many
SELECT * FROM staff_positions
ORDER BY category, title;

-- name: GetStaffPositionByID :one
SELECT * FROM staff_positions
WHERE id = $1;

-- name: CreateStaffPosition :one
INSERT INTO staff_positions (title, category, description)
VALUES (sqlc.arg(title), sqlc.arg(category), sqlc.narg(description))
RETURNING *;
