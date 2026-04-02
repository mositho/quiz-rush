package game

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"quiz-rush/game-backend/internal/questionsapi"
)

type QuestionLoader interface {
	LoadQuestionsBySetID(ctx context.Context, setID string) ([]questionsapi.Question, error)
}

type Service struct {
	questionLoader QuestionLoader
}

func NewService(questionLoader QuestionLoader) *Service {
	return &Service{questionLoader: questionLoader}
}

func (s *Service) StartSession(ctx context.Context, durationSeconds int, selectedQuestionSetIDs []string, now time.Time) (*Session, error) {
	questionDefinitions, err := s.loadQuestions(ctx, selectedQuestionSetIDs)
	if err != nil {
		return nil, err
	}

	return NewSession(SessionConfig{
		DurationSeconds:        durationSeconds,
		SelectedQuestionSetIDs: slices.Clone(selectedQuestionSetIDs),
		ConfigurationKey:       BuildConfigurationKey(durationSeconds, selectedQuestionSetIDs),
	}, questionDefinitions, now)
}

func (s *Service) loadQuestions(ctx context.Context, selectedQuestionSetIDs []string) ([]QuestionDefinition, error) {
	if len(selectedQuestionSetIDs) == 0 {
		return nil, ErrQuestionSetRequired
	}

	questionDefinitions := make([]QuestionDefinition, 0)
	for _, setID := range selectedQuestionSetIDs {
		setQuestions, err := s.questionLoader.LoadQuestionsBySetID(ctx, setID)
		if err != nil {
			return nil, fmt.Errorf("load question set %q: %w", setID, err)
		}

		for _, question := range setQuestions {
			questionDefinitions = append(questionDefinitions, QuestionDefinition{
				ID:                 question.ID,
				QuestionSetID:      setID,
				Difficulty:         question.Difficulty,
				Categories:         slices.Clone(question.Categories),
				Text:               question.Question,
				Options:            slices.Clone(question.Options),
				CorrectAnswerIndex: question.CorrectAnswer,
			})
		}
	}

	return questionDefinitions, nil
}

func BuildConfigurationKey(durationSeconds int, selectedQuestionSetIDs []string) string {
	normalizedSetIDs := slices.Clone(selectedQuestionSetIDs)
	sort.Strings(normalizedSetIDs)

	return fmt.Sprintf(
		"duration=%d|sets=%s",
		durationSeconds,
		strings.Join(normalizedSetIDs, ","),
	)
}
