package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quiz-rush/game-backend/internal/api"
	"quiz-rush/game-backend/internal/middleware"
)

func TestHealthReturnsOKStatusJSON(t *testing.T) {
	router := api.NewRouter(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status field to be ok, got %q", body["status"])
	}
}

func TestLeaderboardRemainsPublicWhenAuthMiddlewareIsConfigured(t *testing.T) {
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	router := api.NewRouter(nil, authMiddleware)
	req := httptest.NewRequest(http.MethodGet, "/api/leaderboard/general-knowledge", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestCreateResultIsProtectedWhenAuthMiddlewareIsConfigured(t *testing.T) {
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	router := api.NewRouter(nil, authMiddleware)
	req := httptest.NewRequest(http.MethodPost, "/api/results", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestCreateResultReceivesAuthenticatedUserFromMiddleware(t *testing.T) {
	injectingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := middleware.AuthenticatedUser{
				Subject:           "user-123",
				PreferredUsername: "alice",
			}

			next.ServeHTTP(w, r.WithContext(middleware.WithAuthenticatedUser(r.Context(), user)))
		})
	}

	router := api.NewRouter(nil, injectingMiddleware)
	req := httptest.NewRequest(http.MethodPost, "/api/results", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var body struct {
		Status string `json:"status"`
		User   struct {
			Subject  string `json:"subject"`
			Username string `json:"username"`
		} `json:"user"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	if body.Status != "created" {
		t.Fatalf("expected status field to be created, got %q", body.Status)
	}

	if body.User.Subject != "user-123" {
		t.Fatalf("expected subject to be user-123, got %q", body.User.Subject)
	}

	if body.User.Username != "alice" {
		t.Fatalf("expected username to be alice, got %q", body.User.Username)
	}
}
