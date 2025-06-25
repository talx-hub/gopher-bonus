BEGIN TRANSACTION;

    DROP TABLE user_hashes;
    DROP TABLE password_hashes;
    DROP TABLE statuses;
    DROP TABLE accrued_orders;
    DROP TABLE withdrawn_orders;

COMMIT;
