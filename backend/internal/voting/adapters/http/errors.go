package http

import (
	"errors"
	"net/http"

	"xugeaneeu/pollify/internal/pkg/apierror"
	polls "xugeaneeu/pollify/internal/polls/core"
	voting "xugeaneeu/pollify/internal/voting/core"
)

func mapError(err error) *apierror.Error {
	if err == nil {
		return nil
	}
	var ve voting.ValidationError
	if errors.As(err, &ve) {
		return apierror.WithField(http.StatusUnprocessableEntity, apierror.CodePollVotePayloadInvalid, ve.Field, ve.Message)
	}
	switch {
	case errors.Is(err, voting.ErrPollAlreadyVoted):
		return apierror.New(http.StatusConflict, apierror.CodePollAlreadyVoted, "user has already participated in this poll")
	case errors.Is(err, voting.ErrPollHidden):
		return apierror.New(http.StatusConflict, apierror.CodePollHidden, "poll is hidden")
	case errors.Is(err, voting.ErrPollNotActive):
		return apierror.New(http.StatusConflict, apierror.CodePollNotActive, "poll is not active")
	case errors.Is(err, polls.ErrPollNotFound):
		return apierror.New(http.StatusNotFound, apierror.CodeNotFound, "poll not found")
	}
	return apierror.New(http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
}
