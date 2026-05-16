-- +goose Up
-- +goose StatementBegin

CREATE TYPE timeline_event_type AS ENUM (
  'case_created', 'case_status_changed', 'case_assigned', 'case_tagged',
  'document_uploaded', 'document_processed', 'document_ai_analyzed',
  'note_added', 'note_edited', 'comment_added',
  'task_created', 'task_completed', 'task_overdue',
  'ai_query', 'ai_summary_generated', 'ai_action_suggested',
  'user_login', 'user_action',
  'sla_warning', 'sla_breached',
  'notification_sent'
);

CREATE TYPE timeline_event_source AS ENUM ('system', 'user', 'ai');

CREATE TABLE timeline_events (
  id           UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at   TIMESTAMPTZ              NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ              NOT NULL DEFAULT now(),
  deleted_at   TIMESTAMPTZ,

  case_id      UUID                     NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  event_type   timeline_event_type      NOT NULL,
  source       timeline_event_source    NOT NULL,
  actor_id     UUID                     REFERENCES users(id) ON DELETE SET NULL,

  -- real-world time (sortable, back-datable)
  occurred_at  TIMESTAMPTZ              NOT NULL,

  payload      JSONB                    NOT NULL DEFAULT '{}',
  metadata     JSONB                    NOT NULL DEFAULT '{}'
);

-- primary read pattern: all events for a case newest-first
CREATE INDEX idx_timeline_case_occurred
  ON timeline_events (case_id, occurred_at DESC)
  WHERE deleted_at IS NULL;

-- filter by event_type within a case
CREATE INDEX idx_timeline_case_event_type
  ON timeline_events (case_id, event_type)
  WHERE deleted_at IS NULL;

-- filter by source within a case
CREATE INDEX idx_timeline_source
  ON timeline_events (case_id, source)
  WHERE deleted_at IS NULL;

-- actor-level lookup
CREATE INDEX idx_timeline_actor
  ON timeline_events (actor_id, occurred_at DESC)
  WHERE actor_id IS NOT NULL AND deleted_at IS NULL;

-- JSONB payload search
CREATE INDEX idx_timeline_payload_gin
  ON timeline_events USING gin (payload);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS timeline_events;
DROP TYPE IF EXISTS timeline_event_source;
DROP TYPE IF EXISTS timeline_event_type;
-- +goose StatementEnd
