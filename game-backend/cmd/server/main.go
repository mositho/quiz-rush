package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

	if migrationErr := db.RunMigrations(ctx, pool); migrationErr != nil {
		log.Fatalf("Failed to run migrations: %v", migrationErr)
	}

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
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Invalid PORT value: %v", err)
	}
	port = strconv.Itoa(portNumber)

	router := api.NewRouter(pool, authMiddleware)
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Backend running on :%d", portNumber)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadBackendEnv() {
	if err := godotenv.Load("game-backend/.env"); err != nil {
		log.Printf("WARNING: Missing game-backend/.env: %v", err)
	}
}
