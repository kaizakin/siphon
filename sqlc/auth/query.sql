-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash,
    role
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: UpdateUserRole :one
UPDATE users
SET role = $2, updated_at = NOW()
WHERE email = $1
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE
email = $1
LIMIT 1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    user_id,
    token,
    expires_at
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1
LIMIT 1;

-- name: DeleteRefreshToken :exec
DELETE from refresh_tokens
WHERE token = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;