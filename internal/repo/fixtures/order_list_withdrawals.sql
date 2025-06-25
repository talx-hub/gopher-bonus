TRUNCATE TABLE withdrawn_orders RESTART IDENTITY CASCADE;
TRUNCATE TABLE accrued_orders RESTART IDENTITY CASCADE;
TRUNCATE TABLE password_hashes RESTART IDENTITY CASCADE;
TRUNCATE TABLE user_hashes RESTART IDENTITY CASCADE;
TRUNCATE TABLE statuses RESTART IDENTITY CASCADE;

INSERT INTO user_hashes (id_user, hash_login) VALUES
    ('1', 'user1hash'),
    ('2', 'user2hash');

INSERT INTO password_hashes (id_user, hash_password) VALUES
    ('1', 'user1password-hash'),
    ('2', 'user2password-hash');

INSERT INTO withdrawn_orders (id_user, name_order, processed_at, amount) VALUES
    ('1', 'order-1', NOW(), 100.50),
    ('1', 'order-2', NOW(), 50.00),
    ('1', 'order-3', NOW(), 200.75),
    ('2', 'order-4', NOW(), 10.00);
