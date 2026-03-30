package api

import (
	"net/http"

	"quiz-rush/game-backend/internal/httpjson"
)

func Health(w http.ResponseWriter, r *http.Request) {
	httpjson.Write(w, http.StatusOK, map[string]string{"status": "ok"})
}
