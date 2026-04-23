package polls

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type Repository interface {
	Create(ctx context.Context, poll Poll) (Poll, error)
	Update(ctx context.Context, poll Poll) (Poll, error)
	GetByID(ctx context.Context, pollID string) (Poll, error)
	List(ctx context.Context, filter ListFilter) ([]Poll, error)
}

type ParticipationRepository interface {
	HasParticipants(ctx context.Context, pollID string) (bool, error)
}
