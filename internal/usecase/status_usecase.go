package usecase

import (
	"context"
	"strings"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/model"
)

type StatusUsecase struct {
	statuses domain.TaskStatusRepository
	tx       domain.TxManager
}

func NewStatusUsecase(statuses domain.TaskStatusRepository, tx domain.TxManager) *StatusUsecase {
	return &StatusUsecase{statuses: statuses, tx: tx}
}

func (u *StatusUsecase) List(ctx context.Context, userID string) ([]domain.TaskStatus, error) {
	return u.statuses.ListByUser(ctx, userID)
}

// Create adds a step. Without a position it appends; with a position it is
// inserted at that index and the ordering is normalized to 0..n-1.
func (u *StatusUsecase) Create(ctx context.Context, in model.CreateStatusInput) (*domain.TaskStatus, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, domain.ErrValidation.WithMessage("name is required")
	}

	slug, err := normalizeSlug(in.Slug, name)
	if err != nil {
		return nil, err
	}

	var created *domain.TaskStatus
	err = u.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		max, err := u.statuses.MaxPosition(txCtx, in.UserID)
		if err != nil {
			return err
		}

		status, err := u.statuses.Create(txCtx, &domain.TaskStatus{
			UserID:   in.UserID,
			Name:     name,
			Slug:     slug,
			Position: max + 1,
		})
		if err != nil {
			return mapSlugConflict(err)
		}

		if in.Position != nil {
			if err := u.moveToIndex(txCtx, in.UserID, status.ID, *in.Position); err != nil {
				return err
			}
		}
		created = status
		return nil
	})
	if err != nil {
		return nil, err
	}
	if created != nil {
		return u.statuses.GetByID(ctx, in.UserID, created.ID)
	}
	return created, nil
}

// Update changes name, slug and/or position. A position is treated as the target
// index, so the item is moved there and the ordering is renumbered.
func (u *StatusUsecase) Update(ctx context.Context, in model.UpdateStatusInput) (*domain.TaskStatus, error) {
	var updated *domain.TaskStatus
	err := u.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := u.statuses.GetByID(txCtx, in.UserID, in.StatusID)
		if err != nil {
			return err
		}

		if in.Name != nil {
			name := strings.TrimSpace(*in.Name)
			if name == "" {
				return domain.ErrValidation.WithMessage("name cannot be empty")
			}
			existing.Name = name
		}
		if in.Slug != nil {
			slug := slugify(*in.Slug)
			if slug == "" {
				return domain.ErrValidation.WithMessage("slug cannot be empty")
			}
			existing.Slug = slug
		}

		status, err := u.statuses.Update(txCtx, existing)
		if err != nil {
			return mapSlugConflict(err)
		}

		if in.Position != nil {
			if err := u.moveToIndex(txCtx, in.UserID, existing.ID, *in.Position); err != nil {
				return err
			}
		}
		updated = status
		return nil
	})
	if err != nil {
		return nil, err
	}
	if updated != nil {
		return u.statuses.GetByID(ctx, in.UserID, updated.ID)
	}
	return updated, nil
}

// Delete removes a step, refusing when it is still referenced by a task or when
// it is the user's last remaining status.
func (u *StatusUsecase) Delete(ctx context.Context, userID, statusID string) error {
	return u.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if _, err := u.statuses.GetByID(txCtx, userID, statusID); err != nil {
			return err
		}

		inUse, err := u.statuses.IsInUse(txCtx, statusID)
		if err != nil {
			return err
		}
		if inUse {
			return domain.ErrConflict.WithMessage("status is used by one or more tasks")
		}

		count, err := u.statuses.CountByUser(txCtx, userID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return domain.ErrConflict.WithMessage("cannot delete the last status")
		}

		if err := u.statuses.Delete(txCtx, userID, statusID); err != nil {
			return err
		}
		return u.normalize(txCtx, userID)
	})
}

// moveToIndex places statusID at the given index in the user's ordered list and
// renumbers every status to 0..n-1.
func (u *StatusUsecase) moveToIndex(ctx context.Context, userID, statusID string, index int32) error {
	list, err := u.statuses.ListByUser(ctx, userID)
	if err != nil {
		return err
	}

	moving := -1
	others := make([]domain.TaskStatus, 0, len(list))
	for i := range list {
		if list[i].ID == statusID {
			moving = i
			continue
		}
		others = append(others, list[i])
	}
	if moving == -1 {
		return domain.ErrNotFound
	}

	if index < 0 {
		index = 0
	}
	if int(index) > len(others) {
		index = int32(len(others))
	}

	reordered := make([]domain.TaskStatus, 0, len(list))
	reordered = append(reordered, others[:index]...)
	reordered = append(reordered, list[moving])
	reordered = append(reordered, others[index:]...)

	for i := range reordered {
		if reordered[i].Position == int32(i) {
			continue
		}
		if err := u.statuses.SetPosition(ctx, userID, reordered[i].ID, int32(i)); err != nil {
			return err
		}
	}
	return nil
}

// normalize renumbers the user's statuses to 0..n-1 in their current order.
func (u *StatusUsecase) normalize(ctx context.Context, userID string) error {
	list, err := u.statuses.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	for i, s := range list {
		if s.Position == int32(i) {
			continue
		}
		if err := u.statuses.SetPosition(ctx, userID, s.ID, int32(i)); err != nil {
			return err
		}
	}
	return nil
}

func mapSlugConflict(err error) error {
	if appErr, ok := domain.AsAppError(err); ok && appErr.Code == domain.CodeConflict {
		return appErr.WithMessage("status slug already exists")
	}
	return err
}

func normalizeSlug(slug, name string) (string, error) {
	if strings.TrimSpace(slug) == "" {
		slug = slugify(name)
	} else {
		slug = slugify(slug)
	}
	if slug == "" {
		return "", domain.ErrValidation.WithMessage("slug could not be derived from name")
	}
	return slug, nil
}

// slugify lowercases and replaces any run of non-alphanumeric characters with a
// single underscore.
func slugify(s string) string {
	var b strings.Builder
	prevUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
