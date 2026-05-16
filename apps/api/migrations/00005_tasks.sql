-- +goose Up
-- +goose StatementBegin

CREATE TYPE task_status AS ENUM ('todo', 'in_progress', 'done', 'cancelled');
CREATE TYPE task_priority AS ENUM ('critical', 'high', 'medium', 'low');

CREATE TABLE tasks (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id           UUID            NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
    created_by        UUID            NOT NULL REFERENCES users(id),
    assigned_to       UUID            REFERENCES users(id),
    timeline_event_id UUID            REFERENCES timeline_events(id) ON DELETE SET NULL,
    title             TEXT            NOT NULL CHECK (char_length(title) BETWEEN 1 AND 255),
    description       TEXT,
    status            task_status     NOT NULL DEFAULT 'todo',
    priority          task_priority   NOT NULL DEFAULT 'medium',
    due_date          TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    is_overdue        BOOLEAN         NOT NULL DEFAULT FALSE,
    is_ai_suggested   BOOLEAN         NOT NULL DEFAULT FALSE,
    sort_order        INTEGER         NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

CREATE TABLE task_comments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id   UUID        NOT NULL REFERENCES users(id),
    content     TEXT        NOT NULL CHECK (char_length(content) BETWEEN 1 AND 4096),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_tasks_case_id      ON tasks(case_id)              WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_assigned_to  ON tasks(assigned_to)          WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status       ON tasks(status)               WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_due_date     ON tasks(due_date)             WHERE deleted_at IS NULL AND status NOT IN ('done','cancelled');
CREATE INDEX idx_tasks_is_overdue   ON tasks(is_overdue)           WHERE deleted_at IS NULL AND is_overdue = TRUE;
CREATE INDEX idx_tasks_case_status  ON tasks(case_id, status, priority) WHERE deleted_at IS NULL;

CREATE INDEX idx_task_comments_task ON task_comments(task_id) WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER task_comments_updated_at
    BEFORE UPDATE ON task_comments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS task_comments;
DROP TABLE IF EXISTS tasks;
DROP TYPE IF EXISTS task_priority;
DROP TYPE IF EXISTS task_status;
-- +goose StatementEnd
