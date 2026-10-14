-- name: CreateTaskStatus :one
INSERT INTO task_statuses (user_id, name, slug, position, is_default)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListTaskStatuses :many
SELECT * FROM task_statuses
WHERE user_id = $1
ORDER BY position, created_at;

-- name: GetTaskStatusByID :one
SELECT * FROM task_statuses
WHERE id = $1 AND user_id = $2;

-- name: GetTaskStatusBySlug :one
SELECT * FROM task_statuses
WHERE user_id = $1 AND slug = $2;

-- name: GetFirstTaskStatus :one
SELECT * FROM task_statuses
WHERE user_id = $1
ORDER BY position, created_at
LIMIT 1;

-- name: CountTaskStatusesByUser :one
SELECT count(*) FROM task_statuses
WHERE user_id = $1;

-- name: MaxTaskStatusPosition :one
SELECT COALESCE(max(position), -1)::integer FROM task_statuses
WHERE user_id = $1;

-- name: UpdateTaskStatus :one
UPDATE task_statuses
SET name = $3, slug = $4, position = $5, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SetTaskStatusPosition :exec
UPDATE task_statuses
SET position = $3, updated_at = now()
WHERE id = $1 AND user_id = $2;

-- name: DeleteTaskStatus :exec
DELETE FROM task_statuses
WHERE id = $1 AND user_id = $2;

-- name: IsTaskStatusInUse :one
SELECT EXISTS (
    SELECT 1 FROM tasks
    WHERE status_id = $1 AND deleted_at IS NULL
);
