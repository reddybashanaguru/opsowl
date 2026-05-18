-- name: CreateAILog :one
INSERT INTO ai_logs (
    case_id, triggered_by, action_type,
    prompt_text, response_text,
    model_used, provider,
    tokens_input, tokens_output, latency_ms,
    confidence_score, citations
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING *;

-- name: GetAILog :one
SELECT
    l.*,
    u.full_name  AS triggered_by_name,
    u.email      AS triggered_by_email,
    r.full_name  AS reviewed_by_name
FROM ai_logs l
JOIN users u ON u.id = l.triggered_by
LEFT JOIN users r ON r.id = l.reviewed_by
WHERE l.id = $1;

-- name: ListAILogsByCase :many
SELECT
    l.*,
    u.full_name  AS triggered_by_name,
    u.email      AS triggered_by_email,
    r.full_name  AS reviewed_by_name
FROM ai_logs l
JOIN users u ON u.id = l.triggered_by
LEFT JOIN users r ON r.id = l.reviewed_by
WHERE l.case_id = $1
  AND ($2 = '' OR l.human_review_status = $2)
  AND ($3 = '00000000-0000-0000-0000-000000000000'::uuid OR l.id < $3)
ORDER BY l.created_at DESC
LIMIT $4;

-- name: CountAILogsByCase :one
SELECT COUNT(*) FROM ai_logs
WHERE case_id = $1
  AND ($2 = '' OR human_review_status = $2);

-- name: ListPendingAILogs :many
SELECT
    l.*,
    u.full_name  AS triggered_by_name,
    u.email      AS triggered_by_email,
    r.full_name  AS reviewed_by_name
FROM ai_logs l
JOIN users u ON u.id = l.triggered_by
LEFT JOIN users r ON r.id = l.reviewed_by
WHERE l.human_review_status = 'pending'
ORDER BY l.created_at DESC
LIMIT $1;

-- name: ApproveAILog :one
UPDATE ai_logs
SET human_review_status = 'approved',
    reviewed_by         = $2,
    reviewed_at         = NOW()
WHERE id = $1
RETURNING *;

-- name: RejectAILog :one
UPDATE ai_logs
SET human_review_status   = 'rejected',
    reviewed_by           = $2,
    reviewed_at           = NOW(),
    human_edited_response = $3
WHERE id = $1
RETURNING *;

-- name: EditAILog :one
UPDATE ai_logs
SET human_review_status   = 'edited',
    reviewed_by           = $2,
    reviewed_at           = NOW(),
    human_edited_response = $3
WHERE id = $1
RETURNING *;
