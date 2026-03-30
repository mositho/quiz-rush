package api

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"quiz-rush/questions-backend/internal/setloader"
)

func NewRouter(loader *setloader.Indexer) http.Handler {
	r := chi.NewRouter()

	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	handler := NewHandler(loader)

	r.Get("/health", handler.Health)
	r.Route("/api", func(api chi.Router) {
		api.Get("/sets", handler.GetSets)
		api.Get("/sets/{id}", handler.GetSetQuestions)
	})

	return r
}
