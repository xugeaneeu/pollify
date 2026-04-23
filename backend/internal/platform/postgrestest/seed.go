package postgrestest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedUser(t testing.TB, pool *pgxpool.Pool, id string, email string, role string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, email, password_hash, display_name, role, created_at)
		VALUES ($1, $2, 'hash', 'name', $3, $4)
	`, id, email, role, time.Now().UTC().Truncate(time.Second))
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func SeedPoll(t testing.TB, pool *pgxpool.Pool, id string, createdBy string, isAnonymous bool) {
	t.Helper()
	startAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	endAt := startAt.Add(2 * time.Hour)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO polls (
			id, title, description, question, is_anonymous, is_multiple_choice, max_choices,
			allow_custom_answer, is_hidden, created_by, start_at, end_at, created_at
		) VALUES ($1, 'title', 'description', 'question', $2, FALSE, 0, FALSE, FALSE, $3, $4, $5, $6)
	`, id, isAnonymous, createdBy, startAt, endAt, time.Now().UTC().Truncate(time.Second))
	if err != nil {
		t.Fatalf("seed poll: %v", err)
	}
}

func SeedOption(t testing.TB, pool *pgxpool.Pool, id string, pollID string, text string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO options (id, poll_id, text)
		VALUES ($1, $2, $3)
	`, id, pollID, text)
	if err != nil {
		t.Fatalf("seed option: %v", err)
	}
}
