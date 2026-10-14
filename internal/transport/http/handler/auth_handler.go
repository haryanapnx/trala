package handler

import (
	"context"
	"net/http"

	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/model"
	"github.com/haryanapnx/trala/internal/transport/http/dto"
	"github.com/haryanapnx/trala/internal/transport/http/response"
)

// AuthUsecase is the slice of the auth use case the handler needs.
type AuthUsecase interface {
	Register(ctx context.Context, in model.RegisterInput) (*domain.User, error)
	Login(ctx context.Context, in model.LoginInput) (*model.LoginResult, error)
}

type AuthHandler struct {
	auth AuthUsecase
}

func NewAuthHandler(auth AuthUsecase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register handles POST /v1/auth/register.
//
// @Summary      Register a user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Registration payload"
// @Success      201 {object} dto.UserResponse
// @Failure      400 {object} response.ErrorBody
// @Failure      409 {object} response.ErrorBody
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, domain.ErrValidation.WithMessage("invalid request body"))
		return
	}

	user, err := h.auth.Register(r.Context(), model.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, dto.ToUserResponse(user))
}

// Login handles POST /v1/auth/login.
//
// @Summary      Log in
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Credentials"
// @Success      200 {object} dto.LoginResponse
// @Failure      401 {object} response.ErrorBody
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, r, domain.ErrValidation.WithMessage("invalid request body"))
		return
	}

	result, err := h.auth.Login(r.Context(), model.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
	})
}
