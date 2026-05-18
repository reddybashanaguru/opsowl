-- ── P2-6: Document retry ──────────────────────────────────────────────────────

-- name: GetDocumentForRetry :one
SELECT d.id, d.case_id, d.stored_path, d.original_filename, d.file_type, d.mime_type,
       d.processing_status, d.retry_count
FROM documents d
WHERE d.id = $1 AND d.deleted_at IS NULL;

-- name: IncrementDocumentRetry :one
UPDATE documents
SET processing_status = 'pending',
    retry_count       = retry_count + 1,
    error_message     = NULL,
    updated_at        = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, case_id, stored_path, original_filename, file_type, mime_type, retry_count;

-- name: UpdateDocumentCategory :one
UPDATE documents SET doc_category = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- ── P2-2: Case AI insights ────────────────────────────────────────────────────

-- name: UpsertCaseAIInsights :one
INSERT INTO case_ai_insights
    (case_id, risk_score, risk_factors, entities, anomalies, recommended_actions, model_used, confidence, analysed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
RETURNING *;

-- name: GetLatestCaseAIInsights :one
SELECT * FROM case_ai_insights
WHERE case_id = $1
ORDER BY analysed_at DESC
LIMIT 1;

-- name: ListCasesWithRiskScore :many
SELECT c.id, c.case_number, c.title, c.status, c.priority, c.sla_breached,
       i.risk_score, i.risk_level, i.analysed_at
FROM cases c
LEFT JOIN LATERAL (
    SELECT risk_score, risk_level, analysed_at
    FROM case_ai_insights
    WHERE case_id = c.id
    ORDER BY analysed_at DESC
    LIMIT 1
) i ON true
WHERE c.deleted_at IS NULL
  AND c.organization_id = $1
ORDER BY i.risk_score DESC NULLS LAST, c.created_at DESC
LIMIT @row_limit;

-- ── P2-4: Communication logs ──────────────────────────────────────────────────

-- name: CreateCommLog :one
INSERT INTO communication_logs
    (case_id, channel, direction, contact_name, contact_detail, subject, body, outcome, logged_by, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListCommLogs :many
SELECT cl.*,
       u.full_name  AS logged_by_name,
       u.email      AS logged_by_email,
       e.entities   AS ai_entities,
       e.action_items AS ai_actions
FROM communication_logs cl
JOIN users u ON u.id = cl.logged_by
LEFT JOIN comm_ai_extractions e ON e.comm_log_id = cl.id
WHERE cl.case_id = $1
ORDER BY cl.occurred_at DESC
LIMIT @row_limit;

-- name: UpsertCommAIExtraction :one
INSERT INTO comm_ai_extractions (comm_log_id, entities, action_items, sentiment, model_used)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (comm_log_id) DO UPDATE
    SET entities = EXCLUDED.entities,
        action_items = EXCLUDED.action_items,
        sentiment = EXCLUDED.sentiment,
        model_used = EXCLUDED.model_used
RETURNING *;

-- name: DeleteCommLog :exec
DELETE FROM communication_logs WHERE id = $1 AND logged_by = $2;

-- ── P2-3: SLA policies ────────────────────────────────────────────────────────

-- name: ListSLAPolicies :many
SELECT * FROM sla_policies ORDER BY breach_hours ASC;

-- name: UpsertSLAPolicy :one
INSERT INTO sla_policies (priority, warning_hours, breach_hours, warn_pct)
VALUES ($1, $2, $3, $4)
ON CONFLICT (priority) DO UPDATE
    SET warning_hours = EXCLUDED.warning_hours,
        breach_hours  = EXCLUDED.breach_hours,
        warn_pct      = EXCLUDED.warn_pct,
        updated_at    = NOW()
RETURNING *;

-- ── P2-5: Workload analytics ──────────────────────────────────────────────────

-- name: GetAnalystWorkload :many
SELECT
    u.id,
    u.full_name,
    u.email,
    u.role,
    COUNT(DISTINCT c.id) FILTER (WHERE c.status NOT IN ('closed','draft')) AS active_cases,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status IN ('todo','in_progress'))  AS open_tasks,
    COUNT(DISTINCT t.id) FILTER (WHERE t.is_overdue = TRUE AND t.status NOT IN ('done','cancelled')) AS overdue_tasks,
    COALESCE(AVG(EXTRACT(EPOCH FROM (NOW() - c.created_at))/3600)
        FILTER (WHERE c.status NOT IN ('closed','draft'))::NUMERIC(8,1), 0) AS avg_case_age_hours
FROM users u
LEFT JOIN cases c ON c.assigned_to = u.id AND c.deleted_at IS NULL
LEFT JOIN tasks t ON t.assigned_to = u.id AND t.deleted_at IS NULL
WHERE u.role IN ('analyst','manager')
  AND u.is_active = TRUE
GROUP BY u.id, u.full_name, u.email, u.role
ORDER BY active_cases DESC, overdue_tasks DESC;
