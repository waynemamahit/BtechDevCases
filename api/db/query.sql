-- name: CreateUser :exec
INSERT INTO users (id, email, password_hash)
VALUES (?, ?, ?);

-- name: CreateWallet :exec
INSERT INTO wallets (user_id, balance_minor)
VALUES (?, ?);

-- name: GetUserByEmail :one
SELECT id, email, password_hash, last_activity_at
FROM users
WHERE email = ?;

-- name: GetLastActivity :one
SELECT last_activity_at
FROM users
WHERE id = ?;

-- name: SetLastActivity :exec
UPDATE users
SET last_activity_at = ?
WHERE id = ?;

-- name: GetWallet :one
SELECT user_id, balance_minor
FROM wallets
WHERE user_id = ?;

-- name: LockWallet :one
SELECT user_id, balance_minor
FROM wallets
WHERE user_id = ?
FOR UPDATE;

-- name: SetBalance :exec
UPDATE wallets
SET balance_minor = ?
WHERE user_id = ?;

-- name: InsertTransfer :exec
INSERT INTO transfers (
  sender_id,
  id,
  recipient_id,
  amount_minor,
  notes,
  created_at
) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTransferBySenderAndID :one
SELECT
  sender_id,
  id,
  recipient_id,
  amount_minor,
  notes,
  created_at
FROM transfers
WHERE sender_id = ? AND id = ?
FOR UPDATE;

-- name: GetEmailByID :one
SELECT email
FROM users
WHERE id = ?;

-- name: ListTransfersForUser :many
SELECT
  t.sender_id,
  t.id,
  t.recipient_id,
  t.amount_minor,
  t.notes,
  t.created_at,
  sender.email AS sender_email,
  recipient.email AS recipient_email
FROM transfers t
JOIN users sender ON sender.id = t.sender_id
JOIN users recipient ON recipient.id = t.recipient_id
WHERE t.sender_id = sqlc.arg(user_id) OR t.recipient_id = sqlc.arg(user_id)
ORDER BY t.created_at ASC, t.id ASC;
