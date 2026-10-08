-- name: CountPendingPaymentsByHouseholdID :one
-- Kritis untuk validasi Aturan Bisnis BR-01
SELECT COUNT(*) FROM payments
WHERE household_id = $1 AND status = 'pending';

-- name: CreatePayment :one
INSERT INTO payments (
    household_id,
    waste_id,
    amount,
    status
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments
WHERE id = $1;

-- name: ListPayments :many
SELECT * FROM payments
WHERE 
    (sqlc.narg('status')::payment_status IS NULL OR status = sqlc.narg('status')) AND
    (sqlc.narg('household_id')::uuid IS NULL OR household_id = sqlc.narg('household_id')) AND
    (sqlc.narg('start_date')::timestamptz IS NULL OR created_at >= sqlc.narg('start_date')) AND
    (sqlc.narg('end_date')::timestamptz IS NULL OR created_at <= sqlc.narg('end_date')) AND
    (sqlc.narg('cursor_created_at')::timestamptz IS NULL OR
        (created_at, id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: ConfirmPayment :one
UPDATE payments
SET 
    status = 'paid',
    payment_date = CURRENT_TIMESTAMP,
    proof_file_url = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND status = 'pending'
RETURNING *;
