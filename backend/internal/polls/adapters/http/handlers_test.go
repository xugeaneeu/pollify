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
	polls "xugeaneeu/pollify/internal/polls/core"
	users "xugeaneeu/pollify/internal/users/core"
)

type fakeService struct {
	createFn func(context.Context, polls.CreatePollInput) (polls.Poll, error)
	getFn    func(context.Context, string) (polls.Poll, error)
	listFn   func(context.Context, polls.ListFilter) ([]polls.Poll, error)
	updateFn func(context.Context, string, string, polls.UpdatePollInput) (polls.Poll, error)
}

func (f *fakeService) Create(ctx context.Context, in polls.CreatePollInput) (polls.Poll, error) {
	return f.createFn(ctx, in)
}
func (f *fakeService) Get(ctx context.Context, id string) (polls.Poll, error) {
	return f.getFn(ctx, id)
}
func (f *fakeService) List(ctx context.Context, filter polls.ListFilter) ([]polls.Poll, error) {
	return f.listFn(ctx, filter)
}
func (f *fakeService) Update(ctx context.Context, actorID, pollID string, in polls.UpdatePollInput) (polls.Poll, error) {
	return f.updateFn(ctx, actorID, pollID, in)
}

type fakeStats struct {
	countFn    func(context.Context, string) (int, error)
	hasVotedFn func(context.Context, string, string) (bool, error)
}

func (f *fakeStats) Count(ctx context.Context, pollID string) (int, error) {
	return f.countFn(ctx, pollID)
}
func (f *fakeStats) HasVoted(ctx context.Context, pollID, userID string) (bool, error) {
	return f.hasVotedFn(ctx, pollID, userID)
}

type fakeCounter struct {
	countFn func(context.Context, polls.ListFilter) (int, error)
}

func (f *fakeCounter) Count(ctx context.Context, filter polls.ListFilter) (int, error) {
	return f.countFn(ctx, filter)
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func samplePoll() polls.Poll {
	return polls.Poll{
		ID:          "poll-1",
		Title:       "Lunch?",
		Description: "Where are we going?",
		Question:    "Pick a place",
		Options: []polls.PollOption{
			{ID: "opt-1", Text: "Pizza"},
			{ID: "opt-2", Text: "Sushi"},
		},
		Settings: polls.PollSettings{
			IsAnonymous:      false,
			IsMultipleChoice: false,
			StartAt:          time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC),
			EndAt:            time.Date(2026, 5, 7, 18, 0, 0, 0, time.UTC),
		},
		CreatedBy: "user-1",
		CreatedAt: time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC),
	}
}

func makeRouter(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := auth.ContextWithAuthenticatedUser(req.Context(), users.AuthenticatedUser{UserID: "user-1", Role: users.RoleUser})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.Route("/polls", func(r chi.Router) {
			h.RegisterRoutes(r)
		})
	})
	return r
}

func TestCreatePollSuccess(t *testing.T) {
	svc := &fakeService{createFn: func(_ context.Context, in polls.CreatePollInput) (polls.Poll, error) {
		if in.CreatedBy != "user-1" {
			t.Fatalf("expected created_by user-1, got %s", in.CreatedBy)
		}
		if in.Title != "Lunch?" {
			t.Fatalf("title not propagated: %+v", in)
		}
		if len(in.Options) != 2 {
			t.Fatalf("expected 2 options, got %d", len(in.Options))
		}
		return samplePoll(), nil
	}}
	stats := &fakeStats{countFn: func(context.Context, string) (int, error) { return 0, nil }, hasVotedFn: func(context.Context, string, string) (bool, error) { return false, nil }}
	counter := &fakeCounter{countFn: func(context.Context, polls.ListFilter) (int, error) { return 0, nil }}
	clock := fixedClock{t: time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, stats, counter, clock)))
	defer srv.Close()

	body := `{"title":"Lunch?","question":"Pick a place","options":[{"text":"Pizza"},{"text":"Sushi"}],"settings":{"is_anonymous":false,"is_multiple_choice":false,"allow_custom_answer":false,"start_at":"2026-05-07T10:00:00Z","end_at":"2026-05-07T18:00:00Z"}}`
	resp, err := http.Post(srv.URL+"/polls/", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got pollDetailsDTO
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "active" {
		t.Fatalf("expected active status, got %s", got.Status)
	}
	if len(got.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(got.Options))
	}
}

func TestCreatePollValidationError(t *testing.T) {
	svc := &fakeService{createFn: func(context.Context, polls.CreatePollInput) (polls.Poll, error) {
		return polls.Poll{}, polls.ValidationError{Field: "title", Message: "must not be empty"}
	}}
	stats := &fakeStats{}
	counter := &fakeCounter{}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, stats, counter, fixedClock{t: time.Now()})))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/polls/", "application/json", strings.NewReader(`{"title":"","question":"q","settings":{"is_anonymous":false,"is_multiple_choice":false,"allow_custom_answer":false,"start_at":"2026-05-07T10:00:00Z","end_at":"2026-05-07T18:00:00Z"}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestListPollsPagination(t *testing.T) {
	svc := &fakeService{listFn: func(_ context.Context, filter polls.ListFilter) ([]polls.Poll, error) {
		if filter.Page != 2 || filter.Limit != 5 {
			t.Fatalf("unexpected filter: %+v", filter)
		}
		return []polls.Poll{samplePoll()}, nil
	}}
	stats := &fakeStats{countFn: func(context.Context, string) (int, error) { return 3, nil }, hasVotedFn: func(context.Context, string, string) (bool, error) { return true, nil }}
	counter := &fakeCounter{countFn: func(context.Context, polls.ListFilter) (int, error) { return 11, nil }}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, stats, counter, fixedClock{t: time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)})))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/polls/?page=2&limit=5")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got pollsListResponseDTO
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Pagination.TotalItems != 11 || got.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected pagination: %+v", got.Pagination)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got.Items))
	}
	if got.Items[0].ParticipationSummary.HasVoted != true {
		t.Fatalf("expected has_voted true")
	}
}

func TestGetPollNotFound(t *testing.T) {
	svc := &fakeService{getFn: func(context.Context, string) (polls.Poll, error) {
		return polls.Poll{}, polls.ErrPollNotFound
	}}
	stats := &fakeStats{}
	counter := &fakeCounter{}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, stats, counter, fixedClock{t: time.Now()})))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/polls/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestUpdatePollNotAllowed(t *testing.T) {
	svc := &fakeService{updateFn: func(context.Context, string, string, polls.UpdatePollInput) (polls.Poll, error) {
		return polls.Poll{}, polls.ErrPollUpdateNotAllowed
	}}
	stats := &fakeStats{}
	counter := &fakeCounter{}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, stats, counter, fixedClock{t: time.Now()})))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/polls/poll-1", strings.NewReader(`{"title":"new"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
