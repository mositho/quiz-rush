package game_test

import (
	"context"
	"testing"
	"time"

	"quiz-rush/game-backend/internal/game"
	"quiz-rush/game-backend/internal/questionsapi"
)

func TestServiceBuildsSessionFromQuestionSetsAndNormalizesConfigurationKey(t *testing.T) {
	service := game.NewService(stubQuestionLoader{
		questionsBySetID: map[string][]questionsapi.Question{
			"lf2": {
				{
					ID:            "lf2_q1",
					Difficulty:    1,
					Categories:    []string{"lf2"},
					Question:      "Question two",
					Options:       []string{"A", "B"},
					CorrectAnswer: 0,
				},
			},
			"lf1": {
				{
					ID:            "lf1_q1",
					Difficulty:    2,
					Categories:    []string{"lf1"},
					Question:      "Question one",
					Options:       []string{"A", "B"},
					CorrectAnswer: 1,
				},
			},
		},
	})

	session, err := service.StartSession(context.Background(), 180, []string{"lf2", "lf1"}, time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, session.ConfigurationKey, "duration=180|sets=lf1,lf2")
	assertEqual(t, session.TotalQuestions, 2)
}

type stubQuestionLoader struct {
	questionsBySetID map[string][]questionsapi.Question
}

func (s stubQuestionLoader) LoadQuestionsBySetID(_ context.Context, setID string) ([]questionsapi.Question, error) {
	questions, ok := s.questionsBySetID[setID]
	if !ok {
		return nil, game.ErrNoQuestionsLoaded
	}

	return questions, nil
}
