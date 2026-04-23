package polls

import "time"

type Status string

const (
	StatusScheduled Status = "scheduled"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	StatusHidden    Status = "hidden"
)

type PollOption struct {
	ID   string
	Text string
}

type PollSettings struct {
	IsAnonymous       bool
	IsMultipleChoice  bool
	MaxChoices        int
	AllowCustomAnswer bool
	StartAt           time.Time
	EndAt             time.Time
}

type Poll struct {
	ID          string
	Title       string
	Description string
	Question    string
	Options     []PollOption
	Settings    PollSettings
	IsHidden    bool
	CreatedBy   string
	CreatedAt   time.Time
}

type ParticipationSummary struct {
	ParticipantsCount int
	HasVoted          bool
}

type ListFilter struct {
	Status             *Status
	CreatorID          string
	IsAnonymous        *bool
	IsMultipleChoice   *bool
	AllowCustomAnswer  *bool
	AvailableForVoting *bool
	Page               int
	Limit              int
	Sort               string
	ActorID            string
}

func (p Poll) Status(now time.Time) Status {
	if p.IsHidden {
		return StatusHidden
	}
	if now.Before(p.Settings.StartAt) {
		return StatusScheduled
	}
	if !now.Before(p.Settings.EndAt) {
		return StatusCompleted
	}

	return StatusActive
}

func (p Poll) HasStarted(now time.Time) bool {
	return !now.Before(p.Settings.StartAt)
}
