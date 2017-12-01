BEGIN;

INSERT INTO account (id) VALUES (1);
INSERT INTO api_key (api_key, account_id) VALUES ('api_key', 1);

COMMIT;