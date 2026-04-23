package postgres_test

import (
	"context"
	"testing"
	"time"

	postgrestest "xugeaneeu/pollify/internal/platform/postgrestest"
	pollspg "xugeaneeu/pollify/internal/polls/adapters/postgres"
	polls "xugeaneeu/pollify/internal/polls/core"
)

func TestPollsRepositoryCreateAndLoad(t *testing.T) {
	pool := postgrestest.OpenTestDB(t)
	postgrestest.SeedUser(t, pool, "11111111-1111-1111-1111-111111111111", "owner@example.com", "USER")

	repo := pollspg.NewRepository(pool)
	startAt := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	endAt := startAt.Add(2 * time.Hour)

	created, err := repo.Create(context.Background(), polls.Poll{
		Title:       "Release poll",
		Description: "desc",
		Question:    "Ship it?",
		CreatedBy:   "11111111-1111-1111-1111-111111111111",
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
		Settings: polls.PollSettings{
			IsAnonymous:       false,
			IsMultipleChoice:  false,
			MaxChoices:        0,
			AllowCustomAnswer: false,
			StartAt:           startAt,
			EndAt:             endAt,
		},
		Options: []polls.PollOption{{Text: "Yes"}, {Text: "No"}},
	})
	if err != nil {
		t.Fatalf("create poll: %v", err)
	}
	if len(created.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(created.Options))
	}

	loaded, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get poll: %v", err)
	}
	if loaded.Question != "Ship it?" {
		t.Fatalf("unexpected question: %q", loaded.Question)
	}
	if len(loaded.Options) != 2 {
		t.Fatalf("expected loaded options, got %d", len(loaded.Options))
	}

	hasParticipants, err := repo.HasParticipants(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("has participants: %v", err)
	}
	if hasParticipants {
		t.Fatalf("expected no participants")
	}
}
