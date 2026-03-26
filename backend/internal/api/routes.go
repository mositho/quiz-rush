package api

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	h := NewHandler(db)

	r.Get("/health", h.Health)
	r.Get("/api/packages", h.GetPackages)
	r.Get("/api/packages/{slug}/questions", h.GetQuestionsByPackage)
	r.Post("/api/results", h.CreateResult)
	r.Get("/api/leaderboard/{slug}", h.GetLeaderboard)

	return r
}