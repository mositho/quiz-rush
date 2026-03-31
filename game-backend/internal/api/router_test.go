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
		t.Fatal("unexpected status code")
	}

	if responseRecorder.Header().Get("Content-Type") != "application/json" {
		t.Fatal("unexpected content type")
	}

	var body map[string]string
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if body["status"] != "ok" {
		t.Fatal("unexpected response body")
	}
}

func TestStartSessionRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/game/sessions", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestGetSessionRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/game/sessions/session-123", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestSubmitAnswerRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/game/sessions/session-123/answers", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestFinishSessionRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/game/sessions/session-123/finish", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestQuitSessionRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/game/sessions/session-123/quit", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestLinkAccountRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/game/sessions/session-123/link-account", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestGetScoreRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/game/scores/score-123", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestGetLeaderboardRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/game/leaderboards", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestGetCurrentUserRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/game/users/me", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestGetUserScoresRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/game/users/user_123/scores", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}

func TestGetUserStatsRouteIsRegistered(t *testing.T) {
	router := api.NewRouter(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/game/users/user_123/stats", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusServiceUnavailable {
		t.Fatal("unexpected status code")
	}
}
