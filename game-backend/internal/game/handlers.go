package game

import (
	"net/http"

	"quiz-rush/game-backend/internal/httpjson"
	"quiz-rush/game-backend/internal/questionsclient"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db              *pgxpool.Pool
	questionsClient *questionsclient.Client
}

type CreateResultResponse struct {
	Status string `json:"status"`
}

type LeaderboardEntry struct {
	Player string `json:"player"`
	Score  int    `json:"score"`
}

type LeaderboardResponse struct {
	PackageSlug string             `json:"packageSlug"`
	Entries     []LeaderboardEntry `json:"entries"`
}

func NewHandler(db *pgxpool.Pool, questionsClient *questionsclient.Client) *Handler {
	return &Handler{db: db, questionsClient: questionsClient}
}

func (h *Handler) CreateResult(w http.ResponseWriter, r *http.Request) {
	httpjson.Write(w, http.StatusCreated, CreateResultResponse{Status: "created"})
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	httpjson.Write(w, http.StatusOK, LeaderboardResponse{
		PackageSlug: slug,
		Entries:     []LeaderboardEntry{},
	})
}
