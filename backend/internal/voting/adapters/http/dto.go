package http

import (
	voting "xugeaneeu/pollify/internal/voting/core"
)

type createVoteRequest struct {
	OptionIDs  []string `json:"option_ids,omitempty"`
	CustomText string   `json:"custom_text,omitempty"`
}

type voteAcceptedResponse struct {
	PollID                string `json:"poll_id"`
	ParticipationRecorded bool   `json:"participation_recorded"`
}

type optionResultDTO struct {
	OptionID   string  `json:"option_id"`
	Label      string  `json:"label"`
	VotesCount int     `json:"votes_count"`
	Percentage float64 `json:"percentage"`
}

type textAnswerDTO struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type voterDetailDTO struct {
	UserID            string   `json:"user_id"`
	DisplayName       *string  `json:"display_name"`
	SelectedOptionIDs []string `json:"selected_option_ids"`
	CustomText        *string  `json:"custom_text"`
}

type pollResultsResponse struct {
	PollID            string            `json:"poll_id"`
	Status            string            `json:"status"`
	ParticipantsCount int               `json:"participants_count"`
	TotalVotesCount   int               `json:"total_votes_count"`
	Options           []optionResultDTO `json:"options"`
	CustomAnswers     []textAnswerDTO   `json:"custom_answers"`
	VoterDetails      []voterDetailDTO  `json:"voter_details,omitempty"`
}

func toResultsDTO(r voting.PollResults) pollResultsResponse {
	options := make([]optionResultDTO, 0, len(r.Options))
	for _, opt := range r.Options {
		options = append(options, optionResultDTO{
			OptionID:   opt.OptionID,
			Label:      opt.Label,
			VotesCount: opt.VotesCount,
			Percentage: opt.Percentage,
		})
	}
	answers := make([]textAnswerDTO, 0, len(r.CustomAnswers))
	for _, a := range r.CustomAnswers {
		answers = append(answers, textAnswerDTO{Value: a.Value, Count: a.Count})
	}
	out := pollResultsResponse{
		PollID:            r.PollID,
		Status:            string(r.Status),
		ParticipantsCount: r.ParticipantsCount,
		TotalVotesCount:   r.TotalVotesCount,
		Options:           options,
		CustomAnswers:     answers,
	}
	if len(r.VoterDetails) > 0 {
		voters := make([]voterDetailDTO, 0, len(r.VoterDetails))
		for _, v := range r.VoterDetails {
			detail := voterDetailDTO{
				UserID:            v.UserID,
				SelectedOptionIDs: v.SelectedOptionIDs,
			}
			if v.DisplayName != "" {
				name := v.DisplayName
				detail.DisplayName = &name
			}
			if v.CustomText != "" {
				txt := v.CustomText
				detail.CustomText = &txt
			}
			if detail.SelectedOptionIDs == nil {
				detail.SelectedOptionIDs = []string{}
			}
			voters = append(voters, detail)
		}
		out.VoterDetails = voters
	}
	return out
}
