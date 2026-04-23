package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	postgrestest "xugeaneeu/pollify/internal/platform/postgrestest"
	userspg "xugeaneeu/pollify/internal/users/adapters/postgres"
	users "xugeaneeu/pollify/internal/users/core"
)

func TestUsersRepositoryCreateAndLookup(t *testing.T) {
	pool := postgrestest.OpenTestDB(t)
	repo := userspg.NewRepository(pool)

	created, err := repo.Create(context.Background(), users.CreateUserParams{
		Email:        "alice@example.com",
		PasswordHash: "hash",
		DisplayName:  "Alice",
		Role:         users.RoleUser,
		CreatedAt:    time.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	byID, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if byID.Email != created.Email {
		t.Fatalf("unexpected email: got %q want %q", byID.Email, created.Email)
	}

	byEmail, err := repo.GetByEmail(context.Background(), created.Email)
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Fatalf("unexpected user id: got %q want %q", byEmail.ID, created.ID)
	}

	_, err = repo.Create(context.Background(), users.CreateUserParams{
		Email:        created.Email,
		PasswordHash: "hash-2",
		DisplayName:  "Alice 2",
		Role:         users.RoleUser,
		CreatedAt:    time.Now().UTC(),
	})
	if !errors.Is(err, users.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}
