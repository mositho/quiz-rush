package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	authMiddleware, err := initAuthMiddlewareWithRetry(ctx)
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
	candidates := []string{
		"game-backend/.env",
		".env",
		filepath.Join("..", "..", ".env"),
	}

	for _, candidate := range candidates {
		if err := godotenv.Load(candidate); err == nil {
			return
		}
	}

	log.Printf("WARNING: Missing backend env file. Tried: %v", candidates)
}

func initAuthMiddlewareWithRetry(ctx context.Context) (func(http.Handler) http.Handler, error) {
	issuerURL := os.Getenv("KEYCLOAK_ISSUER_URL")
	internalIssuerURL := os.Getenv("KEYCLOAK_INTERNAL_ISSUER_URL")
	clientID := os.Getenv("KEYCLOAK_CLIENT_ID")

	maxWait := 2 * time.Minute
	if raw := os.Getenv("AUTH_INIT_MAX_WAIT"); raw != "" {
		duration, err := time.ParseDuration(raw)
		if err != nil {
			log.Printf("WARNING: Invalid AUTH_INIT_MAX_WAIT=%q, using default %s", raw, maxWait)
		} else {
			maxWait = duration
		}
	}

	retryInterval := 5 * time.Second
	if raw := os.Getenv("AUTH_INIT_RETRY_INTERVAL"); raw != "" {
		duration, err := time.ParseDuration(raw)
		if err != nil {
			log.Printf("WARNING: Invalid AUTH_INIT_RETRY_INTERVAL=%q, using default %s", raw, retryInterval)
		} else {
			retryInterval = duration
		}
	}

	deadline := time.Now().Add(maxWait)
	var lastErr error

	for attempt := 1; ; attempt++ {
		authMiddleware, err := middleware.NewOIDCAuthMiddleware(ctx, issuerURL, internalIssuerURL, clientID)
		if err == nil {
			if attempt > 1 {
				log.Printf("OIDC auth middleware initialized after %d attempts", attempt)
			}
			return authMiddleware, nil
		}

		lastErr = err
		if time.Now().After(deadline) {
			break
		}

		log.Printf("OIDC initialization attempt %d failed: %v; retrying in %s", attempt, err, retryInterval)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("oidc initialization failed within %s: %w", maxWait, lastErr)
}
