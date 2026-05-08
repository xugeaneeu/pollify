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

	moderation "xugeaneeu/pollify/internal/moderation/core"
	"xugeaneeu/pollify/internal/platform/auth"
	users "xugeaneeu/pollify/internal/users/core"
)

type fakeService struct {
	createFn func(context.Context, string, moderation.CreateReportInput) (moderation.Report, error)
	getFn    func(context.Context, string) (moderation.Report, error)
	listFn   func(context.Context, moderation.ListFilter) ([]moderation.Report, error)
	reviewFn func(context.Context, moderation.ReviewReportInput) (moderation.ReviewOutcome, error)
}

func (f *fakeService) Create(ctx context.Context, actor string, in moderation.CreateReportInput) (moderation.Report, error) {
	return f.createFn(ctx, actor, in)
}
func (f *fakeService) Get(ctx context.Context, id string) (moderation.Report, error) {
	return f.getFn(ctx, id)
}
func (f *fakeService) List(ctx context.Context, filter moderation.ListFilter) ([]moderation.Report, error) {
	return f.listFn(ctx, filter)
}
func (f *fakeService) Review(ctx context.Context, in moderation.ReviewReportInput) (moderation.ReviewOutcome, error) {
	return f.reviewFn(ctx, in)
}

type fakeCounter struct {
	countFn func(context.Context, moderation.ListFilter) (int, error)
}

func (f *fakeCounter) Count(ctx context.Context, filter moderation.ListFilter) (int, error) {
	return f.countFn(ctx, filter)
}

func makeRouter(h *Handler, role users.Role, userID string) http.Handler {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := auth.ContextWithAuthenticatedUser(req.Context(), users.AuthenticatedUser{UserID: userID, Role: role})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.Route("/polls", func(p chi.Router) {
			p.Post("/{pollId}/reports", h.CreateReport)
			p.With(auth.RequireRole(users.RoleAdmin)).Get("/{pollId}/reports", h.ListReportsByPoll)
		})
		r.Route("/reports", func(rep chi.Router) {
			rep.Use(auth.RequireRole(users.RoleAdmin))
			rep.Get("/", h.ListReports)
			rep.Get("/{reportId}", h.GetReport)
			rep.Post("/{reportId}/reviews", h.SubmitReview)
		})
	})
	return r
}

func sampleReport() moderation.Report {
	return moderation.Report{
		ID:        "rep-1",
		PollID:    "poll-1",
		CreatedBy: "user-1",
		Reason:    "spam",
		Status:    moderation.ReportStatusOpen,
		CreatedAt: time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC),
	}
}

func TestCreateReportSuccess(t *testing.T) {
	svc := &fakeService{createFn: func(_ context.Context, actor string, in moderation.CreateReportInput) (moderation.Report, error) {
		if actor != "user-1" || in.PollID != "poll-1" || in.Reason != "spam" {
			t.Fatalf("unexpected args: %s %+v", actor, in)
		}
		return sampleReport(), nil
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, &fakeCounter{}), users.RoleUser, "user-1"))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/polls/poll-1/reports", "application/json", strings.NewReader(`{"reason":"spam"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestCreateReportConflict(t *testing.T) {
	svc := &fakeService{createFn: func(context.Context, string, moderation.CreateReportInput) (moderation.Report, error) {
		return moderation.Report{}, moderation.ErrReportAlreadyExists
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, &fakeCounter{}), users.RoleUser, "user-1"))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/polls/poll-1/reports", "application/json", strings.NewReader(`{"reason":"spam"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestListReportsRequiresAdmin(t *testing.T) {
	svc := &fakeService{}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, &fakeCounter{}), users.RoleUser, "user-1"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/reports/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestListReportsAdmin(t *testing.T) {
	svc := &fakeService{listFn: func(_ context.Context, _ moderation.ListFilter) ([]moderation.Report, error) {
		return []moderation.Report{sampleReport()}, nil
	}}
	counter := &fakeCounter{countFn: func(context.Context, moderation.ListFilter) (int, error) { return 1, nil }}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, counter), users.RoleAdmin, "admin-1"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/reports/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got reportsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Pagination.TotalItems != 1 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestSubmitReviewQuorumOutcome(t *testing.T) {
	svc := &fakeService{reviewFn: func(_ context.Context, in moderation.ReviewReportInput) (moderation.ReviewOutcome, error) {
		if in.Actor.Role != users.RoleAdmin || in.Decision != moderation.ReviewDecisionApprove {
			t.Fatalf("unexpected input: %+v", in)
		}
		report := sampleReport()
		report.Status = moderation.ReportStatusResolved
		report.ApprovalCount = 2
		report.Resolution = "poll_hidden"
		return moderation.ReviewOutcome{
			Review: moderation.ReportReview{ReportID: report.ID, AdminID: in.Actor.UserID, Decision: moderation.ReviewDecisionApprove, CreatedAt: time.Now()},
			Report: report,
		}, nil
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, &fakeCounter{}), users.RoleAdmin, "admin-2"))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/reports/rep-1/reviews", "application/json", strings.NewReader(`{"decision":"APPROVE"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got reviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Report.Status != "RESOLVED" || got.Report.ApprovalCount != 2 {
		t.Fatalf("unexpected outcome: %+v", got)
	}
}

func TestSubmitReviewSelfForbidden(t *testing.T) {
	svc := &fakeService{reviewFn: func(context.Context, moderation.ReviewReportInput) (moderation.ReviewOutcome, error) {
		return moderation.ReviewOutcome{}, moderation.ErrSelfReviewForbidden
	}}
	srv := httptest.NewServer(makeRouter(NewHandler(svc, &fakeCounter{}), users.RoleAdmin, "admin-1"))
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/reports/rep-1/reviews", "application/json", strings.NewReader(`{"decision":"REJECT"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
