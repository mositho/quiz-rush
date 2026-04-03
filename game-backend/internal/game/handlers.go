package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"quiz-rush/game-backend/internal/httpjson"
	"quiz-rush/game-backend/internal/middleware"
	"quiz-rush/game-backend/internal/questionsapi"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db              *pgxpool.Pool
	service         *Service
	questionsClient *questionsapi.Client
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

type publicUserResponse struct {
	PublicUserID string `json:"publicUserId"`
	DisplayName  string `json:"displayName"`
}

type scoreResponse struct {
	ScoreID                string                `json:"scoreId"`
	SessionID              string                `json:"sessionId"`
	FinishedAt             string                `json:"finishedAt"`
	FinishReason           FinishReason          `json:"finishReason"`
	Score                  int                   `json:"score"`
	CorrectQuestions       int                   `json:"correctQuestions"`
	WrongQuestions         int                   `json:"wrongQuestions"`
	AnsweredQuestions      int                   `json:"answeredQuestions"`
	TotalQuestions         int                   `json:"totalQuestions"`
	DurationSeconds        int                   `json:"durationSeconds"`
	PlayedMs               int64                 `json:"playedMs"`
	SelectedQuestionSetIDs []string              `json:"selectedQuestionSetIds"`
	ConfigurationKey       string                `json:"configurationKey"`
	QuestionResults        []ScoreQuestionResult `json:"questionResults"`
	Player                 *publicUserResponse   `json:"player,omitempty"`
}

type scoreSummaryResponse struct {
	ScoreID                string       `json:"scoreId"`
	SessionID              string       `json:"sessionId"`
	FinishedAt             string       `json:"finishedAt"`
	FinishReason           FinishReason `json:"finishReason"`
	Score                  int          `json:"score"`
	CorrectQuestions       int          `json:"correctQuestions"`
	WrongQuestions         int          `json:"wrongQuestions"`
	AnsweredQuestions      int          `json:"answeredQuestions"`
	TotalQuestions         int          `json:"totalQuestions"`
	DurationSeconds        int          `json:"durationSeconds"`
	PlayedMs               int64        `json:"playedMs"`
	SelectedQuestionSetIDs []string     `json:"selectedQuestionSetIds"`
	ConfigurationKey       string       `json:"configurationKey"`
}

type userScoresResponse struct {
	PublicUserID string                 `json:"publicUserId"`
	DisplayName  string                 `json:"displayName"`
	Scores       []scoreSummaryResponse `json:"scores"`
}

type userStatsPayload struct {
	GamesPlayed           int     `json:"gamesPlayed"`
	BestScore             int     `json:"bestScore"`
	AverageScore          float64 `json:"averageScore"`
	TotalCorrectQuestions int     `json:"totalCorrectQuestions"`
}

type userStatsResponse struct {
	PublicUserID string           `json:"publicUserId"`
	DisplayName  string           `json:"displayName"`
	Stats        userStatsPayload `json:"stats"`
}

type leaderboardEntryResponse struct {
	Rank             int                `json:"rank"`
	ScoreID          string             `json:"scoreId"`
	Score            int                `json:"score"`
	FinishedAt       string             `json:"finishedAt"`
	ConfigurationKey string             `json:"configurationKey"`
	Player           publicUserResponse `json:"player"`
}

type leaderboardResponse struct {
	ConfigurationKey *string                    `json:"configurationKey,omitempty"`
	Entries          []leaderboardEntryResponse `json:"entries"`
}

type linkAccountResponse struct {
	SessionID    string `json:"sessionId"`
	ScoreID      string `json:"scoreId"`
	PublicUserID string `json:"publicUserId"`
	DisplayName  string `json:"displayName"`
	Linked       bool   `json:"linked"`
}

func NewHandler(db *pgxpool.Pool, questionsClient *questionsapi.Client) *Handler {
	return &Handler{
		db:              db,
		service:         NewService(questionsClient),
		questionsClient: questionsClient,
		repository:      newSessionRepository(db),
	}
}

func (h *Handler) GetQuestionSets(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}
	if h.questionsClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "questions client is not configured"})
		return
	}

	sets, err := h.questionsClient.ListSets(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to load question sets"})
		return
	}

	writeJSON(w, http.StatusOK, sets)
}

func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	var request startSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	session, err := h.service.StartSession(r.Context(), request.DurationSeconds, request.SelectedQuestionSetIDs, now)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var (
		ownerProfileID *string
		isAnonymous    = true
		saveDeadlineAt *time.Time
	)
	if user, ok := middleware.AuthenticatedUserFromContext(r.Context()); ok {
		profileID, ensureErr := h.repository.EnsureUserProfile(r.Context(), user)
		if ensureErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to ensure user profile"})
			return
		}
		ownerProfileID = &profileID
		isAnonymous = false
	}

	if err := h.repository.CreateSession(r.Context(), session, ownerProfileID, isAnonymous, saveDeadlineAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	writeJSON(w, http.StatusCreated, buildSessionResponse(session, now))
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	now := time.Now().UTC()
	authenticatedProfileID, err := h.authenticatedProfileID(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to ensure authenticated profile"})
		return
	}

	session, _, _, err := h.repository.UpdateLockedSession(
		r.Context(),
		chi.URLParam(r, "sessionId"),
		func(session *Session, ownerProfileID *string, isAnonymous bool) error {
			accessErr := authorizeSessionAccess(authenticatedProfileID, ownerProfileID, isAnonymous)
			if accessErr != nil {
				return accessErr
			}

			session.Sync(now)
			h.applyAnonymousSaveDeadline(session, now)
			return nil
		},
	)
	if errors.Is(err, ErrSessionNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errAuthenticationRequired) {
			statusCode = http.StatusUnauthorized
		} else if errors.Is(err, errSessionForbidden) {
			statusCode = http.StatusForbidden
		}
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, buildSessionResponse(session, now))
}

func (h *Handler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	var request answerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	authenticatedProfileID, err := h.authenticatedProfileID(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to ensure authenticated profile"})
		return
	}
	var result AnswerResult
	session, ownerProfileID, isAnonymous, err := h.repository.UpdateLockedSession(
		r.Context(),
		chi.URLParam(r, "sessionId"),
		func(session *Session, ownerProfileID *string, isAnonymous bool) error {
			accessErr := authorizeSessionAccess(authenticatedProfileID, ownerProfileID, isAnonymous)
			if accessErr != nil {
				return accessErr
			}

			answerResult, submitErr := session.SubmitAnswer(now, request.SelectedAnswerIndex)
			if submitErr != nil {
				return submitErr
			}

			result = answerResult
			h.applyAnonymousSaveDeadline(session, now)
			return nil
		},
	)
	if errors.Is(err, ErrSessionNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, errAuthenticationRequired):
			statusCode = http.StatusUnauthorized
		case errors.Is(err, errSessionForbidden):
			statusCode = http.StatusForbidden
		case errors.Is(err, ErrSessionAlreadyFinished), errors.Is(err, ErrNoCurrentQuestion), errors.Is(err, ErrInvalidAnswerIndex):
			statusCode = http.StatusBadRequest
		}
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}

	if session.Status == SessionStatusFinished {
		if err := h.persistScore(r, session, ownerProfileID, isAnonymous); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create score"})
			return
		}
	}

	writeJSON(w, http.StatusOK, answerResponse{
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

func (h *Handler) LinkAccount(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	authenticatedProfileID, err := h.authenticatedProfileID(r)
	if err != nil || authenticatedProfileID == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": errAuthenticationRequired.Error()})
		return
	}

	now := time.Now().UTC()
	scoreID, err := h.repository.LinkAnonymousSessionScore(r.Context(), chi.URLParam(r, "sessionId"), *authenticatedProfileID, now)
	if errors.Is(err, ErrSessionNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if errors.Is(err, ErrSessionNotFinished) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session is not finished"})
		return
	}
	if errors.Is(err, ErrSessionExpired) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session can no longer be linked"})
		return
	}
	if errors.Is(err, ErrSessionAlreadyLinked) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session is already linked"})
		return
	}
	if errors.Is(err, ErrScoreNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "score not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to link account"})
		return
	}

	profile, err := h.repository.GetProfileByID(r.Context(), *authenticatedProfileID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load linked profile"})
		return
	}

	writeJSON(w, http.StatusOK, linkAccountResponse{
		SessionID:    chi.URLParam(r, "sessionId"),
		ScoreID:      scoreID,
		PublicUserID: profile.PublicUserID,
		DisplayName:  profile.DisplayName,
		Linked:       true,
	})
}

func (h *Handler) GetScore(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	score, err := h.repository.GetScore(r.Context(), chi.URLParam(r, "scoreId"))
	if errors.Is(err, ErrScoreNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "score not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load score"})
		return
	}

	writeJSON(w, http.StatusOK, buildScoreResponse(score))
}

func (h *Handler) GetUserScores(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	profile, scores, err := h.repository.ListScoresByPublicUserID(r.Context(), chi.URLParam(r, "publicUserId"))
	if errors.Is(err, ErrProfileNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load user scores"})
		return
	}

	summaries := make([]scoreSummaryResponse, 0, len(scores))
	for _, score := range scores {
		summaries = append(summaries, buildScoreSummaryResponse(score))
	}

	writeJSON(w, http.StatusOK, userScoresResponse{
		PublicUserID: profile.PublicUserID,
		DisplayName:  profile.DisplayName,
		Scores:       summaries,
	})
}

func (h *Handler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	profile, stats, err := h.repository.GetUserStatsByPublicUserID(r.Context(), chi.URLParam(r, "publicUserId"))
	if errors.Is(err, ErrProfileNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load user stats"})
		return
	}

	writeJSON(w, http.StatusOK, userStatsResponse{
		PublicUserID: profile.PublicUserID,
		DisplayName:  profile.DisplayName,
		Stats:        userStatsPayload(stats),
	})
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	authenticatedProfileID, err := h.authenticatedProfileID(r)
	if err != nil || authenticatedProfileID == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": errAuthenticationRequired.Error()})
		return
	}

	profile, err := h.repository.GetProfileByID(r.Context(), *authenticatedProfileID)
	if errors.Is(err, ErrProfileNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load user"})
		return
	}

	writeJSON(w, http.StatusOK, publicUserResponse{
		PublicUserID: profile.PublicUserID,
		DisplayName:  profile.DisplayName,
	})
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	var configurationKey *string
	if value := r.URL.Query().Get("configurationKey"); value != "" {
		configurationKey = &value
	}

	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		if parsedLimit > 100 {
			parsedLimit = 100
		}
		limit = parsedLimit
	}

	entries, err := h.repository.ListLeaderboard(r.Context(), configurationKey, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load leaderboard"})
		return
	}

	responseEntries := make([]leaderboardEntryResponse, 0, len(entries))
	for _, entry := range entries {
		responseEntries = append(responseEntries, leaderboardEntryResponse{
			Rank:             entry.Rank,
			ScoreID:          entry.ScoreID,
			Score:            entry.Score,
			FinishedAt:       entry.FinishedAt.UTC().Format(time.RFC3339Nano),
			ConfigurationKey: entry.ConfigurationKey,
			Player: publicUserResponse{
				PublicUserID: entry.Player.PublicUserID,
				DisplayName:  entry.Player.DisplayName,
			},
		})
	}

	writeJSON(w, http.StatusOK, leaderboardResponse{
		ConfigurationKey: configurationKey,
		Entries:          responseEntries,
	})
}

func (h *Handler) finishWithReason(w http.ResponseWriter, r *http.Request, reason FinishReason) {
	if h.repository.db == nil {
		writeServiceUnavailable(w)
		return
	}

	now := time.Now().UTC()
	authenticatedProfileID, err := h.authenticatedProfileID(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to ensure authenticated profile"})
		return
	}

	session, ownerProfileID, isAnonymous, err := h.repository.UpdateLockedSession(
		r.Context(),
		chi.URLParam(r, "sessionId"),
		func(session *Session, ownerProfileID *string, isAnonymous bool) error {
			accessErr := authorizeSessionAccess(authenticatedProfileID, ownerProfileID, isAnonymous)
			if accessErr != nil {
				return accessErr
			}

			session.Finish(now, reason)
			h.applyAnonymousSaveDeadline(session, now)
			return nil
		},
	)
	if errors.Is(err, ErrSessionNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errAuthenticationRequired) {
			statusCode = http.StatusUnauthorized
		} else if errors.Is(err, errSessionForbidden) {
			statusCode = http.StatusForbidden
		}
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	if err := h.persistScore(r, session, ownerProfileID, isAnonymous); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create score"})
		return
	}

	writeJSON(w, http.StatusOK, buildSessionResponse(session, now))
}

func (h *Handler) persistScore(r *http.Request, session *Session, ownerProfileID *string, isAnonymous bool) error {
	scoreResult := session.ScoreResult()
	finishedAt := time.Now().UTC()
	if session.FinishedAt != nil {
		finishedAt = session.FinishedAt.UTC()
	}

	var (
		isSaved   bool
		isPublic  bool
		savedAt   *time.Time
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

	return h.repository.CreateScore(r.Context(), session.ID, ownerProfileID, scoreResult, finishedAt, isSaved, isPublic, savedAt, expiresAt)
}

var (
	errAuthenticationRequired = errors.New("authentication required")
	errSessionForbidden       = errors.New("forbidden")
)

func (h *Handler) authenticatedProfileID(r *http.Request) (*string, error) {
	user, ok := middleware.AuthenticatedUserFromContext(r.Context())
	if !ok {
		return nil, nil
	}

	profileID, err := h.repository.EnsureUserProfile(r.Context(), user)
	if err != nil {
		return nil, fmt.Errorf("ensure authenticated profile: %w", err)
	}

	return &profileID, nil
}

func authorizeSessionAccess(authenticatedProfileID *string, ownerProfileID *string, isAnonymous bool) error {
	if isAnonymous {
		return nil
	}
	if authenticatedProfileID == nil {
		return errAuthenticationRequired
	}
	if ownerProfileID == nil || *ownerProfileID != *authenticatedProfileID {
		return errSessionForbidden
	}

	return nil
}

func (h *Handler) applyAnonymousSaveDeadline(session *Session, now time.Time) {
	if session.SaveDeadlineAt != nil || session.Status != SessionStatusFinished {
		return
	}

	deadline := now.Add(15 * time.Minute)
	session.SaveDeadlineAt = &deadline
}

func buildSessionResponse(session *Session, now time.Time) sessionResponse {
	currentQuestion, currentQuestionErr := session.CurrentQuestion(now)
	if currentQuestionErr != nil {
		currentQuestion = nil
	}

	return sessionResponse{
		SessionID:              session.ID,
		Status:                 session.Status,
		FinishReason:           session.FinishReason,
		StartedAt:              session.StartedAt.UTC().Format(time.RFC3339Nano),
		EndsAt:                 session.EndsAt.UTC().Format(time.RFC3339Nano),
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

func buildScoreResponse(score *storedScore) scoreResponse {
	response := scoreResponse{
		ScoreID:                score.ScoreID,
		SessionID:              score.SessionID,
		FinishedAt:             score.FinishedAt.UTC().Format(time.RFC3339Nano),
		FinishReason:           score.FinishReason,
		Score:                  score.Score,
		CorrectQuestions:       score.CorrectQuestions,
		WrongQuestions:         score.WrongQuestions,
		AnsweredQuestions:      score.AnsweredQuestions,
		TotalQuestions:         score.TotalQuestions,
		DurationSeconds:        score.DurationSeconds,
		PlayedMs:               score.PlayedMs,
		SelectedQuestionSetIDs: score.SelectedQuestionSetIDs,
		ConfigurationKey:       score.ConfigurationKey,
		QuestionResults:        score.QuestionResults,
	}
	if score.Player != nil {
		response.Player = &publicUserResponse{
			PublicUserID: score.Player.PublicUserID,
			DisplayName:  score.Player.DisplayName,
		}
	}

	return response
}

func buildScoreSummaryResponse(score storedScore) scoreSummaryResponse {
	return scoreSummaryResponse{
		ScoreID:                score.ScoreID,
		SessionID:              score.SessionID,
		FinishedAt:             score.FinishedAt.UTC().Format(time.RFC3339Nano),
		FinishReason:           score.FinishReason,
		Score:                  score.Score,
		CorrectQuestions:       score.CorrectQuestions,
		WrongQuestions:         score.WrongQuestions,
		AnsweredQuestions:      score.AnsweredQuestions,
		TotalQuestions:         score.TotalQuestions,
		DurationSeconds:        score.DurationSeconds,
		PlayedMs:               score.PlayedMs,
		SelectedQuestionSetIDs: score.SelectedQuestionSetIDs,
		ConfigurationKey:       score.ConfigurationKey,
	}
}

func writeServiceUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "game backend is not fully configured"})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	if err := httpjson.Write(w, statusCode, payload); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
