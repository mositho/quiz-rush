package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"quiz-rush/questions-backend/internal/api"
)

func TestHealthReturnsOKStatusJSON(t *testing.T) {
	router := api.NewRouter(nil)
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
