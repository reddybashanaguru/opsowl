-- +goose Up
-- +goose StatementBegin

-- Case assignment rules: auto-route new cases to users or roles based on
-- priority, case type, or inbound source. First matching rule (by priority_order) wins.
CREATE TABLE IF NOT EXISTS assignment_rules (
  id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name             TEXT         NOT NULL,
  description      TEXT,
  priority_order   INT          NOT NULL DEFAULT 100,
  -- Match criteria (NULL = match any)
  match_priority   TEXT,        -- 'critical' | 'high' | 'medium' | 'low' | NULL
  match_source_id  UUID         REFERENCES inbound_sources(id) ON DELETE SET NULL,
  -- Assignment target — assign to a specific user OR to all users with a role
  assign_to_user   UUID         REFERENCES users(id) ON DELETE SET NULL,
  assign_to_role   TEXT,        -- 'analyst' | 'manager' | NULL
  is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
  created_by       UUID         REFERENCES users(id) ON DELETE SET NULL,
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assignment_rules_active ON assignment_rules (priority_order ASC)
  WHERE is_active = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS assignment_rules;
-- +goose StatementEnd
