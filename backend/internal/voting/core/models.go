package voting

import (
	"time"

	polls "xugeaneeu/pollify/internal/polls/core"
)

type Participation struct {
	PollID  string
	UserID  string
	VotedAt time.Time
}

type VoteRecord struct {
	PollID     string
	OptionID   string
	UserID     string
	CustomText string
	CreatedAt  time.Time
}

type VoteReceipt struct {
	PollID                string
	ParticipationRecorded bool
}

type OptionResult struct {
	OptionID   string
	Label      string
	VotesCount int
	Percentage float64
}

type TextAnswer struct {
	Value string
	Count int
}

type VoterDetail struct {
	UserID            string
	DisplayName       string
	SelectedOptionIDs []string
	CustomText        string
}

type PollResults struct {
	PollID            string
	Status            polls.Status
	ParticipantsCount int
	TotalVotesCount   int
	Options           []OptionResult
	CustomAnswers     []TextAnswer
	VoterDetails      []VoterDetail
}
