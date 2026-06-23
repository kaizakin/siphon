-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
    event_id,
    event_type,
    source,
    version,
    timestamp,
    correlation_id,
    metadata,
    payload,
    status,
    error_message
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    'pending',
    ''
)
RETURNING *;

-- name: GetPendingOutboxEvents :many
SELECT *
FROM outbox_events
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1
OFFSET $2;

-- name: MarkOutboxEventProcessed :exec
UPDATE outbox_events
SET
  status = 'done',
  processed_at = NOW(),
WHERE event_id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET
    status = 'failed',
    error_message = $2
WHERE event_id = $1;

-- name: GetOutboxEventByEventID :one
SELECT *
FROM outbox_events
WHERE event_id = $1;