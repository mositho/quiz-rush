package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"quiz-rush/game-backend/internal/api"
	"quiz-rush/game-backend/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestCreateSessionWithDatabase(t *testing.T) {
	if loadErr := godotenv.Load("../.env", "game-backend/.env"); loadErr != nil {
		t.Logf("skipping optional env file load: %v", loadErr)
	}

	databaseURL := os.Getenv("GAME_BACKEND_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set GAME_BACKEND_TEST_DATABASE_URL to run DB-backed integration tests")
	}
	t.Log("using GAME_BACKEND_TEST_DATABASE_URL")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if pingErr := pool.Ping(ctx); pingErr != nil {
		t.Skipf("test database is not reachable: %v", pingErr)
	}
	t.Log("connected to test database")

	if migrationErr := db.RunMigrations(ctx, pool); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	cleanupIntegrationTables(t, ctx, pool)
	defer cleanupIntegrationTables(t, ctx, pool)
	t.Log("applied migrations and cleared test tables")

	questionsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/sets/lf1" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if encodeErr := json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":            "q1",
				"difficulty":    2,
				"categories":    []string{"lf1"},
				"question":      "Question one",
				"options":       []string{"A", "B", "C", "D"},
				"correctAnswer": 1,
			},
		}); encodeErr != nil {
			t.Fatal(encodeErr)
		}
	}))
	defer questionsServer.Close()

	t.Setenv("QUESTIONS_API_BASE_URL", questionsServer.URL)

	router := api.NewRouter(pool, nil)
	t.Log("created router with real database and stub questions API")

	requestBody := map[string]any{
		"durationSeconds":        180,
		"selectedQuestionSetIds": []string{"lf1"},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/game/sessions", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	responseRecorder := httptest.NewRecorder()

	router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusCreated {
		t.Fatalf("got status %d, want %d", responseRecorder.Code, http.StatusCreated)
	}

	var response struct {
		SessionID       string `json:"sessionId"`
		Status          string `json:"status"`
		CurrentQuestion *struct {
			QuestionID string `json:"questionId"`
		} `json:"currentQuestion"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SessionID == "" {
		t.Fatal("expected session id")
	}
	if response.Status != "active" {
		t.Fatalf("got status %q, want %q", response.Status, "active")
	}
	if response.CurrentQuestion == nil || response.CurrentQuestion.QuestionID != "q1" {
		t.Fatalf("got current question %+v, want q1", response.CurrentQuestion)
	}
	t.Log("created session through real HTTP + database flow")
}

func cleanupIntegrationTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	statements := []string{
		"delete from game_scores",
		"delete from game_session_questions",
		"delete from game_sessions",
		"delete from user_profiles",
	}

	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
}
