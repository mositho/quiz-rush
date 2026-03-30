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

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	handler := NewHandler(db)

	r.Get("/health", handler.Health)
	r.Route("/api", func(api chi.Router) {
		api.Get("/packages", handler.GetPackages)
		api.Get("/packages/{slug}/questions", handler.GetQuestionsByPackage)
	})

	return r
}
