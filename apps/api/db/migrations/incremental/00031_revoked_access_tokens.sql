-- +goose Up
-- +goose StatementBegin
CREATE TABLE revoked_access_tokens (
    jti        UUID        PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);
-- Index for the cleanup job that deletes expired rows.
CREATE INDEX revoked_access_tokens_expires_at_idx ON revoked_access_tokens(expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS revoked_access_tokens;
-- +goose StatementEnd
