package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"quiz-rush/backend/internal/api"
	"quiz-rush/backend/internal/db"

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := api.NewRouter(pool)

	log.Printf("Backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadBackendEnv() {
	if err := godotenv.Load("backend/.env"); err != nil {
		log.Fatalf("Missing backend/.env: %v", err)
	}
}
