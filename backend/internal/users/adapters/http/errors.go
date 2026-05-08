package http

import (
	"errors"
	"net/http"

	"xugeaneeu/pollify/internal/pkg/apierror"
	users "xugeaneeu/pollify/internal/users/core"
)

func mapError(err error) *apierror.Error {
	if err == nil {
		return nil
	}
	var ve users.ValidationError
	if errors.As(err, &ve) {
		return apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, ve.Field, ve.Message)
	}
	switch {
	case errors.Is(err, users.ErrEmailAlreadyExists):
		return apierror.New(http.StatusConflict, apierror.CodeConflict, "email already exists")
	case errors.Is(err, users.ErrInvalidCredentials):
		return apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "invalid credentials")
	case errors.Is(err, users.ErrUnauthorized):
		return apierror.New(http.StatusUnauthorized, apierror.CodeUnauthorized, "authentication required")
	case errors.Is(err, users.ErrUserNotFound):
		return apierror.New(http.StatusNotFound, apierror.CodeNotFound, "user not found")
	}
	return apierror.New(http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
}
