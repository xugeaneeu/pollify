package polls

import (
	"strconv"
	"strings"
	"time"
)

func ValidateCreateInput(input CreatePollInput) error {
	if strings.TrimSpace(input.CreatedBy) == "" {
		return ValidationError{Field: "created_by", Message: "must not be empty"}
	}
	if strings.TrimSpace(input.Title) == "" {
		return ValidationError{Field: "title", Message: "must not be empty"}
	}
	if strings.TrimSpace(input.Question) == "" {
		return ValidationError{Field: "question", Message: "must not be empty"}
	}
	if !input.Settings.EndAt.After(input.Settings.StartAt) {
		return ValidationError{Field: "end_at", Message: "must be later than start_at"}
	}
	if input.Settings.IsMultipleChoice {
		if input.Settings.AllowCustomAnswer {
			return ValidationError{Field: "settings", Message: "multiple choice and custom answers cannot be enabled together"}
		}
		if input.Settings.MaxChoices < 2 {
			return ValidationError{Field: "max_choices", Message: "must be at least 2 for multiple choice polls"}
		}
	}

	if input.Settings.AllowCustomAnswer {
		if len(input.Options) > 0 {
			return ValidationError{Field: "options", Message: "mixed mode polls are not supported in the initial version"}
		}
	} else if len(input.Options) == 0 {
		return ValidationError{Field: "options", Message: "must contain at least one option for option-based polls"}
	}

	for index, option := range input.Options {
		if strings.TrimSpace(option.Text) == "" {
			return ValidationError{Field: "options[" + strconv.Itoa(index) + "].text", Message: "must not be empty"}
		}
	}

	return nil
}

func ValidateUpdateInput(existing Poll, input UpdatePollInput, now time.Time, hasParticipants bool) error {
	if input.Title == nil && input.Description == nil && input.StartAt == nil && input.EndAt == nil {
		return ValidationError{Field: "body", Message: "must contain at least one mutable field"}
	}
	if hasParticipants || existing.HasStarted(now) {
		return ErrPollUpdateNotAllowed
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		return ValidationError{Field: "title", Message: "must not be empty"}
	}

	startAt := existing.Settings.StartAt
	if input.StartAt != nil {
		startAt = *input.StartAt
	}

	endAt := existing.Settings.EndAt
	if input.EndAt != nil {
		endAt = *input.EndAt
	}

	if !endAt.After(startAt) {
		return ValidationError{Field: "end_at", Message: "must be later than start_at"}
	}

	return nil
}
