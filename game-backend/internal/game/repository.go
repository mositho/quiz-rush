package game

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"quiz-rush/game-backend/internal/middleware"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")

type sessionRepository struct {
	db *pgxpool.Pool
}

type dbTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func newSessionRepository(db *pgxpool.Pool) *sessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) EnsureUserProfile(ctx context.Context, user middleware.AuthenticatedUser) (string, error) {
	displayName := user.PreferredUsername
	if displayName == "" {
		displayName = user.Subject
	}

	publicUserID := newPublicUserID()

	var id string
	err := r.db.QueryRow(
		ctx,
		`
		insert into user_profiles (public_user_id, keycloak_subject, display_name)
		values ($1, $2, $3)
		on conflict (keycloak_subject) do update
		set display_name = excluded.display_name,
		    updated_at = now()
		returning id
		`,
		publicUserID,
		user.Subject,
		displayName,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("ensure user profile: %w", err)
	}

	return id, nil
}

func (r *sessionRepository) CreateSession(
	ctx context.Context,
	session *Session,
	ownerProfileID *string,
	isAnonymous bool,
	saveDeadlineAt *time.Time,
) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin session transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	sessionID, err := insertSessionRow(ctx, tx, session, ownerProfileID, isAnonymous, saveDeadlineAt)
	if err != nil {
		return err
	}

	session.ID = sessionID

	if err := insertSessionQuestions(ctx, tx, sessionID, session.SessionQuestions); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit session transaction: %w", err)
	}

	return nil
}

func (r *sessionRepository) LoadSession(ctx context.Context, sessionID string) (*Session, *string, bool, error) {
	session, ownerProfileID, isAnonymous, err := loadSessionRow(ctx, r.db, sessionID)
	if err != nil {
		return nil, nil, false, err
	}

	sessionQuestions, err := loadSessionQuestions(ctx, r.db, sessionID)
	if err != nil {
		return nil, nil, false, err
	}

	session.SessionQuestions = sessionQuestions

	return session, ownerProfileID, isAnonymous, nil
}

func (r *sessionRepository) UpdateSession(ctx context.Context, session *Session) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin update session transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := updateSessionRow(ctx, tx, session); err != nil {
		return err
	}

	if err := updateSessionQuestions(ctx, tx, session.ID, session.SessionQuestions); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update session transaction: %w", err)
	}

	return nil
}

func (r *sessionRepository) CreateScore(
	ctx context.Context,
	sessionID string,
	ownerProfileID *string,
	scoreResult ScoreResult,
	isSaved bool,
	isPublic bool,
	savedAt *time.Time,
	expiresAt *time.Time,
) error {
	questionResultsJSON, err := json.Marshal(scoreResult.QuestionResults)
	if err != nil {
		return fmt.Errorf("marshal question results: %w", err)
	}
	selectedQuestionSetIDs, err := json.Marshal(scoreResult.SelectedQuestionSetIDs)
	if err != nil {
		return fmt.Errorf("marshal selected question set ids: %w", err)
	}

	_, err = r.db.Exec(
		ctx,
		`
		insert into game_scores (
			session_id,
			owner_profile_id,
			is_saved,
			is_public,
			finished_at,
			saved_at,
			expires_at,
			finish_reason,
			score,
			correct_questions,
			wrong_questions,
			answered_questions,
			total_questions,
			duration_seconds,
			played_ms,
			selected_question_set_ids,
			configuration_key,
			question_results_json
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16::jsonb, $17, $18::jsonb
		)
		on conflict (session_id) do nothing
		`,
		sessionID,
		ownerProfileID,
		isSaved,
		isPublic,
		time.Now().UTC(),
		savedAt,
		expiresAt,
		string(scoreResult.FinishReason),
		scoreResult.Score,
		scoreResult.CorrectQuestions,
		scoreResult.WrongQuestions,
		scoreResult.AnsweredQuestions,
		scoreResult.TotalQuestions,
		scoreResult.DurationSeconds,
		scoreResult.PlayedMs,
		selectedQuestionSetIDs,
		scoreResult.ConfigurationKey,
		questionResultsJSON,
	)
	if err != nil {
		return fmt.Errorf("insert score: %w", err)
	}

	return nil
}

func insertSessionRow(
	ctx context.Context,
	tx dbTX,
	session *Session,
	ownerProfileID *string,
	isAnonymous bool,
	saveDeadlineAt *time.Time,
) (string, error) {
	selectedQuestionSetIDs, err := json.Marshal(session.SelectedQuestionSetIDs)
	if err != nil {
		return "", fmt.Errorf("marshal selected question set ids: %w", err)
	}

	var sessionID string
	err = tx.QueryRow(
		ctx,
		`
		insert into game_sessions (
			owner_profile_id,
			is_anonymous,
			status,
			finish_reason,
			started_at,
			ends_at,
			cooldown_until,
			finished_at,
			save_deadline_at,
			duration_seconds,
			selected_question_set_ids,
			configuration_key,
			current_question_index,
			total_questions,
			answered_questions,
			correct_questions,
			wrong_questions,
			current_score,
			updated_at
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11::jsonb, $12, $13, $14, $15, $16, $17, $18, now()
		)
		returning id
		`,
		ownerProfileID,
		isAnonymous,
		string(session.Status),
		finishReasonValue(session.FinishReason),
		session.StartedAt,
		session.EndsAt,
		session.CooldownUntil,
		session.FinishedAt,
		saveDeadlineAt,
		session.DurationSeconds,
		selectedQuestionSetIDs,
		session.ConfigurationKey,
		session.CurrentQuestionIndex,
		session.TotalQuestions,
		session.AnsweredQuestions,
		session.CorrectQuestions,
		session.WrongQuestions,
		session.CurrentScore,
	).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}

	return sessionID, nil
}

func insertSessionQuestions(ctx context.Context, tx dbTX, sessionID string, sessionQuestions []SessionQuestion) error {
	for _, sessionQuestion := range sessionQuestions {
		if err := insertSessionQuestionRow(ctx, tx, sessionID, sessionQuestion); err != nil {
			return err
		}
	}

	return nil
}

func insertSessionQuestionRow(ctx context.Context, tx dbTX, sessionID string, sessionQuestion SessionQuestion) error {
	questionCategories, err := json.Marshal(sessionQuestion.QuestionCategories)
	if err != nil {
		return fmt.Errorf("marshal question categories: %w", err)
	}
	optionsJSON, err := json.Marshal(sessionQuestion.Options)
	if err != nil {
		return fmt.Errorf("marshal question options: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`
		insert into game_session_questions (
			session_id,
			position,
			question_id,
			question_set_id,
			difficulty,
			question_categories,
			question_text,
			options_json,
			correct_answer_index,
			activated_at,
			answered_at,
			selected_answer_index,
			is_correct,
			awarded_points,
			response_time_ms,
			cooldown_applied_ms,
			updated_at
		)
		values (
			$1, $2, $3, $4, $5, $6::jsonb, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16, now()
		)
		`,
		sessionID,
		sessionQuestion.Position,
		sessionQuestion.QuestionID,
		sessionQuestion.QuestionSetID,
		sessionQuestion.Difficulty,
		questionCategories,
		sessionQuestion.QuestionText,
		optionsJSON,
		sessionQuestion.CorrectAnswerIndex,
		sessionQuestion.ActivatedAt,
		sessionQuestion.AnsweredAt,
		sessionQuestion.SelectedAnswerIndex,
		sessionQuestion.Correct,
		sessionQuestion.AwardedPoints,
		durationMilliseconds(sessionQuestion.ResponseTime),
		durationMilliseconds(sessionQuestion.CooldownApplied),
	)
	if err != nil {
		return fmt.Errorf("insert session question: %w", err)
	}

	return nil
}

func loadSessionRow(ctx context.Context, db *pgxpool.Pool, sessionID string) (*Session, *string, bool, error) {
	row := db.QueryRow(
		ctx,
		`
		select
			id,
			owner_profile_id,
			is_anonymous,
			status,
			finish_reason,
			started_at,
			ends_at,
			cooldown_until,
			finished_at,
			save_deadline_at,
			duration_seconds,
			selected_question_set_ids,
			configuration_key,
			current_question_index,
			total_questions,
			answered_questions,
			correct_questions,
			wrong_questions,
			current_score
		from game_sessions
		where id = $1
		`,
		sessionID,
	)

	var (
		session                   Session
		ownerProfileID            *string
		isAnonymous               bool
		status                    string
		finishReason              *string
		selectedQuestionSetIDsRaw []byte
	)
	err := row.Scan(
		&session.ID,
		&ownerProfileID,
		&isAnonymous,
		&status,
		&finishReason,
		&session.StartedAt,
		&session.EndsAt,
		&session.CooldownUntil,
		&session.FinishedAt,
		&session.SaveDeadlineAt,
		&session.DurationSeconds,
		&selectedQuestionSetIDsRaw,
		&session.ConfigurationKey,
		&session.CurrentQuestionIndex,
		&session.TotalQuestions,
		&session.AnsweredQuestions,
		&session.CorrectQuestions,
		&session.WrongQuestions,
		&session.CurrentScore,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, false, ErrSessionNotFound
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("load session: %w", err)
	}

	session.Status = SessionStatus(status)
	if finishReason != nil {
		reason := FinishReason(*finishReason)
		session.FinishReason = &reason
	}
	if err := json.Unmarshal(selectedQuestionSetIDsRaw, &session.SelectedQuestionSetIDs); err != nil {
		return nil, nil, false, fmt.Errorf("unmarshal selected question set ids: %w", err)
	}

	return &session, ownerProfileID, isAnonymous, nil
}

func loadSessionQuestions(ctx context.Context, db *pgxpool.Pool, sessionID string) ([]SessionQuestion, error) {
	questionRows, err := db.Query(
		ctx,
		`
		select
			position,
			question_id,
			question_set_id,
			difficulty,
			question_categories,
			question_text,
			options_json,
			correct_answer_index,
			activated_at,
			answered_at,
			selected_answer_index,
			is_correct,
			awarded_points,
			response_time_ms,
			cooldown_applied_ms
		from game_session_questions
		where session_id = $1
		order by position asc
		`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query session questions: %w", err)
	}
	defer questionRows.Close()

	sessionQuestions := make([]SessionQuestion, 0)
	for questionRows.Next() {
		sessionQuestion, err := scanSessionQuestion(questionRows)
		if err != nil {
			return nil, err
		}

		sessionQuestions = append(sessionQuestions, sessionQuestion)
	}
	if err := questionRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session questions: %w", err)
	}

	return sessionQuestions, nil
}

func scanSessionQuestion(rows pgx.Rows) (SessionQuestion, error) {
	var (
		sessionQuestion       SessionQuestion
		questionCategoriesRaw []byte
		optionsRaw            []byte
		responseTimeMS        *int
		cooldownAppliedMS     *int
	)

	err := rows.Scan(
		&sessionQuestion.Position,
		&sessionQuestion.QuestionID,
		&sessionQuestion.QuestionSetID,
		&sessionQuestion.Difficulty,
		&questionCategoriesRaw,
		&sessionQuestion.QuestionText,
		&optionsRaw,
		&sessionQuestion.CorrectAnswerIndex,
		&sessionQuestion.ActivatedAt,
		&sessionQuestion.AnsweredAt,
		&sessionQuestion.SelectedAnswerIndex,
		&sessionQuestion.Correct,
		&sessionQuestion.AwardedPoints,
		&responseTimeMS,
		&cooldownAppliedMS,
	)
	if err != nil {
		return SessionQuestion{}, fmt.Errorf("scan session question: %w", err)
	}

	if err := json.Unmarshal(questionCategoriesRaw, &sessionQuestion.QuestionCategories); err != nil {
		return SessionQuestion{}, fmt.Errorf("unmarshal question categories: %w", err)
	}
	if err := json.Unmarshal(optionsRaw, &sessionQuestion.Options); err != nil {
		return SessionQuestion{}, fmt.Errorf("unmarshal question options: %w", err)
	}
	if responseTimeMS != nil {
		duration := time.Duration(*responseTimeMS) * time.Millisecond
		sessionQuestion.ResponseTime = &duration
	}
	if cooldownAppliedMS != nil {
		duration := time.Duration(*cooldownAppliedMS) * time.Millisecond
		sessionQuestion.CooldownApplied = &duration
	}

	return sessionQuestion, nil
}

func updateSessionRow(ctx context.Context, tx dbTX, session *Session) error {
	_, err := tx.Exec(
		ctx,
		`
		update game_sessions
		set
			status = $2,
			finish_reason = $3,
			cooldown_until = $4,
			finished_at = $5,
			save_deadline_at = $6,
			current_question_index = $7,
			answered_questions = $8,
			correct_questions = $9,
			wrong_questions = $10,
			current_score = $11,
			updated_at = now()
		where id = $1
		`,
		session.ID,
		string(session.Status),
		finishReasonValue(session.FinishReason),
		session.CooldownUntil,
		session.FinishedAt,
		session.SaveDeadlineAt,
		session.CurrentQuestionIndex,
		session.AnsweredQuestions,
		session.CorrectQuestions,
		session.WrongQuestions,
		session.CurrentScore,
	)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	return nil
}

func updateSessionQuestions(ctx context.Context, tx dbTX, sessionID string, sessionQuestions []SessionQuestion) error {
	for _, sessionQuestion := range sessionQuestions {
		if err := updateSessionQuestionRow(ctx, tx, sessionID, sessionQuestion); err != nil {
			return err
		}
	}

	return nil
}

func updateSessionQuestionRow(ctx context.Context, tx dbTX, sessionID string, sessionQuestion SessionQuestion) error {
	_, err := tx.Exec(
		ctx,
		`
		update game_session_questions
		set
			activated_at = $3,
			answered_at = $4,
			selected_answer_index = $5,
			is_correct = $6,
			awarded_points = $7,
			response_time_ms = $8,
			cooldown_applied_ms = $9,
			updated_at = now()
		where session_id = $1 and position = $2
		`,
		sessionID,
		sessionQuestion.Position,
		sessionQuestion.ActivatedAt,
		sessionQuestion.AnsweredAt,
		sessionQuestion.SelectedAnswerIndex,
		sessionQuestion.Correct,
		sessionQuestion.AwardedPoints,
		durationMilliseconds(sessionQuestion.ResponseTime),
		durationMilliseconds(sessionQuestion.CooldownApplied),
	)
	if err != nil {
		return fmt.Errorf("update session question: %w", err)
	}

	return nil
}

func finishReasonValue(reason *FinishReason) *string {
	if reason == nil {
		return nil
	}

	value := string(*reason)
	return &value
}

func durationMilliseconds(duration *time.Duration) *int {
	if duration == nil {
		return nil
	}

	value := int(duration.Milliseconds())
	return &value
}

func newPublicUserID() string {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "user_fallback"
	}

	return "user_" + hex.EncodeToString(randomBytes)
}
