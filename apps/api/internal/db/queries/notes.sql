-- name: CreateNote :one
INSERT INTO notes (case_id, author_id, title, content, content_text, note_type, is_pinned, ai_generated)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, case_id, author_id, title, content, content_text, note_type, is_pinned, ai_generated, ai_assisted, created_at, updated_at, deleted_at;

-- name: GetNoteByID :one
SELECT
    n.id, n.case_id, n.author_id, n.title, n.content, n.content_text,
    n.note_type, n.is_pinned, n.ai_generated, n.ai_assisted,
    n.created_at, n.updated_at, n.deleted_at,
    u.full_name AS author_name,
    (SELECT COUNT(*) FROM comments c WHERE c.note_id = n.id AND c.deleted_at IS NULL) AS comment_count
FROM notes n
JOIN users u ON u.id = n.author_id
WHERE n.id = $1 AND n.deleted_at IS NULL;

-- name: ListNotesByCase :many
SELECT
    n.id, n.case_id, n.author_id, n.title, n.content, n.content_text,
    n.note_type, n.is_pinned, n.ai_generated, n.ai_assisted,
    n.created_at, n.updated_at,
    u.full_name AS author_name,
    (SELECT COUNT(*) FROM comments c WHERE c.note_id = n.id AND c.deleted_at IS NULL) AS comment_count
FROM notes n
JOIN users u ON u.id = n.author_id
WHERE n.case_id = $1
  AND n.deleted_at IS NULL
  AND ($2 = '' OR n.note_type::TEXT = $2)
  AND ($3::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR n.id < $3)
ORDER BY n.is_pinned DESC, n.created_at DESC
LIMIT $4;

-- name: CountNotesByCase :one
SELECT COUNT(*) FROM notes
WHERE case_id = $1 AND deleted_at IS NULL
  AND ($2 = '' OR note_type::TEXT = $2);

-- name: UpdateNote :one
UPDATE notes SET
    title        = CASE WHEN $2 = '' THEN title ELSE $2 END,
    content      = CASE WHEN $3 = '' THEN content ELSE $3 END,
    content_text = CASE WHEN $4 = '' THEN content_text ELSE $4 END,
    note_type    = CASE WHEN $5 = '' THEN note_type ELSE $5::note_type_enum END,
    is_pinned    = $6,
    ai_generated = CASE WHEN $7 AND ai_generated THEN FALSE ELSE ai_generated END,
    ai_assisted  = CASE WHEN $7 AND ai_generated THEN TRUE ELSE ai_assisted END,
    updated_at   = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, case_id, author_id, title, content, content_text, note_type, is_pinned, ai_generated, ai_assisted, created_at, updated_at, deleted_at;

-- name: SoftDeleteNote :one
UPDATE notes SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;

-- name: CreateComment :one
INSERT INTO comments (case_id, timeline_event_id, note_id, author_id, parent_id, content)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, case_id, timeline_event_id, note_id, author_id, parent_id, content, created_at, updated_at, deleted_at;

-- name: ListCommentsByNote :many
SELECT
    c.id, c.note_id, c.timeline_event_id, c.author_id, c.parent_id, c.content, c.created_at,
    u.full_name AS author_name
FROM comments c
JOIN users u ON u.id = c.author_id
WHERE c.note_id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at ASC;

-- name: ListCommentsByTimelineEvent :many
SELECT
    c.id, c.note_id, c.timeline_event_id, c.author_id, c.parent_id, c.content, c.created_at,
    u.full_name AS author_name
FROM comments c
JOIN users u ON u.id = c.author_id
WHERE c.timeline_event_id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at ASC;

-- name: SoftDeleteComment :one
UPDATE comments SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;

-- name: CreateMention :one
INSERT INTO mentions (comment_id, note_id, mentioned_user_id, mentioned_by)
VALUES ($1, $2, $3, $4)
RETURNING id, comment_id, note_id, mentioned_user_id, mentioned_by, is_read, read_at, created_at;

-- name: MarkMentionRead :one
UPDATE mentions SET is_read = TRUE, read_at = NOW()
WHERE id = $1 AND mentioned_user_id = $2
RETURNING id;

-- name: GetUnreadMentionCount :one
SELECT COUNT(*) FROM mentions
WHERE mentioned_user_id = $1 AND is_read = FALSE;

-- name: ListCaseActivity :many
SELECT
    n.id,
    n.case_id,
    'note'        AS activity_type,
    n.author_id,
    n.title       AS content_preview,
    n.note_type::TEXT AS sub_type,
    n.created_at
FROM notes n
WHERE n.case_id = $1 AND n.deleted_at IS NULL
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR n.id < $2)
UNION ALL
SELECT
    c.id,
    c.case_id,
    'comment'              AS activity_type,
    c.author_id,
    LEFT(c.content, 80)   AS content_preview,
    ''                     AS sub_type,
    c.created_at
FROM comments c
WHERE c.case_id = $1 AND c.deleted_at IS NULL
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR c.id < $2)
ORDER BY created_at DESC
LIMIT $3;
