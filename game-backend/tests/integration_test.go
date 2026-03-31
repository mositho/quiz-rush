package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quiz-rush/game-backend/internal/api"
	"quiz-rush/game-backend/internal/middleware"
)

// TestLeaderboardResponseIncludesSlug verifies that the leaderboard endpoint
// reflects the slug URL parameter in the response body, testing the full
// routing and handler pipeline.
func TestLeaderboardResponseIncludesSlug(t *testing.T) {
	router := api.NewRouter(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/leaderboard/trivia-night", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body struct {
		PackageSlug string `json:"packageSlug"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	if body.PackageSlug != "trivia-night" {
		t.Fatalf("expected packageSlug to be trivia-night, got %q", body.PackageSlug)
	}
}

// TestLeaderboardEntriesIsNeverNull verifies that the entries field is always
// a JSON array and not null, even when the leaderboard is empty.
func TestLeaderboardEntriesIsNeverNull(t *testing.T) {
	router := api.NewRouter(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/leaderboard/empty-board", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	entries, ok := body["entries"]
	if !ok {
		t.Fatalf("expected entries field in response")
	}
	if entries == nil {
		t.Fatalf("expected entries to be an array, not null")
	}
	if _, ok := entries.([]any); !ok {
		t.Fatalf("expected entries to be a JSON array, got %T", entries)
	}
}

// TestCreateResultIncludesEmailFromMiddleware verifies the full response body
// when an authenticated user with all fields set submits a result.
func TestCreateResultIncludesEmailFromMiddleware(t *testing.T) {
	injectingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := middleware.AuthenticatedUser{
				Subject:           "user-456",
				PreferredUsername: "bob",
				Email:             "bob@example.com",
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
			Email    string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	if body.Status != "created" {
		t.Fatalf("expected status to be created, got %q", body.Status)
	}
	if body.User.Subject != "user-456" {
		t.Fatalf("expected subject to be user-456, got %q", body.User.Subject)
	}
	if body.User.Username != "bob" {
		t.Fatalf("expected username to be bob, got %q", body.User.Username)
	}
	if body.User.Email != "bob@example.com" {
		t.Fatalf("expected email to be bob@example.com, got %q", body.User.Email)
	}
}

// TestCreateResultOmitsUserWhenNotAuthenticated verifies that the user field is
// absent from the response when no middleware injects an authenticated user.
func TestCreateResultOmitsUserWhenNotAuthenticated(t *testing.T) {
	router := api.NewRouter(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/results", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	if body["status"] != "created" {
		t.Fatalf("expected status to be created, got %v", body["status"])
	}
	if _, ok := body["user"]; ok {
		t.Fatalf("expected user field to be absent when not authenticated, but it was present")
	}
}

// TestGameBackendInitializesWithMockQuestionsServerURL verifies that the router
// starts successfully when given a custom questions backend URL via the
// QUESTIONS_API_BASE_URL environment variable.
func TestGameBackendInitializesWithMockQuestionsServerURL(t *testing.T) {
	mockQuestionsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockQuestionsServer.Close()

	t.Setenv("QUESTIONS_API_BASE_URL", mockQuestionsServer.URL)

	router := api.NewRouter(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}
