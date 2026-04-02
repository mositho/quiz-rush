package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	env := setupIntegrationTest(t)

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

	env.router.ServeHTTP(responseRecorder, request)

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

func TestSubmitAnswerPersistsFinishedAnonymousScoreWithDatabase(t *testing.T) {
	env := setupIntegrationTest(t)

	sessionID := createSession(t, env.router)

	requestBody := map[string]any{
		"selectedAnswerIndex": 1,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/game/sessions/%s/answers", sessionID),
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	responseRecorder := httptest.NewRecorder()

	env.router.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", responseRecorder.Code, http.StatusOK)
	}

	var response struct {
		Session struct {
			Status          string `json:"status"`
			AnsweredCount   int    `json:"answeredQuestions"`
			CurrentScore    int    `json:"currentScore"`
			CurrentQuestion *struct {
				QuestionID string `json:"questionId"`
			} `json:"currentQuestion"`
		} `json:"session"`
		Result struct {
			Correct      bool    `json:"correct"`
			Finished     bool    `json:"finished"`
			FinishReason *string `json:"finishReason"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	if !response.Result.Correct {
		t.Fatal("expected answer to be correct")
	}
	if !response.Result.Finished {
		t.Fatal("expected session to finish after answering the only question")
	}
	if response.Session.Status != "finished" {
		t.Fatalf("got session status %q, want %q", response.Session.Status, "finished")
	}
	if response.Session.AnsweredCount != 1 {
		t.Fatalf("got answered questions %d, want %d", response.Session.AnsweredCount, 1)
	}
	if response.Session.CurrentScore <= 0 {
		t.Fatalf("got current score %d, want positive score", response.Session.CurrentScore)
	}
	if response.Session.CurrentQuestion != nil {
		t.Fatalf("got current question %+v, want nil", response.Session.CurrentQuestion)
	}

	var (
		scoreCount int
		isSaved    bool
		expiresAt  *time.Time
	)
	if err := env.pool.QueryRow(
		context.Background(),
		`
		select count(*), coalesce(bool_or(is_saved), false), max(expires_at)
		from game_scores
		where session_id = $1
		`,
		sessionID,
	).Scan(&scoreCount, &isSaved, &expiresAt); err != nil {
		t.Fatal(err)
	}
	if scoreCount != 1 {
		t.Fatalf("got score count %d, want %d", scoreCount, 1)
	}
	if isSaved {
		t.Fatal("expected anonymous score to remain unsaved")
	}
	if expiresAt == nil {
		t.Fatal("expected anonymous score expiry to be set")
	}
}

type integrationEnv struct {
	pool   *pgxpool.Pool
	router http.Handler
}

func setupIntegrationTest(t *testing.T) integrationEnv {
	t.Helper()

	if loadErr := godotenv.Load("../.env", "game-backend/.env"); loadErr != nil {
		t.Logf("skipping optional env file load: %v", loadErr)
	}

	ctx := context.Background()
	databaseURL, terminate, err := startTestDatabase(ctx)
	if err != nil {
		t.Skipf("unable to start test database container: %v", err)
	}
	t.Cleanup(func() {
		if terminateErr := terminate(ctx); terminateErr != nil {
			t.Logf("failed to terminate test database container: %v", terminateErr)
		}
	})
	t.Log("started test database container")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if pingErr := pool.Ping(ctx); pingErr != nil {
		t.Skipf("test database is not reachable: %v", pingErr)
	}
	t.Log("connected to test database")

	if migrationErr := db.RunMigrations(ctx, pool); migrationErr != nil {
		t.Fatal(migrationErr)
	}
	cleanupIntegrationTables(t, ctx, pool)
	t.Cleanup(func() { cleanupIntegrationTables(t, ctx, pool) })
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
	t.Cleanup(questionsServer.Close)

	t.Setenv("QUESTIONS_API_BASE_URL", questionsServer.URL)

	return integrationEnv{
		pool:   pool,
		router: api.NewRouter(pool, nil),
	}
}

func createSession(t *testing.T, router http.Handler) string {
	t.Helper()

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
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SessionID == "" {
		t.Fatal("expected session id")
	}

	return response.SessionID
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

func startTestDatabase(ctx context.Context) (connectionString string, terminate func(context.Context) error, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicErr, ok := recovered.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", recovered)
			}
			connectionString = ""
			terminate = nil
			err = panicErr
		}
	}()

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

	connectionString, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		if terminateErr := container.Terminate(ctx); terminateErr != nil {
			err = errors.Join(err, terminateErr)
		}
		return "", nil, err
	}

	terminate = func(terminateCtx context.Context) error {
		return container.Terminate(terminateCtx)
	}

	return connectionString, terminate, nil
}
