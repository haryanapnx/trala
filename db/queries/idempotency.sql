-- name: GetIdempotencyKey :one
SELECT * FROM idempotency_keys
WHERE key = $1;

-- name: CreateIdempotencyKey :one
INSERT INTO idempotency_keys (key, user_id, request_hash, response_status, response_body)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (key) DO NOTHING
RETURNING *;

-- name: DeleteIdempotencyKey :exec
DELETE FROM idempotency_keys
WHERE key = $1;
