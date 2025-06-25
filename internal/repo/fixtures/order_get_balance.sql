TRUNCATE TABLE withdrawn_orders RESTART IDENTITY CASCADE;
TRUNCATE TABLE accrued_orders RESTART IDENTITY CASCADE;
TRUNCATE TABLE statuses RESTART IDENTITY CASCADE;
TRUNCATE TABLE password_hashes RESTART IDENTITY CASCADE;
TRUNCATE TABLE user_hashes RESTART IDENTITY CASCADE;

INSERT INTO statuses (name_status)
VALUES
    ('NEW'),
    ('PROCESSED'),
    ('INVALID'),
    ('PROCESSING');

INSERT INTO user_hashes (id_user, hash_login)
VALUES
    ('1', 'user1'),
    ('2', 'user2'),
    ('3', 'user3'),
    ('4', 'user4'),
    ('5', 'user5');

INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status, amount)
VALUES
    ('1', 'order-1a', NOW(), 2, 100.00),
    ('1', 'order-1b', NOW(), 2, 50.50);

INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status, amount)
VALUES
    ('3', 'order-3a', NOW(), 2, 100.00);

INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status, amount)
VALUES
    ('4', 'order-4a', NOW(), 2, 50.00);

INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status, amount)
VALUES
    ('5', 'order-5a', NOW(), 2, 300.00);

INSERT INTO withdrawn_orders (id_user, name_order, processed_at, amount)
VALUES
    ('3', 'withdraw-3a', NOW(), 100.00);

INSERT INTO withdrawn_orders (id_user, name_order, processed_at, amount)
VALUES
    ('4', 'withdraw-4a', NOW(), 70.00);

INSERT INTO withdrawn_orders (id_user, name_order, processed_at, amount)
VALUES
    ('5', 'withdraw-5a', NOW(), 100.50);
