-- name: CreateAccrual :exec
INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status)
VALUES ($1, $2, $3,
        (SELECT public.statuses.id_status
         FROM statuses
         WHERE name_status=$4));

-- name: CreateWithdrawal :exec
INSERT INTO withdrawn_orders(id_user, name_order, processed_at, amount)
VALUES ($1, $2, $3, $4);

-- name: FindOrderByID :one
SELECT id_user FROM accrued_orders
WHERE name_order=$1;

-- name: ListAccrualsByUserID :many
SELECT
    name_order,
    statuses.name_status,
    uploaded_at,
    COALESCE(amount, 0) AS accrual
FROM accrued_orders AS acc_o
         JOIN statuses ON acc_o.id_status = statuses.id_status
WHERE acc_o.id_user=$1
ORDER BY uploaded_at DESC;

-- name: UpdateAccrualStatus :execresult
UPDATE accrued_orders
SET id_status=(
    SELECT id_status
    FROM statuses
    WHERE name_status=$2),
    amount=$3
WHERE name_order=$1;

-- name: GetAccruedAmount :one
SELECT sum(amount)::decimal(12,2) as accrued
FROM accrued_orders
WHERE id_user=$1
GROUP BY id_user;

-- name: GetWithdrawnAmount :one
SELECT sum(amount)::decimal(12,2) as withdrawn
FROM withdrawn_orders
WHERE id_user=$1
GROUP BY id_user;

-- name: ListWithdrawalsByUser :many
SELECT name_order, amount, processed_at
FROM withdrawn_orders
WHERE id_user=$1
ORDER BY processed_at DESC;

-- name: SelectOrdersForProcessing :many
SELECT name_order FROM accrued_orders
WHERE id_status IN (
    SELECT id_status
    FROM statuses
    WHERE name_status IN ('NEW', 'PROCESSING')
);
