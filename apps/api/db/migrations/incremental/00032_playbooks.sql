-- +goose Up
-- +goose StatementBegin

ALTER TABLE cases ADD COLUMN IF NOT EXISTS case_type TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_cases_case_type ON cases (case_type) WHERE deleted_at IS NULL AND case_type != '';

CREATE TABLE playbooks (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  case_type   TEXT NOT NULL,
  is_active   BOOLEAN NOT NULL DEFAULT true,
  org_id      UUID NOT NULL,
  created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_playbooks_case_type ON playbooks (case_type, org_id) WHERE is_active = true;

CREATE TABLE playbook_steps (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  playbook_id      UUID NOT NULL REFERENCES playbooks(id) ON DELETE CASCADE,
  title            TEXT NOT NULL,
  description      TEXT NOT NULL DEFAULT '',
  priority         task_priority NOT NULL DEFAULT 'medium',
  step_order       INT NOT NULL DEFAULT 0,
  due_hours        INT,
  depends_on_step  INT NOT NULL DEFAULT 0,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_playbook_steps_playbook ON playbook_steps (playbook_id, step_order);

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS playbook_step_id    UUID REFERENCES playbook_steps(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS playbook_step_order INT NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS depends_on_step     INT NOT NULL DEFAULT 0;
CREATE INDEX idx_tasks_playbook ON tasks (case_id, playbook_step_order) WHERE playbook_step_order > 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_tasks_playbook;
ALTER TABLE tasks DROP COLUMN IF EXISTS depends_on_step;
ALTER TABLE tasks DROP COLUMN IF EXISTS playbook_step_order;
ALTER TABLE tasks DROP COLUMN IF EXISTS playbook_step_id;
DROP TABLE IF EXISTS playbook_steps;
DROP TABLE IF EXISTS playbooks;
DROP INDEX IF EXISTS idx_cases_case_type;
ALTER TABLE cases DROP COLUMN IF EXISTS case_type;

-- +goose StatementEnd
