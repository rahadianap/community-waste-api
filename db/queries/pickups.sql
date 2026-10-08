-- name: CreatePickup :one
INSERT INTO waste_pickups (
    household_id,
    type,
    safety_check,
    status
) VALUES (
    $1, $2, $3, 'pending'
) RETURNING *;

-- name: GetPickupByID :one
SELECT * FROM waste_pickups
WHERE id = $1;

-- name: ListPickups :many
SELECT * FROM waste_pickups
WHERE 
    (sqlc.narg('status')::pickup_status IS NULL OR status = sqlc.narg('status')) AND
    (sqlc.narg('household_id')::uuid IS NULL OR household_id = sqlc.narg('household_id')) AND
    (sqlc.narg('cursor_created_at')::timestamptz IS NULL OR
        (created_at, id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: UpdatePickupSchedule :one
UPDATE waste_pickups
SET 
    status = 'scheduled',
    pickup_date = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: UpdatePickupStatus :one
UPDATE waste_pickups
SET 
    status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
