package usecase

import (
	"context"
	"net/mail"
	"strings"

	"github.com/haryfun/task-management-service/internal/domain"
	"github.com/haryfun/task-management-service/internal/model"
)

type AuthUsecase struct {
	users    domain.UserRepository
	statuses domain.TaskStatusRepository
	tx       domain.TxManager
	hasher   PasswordHasher
	tokens   TokenIssuer
}

func NewAuthUsecase(
	users domain.UserRepository,
	statuses domain.TaskStatusRepository,
	tx domain.TxManager,
	hasher PasswordHasher,
	tokens TokenIssuer,
) *AuthUsecase {
	return &AuthUsecase{users: users, statuses: statuses, tx: tx, hasher: hasher, tokens: tokens}
}

// Register creates a user and seeds their three default statuses inside a
// single transaction.
func (u *AuthUsecase) Register(ctx context.Context, in model.RegisterInput) (*domain.User, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)

	if err := validateRegistration(in); err != nil {
		return nil, err
	}

	hash, err := u.hasher.Hash(in.Password)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}

	var created *domain.User
	err = u.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		user, err := u.users.Create(txCtx, &domain.User{
			Email:        in.Email,
			PasswordHash: hash,
			Name:         in.Name,
		})
		if err != nil {
			return err
		}

		for _, status := range defaultStatuses(user.ID) {
			if _, err := u.statuses.Create(txCtx, status); err != nil {
				return err
			}
		}

		created = user
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

// Login verifies credentials and issues a token. Unknown email and wrong
// password both return the same unauthorized error to avoid user enumeration.
func (u *AuthUsecase) Login(ctx context.Context, in model.LoginInput) (*model.LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	user, err := u.users.GetByEmail(ctx, email)
	if err != nil {
		if appErr, ok := domain.AsAppError(err); ok && appErr.Code == domain.CodeNotFound {
			return nil, invalidCredentials()
		}
		return nil, err
	}

	if err := u.hasher.Compare(user.PasswordHash, in.Password); err != nil {
		return nil, invalidCredentials()
	}

	token, expiresIn, err := u.tokens.Issue(user.ID)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}

	return &model.LoginResult{AccessToken: token, TokenType: "Bearer", ExpiresIn: expiresIn}, nil
}

func invalidCredentials() error {
	return domain.ErrUnauthorized.WithMessage("invalid email or password")
}

func validateRegistration(in model.RegisterInput) error {
	if in.Name == "" {
		return domain.ErrValidation.WithMessage("name is required")
	}
	if len(in.Password) < 8 {
		return domain.ErrValidation.WithMessage("password must be at least 8 characters")
	}
	addr, err := mail.ParseAddress(in.Email)
	if err != nil || addr.Address != in.Email {
		return domain.ErrValidation.WithMessage("invalid email address")
	}
	return nil
}

func defaultStatuses(userID string) []*domain.TaskStatus {
	return []*domain.TaskStatus{
		{UserID: userID, Name: "Todo", Slug: "todo", Position: 0, IsDefault: true},
		{UserID: userID, Name: "In Progress", Slug: "in_progress", Position: 1, IsDefault: true},
		{UserID: userID, Name: "Done", Slug: "done", Position: 2, IsDefault: true},
	}
}
