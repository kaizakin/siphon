-- +goose Up
ALTER TABLE outbox_events
ADD COLUMN recipient TEXT;

-- +goose Down
ALTER TABLE outbox_events
DROP COLUMN recipient;
