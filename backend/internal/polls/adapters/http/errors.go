package http

import (
	"errors"
	"net/http"

	"xugeaneeu/pollify/internal/pkg/apierror"
	polls "xugeaneeu/pollify/internal/polls/core"
)

func mapError(err error) *apierror.Error {
	if err == nil {
		return nil
	}
	var ve polls.ValidationError
	if errors.As(err, &ve) {
		return apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, ve.Field, ve.Message)
	}
	switch {
	case errors.Is(err, polls.ErrPollNotFound):
		return apierror.New(http.StatusNotFound, apierror.CodeNotFound, "poll not found")
	case errors.Is(err, polls.ErrPollUpdateNotAllowed):
		return apierror.New(http.StatusConflict, apierror.CodePollUpdateNotAllowed, "poll cannot be updated")
	}
	return apierror.New(http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
}
