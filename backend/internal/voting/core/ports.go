package voting

import (
	"context"
	"time"

	polls "xugeaneeu/pollify/internal/polls/core"
)

type Clock interface {
	Now() time.Time
}

type PollReader interface {
	GetByID(ctx context.Context, pollID string) (polls.Poll, error)
}

type Repository interface {
	HasParticipated(ctx context.Context, pollID string, userID string) (bool, error)
	SaveVote(ctx context.Context, participation Participation, votes []VoteRecord) error
	GetResults(ctx context.Context, pollID string, includeVoters bool) (PollResults, error)
}
