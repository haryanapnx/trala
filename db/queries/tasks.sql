-- name: CreateTask :one
INSERT INTO tasks (team_id, creator_id, assignee_id, status_id, title, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE deleted_at IS NULL
  AND (creator_id = @user_id OR assignee_id = @user_id)
  AND status_id = COALESCE(sqlc.narg('status_id')::uuid, status_id)
  AND title ILIKE '%' || COALESCE(sqlc.narg('search')::text, '') || '%'
ORDER BY created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: CountTasks :one
SELECT count(*) FROM tasks
WHERE deleted_at IS NULL
  AND (creator_id = @user_id OR assignee_id = @user_id)
  AND status_id = COALESCE(sqlc.narg('status_id')::uuid, status_id)
  AND title ILIKE '%' || COALESCE(sqlc.narg('search')::text, '') || '%';

-- name: UpdateTask :one
UPDATE tasks
SET title = $3, description = $4, status_id = $5, updated_at = now()
WHERE id = $1 AND creator_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: AssignTask :one
UPDATE tasks
SET assignee_id = $3, updated_at = now()
WHERE id = $1 AND creator_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTask :exec
UPDATE tasks
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND creator_id = $2 AND deleted_at IS NULL;
