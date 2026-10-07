-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash)
VALUES (?, ?, ?);

-- name: GetUserByEmail :one
SELECT id, email, password_hash
FROM users
WHERE email = ?;
