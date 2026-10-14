package postgres

import (
	"context"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/repository/postgres/sqlc"
)

type TeamRepository struct {
	db *DB
}

func NewTeamRepository(db *DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) IsMember(ctx context.Context, teamID, userID string) (bool, error) {
	tid, err := parseUUID(teamID)
	if err != nil {
		return false, domain.ErrValidation.WithMessage("team_id is invalid")
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return false, err
	}

	return r.db.Queries(ctx).IsTeamMember(ctx, sqlc.IsTeamMemberParams{TeamID: tid, UserID: uid})
}

var _ domain.TeamRepository = (*TeamRepository)(nil)
