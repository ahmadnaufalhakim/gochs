package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/entity"
)

func TestTrainingRatingRecordsFirstAttemptOnly(t *testing.T) {
	db, err := sql.Open("sqlite", "file:training-repository-test?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := CreateTrainingSchema(db); err != nil {
		t.Fatal(err)
	}
	repository := NewTrainingRepository(db)
	now := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	profile, err := repository.SyncLichessProfile(entity.LichessAccount{ID: "player", Username: "example", Puzzle: 1500}, now)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Rating != 1500 || profile.BaselineRating != 1500 {
		t.Errorf("profile = %#v", profile)
	}

	puzzle := entity.Puzzle{ID: "puzzle-1", Rating: 1700}
	result, err := repository.RecordAttempt(puzzle, entity.PuzzleFailed, now)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Rated || result.Change >= 0 || result.After >= result.Before {
		t.Errorf("failed attempt = %#v", result)
	}

	replay, err := repository.RecordAttempt(puzzle, entity.PuzzleSolved, now)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Rated || replay.After != result.After {
		t.Errorf("replay result = %#v", replay)
	}
	active, ok, err := repository.ActiveProfile()
	if err != nil || !ok {
		t.Fatalf("ActiveProfile() = %#v, %v, %v", active, ok, err)
	}
	if active.GamesPlayed != 1 || active.Failed != 1 || active.Solved != 0 || active.Rating != result.After {
		t.Errorf("active profile = %#v", active)
	}
}

func TestTrainingSyncRebasesExistingProfile(t *testing.T) {
	db, err := sql.Open("sqlite", "file:training-sync-test?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := CreateTrainingSchema(db); err != nil {
		t.Fatal(err)
	}
	repository := NewTrainingRepository(db)
	now := time.Now()
	if _, err := repository.SyncLichessProfile(entity.LichessAccount{ID: "player", Username: "old-name", Puzzle: 1500}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.RecordAttempt(entity.Puzzle{ID: "puzzle-1", Rating: 1500}, entity.PuzzleSolved, now); err != nil {
		t.Fatal(err)
	}
	profile, err := repository.SyncLichessProfile(entity.LichessAccount{ID: "player", Username: "new-name", Puzzle: 1800}, now)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Username != "new-name" || profile.Rating != 1800 || profile.BaselineRating != 1800 || profile.GamesPlayed != 1 {
		t.Errorf("rebased profile = %#v", profile)
	}
}
