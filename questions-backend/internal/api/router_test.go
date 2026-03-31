package api_test

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

func TestGetSetsReturnsMetadata(t *testing.T) {
	indexer := loadTestIndexer(t)

	router := api.NewRouter(indexer)
	request := httptest.NewRequest(http.MethodGet, "/api/sets", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatal("unexpected status code")
	}

	var body []map[string]any
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if len(body) == 0 {
		t.Fatal("unexpected response body")
	}

	first := body[0]
	if _, ok := first["id"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["name"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["description"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["length"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["questions"]; ok {
		t.Fatal("unexpected field")
	}
}

func TestGetSetQuestionsReturnsArray(t *testing.T) {
	indexer := loadTestIndexer(t)

	router := api.NewRouter(indexer)
	request := httptest.NewRequest(http.MethodGet, "/api/sets/lf1", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatal("unexpected status code")
	}

	var body []map[string]any
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if len(body) == 0 {
		t.Fatal("unexpected response body")
	}

	first := body[0]
	if _, ok := first["id"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["difficulty"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["categories"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["question"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["options"]; !ok {
		t.Fatal("missing field")
	}
	if _, ok := first["correctAnswer"]; !ok {
		t.Fatal("missing field")
	}
}

func TestGetSetQuestionsUnknownIDReturns404(t *testing.T) {
	indexer := loadTestIndexer(t)

	router := api.NewRouter(indexer)
	request := httptest.NewRequest(http.MethodGet, "/api/sets/does-not-exist", nil)
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusNotFound {
		t.Fatal("unexpected status code")
	}
}

func loadTestIndexer(t *testing.T) *setloader.Indexer {
	t.Helper()

	candidates := []string{"questionsets", "../questionsets", "../../questionsets"}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			indexer := setloader.NewIndexer(candidate)
			if _, err := indexer.LoadAllMetadata(); err != nil {
				t.Fatal(err)
			}
			return indexer
		}
	}

	t.Fatal("questionsets directory not found")
	return nil
}
