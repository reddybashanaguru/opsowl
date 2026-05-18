-- Tasks domain: tasks per case with state machine and threaded comments.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE tasks (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id      UUID          NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    title        TEXT          NOT NULL,
    description  TEXT          NOT NULL DEFAULT '',
    status       TEXT          NOT NULL DEFAULT 'todo'
                     CHECK (status IN ('todo','in_progress','done','cancelled')),
    priority     case_priority NOT NULL DEFAULT 'medium',
    assigned_to  UUID          REFERENCES users(id),
    created_by   UUID          NOT NULL REFERENCES users(id),
    due_date     TIMESTAMPTZ,
    is_overdue   BOOLEAN       NOT NULL DEFAULT FALSE,
    sort_order   INT           NOT NULL DEFAULT 0,
    search_vector TSVECTOR,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX idx_tasks_case_id     ON tasks (case_id)     WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_assigned_to ON tasks (assigned_to) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status      ON tasks (status)      WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_due_date    ON tasks (due_date)    WHERE deleted_at IS NULL AND is_overdue = FALSE;
CREATE INDEX idx_tasks_search      ON tasks USING GIN (search_vector);
-- Analytics index (Phase 2)
CREATE INDEX idx_tasks_assigned_status ON tasks(assigned_to, status) WHERE deleted_at IS NULL;

CREATE TABLE task_comments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id  UUID        NOT NULL REFERENCES users(id),
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_tc_task_id ON task_comments (task_id) WHERE deleted_at IS NULL;
