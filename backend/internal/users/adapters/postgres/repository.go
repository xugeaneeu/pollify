package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	platformpg "xugeaneeu/pollify/internal/platform/postgres"
	users "xugeaneeu/pollify/internal/users/core"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id string) (users.User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, role, created_at
		FROM users
		WHERE id = $1
	`, strings.TrimSpace(id)))
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (users.User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, role, created_at
		FROM users
		WHERE email = $1
	`, strings.TrimSpace(email)))
}

func (r *Repository) Create(ctx context.Context, params users.CreateUserParams) (users.User, error) {
	user, err := scanUser(r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, password_hash, display_name, role, created_at
	`, params.Email, params.PasswordHash, params.DisplayName, string(params.Role), params.CreatedAt))
	if err != nil {
		if platformpg.ErrorCode(err) == "23505" {
			return users.User{}, users.ErrEmailAlreadyExists
		}
		return users.User{}, err
	}

	return user, nil
}

func scanUser(row pgx.Row) (users.User, error) {
	var user users.User
	var role string
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.DisplayName, &role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return users.User{}, users.ErrUserNotFound
		}
		return users.User{}, err
	}

	user.Role = users.Role(role)
	return user, nil
}
