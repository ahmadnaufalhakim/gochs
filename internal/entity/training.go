package entity

import "time"

type PuzzleOutcome string

const (
	PuzzleSolved PuzzleOutcome = "solved"
	PuzzleFailed PuzzleOutcome = "failed"
)

type TrainingProfile struct {
	ID                string
	LichessUserID     string
	Username          string
	BaselineRating    int
	Rating            int
	GamesPlayed       int
	Solved            int
	Failed            int
	LastLichessRating int
	LastSyncedAt      time.Time
}

type TrainingResult struct {
	Rated   bool
	Outcome PuzzleOutcome
	Before  int
	After   int
	Change  int
}
