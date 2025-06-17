-- name: CreateOrder :exec
INSERT INTO accrued_orders (id_user, order_no, uploaded_at, id_status)
VALUES ($1, $2, $3,
        (SELECT public.statuses.id_status
         FROM statuses
         WHERE name_status=$4));


-- name: FindOrderByID :one
SELECT id_user FROM accrued_orders
WHERE order_no=$1;

-- name: ListByUserID :many
SELECT
    order_no,
    statuses.name_status,
    uploaded_at,
    COALESCE(accruals.amount, 0) AS accrual
FROM accrued_orders AS acc_o
         JOIN statuses ON acc_o.id_status = statuses.id_status
         LEFT JOIN accruals ON acc_o.id_acc_order = accruals.id_acc_order
WHERE acc_o.id_user=$1
ORDER BY uploaded_at DESC;

-- name: AddAccruedAmount :exec
INSERT INTO accruals (id_acc_order, amount)
VALUES ((SELECT id_acc_order
         FROM accrued_orders
         WHERE order_no=$1),
        $2);

-- name: UpdateStatus :execresult
UPDATE accrued_orders
SET id_status=(
    SELECT id_status
    FROM statuses
    WHERE name_status=$1)
WHERE order_no=$2;

