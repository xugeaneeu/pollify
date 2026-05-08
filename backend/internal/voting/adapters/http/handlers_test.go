package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"xugeaneeu/pollify/internal/platform/auth"
	users "xugeaneeu/pollify/internal/users/core"
	voting "xugeaneeu/pollify/internal/voting/core"
)

type fakeService struct {
	submitFn  func(context.Context, string, voting.SubmitVoteInput) (voting.VoteReceipt, error)
	resultsFn func(context.Context, string, string, bool) (voting.PollResults, error)
}

func (f *fakeService) SubmitVote(ctx context.Context, actor string, in voting.SubmitVoteInput) (voting.VoteReceipt, error) {
	return f.submitFn(ctx, actor, in)
}
func (f *fakeService) Results(ctx context.Context, actor, pollID string, include bool) (voting.PollResults, error) {
	return f.resultsFn(ctx, actor, pollID, include)
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

func TestSubmitVoteSuccess(t *testing.T) {
	svc := &fakeService{submitFn: func(_ context.Context, actor string, in voting.SubmitVoteInput) (voting.VoteReceipt, error) {
		if actor != "user-1" || in.PollID != "poll-1" || len(in.OptionIDs) != 1 || in.OptionIDs[0] != "opt-a" {
			t.Fatalf("unexpected args: %s %+v", actor, in)
		}
		return voting.VoteReceipt{PollID: "poll-1", ParticipationRecorded: true}, nil
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/polls/poll-1/votes", "application/json", strings.NewReader(`{"option_ids":["opt-a"]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got voteAcceptedResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.PollID != "poll-1" || !got.ParticipationRecorded {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestSubmitVoteAlreadyVoted(t *testing.T) {
	svc := &fakeService{submitFn: func(context.Context, string, voting.SubmitVoteInput) (voting.VoteReceipt, error) {
		return voting.VoteReceipt{}, voting.ErrPollAlreadyVoted
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/polls/poll-1/votes", "application/json", strings.NewReader(`{"option_ids":["opt-a"]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestSubmitVoteInvalidPayload(t *testing.T) {
	svc := &fakeService{submitFn: func(context.Context, string, voting.SubmitVoteInput) (voting.VoteReceipt, error) {
		return voting.VoteReceipt{}, voting.ValidationError{Field: "vote", Message: "must contain either option_ids or custom_text"}
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/polls/poll-1/votes", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestGetResultsBasic(t *testing.T) {
	svc := &fakeService{resultsFn: func(_ context.Context, actor, pollID string, include bool) (voting.PollResults, error) {
		if include {
			t.Fatalf("did not expect include=true")
		}
		return voting.PollResults{
			PollID:            pollID,
			Status:            "active",
			ParticipantsCount: 2,
			TotalVotesCount:   3,
			Options: []voting.OptionResult{
				{OptionID: "a", Label: "A", VotesCount: 2, Percentage: 66.67},
				{OptionID: "b", Label: "B", VotesCount: 1, Percentage: 33.33},
			},
		}, nil
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/polls/poll-1/results")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got pollResultsResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ParticipantsCount != 2 || len(got.Options) != 2 {
		t.Fatalf("unexpected results: %+v", got)
	}
	if got.VoterDetails != nil {
		t.Fatalf("did not expect voter_details when not requested")
	}
}

func TestGetResultsIncludeVoters(t *testing.T) {
	svc := &fakeService{resultsFn: func(_ context.Context, _, pollID string, include bool) (voting.PollResults, error) {
		if !include {
			t.Fatalf("expected include=true")
		}
		return voting.PollResults{
			PollID: pollID,
			Status: "completed",
			VoterDetails: []voting.VoterDetail{
				{UserID: "user-2", DisplayName: "Bob", SelectedOptionIDs: []string{"a"}},
			},
		}, nil
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/polls/poll-1/results?include_voters=true")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got pollResultsResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.VoterDetails) != 1 || got.VoterDetails[0].UserID != "user-2" {
		t.Fatalf("unexpected voter details: %+v", got.VoterDetails)
	}
}
