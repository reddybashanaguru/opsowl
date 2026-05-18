-- Timeline domain: real-time event log per case.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE timeline_events (
    id           UUID                   PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id      UUID                   NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    actor_id     UUID                   REFERENCES users(id),
    event_type   TEXT                   NOT NULL,
    source       timeline_event_source  NOT NULL DEFAULT 'system',
    payload      JSONB                  NOT NULL DEFAULT '{}',
    occurred_at  TIMESTAMPTZ            NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ            NOT NULL DEFAULT NOW(),
    is_backdated BOOLEAN                NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_timeline_case_id    ON timeline_events (case_id, occurred_at DESC);
CREATE INDEX idx_timeline_actor_id   ON timeline_events (actor_id);
CREATE INDEX idx_timeline_event_type ON timeline_events (event_type, case_id);
CREATE INDEX idx_timeline_source     ON timeline_events (source, case_id);
