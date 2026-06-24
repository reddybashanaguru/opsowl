-- +goose Up
-- EU Sanctions & PEP screening module.
-- Sources: EU Consolidated List, UN Security Council list, OFAC SDN.
-- Uses pg_trgm for fuzzy name matching (enabled in migration 00049).

CREATE TABLE sanctions_lists (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    source      TEXT        NOT NULL UNIQUE
                            CHECK (source IN ('eu_consolidated','un_consolidated','ofac_sdn')),
    loaded_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    entry_count INT         NOT NULL DEFAULT 0,
    checksum    TEXT        NOT NULL DEFAULT ''
);

CREATE TABLE sanctions_entries (
    id           UUID   PRIMARY KEY DEFAULT gen_random_uuid(),
    list_source  TEXT   NOT NULL,
    external_id  TEXT   NOT NULL,
    entity_type  TEXT   NOT NULL DEFAULT 'individual'
                        CHECK (entity_type IN ('individual','entity','vessel','aircraft')),
    primary_name TEXT   NOT NULL,
    aliases      TEXT[] NOT NULL DEFAULT '{}',
    nationality  TEXT   NOT NULL DEFAULT '',
    birth_date   TEXT   NOT NULL DEFAULT '',
    listed_on    DATE,
    programs     TEXT[] NOT NULL DEFAULT '{}',
    remarks      TEXT   NOT NULL DEFAULT '',
    UNIQUE (list_source, external_id)
);

-- Trigram index on primary_name for fuzzy matching
CREATE INDEX idx_sanctions_entries_trgm ON sanctions_entries
    USING gin(primary_name gin_trgm_ops);
-- GIN index on aliases array for ILIKE ANY lookups
CREATE INDEX idx_sanctions_entries_aliases ON sanctions_entries USING gin(aliases);
CREATE INDEX idx_sanctions_entries_source ON sanctions_entries(list_source);

CREATE TABLE sanctions_hits (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id   UUID        NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    entry_id     UUID        NOT NULL REFERENCES sanctions_entries(id),
    match_type   TEXT        NOT NULL CHECK (match_type IN ('exact','fuzzy','alias')),
    match_score  NUMERIC(4,3) NOT NULL DEFAULT 1.000,
    matched_name TEXT        NOT NULL,
    screened_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    screened_by  UUID        REFERENCES users(id),
    status       TEXT        NOT NULL DEFAULT 'open'
                             CHECK (status IN ('open','false_positive','confirmed','escalated')),
    reviewed_by  UUID        REFERENCES users(id),
    reviewed_at  TIMESTAMPTZ,
    notes        TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX idx_sanctions_hits_subject ON sanctions_hits(subject_id);
CREATE INDEX idx_sanctions_hits_status  ON sanctions_hits(status);

-- Add is_sanctioned flag to subjects if not already there
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS is_sanctioned BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS last_screened_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE subjects DROP COLUMN IF EXISTS last_screened_at;
ALTER TABLE subjects DROP COLUMN IF EXISTS is_sanctioned;
DROP TABLE IF EXISTS sanctions_hits;
DROP TABLE IF EXISTS sanctions_entries;
DROP TABLE IF EXISTS sanctions_lists;
