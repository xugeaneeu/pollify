package users

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type UserRepository interface {
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	Create(ctx context.Context, params CreateUserParams) (User, error)
}

type CreateUserParams struct {
	Email        string
	PasswordHash string
	DisplayName  string
	Role         Role
	CreatedAt    time.Time
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password string, hash string) error
}

type TokenIssuer interface {
	IssueTokens(ctx context.Context, user User) (AuthTokens, error)
}
