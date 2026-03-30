package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"quiz-rush/game-backend/internal/api"
	"quiz-rush/game-backend/internal/db"
	"quiz-rush/game-backend/internal/middleware"

	"github.com/joho/godotenv"
)

func main() {
	loadBackendEnv()

	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	authMiddleware, err := middleware.NewOIDCAuthMiddleware(
		ctx,
		os.Getenv("KEYCLOAK_ISSUER_URL"),
		os.Getenv("KEYCLOAK_INTERNAL_ISSUER_URL"),
		os.Getenv("KEYCLOAK_CLIENT_ID"),
	)
	if err != nil {
		log.Fatalf("Failed to configure auth middleware: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := api.NewRouter(pool, authMiddleware)

	log.Printf("Backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadBackendEnv() {
	if err := godotenv.Load("game-backend/.env"); err != nil {
		log.Printf("WARNING: Missing game-backend/.env: %v", err)
	}
}
