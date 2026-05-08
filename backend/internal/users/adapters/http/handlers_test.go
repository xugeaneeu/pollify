package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"xugeaneeu/pollify/internal/platform/auth"
	users "xugeaneeu/pollify/internal/users/core"
)

type fakeService struct {
	registerFn func(context.Context, users.RegisterInput) (users.AuthSession, error)
	loginFn    func(context.Context, users.LoginInput) (users.AuthSession, error)
	currentFn  func(context.Context, users.AuthenticatedUser) (users.User, error)
}

func (f *fakeService) Register(ctx context.Context, in users.RegisterInput) (users.AuthSession, error) {
	return f.registerFn(ctx, in)
}

func (f *fakeService) Login(ctx context.Context, in users.LoginInput) (users.AuthSession, error) {
	return f.loginFn(ctx, in)
}

func (f *fakeService) CurrentUser(ctx context.Context, actor users.AuthenticatedUser) (users.User, error) {
	return f.currentFn(ctx, actor)
}

func newRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Mount("/auth", h.PublicRoutes())
	r.Mount("/users", h.ProtectedRoutes())
	return r
}

func sampleSession() users.AuthSession {
	return users.AuthSession{
		User: users.User{
			ID:          "11111111-1111-1111-1111-111111111111",
			Email:       "alice@example.com",
			DisplayName: "Alice",
			Role:        users.RoleUser,
			CreatedAt:   time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC),
		},
		Tokens: users.AuthTokens{
			AccessToken:  "access",
			RefreshToken: "refresh",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		},
	}
}

func TestRegisterHandlerReturnsCreated(t *testing.T) {
	svc := &fakeService{registerFn: func(_ context.Context, in users.RegisterInput) (users.AuthSession, error) {
		if in.Email != "alice@example.com" || in.Password != "password123" {
			t.Fatalf("unexpected input: %+v", in)
		}
		return sampleSession(), nil
	}}
	srv := httptest.NewServer(newRouter(NewHandler(svc)))
	defer srv.Close()

	body := strings.NewReader(`{"email":"alice@example.com","password":"password123","display_name":"Alice"}`)
	resp, err := http.Post(srv.URL+"/auth/register", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var got authSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.User.ID != "11111111-1111-1111-1111-111111111111" || got.Tokens.AccessToken != "access" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestRegisterHandlerValidationError(t *testing.T) {
	svc := &fakeService{registerFn: func(context.Context, users.RegisterInput) (users.AuthSession, error) {
		return users.AuthSession{}, users.ValidationError{Field: "password", Message: "too short"}
	}}
	srv := httptest.NewServer(newRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/auth/register", "application/json", strings.NewReader(`{"email":"a@b.c","password":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	errBody, _ := got["error"].(map[string]any)
	if errBody["code"] != "validation_error" {
		t.Fatalf("expected validation_error, got %v", errBody)
	}
}

func TestRegisterHandlerEmailConflict(t *testing.T) {
	svc := &fakeService{registerFn: func(context.Context, users.RegisterInput) (users.AuthSession, error) {
		return users.AuthSession{}, users.ErrEmailAlreadyExists
	}}
	srv := httptest.NewServer(newRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/auth/register", "application/json", strings.NewReader(`{"email":"a@b.c","password":"password123"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestLoginHandlerInvalidCredentials(t *testing.T) {
	svc := &fakeService{loginFn: func(context.Context, users.LoginInput) (users.AuthSession, error) {
		return users.AuthSession{}, users.ErrInvalidCredentials
	}}
	srv := httptest.NewServer(newRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/auth/login", "application/json", strings.NewReader(`{"email":"a@b.c","password":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestMeHandlerReturnsCurrentUser(t *testing.T) {
	svc := &fakeService{currentFn: func(_ context.Context, actor users.AuthenticatedUser) (users.User, error) {
		if actor.UserID != "user-1" {
			t.Fatalf("unexpected actor: %+v", actor)
		}
		return users.User{ID: "user-1", Email: "alice@example.com", Role: users.RoleUser, CreatedAt: time.Now()}, nil
	}}

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := auth.ContextWithAuthenticatedUser(req.Context(), users.AuthenticatedUser{UserID: "user-1", Role: users.RoleUser})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.Mount("/users", NewHandler(svc).ProtectedRoutes())
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/users/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got userSummary
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "user-1" {
		t.Fatalf("expected user-1, got %+v", got)
	}
}

func TestMeHandlerUnauthorized(t *testing.T) {
	svc := &fakeService{}
	srv := httptest.NewServer(newRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/users/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
