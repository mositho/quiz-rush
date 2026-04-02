package game

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

const (
	wrongAnswerPenalty = 3 * time.Second
	speedBonusWindow   = 10 * time.Second
)

var (
	ErrNoQuestionsLoaded      = errors.New("no questions loaded")
	ErrQuestionSetRequired    = errors.New("at least one question set is required")
	ErrSessionAlreadyFinished = errors.New("session is already finished")
	ErrNoCurrentQuestion      = errors.New("no current question available")
	ErrInvalidAnswerIndex     = errors.New("selected answer index is invalid")
)

type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusFinished SessionStatus = "finished"
)

type FinishReason string

const (
	FinishReasonTimerElapsed          FinishReason = "timer_elapsed"
	FinishReasonQuestionPoolExhausted FinishReason = "question_pool_exhausted"
	FinishReasonManualFinish          FinishReason = "manual_finish"
	FinishReasonQuit                  FinishReason = "quit"
)

type SessionConfig struct {
	DurationSeconds        int
	SelectedQuestionSetIDs []string
	ConfigurationKey       string
}

type QuestionDefinition struct {
	ID                 string
	QuestionSetID      string
	Difficulty         int
	Categories         []string
	Text               string
	Options            []string
	CorrectAnswerIndex int
}

type SessionQuestion struct {
	Position            int            `db:"position"`
	QuestionID          string         `db:"question_id"`
	QuestionSetID       string         `db:"question_set_id"`
	Difficulty          int            `db:"difficulty"`
	QuestionCategories  []string       `db:"question_categories"`
	QuestionText        string         `db:"question_text"`
	Options             []string       `db:"options_json"`
	CorrectAnswerIndex  int            `db:"correct_answer_index"`
	ActivatedAt         *time.Time     `db:"activated_at"`
	AnsweredAt          *time.Time     `db:"answered_at"`
	SelectedAnswerIndex *int           `db:"selected_answer_index"`
	Correct             *bool          `db:"is_correct"`
	AwardedPoints       *int           `db:"awarded_points"`
	ResponseTime        *time.Duration `db:"-"`
}

type Session struct {
	ID                     string            `db:"id"`
	Status                 SessionStatus     `db:"status"`
	FinishReason           *FinishReason     `db:"finish_reason"`
	StartedAt              time.Time         `db:"started_at"`
	EndsAt                 time.Time         `db:"ends_at"`
	FinishedAt             *time.Time        `db:"finished_at"`
	SaveDeadlineAt         *time.Time        `db:"save_deadline_at"`
	DurationSeconds        int               `db:"duration_seconds"`
	SelectedQuestionSetIDs []string          `db:"selected_question_set_ids"`
	ConfigurationKey       string            `db:"configuration_key"`
	CurrentQuestionIndex   *int              `db:"current_question_index"`
	TotalQuestions         int               `db:"total_questions"`
	AnsweredQuestions      int               `db:"answered_questions"`
	CorrectQuestions       int               `db:"correct_questions"`
	WrongQuestions         int               `db:"wrong_questions"`
	CurrentScore           int               `db:"current_score"`
	SessionQuestions       []SessionQuestion `db:"-"`
}

type Question struct {
	Position      int      `json:"position"`
	QuestionID    string   `json:"questionId"`
	QuestionSetID string   `json:"questionSetId"`
	Difficulty    int      `json:"difficulty"`
	Categories    []string `json:"categories"`
	Text          string   `json:"text"`
	Options       []string `json:"options"`
}

type AnswerResult struct {
	Correct        bool          `json:"correct"`
	AwardedPoints  int           `json:"awardedPoints"`
	ResponseTimeMs int64         `json:"responseTimeMs"`
	NextQuestion   *Question     `json:"nextQuestion,omitempty"`
	Finished       bool          `json:"finished"`
	FinishReason   *FinishReason `json:"finishReason,omitempty"`
}

type ScoreQuestionResult struct {
	QuestionID          string   `json:"questionId"`
	QuestionSetID       string   `json:"questionSetId"`
	Difficulty          int      `json:"difficulty"`
	QuestionCategories  []string `json:"questionCategories"`
	Position            int      `json:"position"`
	SelectedAnswerIndex *int     `json:"selectedAnswerIndex,omitempty"`
	IsCorrect           *bool    `json:"isCorrect,omitempty"`
	AwardedPoints       *int     `json:"awardedPoints,omitempty"`
	ResponseTimeMs      *int64   `json:"responseTimeMs,omitempty"`
}

type ScoreResult struct {
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
}

func NewSession(config SessionConfig, questionDefinitions []QuestionDefinition, now time.Time) (*Session, error) {
	if len(config.SelectedQuestionSetIDs) == 0 {
		return nil, ErrQuestionSetRequired
	}
	if len(questionDefinitions) == 0 {
		return nil, ErrNoQuestionsLoaded
	}
	if config.DurationSeconds <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}

	sessionQuestions := make([]SessionQuestion, 0, len(questionDefinitions))
	for i, questionDefinition := range questionDefinitions {
		sessionQuestions = append(sessionQuestions, SessionQuestion{
			Position:           i,
			QuestionID:         questionDefinition.ID,
			QuestionSetID:      questionDefinition.QuestionSetID,
			Difficulty:         max(questionDefinition.Difficulty, 1),
			QuestionCategories: slices.Clone(questionDefinition.Categories),
			QuestionText:       questionDefinition.Text,
			Options:            slices.Clone(questionDefinition.Options),
			CorrectAnswerIndex: questionDefinition.CorrectAnswerIndex,
		})
	}

	currentIndex := 0
	sessionQuestions[0].ActivatedAt = timePointer(now)

	return &Session{
		Status:                 SessionStatusActive,
		StartedAt:              now,
		EndsAt:                 now.Add(time.Duration(config.DurationSeconds) * time.Second),
		DurationSeconds:        config.DurationSeconds,
		SelectedQuestionSetIDs: slices.Clone(config.SelectedQuestionSetIDs),
		ConfigurationKey:       config.ConfigurationKey,
		CurrentQuestionIndex:   &currentIndex,
		TotalQuestions:         len(sessionQuestions),
		SessionQuestions:       sessionQuestions,
	}, nil
}

func (s *Session) Sync(now time.Time) {
	if s.Status == SessionStatusFinished {
		return
	}

	if !now.Before(s.EndsAt) {
		s.finish(now, FinishReasonTimerElapsed)
	}
}

func (s *Session) CurrentQuestion(now time.Time) (*Question, error) {
	s.Sync(now)

	if s.Status != SessionStatusActive {
		return nil, ErrNoCurrentQuestion
	}

	index, ok := s.currentQuestionIndexValue()
	if !ok || index >= len(s.SessionQuestions) {
		return nil, ErrNoCurrentQuestion
	}

	return s.SessionQuestions[index].Question(), nil
}

func (s *Session) SubmitAnswer(now time.Time, selectedAnswerIndex int) (AnswerResult, error) {
	s.Sync(now)

	if s.Status == SessionStatusFinished {
		return AnswerResult{Finished: true, FinishReason: s.FinishReason}, ErrSessionAlreadyFinished
	}

	index, ok := s.currentQuestionIndexValue()
	if !ok || index >= len(s.SessionQuestions) {
		return AnswerResult{}, ErrNoCurrentQuestion
	}

	current := &s.SessionQuestions[index]
	if selectedAnswerIndex < 0 || selectedAnswerIndex >= len(current.Options) {
		return AnswerResult{}, ErrInvalidAnswerIndex
	}
	if current.ActivatedAt == nil {
		current.ActivatedAt = timePointer(now)
	}

	answeredAt := now
	selected := selectedAnswerIndex
	responseTime := answeredAt.Sub(*current.ActivatedAt)
	isCorrect := current.CorrectAnswerIndex == selectedAnswerIndex
	points := 0

	current.AnsweredAt = &answeredAt
	current.SelectedAnswerIndex = &selected
	current.Correct = boolPointer(isCorrect)
	current.ResponseTime = durationPointer(responseTime)

	s.AnsweredQuestions++
	if isCorrect {
		points = calculateScore(current.Difficulty, responseTime)
		current.AwardedPoints = intPointer(points)
		s.CorrectQuestions++
		s.CurrentScore += points
	} else {
		current.AwardedPoints = intPointer(0)
		s.WrongQuestions++
		s.EndsAt = s.EndsAt.Add(-wrongAnswerPenalty)
	}

	s.advanceCurrentQuestionIndex()

	// Re-check timer after potential EndsAt deduction
	if !now.Before(s.EndsAt) {
		s.finish(now, FinishReasonTimerElapsed)
		return AnswerResult{
			Correct:        isCorrect,
			AwardedPoints:  points,
			ResponseTimeMs: responseTime.Milliseconds(),
			Finished:       true,
			FinishReason:   s.FinishReason,
		}, nil
	}

	if !s.activateCurrentQuestion(answeredAt) {
		s.finish(answeredAt, FinishReasonQuestionPoolExhausted)
		return AnswerResult{
			Correct:        isCorrect,
			AwardedPoints:  points,
			ResponseTimeMs: responseTime.Milliseconds(),
			Finished:       true,
			FinishReason:   s.FinishReason,
		}, nil
	}

	result := AnswerResult{
		Correct:        isCorrect,
		AwardedPoints:  points,
		ResponseTimeMs: responseTime.Milliseconds(),
	}

	nextQuestion, err := s.CurrentQuestion(answeredAt)
	if err == nil {
		result.NextQuestion = nextQuestion
	}

	return result, nil
}

func (s *Session) Finish(now time.Time, reason FinishReason) ScoreResult {
	s.Sync(now)
	if s.Status != SessionStatusFinished {
		s.finish(now, reason)
	}
	return s.ScoreResult()
}

func (s *Session) ScoreResult() ScoreResult {
	finishReason := FinishReasonTimerElapsed
	if s.FinishReason != nil {
		finishReason = *s.FinishReason
	}

	playedUntil := s.EndsAt
	if s.FinishedAt != nil {
		playedUntil = *s.FinishedAt
	}
	if playedUntil.Before(s.StartedAt) {
		playedUntil = s.StartedAt
	}

	results := make([]ScoreQuestionResult, 0, len(s.SessionQuestions))
	for _, question := range s.SessionQuestions {
		var responseTimeMs *int64
		if question.ResponseTime != nil {
			value := question.ResponseTime.Milliseconds()
			responseTimeMs = &value
		}

		results = append(results, ScoreQuestionResult{
			QuestionID:          question.QuestionID,
			QuestionSetID:       question.QuestionSetID,
			Difficulty:          question.Difficulty,
			QuestionCategories:  slices.Clone(question.QuestionCategories),
			Position:            question.Position,
			SelectedAnswerIndex: question.SelectedAnswerIndex,
			IsCorrect:           question.Correct,
			AwardedPoints:       question.AwardedPoints,
			ResponseTimeMs:      responseTimeMs,
		})
	}

	return ScoreResult{
		FinishReason:           finishReason,
		Score:                  s.CurrentScore,
		CorrectQuestions:       s.CorrectQuestions,
		WrongQuestions:         s.WrongQuestions,
		AnsweredQuestions:      s.AnsweredQuestions,
		TotalQuestions:         s.TotalQuestions,
		DurationSeconds:        s.DurationSeconds,
		PlayedMs:               playedUntil.Sub(s.StartedAt).Milliseconds(),
		SelectedQuestionSetIDs: slices.Clone(s.SelectedQuestionSetIDs),
		ConfigurationKey:       s.ConfigurationKey,
		QuestionResults:        results,
	}
}

func (q SessionQuestion) Question() *Question {
	return &Question{
		Position:      q.Position,
		QuestionID:    q.QuestionID,
		QuestionSetID: q.QuestionSetID,
		Difficulty:    q.Difficulty,
		Categories:    slices.Clone(q.QuestionCategories),
		Text:          q.QuestionText,
		Options:       slices.Clone(q.Options),
	}
}

func (s *Session) finish(now time.Time, reason FinishReason) {
	if s.Status == SessionStatusFinished {
		return
	}

	s.Status = SessionStatusFinished
	s.FinishReason = &reason
	s.FinishedAt = timePointer(now)
	s.CurrentQuestionIndex = nil
}

func (s *Session) activateCurrentQuestion(now time.Time) bool {
	index, ok := s.currentQuestionIndexValue()
	if !ok || index >= len(s.SessionQuestions) {
		s.CurrentQuestionIndex = nil
		return false
	}

	question := &s.SessionQuestions[index]
	if question.ActivatedAt == nil {
		question.ActivatedAt = timePointer(now)
	}

	s.Status = SessionStatusActive
	return true
}

func (s *Session) advanceCurrentQuestionIndex() {
	index, ok := s.currentQuestionIndexValue()
	if !ok {
		s.CurrentQuestionIndex = nil
		return
	}

	nextIndex := index + 1
	if nextIndex >= len(s.SessionQuestions) {
		s.CurrentQuestionIndex = nil
		return
	}

	s.CurrentQuestionIndex = &nextIndex
}

func (s *Session) currentQuestionIndexValue() (int, bool) {
	if s.CurrentQuestionIndex == nil {
		return 0, false
	}

	return *s.CurrentQuestionIndex, true
}

func calculateScore(difficulty int, responseTime time.Duration) int {
	basePoints := max(difficulty, 1) * 100
	bonusWindowMs := float64(speedBonusWindow.Milliseconds())
	responseMs := math.Min(float64(maxInt64(responseTime.Milliseconds(), 0)), bonusWindowMs)
	bonusRatio := (bonusWindowMs - responseMs) / bonusWindowMs
	speedBonus := int(math.Round(float64(basePoints) * bonusRatio))
	return basePoints + speedBonus
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}

func intPointer(value int) *int {
	return &value
}

func durationPointer(value time.Duration) *time.Duration {
	return &value
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}
