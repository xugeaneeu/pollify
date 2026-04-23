package users

import "time"

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	Role         Role
	CreatedAt    time.Time
}

type AuthenticatedUser struct {
	UserID string
	Role   Role
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

type AuthSession struct {
	User   User
	Tokens AuthTokens
}
