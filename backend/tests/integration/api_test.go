package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"xugeaneeu/pollify/internal/platform/app"
	"xugeaneeu/pollify/internal/platform/auth"
	"xugeaneeu/pollify/internal/platform/config"
	"xugeaneeu/pollify/internal/platform/postgrestest"
	users "xugeaneeu/pollify/internal/users/core"
)

const jwtSecret = "integration-test-secret"

type apiClient struct {
	t      *testing.T
	server *httptest.Server
	token  string
}

func (c *apiClient) request(method, path string, body any) *http.Response {
	c.t.Helper()
	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, c.server.URL+path, rdr)
	if err != nil {
		c.t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("do request: %v", err)
	}
	return resp
}

func (c *apiClient) decode(resp *http.Response, into any) {
	c.t.Helper()
	defer resp.Body.Close()
	if into == nil {
		return
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		c.t.Fatalf("decode response: %v", err)
	}
}

func TestEndToEndModerationFlow(t *testing.T) {
	pool := postgrestest.OpenTestDB(t)

	cfg := config.Config{HTTPAddress: ":0", JWTSecret: jwtSecret, OpenAPIPath: "../../api/openapi.yaml"}
	handler, err := app.BuildHandler(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), pool)
	if err != nil {
		t.Fatalf("build handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	c := &apiClient{t: t, server: server}

	// register voter
	resp := c.request(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":        "voter@example.com",
		"password":     "supersecret",
		"display_name": "Voter",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register voter: status %d", resp.StatusCode)
	}
	var voterSession struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	c.decode(resp, &voterSession)

	// register reporter
	resp = c.request(http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":        "reporter@example.com",
		"password":     "supersecret",
		"display_name": "Reporter",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register reporter: status %d", resp.StatusCode)
	}
	var reporterSession struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	c.decode(resp, &reporterSession)

	// seed two admins directly + mint tokens
	admin1ID := "11111111-aaaa-aaaa-aaaa-111111111111"
	admin2ID := "22222222-aaaa-aaaa-aaaa-222222222222"
	postgrestest.SeedUser(t, pool, admin1ID, "admin1@example.com", "ADMIN")
	postgrestest.SeedUser(t, pool, admin2ID, "admin2@example.com", "ADMIN")
	issuer := auth.NewHMACIssuer([]byte(jwtSecret), time.Hour, time.Now)
	admin1Tokens, err := issuer.IssueTokens(context.Background(), users.User{ID: admin1ID, Role: users.RoleAdmin})
	if err != nil {
		t.Fatalf("issue admin1: %v", err)
	}
	admin2Tokens, err := issuer.IssueTokens(context.Background(), users.User{ID: admin2ID, Role: users.RoleAdmin})
	if err != nil {
		t.Fatalf("issue admin2: %v", err)
	}

	// create poll as voter
	c.token = voterSession.Tokens.AccessToken
	startAt := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	endAt := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	resp = c.request(http.MethodPost, "/api/v1/polls/", map[string]any{
		"title":    "Lunch?",
		"question": "Pick a place",
		"options":  []map[string]any{{"text": "Pizza"}, {"text": "Sushi"}},
		"settings": map[string]any{
			"is_anonymous":        false,
			"is_multiple_choice":  false,
			"allow_custom_answer": false,
			"start_at":            startAt,
			"end_at":              endAt,
		},
	})
	if resp.StatusCode != http.StatusCreated {
		dumpBody(t, resp)
		t.Fatalf("create poll: status %d", resp.StatusCode)
	}
	var pollDetails struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Options []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"options"`
	}
	c.decode(resp, &pollDetails)
	if pollDetails.Status != "active" {
		t.Fatalf("expected active poll, got %s", pollDetails.Status)
	}
	if len(pollDetails.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(pollDetails.Options))
	}

	// vote as voter (single choice)
	resp = c.request(http.MethodPost, "/api/v1/polls/"+pollDetails.ID+"/votes", map[string]any{
		"option_ids": []string{pollDetails.Options[0].ID},
	})
	if resp.StatusCode != http.StatusCreated {
		dumpBody(t, resp)
		t.Fatalf("submit vote: status %d", resp.StatusCode)
	}

	// double vote rejected
	resp = c.request(http.MethodPost, "/api/v1/polls/"+pollDetails.ID+"/votes", map[string]any{
		"option_ids": []string{pollDetails.Options[0].ID},
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 on double vote, got %d", resp.StatusCode)
	}

	// reporter files report
	c.token = reporterSession.Tokens.AccessToken
	resp = c.request(http.MethodPost, "/api/v1/polls/"+pollDetails.ID+"/reports", map[string]any{
		"reason": "inappropriate",
	})
	if resp.StatusCode != http.StatusCreated {
		dumpBody(t, resp)
		t.Fatalf("create report: status %d", resp.StatusCode)
	}
	var reportCreated struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	c.decode(resp, &reportCreated)

	// admin1 approves
	c.token = admin1Tokens.AccessToken
	resp = c.request(http.MethodPost, "/api/v1/reports/"+reportCreated.ID+"/reviews", map[string]any{
		"decision": "APPROVE",
	})
	if resp.StatusCode != http.StatusCreated {
		dumpBody(t, resp)
		t.Fatalf("admin1 review: status %d", resp.StatusCode)
	}
	var firstReview struct {
		Report struct {
			Status        string `json:"status"`
			ApprovalCount int    `json:"approval_count"`
		} `json:"report"`
	}
	c.decode(resp, &firstReview)
	if firstReview.Report.Status != "IN_REVIEW" || firstReview.Report.ApprovalCount != 1 {
		t.Fatalf("after admin1 review unexpected report: %+v", firstReview.Report)
	}

	// admin1 cannot review twice
	resp = c.request(http.MethodPost, "/api/v1/reports/"+reportCreated.ID+"/reviews", map[string]any{
		"decision": "APPROVE",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate review, got %d", resp.StatusCode)
	}

	// admin2 approves -> quorum reached
	c.token = admin2Tokens.AccessToken
	resp = c.request(http.MethodPost, "/api/v1/reports/"+reportCreated.ID+"/reviews", map[string]any{
		"decision": "APPROVE",
	})
	if resp.StatusCode != http.StatusCreated {
		dumpBody(t, resp)
		t.Fatalf("admin2 review: status %d", resp.StatusCode)
	}
	var resolved struct {
		Report struct {
			Status     string `json:"status"`
			Resolution string `json:"resolution"`
		} `json:"report"`
	}
	c.decode(resp, &resolved)
	if resolved.Report.Status != "RESOLVED" || resolved.Report.Resolution != "poll_hidden" {
		t.Fatalf("expected resolved+poll_hidden, got %+v", resolved.Report)
	}

	// poll is now hidden — voter sees status hidden
	c.token = voterSession.Tokens.AccessToken
	resp = c.request(http.MethodGet, "/api/v1/polls/"+pollDetails.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get hidden poll: status %d", resp.StatusCode)
	}
	var fetched struct {
		Status   string `json:"status"`
		IsHidden bool   `json:"is_hidden"`
	}
	c.decode(resp, &fetched)
	if !fetched.IsHidden || fetched.Status != "hidden" {
		t.Fatalf("expected poll hidden, got %+v", fetched)
	}

	// non-anonymous results: voter_details available with include_voters=true
	resp = c.request(http.MethodGet, "/api/v1/polls/"+pollDetails.ID+"/results?include_voters=true", nil)
	if resp.StatusCode != http.StatusOK {
		dumpBody(t, resp)
		t.Fatalf("get results: status %d", resp.StatusCode)
	}
	var results struct {
		ParticipantsCount int `json:"participants_count"`
		VoterDetails      []struct {
			UserID string `json:"user_id"`
		} `json:"voter_details"`
	}
	c.decode(resp, &results)
	if results.ParticipantsCount != 1 || len(results.VoterDetails) != 1 || results.VoterDetails[0].UserID != voterSession.User.ID {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func dumpBody(t *testing.T, resp *http.Response) {
	t.Helper()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("response body: %s", string(body))
}
