package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	newHandler(nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected health payload, got %q", rr.Body.String())
	}
}

func TestProtectedRouteRequiresAuthorization(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rr := httptest.NewRecorder()

	newHandler(nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}
