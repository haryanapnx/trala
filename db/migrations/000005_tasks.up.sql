CREATE TABLE tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID NOT NULL REFERENCES teams(id),
    creator_id  UUID NOT NULL REFERENCES users(id),
    assignee_id UUID REFERENCES users(id),
    status_id   UUID NOT NULL REFERENCES task_statuses(id) ON DELETE RESTRICT,
    title       TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- Soft-deleted rows are excluded from every read path, so indexes are partial.
CREATE INDEX tasks_creator_created_idx  ON tasks (creator_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX tasks_assignee_created_idx ON tasks (assignee_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX tasks_team_status_idx      ON tasks (team_id, status_id)          WHERE deleted_at IS NULL;

-- Trigram index makes the case-insensitive title search efficient.
CREATE INDEX tasks_title_trgm_idx ON tasks USING gin (title gin_trgm_ops) WHERE deleted_at IS NULL;
