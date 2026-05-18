-- Notes domain: rich-text notes, threaded comments, @mentions.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE notes (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id           UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    author_id         UUID        NOT NULL REFERENCES users(id),
    content           TEXT        NOT NULL,
    note_type         TEXT        NOT NULL DEFAULT 'internal'
                          CHECK (note_type IN ('internal', 'formal')),
    is_pinned         BOOLEAN     NOT NULL DEFAULT FALSE,
    is_ai_generated   BOOLEAN     NOT NULL DEFAULT FALSE,
    is_ai_assisted    BOOLEAN     NOT NULL DEFAULT FALSE,
    search_vector     TSVECTOR,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);
CREATE INDEX idx_notes_case_id ON notes (case_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_author  ON notes (author_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_notes_pinned  ON notes (case_id) WHERE is_pinned = TRUE AND deleted_at IS NULL;
CREATE INDEX idx_notes_search  ON notes USING GIN (search_vector);

CREATE TABLE comments (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id             UUID        REFERENCES notes(id) ON DELETE CASCADE,
    timeline_event_id   UUID        REFERENCES timeline_events(id) ON DELETE CASCADE,
    author_id           UUID        NOT NULL REFERENCES users(id),
    content             TEXT        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CHECK (num_nonnulls(note_id, timeline_event_id) = 1)
);
CREATE INDEX idx_comments_note_id  ON comments (note_id)           WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_event_id ON comments (timeline_event_id) WHERE deleted_at IS NULL;

CREATE TABLE mentions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id           UUID        NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    mentioned_user_id UUID        NOT NULL REFERENCES users(id),
    is_read           BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mentions_user ON mentions (mentioned_user_id) WHERE is_read = FALSE;
CREATE INDEX idx_mentions_note ON mentions (note_id);
