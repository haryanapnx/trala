package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/model"
)

func newAuthUsecase() (*AuthUsecase, *fakeStatusRepo) {
	users := newFakeUserRepo()
	statuses := &fakeStatusRepo{}
	uc := NewAuthUsecase(users, statuses, fakeTxManager{}, fakeHasher{}, fakeTokenIssuer{})
	return uc, statuses
}

func TestRegisterCreatesUserAndSeedsDefaultStatuses(t *testing.T) {
	uc, statuses := newAuthUsecase()

	user, err := uc.Register(context.Background(), model.RegisterInput{
		Email: "Alice@Example.com", Password: "password123", Name: "Alice",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "alice@example.com", user.Email, "email should be normalized")
	assert.Equal(t, "hashed:password123", user.PasswordHash)
	assert.NotEqual(t, "password123", user.PasswordHash)

	list, err := statuses.ListByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, []string{"todo", "in_progress", "done"},
		[]string{list[0].Slug, list[1].Slug, list[2].Slug})
	for _, s := range list {
		assert.True(t, s.IsDefault)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	uc, _ := newAuthUsecase()
	_, err := uc.Register(context.Background(), model.RegisterInput{Email: "a@b.com", Password: "password123", Name: "A"})
	require.NoError(t, err)

	_, err = uc.Register(context.Background(), model.RegisterInput{Email: "a@b.com", Password: "password123", Name: "A"})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeConflict, appErr.Code)
}

func TestRegisterValidation(t *testing.T) {
	uc, _ := newAuthUsecase()
	cases := map[string]model.RegisterInput{
		"empty name": {Email: "a@b.com", Password: "password123", Name: "   "},
		"short pw":   {Email: "a@b.com", Password: "short", Name: "A"},
		"bad email":  {Email: "not-an-email", Password: "password123", Name: "A"},
	}

	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := uc.Register(context.Background(), in)
			appErr, ok := domain.AsAppError(err)
			require.True(t, ok)
			assert.Equal(t, domain.CodeValidation, appErr.Code)
		})
	}
}

func TestLoginSuccess(t *testing.T) {
	uc, _ := newAuthUsecase()
	_, err := uc.Register(context.Background(), model.RegisterInput{Email: "a@b.com", Password: "password123", Name: "A"})
	require.NoError(t, err)

	res, err := uc.Login(context.Background(), model.LoginInput{Email: "A@B.com", Password: "password123"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.Equal(t, "Bearer", res.TokenType)
	assert.Greater(t, res.ExpiresIn, int64(0))
}

func TestLoginWrongPasswordIsUnauthorized(t *testing.T) {
	uc, _ := newAuthUsecase()
	_, err := uc.Register(context.Background(), model.RegisterInput{Email: "a@b.com", Password: "password123", Name: "A"})
	require.NoError(t, err)

	_, err = uc.Login(context.Background(), model.LoginInput{Email: "a@b.com", Password: "wrong-password"})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeUnauthorized, appErr.Code)
}

func TestLoginUnknownEmailIsUnauthorized(t *testing.T) {
	uc, _ := newAuthUsecase()

	_, err := uc.Login(context.Background(), model.LoginInput{Email: "nobody@b.com", Password: "password123"})
	appErr, ok := domain.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, domain.CodeUnauthorized, appErr.Code, "unknown email must not leak as not-found")
}
