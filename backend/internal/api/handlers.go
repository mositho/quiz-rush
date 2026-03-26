package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetPackages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (h *Handler) GetQuestionsByPackage(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "slug")
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (h *Handler) CreateResult(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "slug")
	writeJSON(w, http.StatusOK, []any{})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}