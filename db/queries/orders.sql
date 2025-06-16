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
SELECT acc_o.order_no, statuses.name_status, uploaded_at, public.accruals.amount AS accrual
FROM accrued_orders AS acc_o
    JOIN statuses ON acc_o.id_status = statuses.id_status
    JOIN public.accruals ON acc_o.id_acc_order = public.accruals.id_acc_order
WHERE acc_o.id_user=$1;

-- name: AddAccruedAmount :exec
INSERT INTO accruals (id_acc_order, amount)
VALUES ((SELECT id_acc_order
         FROM accrued_orders
         WHERE order_no=$1),
        $2);

-- name: UpdateStatus :exec
UPDATE accrued_orders
SET id_status=(
    SELECT id_status
    FROM statuses
    WHERE name_status=$1)
WHERE order_no=$2;
