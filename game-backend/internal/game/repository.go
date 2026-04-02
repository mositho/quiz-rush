package game

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"quiz-rush/game-backend/internal/middleware"

	"github.com/georgysavva/scany/v2/pgxscan"
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
	ID           string `db:"id"`
	PublicUserID string `db:"public_user_id"`
	DisplayName  string `db:"display_name"`
}

type storedScore struct {
	ScoreID                string                `db:"id"`
	SessionID              string                `db:"session_id"`
	FinishedAt             time.Time             `db:"finished_at"`
	FinishReason           FinishReason          `db:"finish_reason"`
	Score                  int                   `db:"score"`
	CorrectQuestions       int                   `db:"correct_questions"`
	WrongQuestions         int                   `db:"wrong_questions"`
	AnsweredQuestions      int                   `db:"answered_questions"`
	TotalQuestions         int                   `db:"total_questions"`
	DurationSeconds        int                   `db:"duration_seconds"`
	PlayedMs               int64                 `db:"played_ms"`
	SelectedQuestionSetIDs []string              `db:"selected_question_set_ids"`
	ConfigurationKey       string                `db:"configuration_key"`
	QuestionResults        []ScoreQuestionResult `db:"question_results_json"`
	Player                 *userProfile          `db:"-"`
}

type leaderboardEntry struct {
	Rank             int         `db:"-"`
	ScoreID          string      `db:"id"`
	Score            int         `db:"score"`
	FinishedAt       time.Time   `db:"finished_at"`
	ConfigurationKey string      `db:"configuration_key"`
	Player           userProfile `db:"*"`
}

type userStats struct {
	GamesPlayed           int
	BestScore             int
	AverageScore          float64
	TotalCorrectQuestions int
}

type dbTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
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

	publicUserID, err := newPublicUserID()
	if err != nil {
		return "", fmt.Errorf("generate public user id: %w", err)
	}

	var id string
	err = r.db.QueryRow(
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
	var profile userProfile
	err := pgxscan.Get(
		ctx,
		r.db,
		&profile,
		`
		select id, public_user_id, display_name
		from user_profiles
		where id = $1
		`,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get profile by id: %w", err)
	}

	return &profile, nil
}

func (r *sessionRepository) GetProfileByPublicUserID(ctx context.Context, publicUserID string) (*userProfile, error) {
	var profile userProfile
	err := pgxscan.Get(
		ctx,
		r.db,
		&profile,
		`
		select id, public_user_id, display_name
		from user_profiles
		where public_user_id = $1
		`,
		publicUserID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
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
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			log.Printf("failed to rollback create session transaction: %v", rollbackErr)
		}
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
	session, ownerProfileID, isAnonymous, err := loadSessionRow(ctx, r.db, sessionID, false)
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

func (r *sessionRepository) UpdateLockedSession(
	ctx context.Context,
	sessionID string,
	mutate func(session *Session, ownerProfileID *string, isAnonymous bool) error,
) (*Session, *string, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, false, fmt.Errorf("begin locked session transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			log.Printf("failed to rollback locked session transaction: %v", rollbackErr)
		}
	}()

	session, ownerProfileID, isAnonymous, err := loadSessionRow(ctx, tx, sessionID, true)
	if err != nil {
		return nil, nil, false, err
	}

	sessionQuestions, err := loadSessionQuestions(ctx, tx, sessionID)
	if err != nil {
		return nil, nil, false, err
	}
	session.SessionQuestions = sessionQuestions

	if err := mutate(session, ownerProfileID, isAnonymous); err != nil {
		return nil, nil, false, err
	}

	if err := updateSessionRow(ctx, tx, session); err != nil {
		return nil, nil, false, err
	}
	if err := updateSessionQuestions(ctx, tx, session.ID, session.SessionQuestions); err != nil {
		return nil, nil, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, false, fmt.Errorf("commit locked session transaction: %w", err)
	}

	return session, ownerProfileID, isAnonymous, nil
}

func (r *sessionRepository) UpdateSession(ctx context.Context, session *Session) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin update session transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			log.Printf("failed to rollback update session transaction: %v", rollbackErr)
		}
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
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			log.Printf("failed to rollback link session score transaction: %v", rollbackErr)
		}
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
			scoreID, scoreErr := r.scoreIDBySessionID(ctx, tx, sessionID)
			if scoreErr != nil {
				return "", scoreErr
			}
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return "", fmt.Errorf("commit idempotent link session score transaction: %w", commitErr)
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
	var scoreRow struct {
		storedScore
		PlayerID          string `db:"player_id"`
		PlayerPublicID    string `db:"player_public_user_id"`
		PlayerDisplayName string `db:"player_display_name"`
	}

	err := pgxscan.Get(
		ctx,
		r.db,
		&scoreRow,
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
			u.id as player_id,
			u.public_user_id as player_public_user_id,
			u.display_name as player_display_name
		from game_scores s
		join user_profiles u on u.id = s.owner_profile_id
		where s.id = $1 and s.is_public = true and s.is_saved = true
		`,
		scoreID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrScoreNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get score: %w", err)
	}

	score := scoreRow.storedScore
	score.Player = &userProfile{
		ID:           scoreRow.PlayerID,
		PublicUserID: scoreRow.PlayerPublicID,
		DisplayName:  scoreRow.PlayerDisplayName,
	}

	return &score, nil
}

func (r *sessionRepository) ListScoresByPublicUserID(ctx context.Context, publicUserID string) (*userProfile, []storedScore, error) {
	profile, err := r.GetProfileByPublicUserID(ctx, publicUserID)
	if err != nil {
		return nil, nil, err
	}

	var scores []storedScore
	err = pgxscan.Select(
		ctx,
		r.db,
		&scores,
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

	return profile, scores, nil
}

func (r *sessionRepository) GetUserStatsByPublicUserID(ctx context.Context, publicUserID string) (*userProfile, userStats, error) {
	profile, err := r.GetProfileByPublicUserID(ctx, publicUserID)
	if err != nil {
		return nil, userStats{}, err
	}

	var statsRow struct {
		GamesPlayed           int     `db:"count"`
		BestScore             int     `db:"max"`
		AverageScore          float64 `db:"avg"`
		TotalCorrectQuestions int     `db:"sum"`
	}

	err = pgxscan.Get(
		ctx,
		r.db,
		&statsRow,
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
	if err != nil {
		return nil, userStats{}, fmt.Errorf("get user stats by public user id: %w", err)
	}

	stats := userStats{
		GamesPlayed:           statsRow.GamesPlayed,
		BestScore:             statsRow.BestScore,
		AverageScore:          statsRow.AverageScore,
		TotalCorrectQuestions: statsRow.TotalCorrectQuestions,
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

	var entries []leaderboardEntry
	err := pgxscan.Select(ctx, r.db, &entries, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list leaderboard: %w", err)
	}

	// Add rank to each entry
	for i := range entries {
		entries[i].Rank = i + 1
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

func loadSessionRow(ctx context.Context, db dbTX, sessionID string, lock bool) (*Session, *string, bool, error) {
	type sessionRow struct {
		ID                     string     `db:"id"`
		OwnerProfileID         *string    `db:"owner_profile_id"`
		IsAnonymous            bool       `db:"is_anonymous"`
		Status                 string     `db:"status"`
		FinishReason           *string    `db:"finish_reason"`
		StartedAt              time.Time  `db:"started_at"`
		EndsAt                 time.Time  `db:"ends_at"`
		CooldownUntil          *time.Time `db:"cooldown_until"`
		FinishedAt             *time.Time `db:"finished_at"`
		SaveDeadlineAt         *time.Time `db:"save_deadline_at"`
		DurationSeconds        int        `db:"duration_seconds"`
		SelectedQuestionSetIDs []string   `db:"selected_question_set_ids"`
		ConfigurationKey       string     `db:"configuration_key"`
		CurrentQuestionIndex   *int       `db:"current_question_index"`
		TotalQuestions         int        `db:"total_questions"`
		AnsweredQuestions      int        `db:"answered_questions"`
		CorrectQuestions       int        `db:"correct_questions"`
		WrongQuestions         int        `db:"wrong_questions"`
		CurrentScore           int        `db:"current_score"`
	}

	query := `
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
	`
	if lock {
		query += " for update"
	}

	var row sessionRow
	err := pgxscan.Get(ctx, db, &row, query, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, false, ErrSessionNotFound
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("load session: %w", err)
	}

	session := &Session{
		ID:                     row.ID,
		Status:                 SessionStatus(row.Status),
		StartedAt:              row.StartedAt,
		EndsAt:                 row.EndsAt,
		CooldownUntil:          row.CooldownUntil,
		FinishedAt:             row.FinishedAt,
		SaveDeadlineAt:         row.SaveDeadlineAt,
		DurationSeconds:        row.DurationSeconds,
		SelectedQuestionSetIDs: row.SelectedQuestionSetIDs,
		ConfigurationKey:       row.ConfigurationKey,
		CurrentQuestionIndex:   row.CurrentQuestionIndex,
		TotalQuestions:         row.TotalQuestions,
		AnsweredQuestions:      row.AnsweredQuestions,
		CorrectQuestions:       row.CorrectQuestions,
		WrongQuestions:         row.WrongQuestions,
		CurrentScore:           row.CurrentScore,
	}
	if row.FinishReason != nil {
		reason := FinishReason(*row.FinishReason)
		session.FinishReason = &reason
	}

	return session, row.OwnerProfileID, row.IsAnonymous, nil
}

func loadSessionQuestions(ctx context.Context, db dbTX, sessionID string) ([]SessionQuestion, error) {
	type tempSessionQuestion struct {
		Position            int        `db:"position"`
		QuestionID          string     `db:"question_id"`
		QuestionSetID       string     `db:"question_set_id"`
		Difficulty          int        `db:"difficulty"`
		QuestionCategories  []byte     `db:"question_categories"`
		QuestionText        string     `db:"question_text"`
		Options             []byte     `db:"options_json"`
		CorrectAnswerIndex  int        `db:"correct_answer_index"`
		ActivatedAt         *time.Time `db:"activated_at"`
		AnsweredAt          *time.Time `db:"answered_at"`
		SelectedAnswerIndex *int       `db:"selected_answer_index"`
		Correct             *bool      `db:"is_correct"`
		AwardedPoints       *int       `db:"awarded_points"`
		ResponseTimeMS      *int       `db:"response_time_ms"`
		CooldownAppliedMS   *int       `db:"cooldown_applied_ms"`
	}

	var tempQuestions []tempSessionQuestion
	err := pgxscan.Select(
		ctx,
		db,
		&tempQuestions,
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

	// Convert temp questions to SessionQuestion with proper type conversions
	sessionQuestions := make([]SessionQuestion, 0, len(tempQuestions))
	for _, tmp := range tempQuestions {
		sq := SessionQuestion{
			Position:            tmp.Position,
			QuestionID:          tmp.QuestionID,
			QuestionSetID:       tmp.QuestionSetID,
			Difficulty:          tmp.Difficulty,
			QuestionText:        tmp.QuestionText,
			CorrectAnswerIndex:  tmp.CorrectAnswerIndex,
			ActivatedAt:         tmp.ActivatedAt,
			AnsweredAt:          tmp.AnsweredAt,
			SelectedAnswerIndex: tmp.SelectedAnswerIndex,
			Correct:             tmp.Correct,
			AwardedPoints:       tmp.AwardedPoints,
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(tmp.QuestionCategories, &sq.QuestionCategories); err != nil {
			return nil, fmt.Errorf("unmarshal question categories: %w", err)
		}
		if err := json.Unmarshal(tmp.Options, &sq.Options); err != nil {
			return nil, fmt.Errorf("unmarshal question options: %w", err)
		}

		// Convert milliseconds to duration
		if tmp.ResponseTimeMS != nil {
			duration := time.Duration(*tmp.ResponseTimeMS) * time.Millisecond
			sq.ResponseTime = &duration
		}
		if tmp.CooldownAppliedMS != nil {
			duration := time.Duration(*tmp.CooldownAppliedMS) * time.Millisecond
			sq.CooldownApplied = &duration
		}

		sessionQuestions = append(sessionQuestions, sq)
	}

	return sessionQuestions, nil
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

func newPublicUserID() (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return "user_" + hex.EncodeToString(randomBytes), nil
}
