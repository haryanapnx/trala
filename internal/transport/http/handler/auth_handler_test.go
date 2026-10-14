package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/model"
	"github.com/haryanapnx/trala/internal/transport/http/dto"
)

type stubAuthUsecase struct {
	registerFn func(context.Context, model.RegisterInput) (*domain.User, error)
	loginFn    func(context.Context, model.LoginInput) (*model.LoginResult, error)
}

func (s stubAuthUsecase) Register(ctx context.Context, in model.RegisterInput) (*domain.User, error) {
	return s.registerFn(ctx, in)
}

func (s stubAuthUsecase) Login(ctx context.Context, in model.LoginInput) (*model.LoginResult, error) {
	return s.loginFn(ctx, in)
}

var _ AuthUsecase = stubAuthUsecase{}

func TestAuthHandlerRegisterReturns201WithoutPasswordHash(t *testing.T) {
	h := NewAuthHandler(stubAuthUsecase{
		registerFn: func(_ context.Context, in model.RegisterInput) (*domain.User, error) {
			return &domain.User{
				ID: "u1", Email: in.Email, Name: in.Name,
				PasswordHash: "secret-hash", CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register",
		bytes.NewBufferString(`{"email":"a@b.com","password":"password123","name":"A"}`))

	h.Register(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	assert.NotContains(t, rr.Body.String(), "secret-hash")
	assert.NotContains(t, rr.Body.String(), "password_hash")

	var resp dto.UserResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "u1", resp.ID)
	assert.Equal(t, "a@b.com", resp.Email)
}

func TestAuthHandlerRegisterMapsConflict(t *testing.T) {
	h := NewAuthHandler(stubAuthUsecase{
		registerFn: func(context.Context, model.RegisterInput) (*domain.User, error) {
			return nil, domain.ErrConflict.WithMessage("email already registered")
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register",
		bytes.NewBufferString(`{"email":"a@b.com","password":"password123","name":"A"}`))

	h.Register(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "CONFLICT")
}

func TestAuthHandlerLoginReturnsToken(t *testing.T) {
	h := NewAuthHandler(stubAuthUsecase{
		loginFn: func(context.Context, model.LoginInput) (*model.LoginResult, error) {
			return &model.LoginResult{AccessToken: "tok", TokenType: "Bearer", ExpiresIn: 3600}, nil
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
		bytes.NewBufferString(`{"email":"a@b.com","password":"password123"}`))

	h.Login(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp dto.LoginResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "tok", resp.AccessToken)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
}

func TestAuthHandlerLoginMapsUnauthorized(t *testing.T) {
	h := NewAuthHandler(stubAuthUsecase{
		loginFn: func(context.Context, model.LoginInput) (*model.LoginResult, error) {
			return nil, domain.ErrUnauthorized.WithMessage("invalid email or password")
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
		bytes.NewBufferString(`{"email":"a@b.com","password":"x"}`))

	h.Login(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "UNAUTHORIZED")
}

func TestAuthHandlerRejectsInvalidJSON(t *testing.T) {
	h := NewAuthHandler(stubAuthUsecase{})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{not json`))

	h.Register(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "VALIDATION_ERROR")
}
