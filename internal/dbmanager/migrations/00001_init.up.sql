BEGIN TRANSACTION;

    CREATE TABLE user_hashes(
        id_user INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        hash_login VARCHAR(200) NOT NULL);

    CREATE TABLE password_hashes(
        id_password INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user INT REFERENCES user_hashes(id_user) NOT NULL,
        hash_password VARCHAR(200) NOT NULL);

    CREATE TABLE statuses(
        id_status INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        name_status VARCHAR(30) NOT NULL);

    CREATE TABLE accrued_orders(
        id_acc_order INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user INT REFERENCES user_hashes(id_user) NOT NULL,
        order_no VARCHAR(24) NOT NULL,
        uploaded_at timestamp with time zone NOT NULL,
        id_status INT REFERENCES statuses(id_status) NOT NULL);

    CREATE TABLE withdrawn_orders(
        id_withdrawn_order INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user INT REFERENCES user_hashes(id_user) NOT NULL,
        order_no VARCHAR(24) NOT NULL,
        processed_at timestamp with time zone NOT NULL);

    CREATE TABLE accruals(
        id_acc INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_acc_order INT REFERENCES accrued_orders(id_acc_order) NOT NULL,
        amount DECIMAL(12, 2) NOT NULL);

    CREATE TABLE withdraws(
        id_withdraw INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_withdrawn_order INT REFERENCES withdrawn_orders(id_withdrawn_order),
        amount DECIMAL(12, 2) NOT NULL);

ALTER TABLE user_hashes ADD CONSTRAINT unique_login UNIQUE("hash_login");
ALTER TABLE password_hashes ADD CONSTRAINT unique_hash UNIQUE("hash_password");
ALTER TABLE password_hashes ADD CONSTRAINT unique_user_id UNIQUE("id_user");
ALTER TABLE statuses ADD CONSTRAINT unique_name_status UNIQUE ("name_status");
ALTER TABLE accrued_orders ADD CONSTRAINT unique_accrual_no UNIQUE ("order_no");
ALTER TABLE withdrawn_orders ADD CONSTRAINT unique_withdrawn_no UNIQUE ("order_no");
ALTER TABLE accruals ADD CONSTRAINT non_negative_bonus_amount CHECK (amount::numeric >= 0);
ALTER TABLE accruals ADD CONSTRAINT unique_accrual_id UNIQUE ("id_acc_order");
ALTER TABLE withdraws ADD CONSTRAINT non_negative_bonus_amount CHECK (amount::numeric >= 0);
ALTER TABLE withdraws ADD CONSTRAINT unique_withdrawn_id UNIQUE ("id_withdrawn_order");

COMMIT;
