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
	_ = godotenv.Load()

	ctx := context.Background()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := api.NewRouter(pool)

	log.Printf("backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}