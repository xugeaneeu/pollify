package voting

import "errors"

var (
	ErrPollAlreadyVoted = errors.New("voting: user has already participated in this poll")
	ErrPollNotActive    = errors.New("voting: poll is not active")
	ErrPollHidden       = errors.New("voting: poll is hidden")
	ErrRepositoryNil    = errors.New("voting: repository is required")
	ErrPollReaderNil    = errors.New("voting: poll reader is required")
	ErrClockNil         = errors.New("voting: clock is required")
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
