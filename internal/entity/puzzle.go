package entity

import "github.com/ahmadnaufalhakim/gochs/internal/chess"

type Puzzle struct {
	ID              string
	FEN             string
	Moves           []chess.Move
	Rating          int
	RatingDeviation int
	Popularity      int
	PlayCount       int
	Themes          []Theme
	GameURL         string
	OpeningTags     []string
	DailyDate       *int64
}

type Theme struct {
	ID          string
	DisplayName string
	PuzzleCount int
}
