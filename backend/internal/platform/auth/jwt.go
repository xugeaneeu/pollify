package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	users "xugeaneeu/pollify/internal/users/core"
)

var (
	ErrMissingAuthorizationHeader = errors.New("auth: missing authorization header")
	ErrInvalidAuthorizationHeader = errors.New("auth: invalid authorization header")
	ErrInvalidToken               = errors.New("auth: invalid token")
	ErrUnsupportedAlgorithm       = errors.New("auth: unsupported jwt algorithm")
	ErrExpiredToken               = errors.New("auth: token expired")
	ErrVerifierUnbound            = errors.New("auth: token verifier is required")
)

type Claims struct {
	UserID string
	Role   users.Role
	Exp    int64
}

type TokenVerifier interface {
	VerifyToken(ctx context.Context, rawToken string) (Claims, error)
}

type HMACVerifier struct {
	secret []byte
	now    func() time.Time
}

type jwtClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func NewHMACVerifier(secret []byte, now func() time.Time) *HMACVerifier {
	if now == nil {
		now = time.Now
	}

	return &HMACVerifier{secret: append([]byte(nil), secret...), now: now}
}

func (v *HMACVerifier) VerifyToken(_ context.Context, rawToken string) (Claims, error) {
	if len(v.secret) == 0 {
		return Claims{}, ErrInvalidToken
	}

	var payload jwtClaims
	_, err := jwt.ParseWithClaims(rawToken, &payload, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrUnsupportedAlgorithm
		}

		return v.secret, nil
	}, jwt.WithTimeFunc(v.now))
	if err != nil {
		if errors.Is(err, ErrUnsupportedAlgorithm) {
			return Claims{}, err
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Claims{}, ErrExpiredToken
		}
		return Claims{}, ErrInvalidToken
	}

	if strings.TrimSpace(payload.Subject) == "" {
		return Claims{}, ErrInvalidToken
	}

	role := users.Role(strings.TrimSpace(payload.Role))
	if role != users.RoleUser && role != users.RoleAdmin {
		return Claims{}, ErrInvalidToken
	}

	var exp int64
	if payload.ExpiresAt != nil {
		exp = payload.ExpiresAt.Unix()
	}

	return Claims{UserID: payload.Subject, Role: role, Exp: exp}, nil
}

func ParseBearerToken(headerValue string) (string, error) {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return "", ErrMissingAuthorizationHeader
	}

	parts := strings.SplitN(headerValue, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", ErrInvalidAuthorizationHeader
	}

	return strings.TrimSpace(parts[1]), nil
}
