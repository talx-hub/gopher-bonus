-- name: InsertLoginHash :one
INSERT INTO user_hashes (hash_login)
VALUES ($1)
RETURNING id_user;

-- name: InsertPasswordHash :exec
INSERT INTO password_hashes (id_user, hash_password)
VALUES ($1, $2);

-- name: Exists :one
SELECT EXISTS(SELECT 1
              FROM user_hashes
              WHERE hash_login = $1);

-- name: FindByLogin :one
SELECT user_hashes.id_user, user_hashes.hash_login, ph.hash_password
FROM user_hashes JOIN password_hashes ph on user_hashes.id_user = ph.id_user
WHERE hash_login = $1;

-- name: FindByID :one
SELECT user_hashes.id_user, user_hashes.hash_login, ph.hash_password
FROM user_hashes JOIN password_hashes ph on user_hashes.id_user = ph.id_user
WHERE user_hashes.id_user = $1;
