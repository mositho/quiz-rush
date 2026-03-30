package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"quiz-rush/questions-backend/internal/api"
	"quiz-rush/questions-backend/internal/db"

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

	log.Printf("Questions backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadBackendEnv() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("questions-backend/.env")
}
