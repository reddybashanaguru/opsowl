-- +goose Up

-- ── inbound_sources ─────────────────────────────────────────────────────────
-- Every external system that can push events into OpsOwl must be registered
-- here first. The signing_secret is used for HMAC-SHA256 request validation.
CREATE TABLE inbound_sources (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT        NOT NULL,
    slug           TEXT        NOT NULL UNIQUE,         -- short identifier e.g. "finacle", "bancs"
    description    TEXT        NOT NULL DEFAULT '',
    signing_secret TEXT        NOT NULL,                -- 32-byte hex secret; used for HMAC validation
    is_active      BOOLEAN     NOT NULL DEFAULT true,
    created_by     UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

-- ── inbound_events ───────────────────────────────────────────────────────────
-- Every event received from an external source is stored here before (and
-- after) processing. The (source_id, idempotency_key) UNIQUE constraint
-- guarantees that re-submitted events never create duplicate cases.
CREATE TABLE inbound_events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id       UUID        NOT NULL REFERENCES inbound_sources(id),
    idempotency_key TEXT        NOT NULL,
    event_type      TEXT        NOT NULL,
    payload         JSONB       NOT NULL DEFAULT '{}',
    status          TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','processed','failed','ignored')),
    case_id         UUID        REFERENCES cases(id) ON DELETE SET NULL,
    error_message   TEXT,
    attempt_count   SMALLINT    NOT NULL DEFAULT 0,
    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, idempotency_key)
);

-- ── inbound_rules ────────────────────────────────────────────────────────────
-- Rules tell OpsOwl how to turn an inbound event into a case.
-- event_type_pattern supports exact match ("fraud_alert") or wildcard
-- suffix ("fraud_*"). Rules are evaluated by sort_order (lowest first);
-- the first matching active rule wins.
CREATE TABLE inbound_rules (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id            UUID        NOT NULL REFERENCES inbound_sources(id) ON DELETE CASCADE,
    event_type_pattern   TEXT        NOT NULL,
    case_priority        TEXT        NOT NULL DEFAULT 'medium'
                         CHECK (case_priority IN ('critical','high','medium','low')),
    tags                 TEXT[]      NOT NULL DEFAULT '{}',
    title_template       TEXT        NOT NULL DEFAULT '{{event_type}} from {{source_name}}',
    description_template TEXT        NOT NULL DEFAULT 'Source: {{source_name}}\nEvent: {{event_type}}\n\n{{payload}}',
    auto_assign_user_id  UUID        REFERENCES users(id) ON DELETE SET NULL,
    is_active            BOOLEAN     NOT NULL DEFAULT true,
    sort_order           INT         NOT NULL DEFAULT 0,
    created_by           UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbound_sources_slug        ON inbound_sources(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_inbound_events_source       ON inbound_events(source_id, created_at DESC);
CREATE INDEX idx_inbound_events_status       ON inbound_events(status)   WHERE status IN ('pending','failed');
CREATE INDEX idx_inbound_rules_source_order  ON inbound_rules(source_id, sort_order) WHERE is_active = true;

-- +goose Down
DROP TABLE IF EXISTS inbound_rules;
DROP TABLE IF EXISTS inbound_events;
DROP TABLE IF EXISTS inbound_sources;
