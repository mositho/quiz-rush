package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quiz-rush/game-backend/internal/api"
)

func TestHealthReturnsOKStatusJSON(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", responseRecorder.Code, http.StatusOK)
	}
	if responseRecorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("got content type %q, want application/json", responseRecorder.Header().Get("Content-Type"))
	}

	var body map[string]string
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("got body %v, want status=ok", body)
	}
}

func TestGameEndpointsReturnServiceUnavailableWhenDatabaseIsMissing(t *testing.T) {
	router := api.NewRouter(nil, nil)

	testCases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "question sets", method: http.MethodGet, path: "/api/game/question-sets"},
		{name: "start session", method: http.MethodPost, path: "/api/game/sessions"},
		{name: "get session", method: http.MethodGet, path: "/api/game/sessions/session-123"},
		{name: "submit answer", method: http.MethodPost, path: "/api/game/sessions/session-123/answers"},
		{name: "finish session", method: http.MethodPost, path: "/api/game/sessions/session-123/finish"},
		{name: "quit session", method: http.MethodPost, path: "/api/game/sessions/session-123/quit"},
		{name: "link account", method: http.MethodPost, path: "/api/game/sessions/session-123/link-account"},
		{name: "get score", method: http.MethodGet, path: "/api/game/scores/score-123"},
		{name: "leaderboard", method: http.MethodGet, path: "/api/game/leaderboards"},
		{name: "current user", method: http.MethodGet, path: "/api/game/users/me"},
		{name: "user scores", method: http.MethodGet, path: "/api/game/users/user-123/scores"},
		{name: "user stats", method: http.MethodGet, path: "/api/game/users/user-123/stats"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			responseRecorder := httptest.NewRecorder()

			router.ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("got status %d, want %d", responseRecorder.Code, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestRouterSetsConfiguredCORSHeaders(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "http://localhost:5173")

	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf(
			"got allow origin %q, want %q",
			responseRecorder.Header().Get("Access-Control-Allow-Origin"),
			"http://localhost:5173",
		)
	}
}
