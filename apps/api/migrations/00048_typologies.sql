-- +goose Up
-- FATF Typology Library + Case Intelligence
-- Typologies are seeded at server startup via Go (typologies/seed.go).
-- case_typologies is the many-to-many join: investigators tag typologies to cases.

CREATE TABLE IF NOT EXISTS typologies (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code          VARCHAR(30) UNIQUE NOT NULL,
    name          TEXT        NOT NULL,
    category      VARCHAR(20) NOT NULL,
    description   TEXT        NOT NULL,
    red_flags     JSONB       NOT NULL DEFAULT '[]',
    risk_keywords JSONB       NOT NULL DEFAULT '[]',
    fatf_reference TEXT       NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS case_typologies (
    case_id       UUID NOT NULL REFERENCES cases(id)      ON DELETE CASCADE,
    typology_id   UUID NOT NULL REFERENCES typologies(id) ON DELETE CASCADE,
    tagged_by     UUID REFERENCES users(id),
    tagged_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes         TEXT        NOT NULL DEFAULT '',
    is_ai_suggested BOOLEAN   NOT NULL DEFAULT FALSE,
    PRIMARY KEY (case_id, typology_id)
);

CREATE INDEX IF NOT EXISTS idx_case_typologies_case_id ON case_typologies(case_id);
CREATE INDEX IF NOT EXISTS idx_typologies_category      ON typologies(category);

-- +goose Down

DROP TABLE IF EXISTS case_typologies;
DROP TABLE IF EXISTS typologies;
