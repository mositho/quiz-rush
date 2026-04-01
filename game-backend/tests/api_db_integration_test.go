package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"quiz-rush/game-backend/internal/api"
	"quiz-rush/game-backend/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCreateSessionWithDatabase(t *testing.T) {
	if loadErr := godotenv.Load("../.env", "game-backend/.env"); loadErr != nil {
		t.Logf("skipping optional env file load: %v", loadErr)
	}

	ctx := context.Background()
	databaseURL, terminate, err := startTestDatabase(ctx)
	if err != nil {
		t.Skipf("unable to start test database container: %v", err)
	}
	defer func() {
		if terminateErr := terminate(ctx); terminateErr != nil {
			t.Logf("failed to terminate test database container: %v", terminateErr)
		}
	}()
	t.Log("started test database container")

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

func startTestDatabase(ctx context.Context) (string, func(context.Context) error, error) {
	dbName := "quiz_rush_game_test"
	username := "quiz_rush_game_test"
	password := "quiz_rush_game_test"

	container, err := postgres.Run(
		ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
				wait.ForLog("database system is ready to accept connections"),
			).WithDeadline(60*time.Second),
		),
	)
	if err != nil {
		return "", nil, err
	}

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return "", nil, err
	}

	terminate := func(terminateCtx context.Context) error {
		return container.Terminate(terminateCtx)
	}

	return connectionString, terminate, nil
}
