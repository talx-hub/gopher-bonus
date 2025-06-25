TRUNCATE TABLE user_hashes CASCADE ;
TRUNCATE TABLE password_hashes CASCADE;

INSERT INTO user_hashes (id_user, hash_login)
VALUES
    ('1', 'user1hash'),
    ('2', 'user2hash');

INSERT INTO password_hashes (id_user, hash_password)
VALUES
    ('1', 'user1password-hash'),
    ('2', 'user2password-hash');
