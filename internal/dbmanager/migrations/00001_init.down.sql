BEGIN TRANSACTION;

    DROP TABLE users;
    DROP TABLE password_hashes;
    DROP TABLE statuses;
    DROP TABLE accrued_orders;
    DROP TABLE withdrawn_orders;
    DROP TABLE accruals;
    DROP TABLE withdraws;

COMMIT;
