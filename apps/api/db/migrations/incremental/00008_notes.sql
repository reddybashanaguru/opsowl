-- +goose Up

CREATE TYPE note_type_enum AS ENUM ('internal', 'formal');

CREATE TABLE notes (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id      UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    author_id    UUID        NOT NULL REFERENCES users(id),
    title        VARCHAR(255) NOT NULL DEFAULT '',
    content      TEXT        NOT NULL DEFAULT '',       -- Tiptap HTML output
    content_text TEXT        NOT NULL DEFAULT '',       -- plain text for full-text search
    note_type    note_type_enum NOT NULL DEFAULT 'internal',
    is_pinned    BOOLEAN     NOT NULL DEFAULT FALSE,
    ai_generated BOOLEAN     NOT NULL DEFAULT FALSE,
    ai_assisted  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_notes_case_id   ON notes(case_id)              WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_author_id ON notes(author_id)            WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_created   ON notes(case_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_pinned    ON notes(case_id, is_pinned)   WHERE deleted_at IS NULL;

-- --------------------------------------------------------------------
-- comments: on notes or timeline events (one reply level for MVP)
-- --------------------------------------------------------------------
CREATE TABLE comments (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id           UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    timeline_event_id UUID        REFERENCES timeline_events(id) ON DELETE SET NULL,
    note_id           UUID        REFERENCES notes(id) ON DELETE CASCADE,
    author_id         UUID        NOT NULL REFERENCES users(id),
    parent_id         UUID        REFERENCES comments(id) ON DELETE CASCADE,
    content           TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT chk_comment_target CHECK (timeline_event_id IS NOT NULL OR note_id IS NOT NULL)
);

CREATE INDEX idx_comments_case_id           ON comments(case_id)           WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_timeline_event_id ON comments(timeline_event_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_note_id           ON comments(note_id)           WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_parent_id         ON comments(parent_id)         WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_created           ON comments(case_id, created_at DESC) WHERE deleted_at IS NULL;

-- --------------------------------------------------------------------
-- mentions: tracks @mention records for notification delivery
-- --------------------------------------------------------------------
CREATE TABLE mentions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_id        UUID        REFERENCES comments(id) ON DELETE CASCADE,
    note_id           UUID        REFERENCES notes(id) ON DELETE CASCADE,
    mentioned_user_id UUID        NOT NULL REFERENCES users(id),
    mentioned_by      UUID        NOT NULL REFERENCES users(id),
    is_read           BOOLEAN     NOT NULL DEFAULT FALSE,
    read_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mention_source CHECK (comment_id IS NOT NULL OR note_id IS NOT NULL)
);

CREATE INDEX idx_mentions_user    ON mentions(mentioned_user_id, is_read);
CREATE INDEX idx_mentions_comment ON mentions(comment_id);
CREATE INDEX idx_mentions_note    ON mentions(note_id);
CREATE INDEX idx_mentions_created ON mentions(created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS mentions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS notes;
DROP TYPE IF EXISTS note_type_enum;
