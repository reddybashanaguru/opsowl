-- +goose Up
CREATE TABLE ubo_records (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id           UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    entity_name       TEXT        NOT NULL,
    owner_name        TEXT        NOT NULL,
    owner_nationality TEXT        NOT NULL DEFAULT '',
    owner_id_type     TEXT        NOT NULL DEFAULT '',
    owner_id_number   TEXT        NOT NULL DEFAULT '',
    owner_birth_date  TEXT        NOT NULL DEFAULT '',
    ownership_pct     NUMERIC(5,2) NOT NULL,
    control_type      TEXT        NOT NULL DEFAULT 'share'
                                  CHECK (control_type IN ('share','voting_right','other_control')),
    status            TEXT        NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending','verified','discrepant','n/a')),
    registry_url      TEXT        NOT NULL DEFAULT '',
    verified_by       UUID        REFERENCES users(id),
    verified_at       TIMESTAMPTZ,
    discrepancy_note  TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ubo_records_case_id ON ubo_records(case_id);

-- +goose Down
DROP TABLE IF EXISTS ubo_records;
