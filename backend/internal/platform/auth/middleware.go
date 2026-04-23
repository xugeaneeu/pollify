package auth

import (
	"context"
	"encoding/json"
	"net/http"

	users "xugeaneeu/pollify/internal/users/core"
)

type contextKey string

const authenticatedUserContextKey contextKey = "authenticated_user"

func Middleware(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if verifier == nil {
				writeUnauthorized(w, ErrVerifierUnbound)
				return
			}

			token, err := ParseBearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeUnauthorized(w, err)
				return
			}

			claims, err := verifier.VerifyToken(r.Context(), token)
			if err != nil {
				writeUnauthorized(w, err)
				return
			}

			actor := users.AuthenticatedUser{UserID: claims.UserID, Role: claims.Role}
			next.ServeHTTP(w, r.WithContext(ContextWithAuthenticatedUser(r.Context(), actor)))
		})
	}
}

func ContextWithAuthenticatedUser(ctx context.Context, actor users.AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authenticatedUserContextKey, actor)
}

func AuthenticatedUserFromContext(ctx context.Context) (users.AuthenticatedUser, bool) {
	actor, ok := ctx.Value(authenticatedUserContextKey).(users.AuthenticatedUser)
	return actor, ok
}

func writeUnauthorized(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "unauthorized",
			"message": err.Error(),
			"details": map[string]any{},
		},
	})
}
