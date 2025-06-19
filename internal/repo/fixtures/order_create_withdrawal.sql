TRUNCATE TABLE user_hashes CASCADE;
TRUNCATE TABLE password_hashes CASCADE;
TRUNCATE TABLE accrued_orders CASCADE;
TRUNCATE TABLE withdrawn_orders CASCADE;

INSERT INTO user_hashes (id_user, hash_login) VALUES
                                                  ('user1', 'login1'),
                                                  ('user2', 'login2'),
                                                  ('user3', 'login3'),
                                                  ('user4', 'login4'),
                                                  ('user5', 'login5');

INSERT INTO password_hashes (id_user, hash_password) VALUES
                                                         ('user1', 'pass1'),
                                                         ('user2', 'pass2'),
                                                         ('user3', 'pass3'),
                                                         ('user4', 'pass4'),
                                                         ('user5', 'pass5');

INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status, amount)
VALUES ('user1', 'accrual1', now(), 3, 100.00);

INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status, amount)
VALUES ('user3', 'accrual3', now(), 3, 100.00);
INSERT INTO withdrawn_orders (id_user, name_order, processed_at, amount)
VALUES ('user3', 'withdraw3', now(), 100.00);

INSERT INTO withdrawn_orders (id_user, name_order, processed_at, amount)
VALUES ('user4', 'withdraw4', now(), 50.00);

INSERT INTO accrued_orders (id_user, name_order, uploaded_at, id_status, amount)
VALUES ('user5', 'accrual5', now(), 3, 200.00);
INSERT INTO withdrawn_orders (id_user, name_order, processed_at, amount)
VALUES ('user5', 'withdraw5', now(), 50.00);
