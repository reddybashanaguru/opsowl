-- +goose Up
-- +goose StatementBegin

CREATE TABLE case_templates (
  id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  name                 TEXT          NOT NULL,
  description_skeleton TEXT          NOT NULL DEFAULT '',
  priority             case_priority NOT NULL DEFAULT 'medium',
  tags                 TEXT[]        NOT NULL DEFAULT '{}',
  is_active            BOOLEAN       NOT NULL DEFAULT true,
  created_by           UUID          REFERENCES users(id) ON DELETE SET NULL,
  created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Task stubs attached to a template. Applied in sort_order when creating a case.
CREATE TABLE case_template_tasks (
  id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID          NOT NULL REFERENCES case_templates(id) ON DELETE CASCADE,
  title       TEXT          NOT NULL,
  description TEXT          NOT NULL DEFAULT '',
  priority    task_priority NOT NULL DEFAULT 'medium',
  sort_order  INT           NOT NULL DEFAULT 0
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS case_template_tasks;
DROP TABLE IF EXISTS case_templates;
-- +goose StatementEnd
