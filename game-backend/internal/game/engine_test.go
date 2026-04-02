package game_test

import (
	"testing"
	"time"

	"quiz-rush/game-backend/internal/game"
)

func TestNewSessionActivatesFirstQuestion(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	session := mustNewSession(t, game.SessionConfig{
		DurationSeconds:        180,
		SelectedQuestionSetIDs: []string{"lf1"},
		ConfigurationKey:       "duration=180|sets=lf1",
	}, testQuestions(), now)

	assertEqual(t, session.Status, game.SessionStatusActive)

	currentQuestion, err := session.CurrentQuestion(now)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, currentQuestion.QuestionID, "q1")
}

func TestSubmitCorrectAnswerAdvancesAndScores(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	session := mustNewSession(t, game.SessionConfig{
		DurationSeconds:        180,
		SelectedQuestionSetIDs: []string{"lf1"},
		ConfigurationKey:       "duration=180|sets=lf1",
	}, testQuestions(), now)

	result, err := session.SubmitAnswer(now.Add(2*time.Second), 1)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, result.Correct, true)
	if result.AwardedPoints <= 0 {
		t.Fatalf("got awarded points %d, want positive value", result.AwardedPoints)
	}
	assertEqual(t, session.CurrentScore, result.AwardedPoints)
	assertEqual(t, session.CorrectQuestions, 1)
	if result.NextQuestion == nil || result.NextQuestion.QuestionID != "q2" {
		t.Fatalf("got next question %+v, want questionId q2", result.NextQuestion)
	}
}

func TestWrongAnswerDeductsTimeAndAdvancesImmediately(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	session := mustNewSession(t, game.SessionConfig{
		DurationSeconds:        180,
		SelectedQuestionSetIDs: []string{"lf1"},
		ConfigurationKey:       "duration=180|sets=lf1",
	}, testQuestions(), now)

	expectedEndsAt := session.EndsAt.Add(-3 * time.Second)

	result, err := session.SubmitAnswer(now.Add(2*time.Second), 0) // wrong answer (correct is index 1)
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, result.Correct, false)
	assertEqual(t, session.Status, game.SessionStatusActive)
	assertEqual(t, session.EndsAt.UTC(), expectedEndsAt.UTC())

	// Next question is immediately available — no cooldown block
	question, err := session.CurrentQuestion(now.Add(2 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, question.QuestionID, "q2")
}

func TestWrongAnswerWithInsufficientTimeFinishesSession(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	// Only 2 seconds left — a wrong answer (3s penalty) should exhaust the timer
	session := mustNewSession(t, game.SessionConfig{
		DurationSeconds:        2,
		SelectedQuestionSetIDs: []string{"lf1"},
		ConfigurationKey:       "duration=2|sets=lf1",
	}, testQuestions(), now)

	result, err := session.SubmitAnswer(now.Add(1*time.Second), 0) // wrong answer
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, result.Correct, false)
	assertEqual(t, result.Finished, true)
	if result.FinishReason == nil || *result.FinishReason != game.FinishReasonTimerElapsed {
		t.Fatalf("got finish reason %v, want %q", result.FinishReason, game.FinishReasonTimerElapsed)
	}
	assertEqual(t, session.Status, game.SessionStatusFinished)
}

func TestTimerElapsedFinishesSessionAndProducesScoreResult(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	session := mustNewSession(t, game.SessionConfig{
		DurationSeconds:        5,
		SelectedQuestionSetIDs: []string{"lf1"},
		ConfigurationKey:       "duration=5|sets=lf1",
	}, testQuestions(), now)

	_, err := session.SubmitAnswer(now.Add(2*time.Second), 1)
	if err != nil {
		t.Fatal(err)
	}

	session.Sync(now.Add(6 * time.Second))

	assertEqual(t, session.Status, game.SessionStatusFinished)
	if session.FinishReason == nil || *session.FinishReason != game.FinishReasonTimerElapsed {
		t.Fatalf("got finish reason %v, want %q", session.FinishReason, game.FinishReasonTimerElapsed)
	}

	scoreResult := session.ScoreResult()
	if scoreResult.Score <= 0 {
		t.Fatalf("got score %d, want positive value", scoreResult.Score)
	}
	assertEqual(t, scoreResult.AnsweredQuestions, 1)
	assertEqual(t, len(scoreResult.QuestionResults), len(testQuestions()))
}

func testQuestions() []game.QuestionDefinition {
	return []game.QuestionDefinition{
		{
			ID:                 "q1",
			QuestionSetID:      "lf1",
			Difficulty:         2,
			Categories:         []string{"lf1", "basics"},
			Text:               "Question one",
			Options:            []string{"A", "B", "C", "D"},
			CorrectAnswerIndex: 1,
		},
		{
			ID:                 "q2",
			QuestionSetID:      "lf1",
			Difficulty:         1,
			Categories:         []string{"lf1", "followup"},
			Text:               "Question two",
			Options:            []string{"A", "B", "C", "D"},
			CorrectAnswerIndex: 2,
		},
	}
}

func mustNewSession(t *testing.T, config game.SessionConfig, questionDefinitions []game.QuestionDefinition, now time.Time) *game.Session {
	t.Helper()

	session, err := game.NewSession(config, questionDefinitions, now)
	if err != nil {
		t.Fatal(err)
	}

	return session
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()

	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
