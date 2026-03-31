package game

import (
	"log"
	"net/http"

	"quiz-rush/game-backend/internal/httpjson"
	"quiz-rush/game-backend/internal/middleware"
	"quiz-rush/game-backend/internal/questionsclient"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db              *pgxpool.Pool
	questionsClient *questionsclient.Client
}

type CreateResultResponse struct {
	Status string                    `json:"status"`
	User   *CreateResultUserResponse `json:"user,omitempty"`
}

type CreateResultUserResponse struct {
	Subject  string `json:"subject"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
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
	response := CreateResultResponse{Status: "created"}

	if user, ok := middleware.AuthenticatedUserFromContext(r.Context()); ok {
		response.User = &CreateResultUserResponse{
			Subject:  user.Subject,
			Username: user.PreferredUsername,
			Email:    user.Email,
		}
	}

	if err := httpjson.Write(w, http.StatusCreated, response); err != nil {
		log.Printf("failed to write create result response: %v", err)
	}
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	if err := httpjson.Write(w, http.StatusOK, LeaderboardResponse{
		PackageSlug: slug,
		Entries:     []LeaderboardEntry{},
	}); err != nil {
		log.Printf("failed to write leaderboard response: %v", err)
	}
}
