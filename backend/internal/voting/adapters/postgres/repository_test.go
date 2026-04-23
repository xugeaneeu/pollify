package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	postgrestest "xugeaneeu/pollify/internal/platform/postgrestest"
	votingpg "xugeaneeu/pollify/internal/voting/adapters/postgres"
	voting "xugeaneeu/pollify/internal/voting/core"
)

func TestVotingRepositorySaveVoteAndResults(t *testing.T) {
	pool := postgrestest.OpenTestDB(t)
	postgrestest.SeedUser(t, pool, "11111111-1111-1111-1111-111111111111", "owner@example.com", "USER")
	postgrestest.SeedUser(t, pool, "22222222-2222-2222-2222-222222222222", "voter@example.com", "USER")
	postgrestest.SeedPoll(t, pool, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", false)
	postgrestest.SeedOption(t, pool, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Yes")
	postgrestest.SeedOption(t, pool, "cccccccc-cccc-cccc-cccc-cccccccccccc", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "No")

	repo := votingpg.NewRepository(pool)
	err := repo.SaveVote(context.Background(), voting.Participation{
		PollID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:  "22222222-2222-2222-2222-222222222222",
		VotedAt: time.Now().UTC().Truncate(time.Second),
	}, []voting.VoteRecord{{
		PollID:    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		OptionID:  "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:    "22222222-2222-2222-2222-222222222222",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}})
	if err != nil {
		t.Fatalf("save vote: %v", err)
	}

	err = repo.SaveVote(context.Background(), voting.Participation{
		PollID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:  "22222222-2222-2222-2222-222222222222",
		VotedAt: time.Now().UTC().Truncate(time.Second),
	}, []voting.VoteRecord{{
		PollID:    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		OptionID:  "cccccccc-cccc-cccc-cccc-cccccccccccc",
		UserID:    "22222222-2222-2222-2222-222222222222",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
	}})
	if !errors.Is(err, voting.ErrPollAlreadyVoted) {
		t.Fatalf("expected ErrPollAlreadyVoted, got %v", err)
	}

	results, err := repo.GetResults(context.Background(), "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", true)
	if err != nil {
		t.Fatalf("get results: %v", err)
	}
	if results.ParticipantsCount != 1 {
		t.Fatalf("unexpected participants count: %d", results.ParticipantsCount)
	}
	if len(results.Options) != 2 {
		t.Fatalf("expected 2 option results, got %d", len(results.Options))
	}
	if len(results.VoterDetails) != 1 {
		t.Fatalf("expected 1 voter detail, got %d", len(results.VoterDetails))
	}
}
