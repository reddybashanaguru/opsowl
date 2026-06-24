-- +goose Up
CREATE TABLE dsar_requests (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_name    TEXT        NOT NULL,
    subject_email   TEXT        NOT NULL DEFAULT '',
    request_date    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deadline        TIMESTAMPTZ NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'received'
                                CHECK (status IN ('received','in_progress','completed','rejected','deferred')),
    outcome         TEXT        CHECK (outcome IN ('pseudonymised','full_erasure','denied')),
    legal_basis     TEXT        NOT NULL DEFAULT '',
    processed_by    UUID        REFERENCES users(id),
    processed_at    TIMESTAMPTZ,
    cases_affected  UUID[]      NOT NULL DEFAULT '{}',
    notes           TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dsar_status ON dsar_requests(status);
CREATE INDEX idx_dsar_deadline ON dsar_requests(deadline);

CREATE TABLE anonymisation_records (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    dsar_id       UUID        NOT NULL REFERENCES dsar_requests(id) ON DELETE CASCADE,
    subject_id    UUID        NOT NULL REFERENCES subjects(id),
    fields_masked TEXT[]      NOT NULL,
    pseudonym     TEXT        NOT NULL,
    legal_basis   TEXT        NOT NULL,
    anonymised_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    anonymised_by UUID        NOT NULL REFERENCES users(id)
);

-- +goose Down
DROP TABLE IF EXISTS anonymisation_records;
DROP TABLE IF EXISTS dsar_requests;
