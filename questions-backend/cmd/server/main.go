package main

import (
	"log"
	"net/http"
	"os"

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

	setsDir := resolveSetsDir()
	indexer := setloader.NewIndexer(setsDir)
	if _, err := indexer.LoadAllMetadata(); err != nil {
		log.Fatalf("Failed to load question set metadata: %v", err)
	}

	router := api.NewRouter(indexer)

	log.Printf("Questions backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadBackendEnv() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("questions-backend/.env")
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
