package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"quiz-rush/questions-backend/internal/setloader"
)

type Handler struct {
	loader *setloader.Indexer
}

func NewHandler(loader *setloader.Indexer) *Handler {
	return &Handler{loader: loader}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetSets(w http.ResponseWriter, r *http.Request) {
	if h.loader == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "set loader is not configured"})
		return
	}

	writeJSON(w, http.StatusOK, h.loader.ListSets())
}

func (h *Handler) GetSetQuestions(w http.ResponseWriter, r *http.Request) {
	if h.loader == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "set loader is not configured"})
		return
	}

	id := chi.URLParam(r, "id")
	questions, err := h.loader.LoadQuestionsByID(id)
	if err != nil {
		if errors.Is(err, setloader.ErrSetNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "set not found"})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load set questions"})
		return
	}

	writeJSON(w, http.StatusOK, questions)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
