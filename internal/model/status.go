package model

// CreateStatusInput is the input for adding a status step.
type CreateStatusInput struct {
	UserID   string
	Name     string
	Slug     string
	Position *int32
}

// UpdateStatusInput is the input for renaming or reordering a status step.
type UpdateStatusInput struct {
	UserID   string
	StatusID string
	Name     *string
	Slug     *string
	Position *int32
}
