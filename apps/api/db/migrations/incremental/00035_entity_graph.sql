-- +goose Up
-- +goose StatementBegin

CREATE TABLE entity_relationships (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id              UUID        NOT NULL,
  from_subject_id     UUID        NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  to_subject_id       UUID        NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  relationship_type   TEXT        NOT NULL,
  -- Relationship types:
  --   OWNS_ACCOUNT       person → account
  --   TRANSACTS_WITH     person/account ↔ person/account (money flow)
  --   FAMILY_OF          person ↔ person (declared family)
  --   EMPLOYED_BY        person → company
  --   DIRECTOR_OF        person → company
  --   BENEFICIAL_OWNER_OF person → company/account
  --   SAME_ADDRESS       inferred (shared physical address)
  --   SAME_PHONE         inferred (shared phone number)
  --   SAME_IP            inferred (shared device/IP)
  --   SAME_DEVICE        inferred (shared device fingerprint)
  --   GUARANTOR_OF       person → loan/account
  --   INTRODUCED_BY      person → person (referral chain)
  strength            FLOAT       NOT NULL DEFAULT 1.0,  -- 0.0–1.0; used for graph layout weighting
  source              TEXT        NOT NULL DEFAULT 'manual', -- manual | transaction | document | kyc | alert | system
  metadata            JSONB       NOT NULL DEFAULT '{}', -- {shared_value, account_id, transaction_ids, etc.}
  first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_active           BOOLEAN     NOT NULL DEFAULT true,
  created_by          UUID        REFERENCES users(id) ON DELETE SET NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  -- Prevent duplicate edges (same direction + type)
  UNIQUE (org_id, from_subject_id, to_subject_id, relationship_type)
);

CREATE INDEX idx_entity_rel_from   ON entity_relationships (from_subject_id) WHERE is_active = true;
CREATE INDEX idx_entity_rel_to     ON entity_relationships (to_subject_id)   WHERE is_active = true;
CREATE INDEX idx_entity_rel_org    ON entity_relationships (org_id)           WHERE is_active = true;
CREATE INDEX idx_entity_rel_type   ON entity_relationships (org_id, relationship_type) WHERE is_active = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS entity_relationships;
-- +goose StatementEnd
