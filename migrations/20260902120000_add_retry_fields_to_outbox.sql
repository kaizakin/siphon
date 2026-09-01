-- +goose Up
ALTER TABLE outbox_events
ADD COLUMN retry_count INT NOT NULL DEFAULT 0,
ADD COLUMN next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
ADD COLUMN max_retries INT NOT NULL DEFAULT 5;

CREATE INDEX idx_outbox_retry
ON outbox_events(status, next_retry_at)
WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_outbox_retry;
ALTER TABLE outbox_events
DROP COLUMN IF EXISTS max_retries,
DROP COLUMN IF EXISTS next_retry_at,
DROP COLUMN IF EXISTS retry_count;
