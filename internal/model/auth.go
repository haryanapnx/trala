// Package model holds the use-case input and output types (the application
// layer's data contracts). Wiring-level DTOs for JSON live in the transport
// layer instead.
package model

// RegisterInput is the input for registering a user.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// LoginInput is the input for logging in.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult is the issued token returned after a successful login.
type LoginResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
}
