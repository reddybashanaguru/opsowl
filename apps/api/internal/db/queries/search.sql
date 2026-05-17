-- name: SearchCases :many
SELECT
  c.id,
  c.case_number,
  c.title,
  c.description,
  c.status,
  c.priority,
  c.sla_breached,
  c.created_at,
  u.full_name AS created_by_name,
  ts_rank(c.search_vector, websearch_to_tsquery('english', @query)) AS rank
FROM cases c
JOIN users u ON u.id = c.created_by
WHERE c.deleted_at IS NULL
  AND c.organization_id = @organization_id
  AND c.search_vector @@ websearch_to_tsquery('english', @query)
ORDER BY rank DESC, c.created_at DESC
LIMIT @row_limit;

-- name: SearchNotes :many
SELECT
  n.id,
  n.case_id,
  n.title,
  n.content_text,
  n.note_type,
  n.is_pinned,
  n.ai_generated,
  n.created_at,
  c.case_number,
  c.title AS case_title,
  ts_rank(n.search_vector, websearch_to_tsquery('english', @query)) AS rank
FROM notes n
JOIN cases c ON c.id = n.case_id AND c.deleted_at IS NULL
WHERE n.deleted_at IS NULL
  AND c.organization_id = @organization_id
  AND n.search_vector @@ websearch_to_tsquery('english', @query)
ORDER BY rank DESC, n.created_at DESC
LIMIT @row_limit;

-- name: SearchTasks :many
SELECT
  t.id,
  t.case_id,
  t.title,
  t.description,
  t.status,
  t.priority,
  t.is_overdue,
  t.created_at,
  c.case_number,
  c.title AS case_title,
  ts_rank(t.search_vector, websearch_to_tsquery('english', @query)) AS rank
FROM tasks t
JOIN cases c ON c.id = t.case_id AND c.deleted_at IS NULL
WHERE t.deleted_at IS NULL
  AND c.organization_id = @organization_id
  AND t.search_vector @@ websearch_to_tsquery('english', @query)
ORDER BY rank DESC, t.created_at DESC
LIMIT @row_limit;

-- name: SearchDocuments :many
SELECT
  d.id,
  d.case_id,
  d.original_filename,
  d.mime_type,
  d.processing_status,
  d.file_size_bytes,
  d.created_at,
  c.case_number,
  c.title AS case_title,
  ts_rank(d.search_vector, websearch_to_tsquery('english', @query)) AS rank
FROM documents d
JOIN cases c ON c.id = d.case_id AND c.deleted_at IS NULL
WHERE d.deleted_at IS NULL
  AND c.organization_id = @organization_id
  AND d.search_vector @@ websearch_to_tsquery('english', @query)
ORDER BY rank DESC, d.created_at DESC
LIMIT @row_limit;
