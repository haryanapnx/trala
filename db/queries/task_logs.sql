-- name: CreateTaskLog :one
INSERT INTO task_logs (task_id, actor_id, action, field, old_value, new_value)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListTaskLogs :many
SELECT * FROM task_logs
WHERE task_id = $1
ORDER BY created_at DESC;
