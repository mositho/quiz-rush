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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")
var ErrScoreNotFound = errors.New("score not found")
var ErrProfileNotFound = errors.New("profile not found")
var ErrSessionExpired = errors.New("session expired")
var ErrSessionNotFinished = errors.New("session not finished")
var ErrSessionAlreadyLinked = errors.New("session already linked")

type sessionRepository struct {
	db *pgxpool.Pool
}

type userProfile struct {
	ID           string
	PublicUserID string
	DisplayName  string
}

type storedScore struct {
	ScoreID                string
	SessionID              string
	FinishedAt             time.Time
	FinishReason           FinishReason
	Score                  int
	CorrectQuestions       int
	WrongQuestions         int
	AnsweredQuestions      int
	TotalQuestions         int
	DurationSeconds        int
	PlayedMs               int64
	SelectedQuestionSetIDs []string
	ConfigurationKey       string
	QuestionResults        []ScoreQuestionResult
	Player                 *userProfile
}

type leaderboardEntry struct {
	Rank             int
	ScoreID          string
	Score            int
	FinishedAt       time.Time
	ConfigurationKey string
	Player           userProfile
}

type userStats struct {
	GamesPlayed           int
	BestScore             int
	AverageScore          float64
	TotalCorrectQuestions int
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

func (r *sessionRepository) GetProfileByID(ctx context.Context, id string) (*userProfile, error) {
	row := r.db.QueryRow(
		ctx,
		`
		select id, public_user_id, display_name
		from user_profiles
		where id = $1
		`,
		id,
	)

	var profile userProfile
	if err := row.Scan(&profile.ID, &profile.PublicUserID, &profile.DisplayName); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get profile by id: %w", err)
	}

	return &profile, nil
}

func (r *sessionRepository) GetProfileByPublicUserID(ctx context.Context, publicUserID string) (*userProfile, error) {
	row := r.db.QueryRow(
		ctx,
		`
		select id, public_user_id, display_name
		from user_profiles
		where public_user_id = $1
		`,
		publicUserID,
	)

	var profile userProfile
	if err := row.Scan(&profile.ID, &profile.PublicUserID, &profile.DisplayName); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get profile by public user id: %w", err)
	}

	return &profile, nil
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
	finishedAt time.Time,
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
		finishedAt,
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

func (r *sessionRepository) LinkAnonymousSessionScore(
	ctx context.Context,
	sessionID string,
	ownerProfileID string,
	now time.Time,
) (string, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin link session score transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentOwnerProfileID *string
		isAnonymous           bool
		status                string
		saveDeadlineAt        *time.Time
	)
	err = tx.QueryRow(
		ctx,
		`
		select owner_profile_id, is_anonymous, status, save_deadline_at
		from game_sessions
		where id = $1
		for update
		`,
		sessionID,
	).Scan(&currentOwnerProfileID, &isAnonymous, &status, &saveDeadlineAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load session for linking: %w", err)
	}

	if currentOwnerProfileID != nil {
		if *currentOwnerProfileID == ownerProfileID {
			scoreID, err := r.scoreIDBySessionID(ctx, tx, sessionID)
			if err != nil {
				return "", err
			}
			if err := tx.Commit(ctx); err != nil {
				return "", fmt.Errorf("commit idempotent link session score transaction: %w", err)
			}
			return scoreID, nil
		}
		return "", ErrSessionAlreadyLinked
	}
	if SessionStatus(status) != SessionStatusFinished {
		return "", ErrSessionNotFinished
	}
	if !isAnonymous || saveDeadlineAt == nil || now.After(*saveDeadlineAt) {
		return "", ErrSessionExpired
	}

	scoreID, err := r.scoreIDBySessionID(ctx, tx, sessionID)
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(
		ctx,
		`
		update game_sessions
		set
			owner_profile_id = $2,
			is_anonymous = false,
			updated_at = now()
		where id = $1
		`,
		sessionID,
		ownerProfileID,
	); err != nil {
		return "", fmt.Errorf("link session owner: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`
		update game_scores
		set
			owner_profile_id = $2,
			is_saved = true,
			is_public = true,
			saved_at = $3,
			expires_at = null
		where session_id = $1
		`,
		sessionID,
		ownerProfileID,
		now,
	); err != nil {
		return "", fmt.Errorf("link score owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit link session score transaction: %w", err)
	}

	return scoreID, nil
}

func (r *sessionRepository) GetScore(ctx context.Context, scoreID string) (*storedScore, error) {
	row := r.db.QueryRow(
		ctx,
		`
		select
			s.id,
			s.session_id,
			s.finished_at,
			s.finish_reason,
			s.score,
			s.correct_questions,
			s.wrong_questions,
			s.answered_questions,
			s.total_questions,
			s.duration_seconds,
			s.played_ms,
			s.selected_question_set_ids,
			s.configuration_key,
			s.question_results_json,
			u.id,
			u.public_user_id,
			u.display_name
		from game_scores s
		join user_profiles u on u.id = s.owner_profile_id
		where s.id = $1 and s.is_public = true and s.is_saved = true
		`,
		scoreID,
	)

	score, err := scanStoredScoreWithPlayer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrScoreNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get score: %w", err)
	}

	return score, nil
}

func (r *sessionRepository) ListScoresByPublicUserID(ctx context.Context, publicUserID string) (*userProfile, []storedScore, error) {
	profile, err := r.GetProfileByPublicUserID(ctx, publicUserID)
	if err != nil {
		return nil, nil, err
	}

	rows, err := r.db.Query(
		ctx,
		`
		select
			s.id,
			s.session_id,
			s.finished_at,
			s.finish_reason,
			s.score,
			s.correct_questions,
			s.wrong_questions,
			s.answered_questions,
			s.total_questions,
			s.duration_seconds,
			s.played_ms,
			s.selected_question_set_ids,
			s.configuration_key,
			s.question_results_json
		from game_scores s
		where s.owner_profile_id = $1 and s.is_public = true and s.is_saved = true
		order by s.finished_at desc
		`,
		profile.ID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list scores by public user id: %w", err)
	}
	defer rows.Close()

	scores := make([]storedScore, 0)
	for rows.Next() {
		score, err := scanStoredScoreWithoutPlayer(rows)
		if err != nil {
			return nil, nil, err
		}
		scores = append(scores, score)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate scores by public user id: %w", err)
	}

	return profile, scores, nil
}

func (r *sessionRepository) GetUserStatsByPublicUserID(ctx context.Context, publicUserID string) (*userProfile, userStats, error) {
	profile, err := r.GetProfileByPublicUserID(ctx, publicUserID)
	if err != nil {
		return nil, userStats{}, err
	}

	row := r.db.QueryRow(
		ctx,
		`
		select
			count(*),
			coalesce(max(score), 0),
			coalesce(avg(score), 0),
			coalesce(sum(correct_questions), 0)
		from game_scores
		where owner_profile_id = $1 and is_public = true and is_saved = true
		`,
		profile.ID,
	)

	var stats userStats
	if err := row.Scan(&stats.GamesPlayed, &stats.BestScore, &stats.AverageScore, &stats.TotalCorrectQuestions); err != nil {
		return nil, userStats{}, fmt.Errorf("get user stats by public user id: %w", err)
	}

	return profile, stats, nil
}

func (r *sessionRepository) ListLeaderboard(ctx context.Context, configurationKey *string, limit int) ([]leaderboardEntry, error) {
	query := `
		select
			s.id,
			s.score,
			s.finished_at,
			s.configuration_key,
			u.id,
			u.public_user_id,
			u.display_name
		from game_scores s
		join user_profiles u on u.id = s.owner_profile_id
		where s.is_public = true and s.is_saved = true
	`
	args := []any{}
	if configurationKey != nil && *configurationKey != "" {
		query += ` and s.configuration_key = $1`
		args = append(args, *configurationKey)
	}
	if len(args) == 0 {
		query += ` order by s.score desc, s.finished_at asc limit $1`
		args = append(args, limit)
	} else {
		query += ` order by s.score desc, s.finished_at asc limit $2`
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list leaderboard: %w", err)
	}
	defer rows.Close()

	entries := make([]leaderboardEntry, 0)
	rank := 1
	for rows.Next() {
		var entry leaderboardEntry
		if err := rows.Scan(
			&entry.ScoreID,
			&entry.Score,
			&entry.FinishedAt,
			&entry.ConfigurationKey,
			&entry.Player.ID,
			&entry.Player.PublicUserID,
			&entry.Player.DisplayName,
		); err != nil {
			return nil, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		entry.Rank = rank
		rank++
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaderboard entries: %w", err)
	}

	return entries, nil
}

func (r *sessionRepository) scoreIDBySessionID(ctx context.Context, tx pgx.Tx, sessionID string) (string, error) {
	var scoreID string
	err := tx.QueryRow(
		ctx,
		`
		select id
		from game_scores
		where session_id = $1
		for update
		`,
		sessionID,
	).Scan(&scoreID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrScoreNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load score by session id: %w", err)
	}

	return scoreID, nil
}

func scanStoredScoreWithoutPlayer(rows pgx.Rows) (storedScore, error) {
	score, err := scanStoredScoreCore(rows, false)
	if err != nil {
		return storedScore{}, err
	}

	return *score, nil
}

type scoreScanner interface {
	Scan(dest ...any) error
}

func scanStoredScoreWithPlayer(scanner scoreScanner) (*storedScore, error) {
	return scanStoredScoreCore(scanner, true)
}

func scanStoredScoreCore(scanner scoreScanner, includePlayer bool) (*storedScore, error) {
	var (
		score                storedScore
		finishReason         string
		selectedQuestionSets []byte
		questionResultsRaw   []byte
	)

	if includePlayer {
		var player userProfile
		if err := scanner.Scan(
			&score.ScoreID,
			&score.SessionID,
			&score.FinishedAt,
			&finishReason,
			&score.Score,
			&score.CorrectQuestions,
			&score.WrongQuestions,
			&score.AnsweredQuestions,
			&score.TotalQuestions,
			&score.DurationSeconds,
			&score.PlayedMs,
			&selectedQuestionSets,
			&score.ConfigurationKey,
			&questionResultsRaw,
			&player.ID,
			&player.PublicUserID,
			&player.DisplayName,
		); err != nil {
			return nil, err
		}
		score.Player = &player
	} else {
		if err := scanner.Scan(
			&score.ScoreID,
			&score.SessionID,
			&score.FinishedAt,
			&finishReason,
			&score.Score,
			&score.CorrectQuestions,
			&score.WrongQuestions,
			&score.AnsweredQuestions,
			&score.TotalQuestions,
			&score.DurationSeconds,
			&score.PlayedMs,
			&selectedQuestionSets,
			&score.ConfigurationKey,
			&questionResultsRaw,
		); err != nil {
			return nil, err
		}
	}

	score.FinishReason = FinishReason(finishReason)
	if err := json.Unmarshal(selectedQuestionSets, &score.SelectedQuestionSetIDs); err != nil {
		return nil, fmt.Errorf("unmarshal score selected question set ids: %w", err)
	}
	if err := json.Unmarshal(questionResultsRaw, &score.QuestionResults); err != nil {
		return nil, fmt.Errorf("unmarshal score question results: %w", err)
	}

	return &score, nil
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
