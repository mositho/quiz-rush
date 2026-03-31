package game

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"quiz-rush/game-backend/internal/httpjson"
	"quiz-rush/game-backend/internal/middleware"
	"quiz-rush/game-backend/internal/questionsapi"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db              *pgxpool.Pool
	questionsClient *questionsapi.Client
	service         *Service
	repository      *sessionRepository
}

type startSessionRequest struct {
	DurationSeconds        int      `json:"durationSeconds"`
	SelectedQuestionSetIDs []string `json:"selectedQuestionSetIds"`
}

type sessionResponse struct {
	SessionID              string        `json:"sessionId"`
	Status                 SessionStatus `json:"status"`
	FinishReason           *FinishReason `json:"finishReason,omitempty"`
	StartedAt              string        `json:"startedAt"`
	EndsAt                 string        `json:"endsAt"`
	CooldownUntil          *string       `json:"cooldownUntil,omitempty"`
	DurationSeconds        int           `json:"durationSeconds"`
	SelectedQuestionSetIDs []string      `json:"selectedQuestionSetIds"`
	CurrentQuestionIndex   *int          `json:"currentQuestionIndex,omitempty"`
	TotalQuestions         int           `json:"totalQuestions"`
	AnsweredQuestions      int           `json:"answeredQuestions"`
	CorrectQuestions       int           `json:"correctQuestions"`
	WrongQuestions         int           `json:"wrongQuestions"`
	CurrentScore           int           `json:"currentScore"`
	CurrentQuestion        *Question     `json:"currentQuestion,omitempty"`
}

type answerRequest struct {
	SelectedAnswerIndex int `json:"selectedAnswerIndex"`
}

type answerResponse struct {
	Session sessionResponse `json:"session"`
	Result  AnswerResult    `json:"result"`
}

func NewHandler(db *pgxpool.Pool, questionsClient *questionsapi.Client) *Handler {
	return &Handler{
		db:              db,
		questionsClient: questionsClient,
		service:         NewService(questionsClient),
		repository:      newSessionRepository(db),
	}
}

func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	var request startSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	session, err := h.service.StartSession(r.Context(), request.DurationSeconds, request.SelectedQuestionSetIDs, now)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var (
		ownerProfileID *string
		isAnonymous    = true
		saveDeadlineAt *time.Time
	)
	if user, ok := middleware.AuthenticatedUserFromContext(r.Context()); ok {
		profileID, err := h.repository.EnsureUserProfile(r.Context(), user)
		if err != nil {
			httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to ensure user profile"})
			return
		}
		ownerProfileID = &profileID
		isAnonymous = false
	}

	if isAnonymous {
		deadline := now.Add(15 * time.Minute)
		saveDeadlineAt = &deadline
		session.SaveDeadlineAt = saveDeadlineAt
	}

	if err := h.repository.CreateSession(r.Context(), session, ownerProfileID, isAnonymous, saveDeadlineAt); err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	httpjson.Write(w, http.StatusCreated, buildSessionResponse(session, now))
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	now := time.Now().UTC()
	session, _, _, err := h.repository.LoadSession(r.Context(), chi.URLParam(r, "sessionId"))
	if errors.Is(err, ErrSessionNotFound) {
		httpjson.Write(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
		return
	}

	session.Sync(now)
	if err := h.repository.UpdateSession(r.Context(), session); err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to update session"})
		return
	}

	httpjson.Write(w, http.StatusOK, buildSessionResponse(session, now))
}

func (h *Handler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	var request answerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	session, ownerProfileID, isAnonymous, err := h.repository.LoadSession(r.Context(), chi.URLParam(r, "sessionId"))
	if errors.Is(err, ErrSessionNotFound) {
		httpjson.Write(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
		return
	}

	result, err := session.SubmitAnswer(now, request.SelectedAnswerIndex)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.repository.UpdateSession(r.Context(), session); err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to update session"})
		return
	}

	if session.Status == SessionStatusFinished {
		if err := h.persistScore(r, session, ownerProfileID, isAnonymous); err != nil {
			httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to create score"})
			return
		}
	}

	httpjson.Write(w, http.StatusOK, answerResponse{
		Session: buildSessionResponse(session, now),
		Result:  result,
	})
}

func (h *Handler) FinishSession(w http.ResponseWriter, r *http.Request) {
	h.finishWithReason(w, r, FinishReasonManualFinish)
}

func (h *Handler) QuitSession(w http.ResponseWriter, r *http.Request) {
	h.finishWithReason(w, r, FinishReasonQuit)
}

func (h *Handler) finishWithReason(w http.ResponseWriter, r *http.Request, reason FinishReason) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	now := time.Now().UTC()
	session, ownerProfileID, isAnonymous, err := h.repository.LoadSession(r.Context(), chi.URLParam(r, "sessionId"))
	if errors.Is(err, ErrSessionNotFound) {
		httpjson.Write(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
		return
	}

	session.Finish(now, reason)
	if err := h.repository.UpdateSession(r.Context(), session); err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to update session"})
		return
	}
	if err := h.persistScore(r, session, ownerProfileID, isAnonymous); err != nil {
		httpjson.Write(w, http.StatusInternalServerError, map[string]string{"error": "failed to create score"})
		return
	}

	httpjson.Write(w, http.StatusOK, buildSessionResponse(session, now))
}

func (h *Handler) persistScore(r *http.Request, session *Session, ownerProfileID *string, isAnonymous bool) error {
	scoreResult := session.ScoreResult()

	var (
		isSaved  bool
		isPublic bool
		savedAt  *time.Time
		expiresAt *time.Time
	)
	if isAnonymous {
		expiresAt = session.SaveDeadlineAt
	} else {
		isSaved = true
		isPublic = true
		now := time.Now().UTC()
		savedAt = &now
	}

	return h.repository.CreateScore(r.Context(), session.ID, ownerProfileID, scoreResult, isSaved, isPublic, savedAt, expiresAt)
}

func buildSessionResponse(session *Session, now time.Time) sessionResponse {
	var cooldownUntil *string
	if session.CooldownUntil != nil {
		value := session.CooldownUntil.UTC().Format(time.RFC3339Nano)
		cooldownUntil = &value
	}

	currentQuestion, _ := session.CurrentQuestion(now)

	return sessionResponse{
		SessionID:              session.ID,
		Status:                 session.Status,
		FinishReason:           session.FinishReason,
		StartedAt:              session.StartedAt.UTC().Format(time.RFC3339Nano),
		EndsAt:                 session.EndsAt.UTC().Format(time.RFC3339Nano),
		CooldownUntil:          cooldownUntil,
		DurationSeconds:        session.DurationSeconds,
		SelectedQuestionSetIDs: session.SelectedQuestionSetIDs,
		CurrentQuestionIndex:   session.CurrentQuestionIndex,
		TotalQuestions:         session.TotalQuestions,
		AnsweredQuestions:      session.AnsweredQuestions,
		CorrectQuestions:       session.CorrectQuestions,
		WrongQuestions:         session.WrongQuestions,
		CurrentScore:           session.CurrentScore,
		CurrentQuestion:        currentQuestion,
	}
}

func writeServiceUnavailable(w http.ResponseWriter) {
	httpjson.Write(w, http.StatusServiceUnavailable, map[string]string{"error": "game backend is not fully configured"})
}
