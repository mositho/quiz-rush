package api

import (
	"net/http"
	"os"

	game "quiz-rush/game-backend/internal/game"
	"quiz-rush/game-backend/internal/questionsapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(db *pgxpool.Pool, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()

	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	questionsBaseURL := os.Getenv("QUESTIONS_API_BASE_URL")
	if questionsBaseURL == "" {
		questionsBaseURL = "http://localhost:8081"
	}

	gameHandler := game.NewHandler(db, questionsapi.New(questionsBaseURL))

	r.Get("/health", Health)
	r.Route("/api", func(api chi.Router) {
		registerGameRoutes := func(gameRouter chi.Router) {
			gameRouter.Get("/question-sets", gameHandler.GetQuestionSets)
			gameRouter.Post("/sessions", gameHandler.StartSession)
			gameRouter.Get("/sessions/{sessionId}", gameHandler.GetSession)
			gameRouter.Post("/sessions/{sessionId}/answers", gameHandler.SubmitAnswer)
			gameRouter.Post("/sessions/{sessionId}/finish", gameHandler.FinishSession)
			gameRouter.Post("/sessions/{sessionId}/quit", gameHandler.QuitSession)
			gameRouter.Post("/sessions/{sessionId}/link-account", gameHandler.LinkAccount)
			gameRouter.Get("/scores/{scoreId}", gameHandler.GetScore)
			gameRouter.Get("/leaderboards", gameHandler.GetLeaderboard)
			gameRouter.Get("/users/me", gameHandler.GetCurrentUser)
			gameRouter.Get("/users/{publicUserId}/scores", gameHandler.GetUserScores)
			gameRouter.Get("/users/{publicUserId}/stats", gameHandler.GetUserStats)
		}

		if authMiddleware != nil {
			api.With(authMiddleware).Route("/game", registerGameRoutes)
			return
		}

		api.Route("/game", registerGameRoutes)
	})

	return r
}
