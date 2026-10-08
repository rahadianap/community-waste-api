-- name: CreateHousehold :one
INSERT INTO households (
    owner_name,
    address
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetHouseholdByID :one
SELECT * FROM households
WHERE id = $1;

-- name: ListHouseholds :many
SELECT * FROM households
WHERE sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: DeleteHousehold :exec
DELETE FROM households
WHERE id = $1;
