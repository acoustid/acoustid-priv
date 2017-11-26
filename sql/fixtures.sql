BEGIN;

INSERT INTO account (api_key, enabled) VALUES ('api_key', true);
INSERT INTO account (api_key, enabled) VALUES ('disabled_api_key', false);

COMMIT;