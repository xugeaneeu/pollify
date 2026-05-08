package http

import (
	"time"

	polls "xugeaneeu/pollify/internal/polls/core"
)

type pollOptionDTO struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type pollSettingsDTO struct {
	IsAnonymous       bool      `json:"is_anonymous"`
	IsMultipleChoice  bool      `json:"is_multiple_choice"`
	MaxChoices        *int      `json:"max_choices"`
	AllowCustomAnswer bool      `json:"allow_custom_answer"`
	StartAt           time.Time `json:"start_at"`
	EndAt             time.Time `json:"end_at"`
}

type participationSummaryDTO struct {
	ParticipantsCount int  `json:"participants_count"`
	HasVoted          bool `json:"has_voted"`
}

type pollSummaryDTO struct {
	ID                   string                  `json:"id"`
	Title                string                  `json:"title"`
	Description          *string                 `json:"description"`
	Question             string                  `json:"question"`
	Status               string                  `json:"status"`
	IsAnonymous          bool                    `json:"is_anonymous"`
	IsMultipleChoice     bool                    `json:"is_multiple_choice"`
	MaxChoices           *int                    `json:"max_choices"`
	AllowCustomAnswer    bool                    `json:"allow_custom_answer"`
	StartAt              time.Time               `json:"start_at"`
	EndAt                time.Time               `json:"end_at"`
	IsHidden             bool                    `json:"is_hidden"`
	CreatedBy            string                  `json:"created_by"`
	CreatedAt            time.Time               `json:"created_at"`
	ParticipationSummary participationSummaryDTO `json:"participation_summary"`
}

type pollDetailsDTO struct {
	pollSummaryDTO
	Options  []pollOptionDTO `json:"options"`
	Settings pollSettingsDTO `json:"settings"`
}

type paginationMetaDTO struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type pollsListResponseDTO struct {
	Items      []pollSummaryDTO  `json:"items"`
	Pagination paginationMetaDTO `json:"pagination"`
}

type createPollOptionDTO struct {
	Text string `json:"text"`
}

type createPollSettingsDTO struct {
	IsAnonymous       bool      `json:"is_anonymous"`
	IsMultipleChoice  bool      `json:"is_multiple_choice"`
	MaxChoices        *int      `json:"max_choices,omitempty"`
	AllowCustomAnswer bool      `json:"allow_custom_answer"`
	StartAt           time.Time `json:"start_at"`
	EndAt             time.Time `json:"end_at"`
}

type createPollRequest struct {
	Title       string                `json:"title"`
	Description *string               `json:"description,omitempty"`
	Question    string                `json:"question"`
	Options     []createPollOptionDTO `json:"options,omitempty"`
	Settings    createPollSettingsDTO `json:"settings"`
}

type updatePollRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndAt       *time.Time `json:"end_at,omitempty"`
}

func toSettingsDTO(s polls.PollSettings) pollSettingsDTO {
	out := pollSettingsDTO{
		IsAnonymous:       s.IsAnonymous,
		IsMultipleChoice:  s.IsMultipleChoice,
		AllowCustomAnswer: s.AllowCustomAnswer,
		StartAt:           s.StartAt.UTC(),
		EndAt:             s.EndAt.UTC(),
	}
	if s.IsMultipleChoice && s.MaxChoices > 0 {
		mc := s.MaxChoices
		out.MaxChoices = &mc
	}
	return out
}

func toSummaryDTO(p polls.Poll, status polls.Status, summary participationSummaryDTO) pollSummaryDTO {
	dto := pollSummaryDTO{
		ID:                   p.ID,
		Title:                p.Title,
		Question:             p.Question,
		Status:               string(status),
		IsAnonymous:          p.Settings.IsAnonymous,
		IsMultipleChoice:     p.Settings.IsMultipleChoice,
		AllowCustomAnswer:    p.Settings.AllowCustomAnswer,
		StartAt:              p.Settings.StartAt.UTC(),
		EndAt:                p.Settings.EndAt.UTC(),
		IsHidden:             p.IsHidden,
		CreatedBy:            p.CreatedBy,
		CreatedAt:            p.CreatedAt.UTC(),
		ParticipationSummary: summary,
	}
	if p.Description != "" {
		desc := p.Description
		dto.Description = &desc
	}
	if p.Settings.IsMultipleChoice && p.Settings.MaxChoices > 0 {
		mc := p.Settings.MaxChoices
		dto.MaxChoices = &mc
	}
	return dto
}

func toDetailsDTO(p polls.Poll, status polls.Status, summary participationSummaryDTO) pollDetailsDTO {
	options := make([]pollOptionDTO, 0, len(p.Options))
	for _, opt := range p.Options {
		options = append(options, pollOptionDTO{ID: opt.ID, Text: opt.Text})
	}
	return pollDetailsDTO{
		pollSummaryDTO: toSummaryDTO(p, status, summary),
		Options:        options,
		Settings:       toSettingsDTO(p.Settings),
	}
}
