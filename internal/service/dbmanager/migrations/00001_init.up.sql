BEGIN TRANSACTION;

    CREATE TABLE user_hashes(
        id_user TEXT PRIMARY KEY UNIQUE,
        hash_login VARCHAR(200) NOT NULL);

    CREATE TABLE password_hashes(
        id_password INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user TEXT REFERENCES user_hashes(id_user) NOT NULL,
        hash_password VARCHAR(200) NOT NULL);

    CREATE TABLE statuses(
        id_status INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        name_status VARCHAR(30) NOT NULL);

    CREATE TABLE accrued_orders(
        id_acc_order INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user TEXT REFERENCES user_hashes(id_user) NOT NULL,
        name_order VARCHAR(36) NOT NULL,
        uploaded_at timestamp with time zone NOT NULL,
        id_status INT REFERENCES statuses(id_status) NOT NULL,
        amount DECIMAL(12, 2));

    CREATE TABLE withdrawn_orders(
        id_withdrawn_order INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user TEXT REFERENCES user_hashes(id_user) NOT NULL,
        name_order VARCHAR(24) NOT NULL,
        processed_at timestamp with time zone NOT NULL,
        amount DECIMAL(12, 2));

ALTER TABLE user_hashes ADD CONSTRAINT unique_login UNIQUE(hash_login);
ALTER TABLE user_hashes ADD CONSTRAINT check_hash_login_not_empty
CHECK (length(trim(hash_login)) > 0);

ALTER TABLE password_hashes ADD CONSTRAINT check_hash_password_not_empty
CHECK (length(trim(hash_password)) > 0);

ALTER TABLE statuses ADD CONSTRAINT unique_name_status UNIQUE (name_status);

ALTER TABLE accrued_orders ADD CONSTRAINT unique_accrual_no UNIQUE (name_order);
ALTER TABLE accrued_orders ADD CONSTRAINT check_name_order_not_empty
    CHECK (length(trim(name_order)) > 0);
ALTER TABLE accrued_orders ADD CONSTRAINT non_negative_bonus_amount CHECK (amount::numeric >= 0);

ALTER TABLE withdrawn_orders ADD CONSTRAINT unique_withdrawn_no UNIQUE (name_order);
ALTER TABLE withdrawn_orders ADD CONSTRAINT non_negative_bonus_amount CHECK (amount::numeric >= 0);
ALTER TABLE withdrawn_orders ADD CONSTRAINT check_name_order_not_empty
    CHECK (length(trim(name_order)) > 0);

COMMIT;
