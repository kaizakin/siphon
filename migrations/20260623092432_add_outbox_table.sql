-- +goose Up
CREATE TABLE outbox_events (
  event_id UUID PRIMARY KEY,
  event_type TEXT NOT NULL,
  source TEXT NOT NULL,
  version TEXT NOT NULL,
  timestamp TIMESTAMPTZ NOT NULL,
  correlation_id UUID NOT NULL,
  metadata JSONB NOT NULL,
  payload JSONB NOT NULL,
  
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at TIMESTAMPTZ,
  error_message TEXT NOT NULL
);

-- index makes the query faster
CREATE INDEX idx_outbox_status_created_at 
ON outbox_events(status, created_at);
  
-- +goose Down
DROP INDEX IF EXISTS outbox_events;
DROP TABLE IF EXISTS idx_outbox_status_created_at;