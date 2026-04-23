package voting

import (
	"strings"

	polls "xugeaneeu/pollify/internal/polls/core"
)

func ValidateVotePayload(poll polls.Poll, input SubmitVoteInput) error {
	hasOptions := len(input.OptionIDs) > 0
	hasText := strings.TrimSpace(input.CustomText) != ""

	if hasOptions == hasText {
		return ValidationError{Field: "vote", Message: "must contain either option_ids or custom_text"}
	}

	if hasText {
		if !poll.Settings.AllowCustomAnswer {
			return ValidationError{Field: "custom_text", Message: "is not allowed for this poll"}
		}
		return nil
	}

	if poll.Settings.AllowCustomAnswer {
		return ValidationError{Field: "option_ids", Message: "options are not allowed for text-only polls in the initial version"}
	}

	if poll.Settings.IsMultipleChoice {
		if len(input.OptionIDs) > poll.Settings.MaxChoices {
			return ValidationError{Field: "option_ids", Message: "exceeds max_choices"}
		}
	} else if len(input.OptionIDs) != 1 {
		return ValidationError{Field: "option_ids", Message: "single-choice polls require exactly one option id"}
	}

	allowed := make(map[string]struct{}, len(poll.Options))
	for _, option := range poll.Options {
		allowed[option.ID] = struct{}{}
	}

	seen := make(map[string]struct{}, len(input.OptionIDs))
	for _, optionID := range input.OptionIDs {
		optionID = strings.TrimSpace(optionID)
		if optionID == "" {
			return ValidationError{Field: "option_ids", Message: "must not contain empty values"}
		}
		if _, ok := allowed[optionID]; !ok {
			return ValidationError{Field: "option_ids", Message: "contains an option that does not belong to the poll"}
		}
		if _, duplicate := seen[optionID]; duplicate {
			return ValidationError{Field: "option_ids", Message: "must not contain duplicates"}
		}
		seen[optionID] = struct{}{}
	}

	return nil
}
