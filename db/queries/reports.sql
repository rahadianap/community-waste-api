-- name: GetWasteSummaryReport :many
SELECT 
    type,
    status,
    COUNT(*)::bigint AS total
FROM waste_pickups
GROUP BY type, status
ORDER BY type, status;

-- name: GetPaymentSummaryReport :many
SELECT 
    status,
    COUNT(*)::bigint AS count,
    COALESCE(SUM(amount), 0)::numeric AS total_amount
FROM payments
GROUP BY status
ORDER BY status;

-- name: GetTotalRevenue :one
SELECT COALESCE(SUM(amount), 0)::numeric AS total_revenue
FROM payments
WHERE status = 'paid';
