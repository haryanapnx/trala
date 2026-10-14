package domain

import "context"

// TeamRepository is the persistence port for team membership checks.
type TeamRepository interface {
	IsMember(ctx context.Context, teamID, userID string) (bool, error)
}
