-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE
email = $1
LIMIT 1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    id,
    user_id,
    token,
    expires_at
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1
LIMIT 1;

-- name: DeleteRefreshToken :exec
DELETE from refresh_tokens
WHERE token = $1;