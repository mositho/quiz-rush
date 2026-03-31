package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"quiz-rush/questions-backend/internal/api"
	"quiz-rush/questions-backend/internal/setloader"
)

// TestAllExpectedQuestionSetsAreLoaded verifies that the questions backend
// serves all four expected question sets (lf1, lf2, lf3, lf4).
func TestAllExpectedQuestionSetsAreLoaded(t *testing.T) {
	indexer := loadTestIndexer(t)
	router := api.NewRouter(indexer)

	req := httptest.NewRequest(http.MethodGet, "/api/sets", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var body []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}

	expectedIDs := map[string]bool{
		"lf1": false,
		"lf2": false,
		"lf3": false,
		"lf4": false,
	}
	for _, set := range body {
		expectedIDs[set.ID] = true
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("expected question set %q to be present, but it was not found", id)
		}
	}
}

// TestMetadataLengthMatchesActualQuestionCount verifies that the length field
// in each set's metadata matches the actual number of questions that are loaded
// when the set is requested.
func TestMetadataLengthMatchesActualQuestionCount(t *testing.T) {
	indexer := loadTestIndexer(t)

	for _, meta := range indexer.ListSets() {
		questions, err := indexer.LoadQuestionsByID(meta.ID)
		if err != nil {
			t.Fatalf("failed to load questions for set %q: %v", meta.ID, err)
		}

		if meta.Length != len(questions) {
			t.Errorf(
				"set %q: metadata length %d does not match actual question count %d",
				meta.ID, meta.Length, len(questions),
			)
		}
	}
}

// TestQuestionsHaveValidCorrectAnswerIndex verifies that every question's
// correctAnswer index is within the bounds of its options slice.
func TestQuestionsHaveValidCorrectAnswerIndex(t *testing.T) {
	indexer := loadTestIndexer(t)

	for _, meta := range indexer.ListSets() {
		questions, err := indexer.LoadQuestionsByID(meta.ID)
		if err != nil {
			t.Fatalf("failed to load questions for set %q: %v", meta.ID, err)
		}

		for _, q := range questions {
			if q.CorrectAnswer < 0 || q.CorrectAnswer >= len(q.Options) {
				t.Errorf(
					"set %q, question %q: correctAnswer %d is out of bounds for %d options",
					meta.ID, q.ID, q.CorrectAnswer, len(q.Options),
				)
			}
		}
	}
}

// TestSetloaderWithCustomTempData verifies that the Indexer correctly loads
// question sets from a directory created at runtime with known test data.
func TestSetloaderWithCustomTempData(t *testing.T) {
	dir := t.TempDir()

	set := map[string]any{
		"id":          "test-set",
		"name":        "Test Set",
		"description": "A test question set",
		"questions": []map[string]any{
			{
				"id":            "q1",
				"difficulty":    1,
				"categories":    []string{"test"},
				"question":      "What is 1+1?",
				"options":       []string{"1", "2", "3", "4"},
				"correctAnswer": 1,
			},
		},
	}

	data, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("failed to marshal test set: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test-set.json"), data, 0600); err != nil {
		t.Fatalf("failed to write test set file: %v", err)
	}

	indexer := setloader.NewIndexer(dir)
	metadata, err := indexer.LoadAllMetadata()
	if err != nil {
		t.Fatalf("expected no error loading custom set, got: %v", err)
	}

	if len(metadata) != 1 {
		t.Fatalf("expected 1 set, got %d", len(metadata))
	}
	if metadata[0].ID != "test-set" {
		t.Fatalf("expected set ID to be test-set, got %q", metadata[0].ID)
	}
	if metadata[0].Length != 1 {
		t.Fatalf("expected metadata length to be 1, got %d", metadata[0].Length)
	}

	questions, err := indexer.LoadQuestionsByID("test-set")
	if err != nil {
		t.Fatalf("expected no error loading custom set questions, got: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
	if questions[0].ID != "q1" {
		t.Fatalf("expected question ID to be q1, got %q", questions[0].ID)
	}
}

// TestSetloaderRejectsEmptySetID verifies that the Indexer returns an error
// when a question set file is missing the required id field.
func TestSetloaderRejectsEmptySetID(t *testing.T) {
	dir := t.TempDir()

	set := map[string]any{
		"id":        "",
		"name":      "Missing ID Set",
		"questions": []any{},
	}

	data, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("failed to marshal test set: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "missing-id.json"), data, 0600); err != nil {
		t.Fatalf("failed to write test set file: %v", err)
	}

	indexer := setloader.NewIndexer(dir)
	_, err = indexer.LoadAllMetadata()

	if err == nil {
		t.Fatal("expected an error when set ID is empty, but got nil")
	}
}

// TestSetloaderRejectsDuplicateSetIDs verifies that the Indexer returns an error
// when two question set files declare the same ID.
func TestSetloaderRejectsDuplicateSetIDs(t *testing.T) {
	dir := t.TempDir()

	for _, filename := range []string{"a.json", "b.json"} {
		set := map[string]any{
			"id":        "duplicate-id",
			"name":      "Duplicate Set",
			"questions": []any{},
		}
		data, err := json.Marshal(set)
		if err != nil {
			t.Fatalf("failed to marshal test set: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, filename), data, 0600); err != nil {
			t.Fatalf("failed to write test set file: %v", err)
		}
	}

	indexer := setloader.NewIndexer(dir)
	_, err := indexer.LoadAllMetadata()

	if err == nil {
		t.Fatal("expected an error for duplicate set IDs, but got nil")
	}
}
