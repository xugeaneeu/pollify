package moderation

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

type PollVisibilityWriter interface {
	Hide(ctx context.Context, pollID string) error
}

type Repository interface {
	Create(ctx context.Context, report Report) (Report, error)
	GetByID(ctx context.Context, reportID string) (Report, error)
	List(ctx context.Context, filter ListFilter) ([]Report, error)
	HasActiveReport(ctx context.Context, pollID string, userID string) (bool, error)
	SaveReview(ctx context.Context, review ReportReview, report Report) (Report, error)
}
