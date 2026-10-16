package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/entity"
)

const defaultPuzzleRating = 1500

var trainingSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS profiles (
		PROFILE_ID TEXT PRIMARY KEY,
		LICHESS_USER_ID TEXT UNIQUE,
		USERNAME TEXT NOT NULL,
		BASELINE_RATING INTEGER NOT NULL,
		RATING INTEGER NOT NULL,
		GAMES_PLAYED INTEGER NOT NULL DEFAULT 0,
		SOLVED INTEGER NOT NULL DEFAULT 0,
		FAILED INTEGER NOT NULL DEFAULT 0,
		CREATED_AT INTEGER NOT NULL,
		LAST_LICHESS_RATING INTEGER,
		LAST_LICHESS_SYNC_AT INTEGER
	)`,
	`CREATE TABLE IF NOT EXISTS puzzle_attempts (
		PROFILE_ID TEXT NOT NULL,
		PUZZLE_ID TEXT NOT NULL,
		OUTCOME TEXT NOT NULL,
		PUZZLE_RATING INTEGER NOT NULL,
		RATING_BEFORE INTEGER NOT NULL,
		RATING_AFTER INTEGER NOT NULL,
		RATING_CHANGE INTEGER NOT NULL,
		ATTEMPTED_AT INTEGER NOT NULL,
		PRIMARY KEY (PROFILE_ID, PUZZLE_ID),
		FOREIGN KEY (PROFILE_ID) REFERENCES profiles(PROFILE_ID)
	)`,
	`CREATE TABLE IF NOT EXISTS rating_history (
		EVENT_ID INTEGER PRIMARY KEY,
		PROFILE_ID TEXT NOT NULL,
		RATING_BEFORE INTEGER NOT NULL,
		RATING_AFTER INTEGER NOT NULL,
		RATING_CHANGE INTEGER NOT NULL,
		REASON TEXT NOT NULL,
		CREATED_AT INTEGER NOT NULL,
		FOREIGN KEY (PROFILE_ID) REFERENCES profiles(PROFILE_ID)
	)`,
	`CREATE TABLE IF NOT EXISTS training_settings (
		KEY TEXT PRIMARY KEY,
		VALUE TEXT NOT NULL
	)`,
}

type TrainingRepository struct {
	db *sql.DB
}

func NewTrainingRepository(db *sql.DB) *TrainingRepository {
	return &TrainingRepository{db: db}
}

func CreateTrainingSchema(db *sql.DB) error {
	for _, statement := range trainingSchemaStatements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create training schema: %w", err)
		}
	}
	return nil
}

func (r *TrainingRepository) ActiveProfile() (entity.TrainingProfile, bool, error) {
	var profile entity.TrainingProfile
	var syncedAt sql.NullInt64
	err := r.db.QueryRow(`SELECT P.PROFILE_ID, P.LICHESS_USER_ID, P.USERNAME, P.BASELINE_RATING, P.RATING,
		P.GAMES_PLAYED, P.SOLVED, P.FAILED, COALESCE(P.LAST_LICHESS_RATING, 0), P.LAST_LICHESS_SYNC_AT
		FROM profiles P JOIN training_settings S ON S.VALUE = P.PROFILE_ID WHERE S.KEY = 'ACTIVE_PROFILE_ID'`).Scan(
		&profile.ID, &profile.LichessUserID, &profile.Username, &profile.BaselineRating, &profile.Rating,
		&profile.GamesPlayed, &profile.Solved, &profile.Failed, &profile.LastLichessRating, &syncedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.TrainingProfile{}, false, nil
	}
	if err != nil {
		return entity.TrainingProfile{}, false, fmt.Errorf("load active training profile: %w", err)
	}
	if syncedAt.Valid {
		profile.LastSyncedAt = time.UnixMilli(syncedAt.Int64)
	}
	return profile, true, nil
}

func (r *TrainingRepository) SyncLichessProfile(account entity.LichessAccount, now time.Time) (entity.TrainingProfile, error) {
	if account.ID == "" || account.Puzzle == 0 {
		return entity.TrainingProfile{}, fmt.Errorf("Lichess Puzzle rating is unavailable")
	}
	profileID := "lichess:" + account.ID
	timestamp := now.UnixMilli()
	tx, err := r.db.Begin()
	if err != nil {
		return entity.TrainingProfile{}, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO profiles (PROFILE_ID, LICHESS_USER_ID, USERNAME, BASELINE_RATING, RATING, LAST_LICHESS_RATING, LAST_LICHESS_SYNC_AT, CREATED_AT)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(LICHESS_USER_ID) DO UPDATE SET USERNAME = excluded.USERNAME, BASELINE_RATING = excluded.BASELINE_RATING,
		RATING = excluded.RATING, LAST_LICHESS_RATING = excluded.LAST_LICHESS_RATING, LAST_LICHESS_SYNC_AT = excluded.LAST_LICHESS_SYNC_AT`,
		profileID, account.ID, account.Username, account.Puzzle, account.Puzzle, account.Puzzle, timestamp, timestamp)
	if err != nil {
		return entity.TrainingProfile{}, fmt.Errorf("sync training profile: %w", err)
	}
	_, err = tx.Exec(`INSERT INTO training_settings (KEY, VALUE) VALUES ('ACTIVE_PROFILE_ID', ?)
		ON CONFLICT(KEY) DO UPDATE SET VALUE = excluded.VALUE`, profileID)
	if err != nil {
		return entity.TrainingProfile{}, fmt.Errorf("set active training profile: %w", err)
	}
	_, err = tx.Exec(`INSERT INTO rating_history (PROFILE_ID, RATING_BEFORE, RATING_AFTER, RATING_CHANGE, REASON, CREATED_AT)
		VALUES (?, ?, ?, ?, 'lichess_sync', ?)`, profileID, account.Puzzle, account.Puzzle, 0, timestamp)
	if err != nil {
		return entity.TrainingProfile{}, fmt.Errorf("record Lichess sync: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return entity.TrainingProfile{}, err
	}
	return r.ActiveProfileValue()
}

func (r *TrainingRepository) ActiveProfileValue() (entity.TrainingProfile, error) {
	profile, ok, err := r.ActiveProfile()
	if err != nil {
		return entity.TrainingProfile{}, err
	}
	if !ok {
		return entity.TrainingProfile{}, fmt.Errorf("active training profile not found")
	}
	return profile, nil
}

func (r *TrainingRepository) RecordAttempt(puzzle entity.Puzzle, outcome entity.PuzzleOutcome, now time.Time) (entity.TrainingResult, error) {
	profile, ok, err := r.ActiveProfile()
	if err != nil || !ok {
		return entity.TrainingResult{}, err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return entity.TrainingResult{}, err
	}
	defer tx.Rollback()
	var exists int
	err = tx.QueryRow("SELECT 1 FROM puzzle_attempts WHERE PROFILE_ID = ? AND PUZZLE_ID = ?", profile.ID, puzzle.ID).Scan(&exists)
	if err == nil {
		return entity.TrainingResult{Outcome: outcome, Before: profile.Rating, After: profile.Rating}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return entity.TrainingResult{}, fmt.Errorf("check puzzle attempt: %w", err)
	}
	score := 0.0
	if outcome == entity.PuzzleSolved {
		score = 1
	}
	expected := 1 / (1 + math.Pow(10, float64(puzzle.Rating-profile.Rating)/400))
	change := int(math.Round(32 * (score - expected)))
	after := profile.Rating + change
	timestamp := now.UnixMilli()
	_, err = tx.Exec(`INSERT INTO puzzle_attempts (PROFILE_ID, PUZZLE_ID, OUTCOME, PUZZLE_RATING, RATING_BEFORE, RATING_AFTER, RATING_CHANGE, ATTEMPTED_AT)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, profile.ID, puzzle.ID, outcome, puzzle.Rating, profile.Rating, after, change, timestamp)
	if err != nil {
		return entity.TrainingResult{}, fmt.Errorf("record puzzle attempt: %w", err)
	}
	solved, failed := 0, 1
	if outcome == entity.PuzzleSolved {
		solved, failed = 1, 0
	}
	_, err = tx.Exec(`UPDATE profiles SET RATING = ?, GAMES_PLAYED = GAMES_PLAYED + 1, SOLVED = SOLVED + ?, FAILED = FAILED + ? WHERE PROFILE_ID = ?`, after, solved, failed, profile.ID)
	if err != nil {
		return entity.TrainingResult{}, fmt.Errorf("update training profile: %w", err)
	}
	_, err = tx.Exec(`INSERT INTO rating_history (PROFILE_ID, RATING_BEFORE, RATING_AFTER, RATING_CHANGE, REASON, CREATED_AT)
		VALUES (?, ?, ?, ?, ?, ?)`, profile.ID, profile.Rating, after, change, "puzzle_"+string(outcome), timestamp)
	if err != nil {
		return entity.TrainingResult{}, fmt.Errorf("record rating history: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return entity.TrainingResult{}, err
	}
	return entity.TrainingResult{Rated: true, Outcome: outcome, Before: profile.Rating, After: after, Change: change}, nil
}

func DefaultPuzzleRating() int { return defaultPuzzleRating }
