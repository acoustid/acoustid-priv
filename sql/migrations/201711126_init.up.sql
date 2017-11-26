BEGIN;

CREATE TABLE account (
    id      serial PRIMARY KEY,
    api_key text NOT NULL,
    enabled bool NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX catalog_idx_api_key
    ON account (api_key);

CREATE TABLE catalog (
    id         serial PRIMARY KEY,
    account_id int  NOT NULL REFERENCES account (id),
    name       text NOT NULL
);

CREATE UNIQUE INDEX catalog_idx_account_id_name
    ON catalog (account_id, name);

CREATE TABLE track_tpl (
    id               serial PRIMARY KEY,
    external_id      text  NOT NULL,
    fingerprint      bytea NOT NULL,
    fingerprint_sha1 bytea NOT NULL,
    metadata         jsonb
);

CREATE UNIQUE INDEX track_tpl_idx_external_id
    ON track_tpl (external_id);
CREATE INDEX track_tpl_idx_data_sha1
    ON track_tpl (fingerprint_sha1);

CREATE TABLE track_index_tpl (
    track_id int     NOT NULL,
    segment  int     NOT NULL,
    values   int4 [] NOT NULL,
    PRIMARY KEY (track_id, segment)
);

CREATE INDEX track_index_tpl_idx_values
    ON track_index_tpl USING GIN (values gin__int_ops);

COMMIT;
