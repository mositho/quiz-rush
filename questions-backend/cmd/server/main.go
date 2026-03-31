package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"quiz-rush/questions-backend/internal/api"
	"quiz-rush/questions-backend/internal/setloader"

	"github.com/joho/godotenv"
)

func main() {
	loadBackendEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Invalid PORT value: %v", err)
	}
	port = strconv.Itoa(portNumber)

	setsDir := resolveSetsDir()
	indexer := setloader.NewIndexer(setsDir)
	if _, err := indexer.LoadAllMetadata(); err != nil {
		log.Fatalf("Failed to load question set metadata: %v", err)
	}

	router := api.NewRouter(indexer)

	log.Printf("Questions backend running on :%d", portNumber)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadBackendEnv() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("No .env file loaded: %v", err)
	}
	if err := godotenv.Load("questions-backend/.env"); err != nil {
		log.Printf("No questions-backend/.env file loaded: %v", err)
	}
}

func resolveSetsDir() string {
	if envPath := os.Getenv("QUESTION_SETS_DIR"); envPath != "" {
		return envPath
	}

	candidates := []string{"questionsets", "questions-backend/questionsets"}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return "questionsets"
}
