package users

import "errors"

var (
	ErrUserNotFound       = errors.New("users: user not found")
	ErrEmailAlreadyExists = errors.New("users: email already exists")
	ErrInvalidCredentials = errors.New("users: invalid credentials")
	ErrUnauthorized       = errors.New("users: unauthorized")
	ErrRepositoryUnbound  = errors.New("users: repository is required")
	ErrHasherUnbound      = errors.New("users: password hasher is required")
	ErrTokenIssuerUnbound = errors.New("users: token issuer is required")
	ErrClockUnbound       = errors.New("users: clock is required")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}

	return e.Field + ": " + e.Message
}
