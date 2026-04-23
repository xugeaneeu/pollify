package moderation

import "errors"

var (
	ErrReportNotFound            = errors.New("moderation: report not found")
	ErrReportAlreadyExists       = errors.New("moderation: active report already exists")
	ErrReportAlreadyResolved     = errors.New("moderation: report already resolved")
	ErrReportReviewAlreadyExists = errors.New("moderation: report review already exists")
	ErrSelfReviewForbidden       = errors.New("moderation: self review is forbidden")
	ErrAdminRequired             = errors.New("moderation: admin role is required")
	ErrRepositoryUnbound         = errors.New("moderation: repository is required")
	ErrPollReaderUnbound         = errors.New("moderation: poll reader is required")
	ErrPollWriterUnbound         = errors.New("moderation: poll visibility writer is required")
	ErrClockUnbound              = errors.New("moderation: clock is required")
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
