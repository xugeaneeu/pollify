package polls

import "errors"

var (
	ErrPollNotFound         = errors.New("polls: poll not found")
	ErrPollUpdateNotAllowed = errors.New("polls: poll update not allowed")
	ErrRepositoryUnbound    = errors.New("polls: repository is required")
	ErrClockUnbound         = errors.New("polls: clock is required")
	ErrParticipantsUnbound  = errors.New("polls: participation repository is required")
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
