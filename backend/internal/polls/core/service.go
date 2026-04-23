package polls

import (
	"context"
	"strings"
	"time"
)

type Service struct {
	repo         Repository
	participants ParticipationRepository
	clock        Clock
}

type CreatePollOptionInput struct {
	Text string
}

type CreatePollInput struct {
	CreatedBy   string
	Title       string
	Description string
	Question    string
	Options     []CreatePollOptionInput
	Settings    PollSettings
}

type UpdatePollInput struct {
	Title       *string
	Description *string
	StartAt     *time.Time
	EndAt       *time.Time
}

func NewService(repo Repository, participants ParticipationRepository, clock Clock) (*Service, error) {
	if repo == nil {
		return nil, ErrRepositoryUnbound
	}
	if participants == nil {
		return nil, ErrParticipantsUnbound
	}
	if clock == nil {
		return nil, ErrClockUnbound
	}

	return &Service{repo: repo, participants: participants, clock: clock}, nil
}

func (s *Service) Create(ctx context.Context, input CreatePollInput) (Poll, error) {
	if err := ValidateCreateInput(input); err != nil {
		return Poll{}, err
	}

	options := make([]PollOption, 0, len(input.Options))
	for _, option := range input.Options {
		options = append(options, PollOption{Text: strings.TrimSpace(option.Text)})
	}

	settings := input.Settings
	if !settings.IsMultipleChoice {
		settings.MaxChoices = 0
	}

	poll := Poll{
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Question:    strings.TrimSpace(input.Question),
		Options:     options,
		Settings:    settings,
		CreatedBy:   strings.TrimSpace(input.CreatedBy),
		CreatedAt:   s.clock.Now(),
	}

	return s.repo.Create(ctx, poll)
}

func (s *Service) Get(ctx context.Context, pollID string) (Poll, error) {
	return s.repo.GetByID(ctx, strings.TrimSpace(pollID))
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Poll, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) Update(ctx context.Context, actorID string, pollID string, input UpdatePollInput) (Poll, error) {
	poll, err := s.repo.GetByID(ctx, strings.TrimSpace(pollID))
	if err != nil {
		return Poll{}, err
	}
	if poll.CreatedBy != strings.TrimSpace(actorID) {
		return Poll{}, ErrPollUpdateNotAllowed
	}

	hasParticipants, err := s.participants.HasParticipants(ctx, poll.ID)
	if err != nil {
		return Poll{}, err
	}
	if err := ValidateUpdateInput(poll, input, s.clock.Now(), hasParticipants); err != nil {
		return Poll{}, err
	}

	if input.Title != nil {
		poll.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		poll.Description = strings.TrimSpace(*input.Description)
	}
	if input.StartAt != nil {
		poll.Settings.StartAt = *input.StartAt
	}
	if input.EndAt != nil {
		poll.Settings.EndAt = *input.EndAt
	}

	return s.repo.Update(ctx, poll)
}
