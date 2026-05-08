package http

import (
	"errors"
	"net/http"

	moderation "xugeaneeu/pollify/internal/moderation/core"
	"xugeaneeu/pollify/internal/pkg/apierror"
	polls "xugeaneeu/pollify/internal/polls/core"
)

func mapError(err error) *apierror.Error {
	if err == nil {
		return nil
	}
	var ve moderation.ValidationError
	if errors.As(err, &ve) {
		return apierror.WithField(http.StatusUnprocessableEntity, apierror.CodeValidationError, ve.Field, ve.Message)
	}
	switch {
	case errors.Is(err, moderation.ErrReportNotFound):
		return apierror.New(http.StatusNotFound, apierror.CodeNotFound, "report not found")
	case errors.Is(err, polls.ErrPollNotFound):
		return apierror.New(http.StatusNotFound, apierror.CodeNotFound, "poll not found")
	case errors.Is(err, moderation.ErrReportAlreadyExists):
		return apierror.New(http.StatusConflict, apierror.CodeReportAlreadyExists, "active report already exists")
	case errors.Is(err, moderation.ErrReportAlreadyResolved):
		return apierror.New(http.StatusConflict, apierror.CodeReportAlreadyResolved, "report is already resolved")
	case errors.Is(err, moderation.ErrReportReviewAlreadyExists):
		return apierror.New(http.StatusConflict, apierror.CodeReportReviewAlreadyExists, "review already submitted")
	case errors.Is(err, moderation.ErrSelfReviewForbidden):
		return apierror.New(http.StatusForbidden, apierror.CodeReportReviewSelfForbidden, "admin cannot review own report")
	case errors.Is(err, moderation.ErrAdminRequired):
		return apierror.New(http.StatusForbidden, apierror.CodeForbidden, "admin role required")
	}
	return apierror.New(http.StatusInternalServerError, apierror.CodeInternalError, "internal error")
}
