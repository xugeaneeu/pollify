package http

import (
	"time"

	users "xugeaneeu/pollify/internal/users/core"
)

type registerRequest struct {
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName *string `json:"display_name,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userSummary struct {
	ID          string     `json:"id"`
	DisplayName *string    `json:"display_name"`
	Email       *string    `json:"email"`
	Role        string     `json:"role"`
	CreatedAt   *time.Time `json:"created_at"`
}

type authTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type authSessionResponse struct {
	User   userSummary `json:"user"`
	Tokens authTokens  `json:"tokens"`
}

func toUserSummary(u users.User) userSummary {
	out := userSummary{ID: u.ID, Role: string(u.Role)}
	if u.DisplayName != "" {
		dn := u.DisplayName
		out.DisplayName = &dn
	}
	if u.Email != "" {
		em := u.Email
		out.Email = &em
	}
	if !u.CreatedAt.IsZero() {
		ts := u.CreatedAt.UTC()
		out.CreatedAt = &ts
	}
	return out
}

func toSessionResponse(s users.AuthSession) authSessionResponse {
	return authSessionResponse{
		User: toUserSummary(s.User),
		Tokens: authTokens{
			AccessToken:  s.Tokens.AccessToken,
			RefreshToken: s.Tokens.RefreshToken,
			TokenType:    s.Tokens.TokenType,
			ExpiresIn:    s.Tokens.ExpiresIn,
		},
	}
}
