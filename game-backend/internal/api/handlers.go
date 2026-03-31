package api

import (
	"log"
	"net/http"

	"quiz-rush/game-backend/internal/httpjson"
)

func Health(w http.ResponseWriter, r *http.Request) {
	if err := httpjson.Write(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to write health response: %v", err)
	}
}
