-- +goose Up
ALTER TABLE users
ADD COLUMN role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin'));

CREATE INDEX idx_users_role ON users(role);

-- +goose Down
DROP INDEX IF EXISTS idx_users_role;

ALTER TABLE users
DROP COLUMN IF EXISTS role;
