package domain

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable, machine-readable error identifier exposed to clients.
type ErrorCode string

const (
	CodeValidation        ErrorCode = "VALIDATION_ERROR"
	CodeNotFound          ErrorCode = "NOT_FOUND"
	CodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	CodeForbidden         ErrorCode = "FORBIDDEN"
	CodeConflict          ErrorCode = "CONFLICT"
	CodeMethodNotAllowed  ErrorCode = "METHOD_NOT_ALLOWED"
	CodeIdempotencyReused ErrorCode = "IDEMPOTENCY_KEY_REUSED"
	CodeInProgress        ErrorCode = "REQUEST_IN_PROGRESS"
	CodeInternal          ErrorCode = "INTERNAL_ERROR"
)

// AppError is a domain error. It carries no transport concerns (no HTTP status):
// the transport layer maps the Code to a status and response body.
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// New builds an AppError with the given code and message.
func New(code ErrorCode, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// WithMessage returns a copy with a different, client-safe message.
func (e *AppError) WithMessage(message string) *AppError {
	c := *e
	c.Message = message
	return &c
}

// WithCause returns a copy carrying an underlying error for server-side logs.
func (e *AppError) WithCause(err error) *AppError {
	c := *e
	c.Err = err
	return &c
}

// Sentinel errors. Use WithMessage/WithCause to add context without mutating them.
var (
	ErrValidation        = &AppError{Code: CodeValidation, Message: "validation failed"}
	ErrNotFound          = &AppError{Code: CodeNotFound, Message: "resource not found"}
	ErrUnauthorized      = &AppError{Code: CodeUnauthorized, Message: "unauthorized"}
	ErrForbidden         = &AppError{Code: CodeForbidden, Message: "forbidden"}
	ErrConflict          = &AppError{Code: CodeConflict, Message: "conflict"}
	ErrIdempotencyReused = &AppError{Code: CodeIdempotencyReused, Message: "idempotency key reused with a different request"}
	ErrInProgress        = &AppError{Code: CodeInProgress, Message: "a request with this key is still in progress"}
	ErrInternal          = &AppError{Code: CodeInternal, Message: "internal server error"}
)

// AsAppError extracts an *AppError from err, reporting whether one was found.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
