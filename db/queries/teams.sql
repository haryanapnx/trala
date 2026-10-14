-- name: CreateTeam :one
INSERT INTO teams (name)
VALUES ($1)
RETURNING *;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, user_id) DO NOTHING;

-- name: IsTeamMember :one
SELECT EXISTS (
    SELECT 1 FROM team_members
    WHERE team_id = $1 AND user_id = $2
);

-- name: ListTeamMemberIDs :many
SELECT user_id FROM team_members
WHERE team_id = $1;
