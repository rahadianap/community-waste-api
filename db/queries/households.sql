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
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountHouseholds :one
SELECT COUNT(*) FROM households;

-- name: DeleteHousehold :exec
DELETE FROM households
WHERE id = $1;
