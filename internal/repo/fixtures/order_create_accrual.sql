TRUNCATE TABLE withdrawn_orders RESTART IDENTITY CASCADE;
TRUNCATE TABLE accrued_orders RESTART IDENTITY CASCADE;
TRUNCATE TABLE password_hashes RESTART IDENTITY CASCADE;
TRUNCATE TABLE user_hashes RESTART IDENTITY CASCADE;

INSERT INTO user_hashes (id_user, hash_login)
VALUES
    ('1', 'user1hash'),
    ('2', 'user2hash');

INSERT INTO password_hashes (id_user, hash_password)
VALUES
    ('1', 'user1password-hash'),
    ('2', 'user2password-hash');

