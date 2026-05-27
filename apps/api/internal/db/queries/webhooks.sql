-- ── Outbound Webhooks ──────────────────────────────────────────────────────────

-- name: InsertWebhook :one
INSERT INTO outbound_webhooks (name, url, signing_secret, event_types, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetWebhook :one
SELECT * FROM outbound_webhooks
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWebhooks :many
SELECT * FROM outbound_webhooks
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateWebhook :one
UPDATE outbound_webhooks
SET name         = $2,
    url          = $3,
    signing_secret = $4,
    event_types  = $5,
    is_active    = $6,
    updated_at   = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteWebhook :exec
UPDATE outbound_webhooks
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListActiveWebhooksForEvent :many
SELECT * FROM outbound_webhooks
WHERE is_active = true
  AND deleted_at IS NULL
  AND (array_length(event_types, 1) IS NULL OR $1 = ANY(event_types));

-- ── Deliveries ─────────────────────────────────────────────────────────────────

-- name: InsertDelivery :one
INSERT INTO outbound_webhook_deliveries
  (webhook_id, event_type, case_id, payload)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListDeliveriesByWebhook :many
SELECT * FROM outbound_webhook_deliveries
WHERE webhook_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountDeliveriesByWebhook :one
SELECT COUNT(*) FROM outbound_webhook_deliveries WHERE webhook_id = $1;

-- name: GetPendingDeliveries :many
SELECT * FROM outbound_webhook_deliveries
WHERE status = 'pending' AND next_attempt_at <= NOW()
ORDER BY next_attempt_at ASC
LIMIT 100;

-- name: MarkDeliverySuccess :one
UPDATE outbound_webhook_deliveries
SET status        = 'success',
    last_status   = $2,
    delivered_at  = NOW(),
    updated_at    = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkDeliveryFailed :one
UPDATE outbound_webhook_deliveries
SET status          = CASE WHEN attempt_count + 1 >= $4 THEN 'exhausted' ELSE 'pending' END,
    attempt_count   = attempt_count + 1,
    next_attempt_at = $3,
    last_status     = $2,
    last_error      = $5,
    updated_at      = NOW()
WHERE id = $1
RETURNING *;

-- name: RequeueDelivery :one
UPDATE outbound_webhook_deliveries
SET status          = 'pending',
    attempt_count   = 0,
    next_attempt_at = NOW(),
    last_status     = NULL,
    last_error      = NULL,
    delivered_at    = NULL,
    updated_at      = NOW()
WHERE id = $1
RETURNING *;
