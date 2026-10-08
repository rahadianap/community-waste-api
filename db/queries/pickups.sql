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
    (sqlc.narg('household_id')::uuid IS NULL OR household_id = sqlc.narg('household_id'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountPickups :one
SELECT COUNT(*) FROM waste_pickups
WHERE 
    (sqlc.narg('status')::pickup_status IS NULL OR status = sqlc.narg('status')) AND
    (sqlc.narg('household_id')::uuid IS NULL OR household_id = sqlc.narg('household_id'));

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
