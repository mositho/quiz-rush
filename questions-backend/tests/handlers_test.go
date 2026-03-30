package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"quiz-rush/questions-backend/internal/api"
	"quiz-rush/questions-backend/internal/setloader"
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

func TestGetSetsReturnsMetadata(t *testing.T) {
	indexer := loadTestIndexer(t)

	router := api.NewRouter(indexer)
	req := httptest.NewRequest(http.MethodGet, "/api/sets", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	if len(body) == 0 {
		t.Fatalf("expected at least one set metadata entry")
	}

	first := body[0]
	if _, ok := first["id"]; !ok {
		t.Fatalf("expected id field")
	}
	if _, ok := first["name"]; !ok {
		t.Fatalf("expected name field")
	}
	if _, ok := first["description"]; !ok {
		t.Fatalf("expected description field")
	}
	if _, ok := first["length"]; !ok {
		t.Fatalf("expected length field")
	}
	if _, ok := first["questions"]; ok {
		t.Fatalf("did not expect questions field in metadata response")
	}
}

func TestGetSetQuestionsReturnsArray(t *testing.T) {
	indexer := loadTestIndexer(t)

	router := api.NewRouter(indexer)
	req := httptest.NewRequest(http.MethodGet, "/api/sets/lf1", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	if len(body) == 0 {
		t.Fatalf("expected at least one question")
	}

	first := body[0]
	if _, ok := first["id"]; !ok {
		t.Fatalf("expected id field")
	}
	if _, ok := first["difficulty"]; !ok {
		t.Fatalf("expected difficulty field")
	}
	if _, ok := first["categories"]; !ok {
		t.Fatalf("expected categories field")
	}
	if _, ok := first["question"]; !ok {
		t.Fatalf("expected question field")
	}
	if _, ok := first["options"]; !ok {
		t.Fatalf("expected options field")
	}
	if _, ok := first["correctAnswer"]; !ok {
		t.Fatalf("expected correctAnswer field")
	}
}

func TestGetSetQuestionsUnknownIDReturns404(t *testing.T) {
	indexer := loadTestIndexer(t)

	router := api.NewRouter(indexer)
	req := httptest.NewRequest(http.MethodGet, "/api/sets/does-not-exist", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func loadTestIndexer(t *testing.T) *setloader.Indexer {
	t.Helper()

	candidates := []string{"questionsets", "../questionsets"}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			indexer := setloader.NewIndexer(candidate)
			if _, err := indexer.LoadAllMetadata(); err != nil {
				t.Fatalf("failed to load metadata: %v", err)
			}
			return indexer
		}
	}

	t.Fatalf("questionsets directory not found")
	return nil
}
