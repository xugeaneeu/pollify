package voting

import (
	"context"
	"strings"

	polls "xugeaneeu/pollify/internal/polls/core"
)

type Service struct {
	polls PollReader
	repo  Repository
	clock Clock
}

type SubmitVoteInput struct {
	PollID     string
	OptionIDs  []string
	CustomText string
}

func NewService(pollsReader PollReader, repo Repository, clock Clock) (*Service, error) {
	if pollsReader == nil {
		return nil, ErrPollReaderNil
	}
	if repo == nil {
		return nil, ErrRepositoryNil
	}
	if clock == nil {
		return nil, ErrClockNil
	}

	return &Service{polls: pollsReader, repo: repo, clock: clock}, nil
}

func (s *Service) SubmitVote(ctx context.Context, actorID string, input SubmitVoteInput) (VoteReceipt, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return VoteReceipt{}, ValidationError{Field: "actor_id", Message: "must not be empty"}
	}

	poll, err := s.polls.GetByID(ctx, strings.TrimSpace(input.PollID))
	if err != nil {
		return VoteReceipt{}, err
	}

	status := poll.Status(s.clock.Now())
	if status == polls.StatusHidden {
		return VoteReceipt{}, ErrPollHidden
	}
	if status != polls.StatusActive {
		return VoteReceipt{}, ErrPollNotActive
	}

	alreadyVoted, err := s.repo.HasParticipated(ctx, poll.ID, actorID)
	if err != nil {
		return VoteReceipt{}, err
	}
	if alreadyVoted {
		return VoteReceipt{}, ErrPollAlreadyVoted
	}

	if err := ValidateVotePayload(poll, input); err != nil {
		return VoteReceipt{}, err
	}

	now := s.clock.Now()
	participation := Participation{PollID: poll.ID, UserID: actorID, VotedAt: now}
	votes := make([]VoteRecord, 0, max(1, len(input.OptionIDs)))
	voteUserID := actorID
	if poll.Settings.IsAnonymous {
		voteUserID = ""
	}

	if strings.TrimSpace(input.CustomText) != "" {
		votes = append(votes, VoteRecord{
			PollID:     poll.ID,
			UserID:     voteUserID,
			CustomText: strings.TrimSpace(input.CustomText),
			CreatedAt:  now,
		})
	} else {
		for _, optionID := range input.OptionIDs {
			votes = append(votes, VoteRecord{
				PollID:    poll.ID,
				OptionID:  strings.TrimSpace(optionID),
				UserID:    voteUserID,
				CreatedAt: now,
			})
		}
	}

	if err := s.repo.SaveVote(ctx, participation, votes); err != nil {
		return VoteReceipt{}, err
	}

	return VoteReceipt{PollID: poll.ID, ParticipationRecorded: true}, nil
}

func (s *Service) Results(ctx context.Context, actorID string, pollID string, includeVoters bool) (PollResults, error) {
	if strings.TrimSpace(actorID) == "" {
		return PollResults{}, ValidationError{Field: "actor_id", Message: "must not be empty"}
	}

	poll, err := s.polls.GetByID(ctx, strings.TrimSpace(pollID))
	if err != nil {
		return PollResults{}, err
	}
	if includeVoters && poll.Settings.IsAnonymous {
		return PollResults{}, ValidationError{Field: "include_voters", Message: "is allowed only for non-anonymous polls"}
	}

	results, err := s.repo.GetResults(ctx, poll.ID, includeVoters)
	if err != nil {
		return PollResults{}, err
	}

	results.PollID = poll.ID
	results.Status = poll.Status(s.clock.Now())
	if poll.Settings.IsAnonymous {
		results.VoterDetails = nil
	}

	return results, nil
}
