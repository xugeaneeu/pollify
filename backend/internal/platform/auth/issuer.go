package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	users "xugeaneeu/pollify/internal/users/core"
)

var ErrIssuerSecretMissing = errors.New("auth: issuer secret is empty")

type HMACIssuer struct {
	secret    []byte
	now       func() time.Time
	accessTTL time.Duration
}

func NewHMACIssuer(secret []byte, accessTTL time.Duration, now func() time.Time) *HMACIssuer {
	if now == nil {
		now = time.Now
	}
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}
	return &HMACIssuer{
		secret:    append([]byte(nil), secret...),
		now:       now,
		accessTTL: accessTTL,
	}
}

func (i *HMACIssuer) IssueTokens(_ context.Context, user users.User) (users.AuthTokens, error) {
	if len(i.secret) == 0 {
		return users.AuthTokens{}, ErrIssuerSecretMissing
	}

	now := i.now()
	access, err := i.signAccessToken(user, now)
	if err != nil {
		return users.AuthTokens{}, err
	}

	refresh, err := randomToken(32)
	if err != nil {
		return users.AuthTokens{}, err
	}

	return users.AuthTokens{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(i.accessTTL.Seconds()),
	}, nil
}

func (i *HMACIssuer) signAccessToken(user users.User, now time.Time) (string, error) {
	claims := jwtClaims{
		Role: string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.accessTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}

func randomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
