-- +goose Up
CREATE INDEX idx_outbox_events_event_id
ON outbox_events(event_id);

-- +goose Down
DROP INDEX IF EXISTS idx_outbox_events_event_id;
