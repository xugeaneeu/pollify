package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	users "xugeaneeu/pollify/internal/users/core"
)

func TestMiddlewareRejectsMissingAuthorization(t *testing.T) {
	verifier := NewHMACVerifier([]byte("secret"), func() time.Time { return time.Unix(1_700_000_000, 0) })
	handler := Middleware(verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestMiddlewareInjectsAuthenticatedUser(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	verifier := NewHMACVerifier([]byte("secret"), func() time.Time { return now })
	token := signedToken(t, []byte("secret"), map[string]any{
		"sub":  "user-1",
		"role": string(users.RoleAdmin),
		"exp":  now.Add(time.Hour).Unix(),
	})

	handler := Middleware(verifier)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := AuthenticatedUserFromContext(r.Context())
		if !ok {
			t.Fatalf("expected actor in context")
		}
		if actor.UserID != "user-1" {
			t.Fatalf("expected user id user-1, got %q", actor.UserID)
		}
		if actor.Role != users.RoleAdmin {
			t.Fatalf("expected role %q, got %q", users.RoleAdmin, actor.Role)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func signedToken(t *testing.T, secret []byte, claims map[string]any) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return signed
}
