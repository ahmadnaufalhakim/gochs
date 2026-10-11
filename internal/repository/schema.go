package repository

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

var puzzleSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS puzzles (
		PUZZLE_ID TEXT PRIMARY KEY,
		FEN TEXT NOT NULL,
		MOVES TEXT NOT NULL,
		RATING INTEGER NOT NULL,
		RATING_DEVIATION INTEGER NOT NULL,
		POPULARITY INTEGER NOT NULL,
		PLAY_COUNT INTEGER NOT NULL,
		THEME_IDS TEXT NOT NULL,
		GAME_URL TEXT NOT NULL,
		OPENING_TAGS TEXT NOT NULL,
		DAILY_DATE INTEGER
	)`,
	`CREATE TABLE IF NOT EXISTS themes (
		THEME_ID TEXT PRIMARY KEY,
		DISPLAY_NAME TEXT NOT NULL,
		PUZZLE_COUNT INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS puzzle_themes (
		PUZZLE_ID TEXT NOT NULL,
		THEME_ID TEXT NOT NULL,
		PRIMARY KEY (PUZZLE_ID, THEME_ID),
		FOREIGN KEY (PUZZLE_ID) REFERENCES puzzles(PUZZLE_ID),
		FOREIGN KEY (THEME_ID) REFERENCES themes(THEME_ID)
	)`,
}

var puzzleIndexStatements = []string{
	`CREATE INDEX IF NOT EXISTS puzzles_by_rating ON puzzles(RATING)`,
	`CREATE INDEX IF NOT EXISTS puzzle_themes_by_theme ON puzzle_themes(THEME_ID, PUZZLE_ID)`,
}

var dropPuzzleIndexStatements = []string{
	`DROP INDEX IF EXISTS puzzles_by_rating`,
	`DROP INDEX IF EXISTS puzzle_themes_by_theme`,
}

func CreatePuzzleSchema(db *sql.DB) error {
	for _, statement := range puzzleSchemaStatements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create puzzle schema: %w", err)
		}
	}
	return nil
}

func CreatePuzzleIndexes(db *sql.DB) error {
	for _, statement := range puzzleIndexStatements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create puzzle indexes: %w", err)
		}
	}
	return nil
}

func DropPuzzleIndexes(db *sql.DB) error {
	for _, statement := range dropPuzzleIndexStatements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("drop puzzle indexes: %w", err)
		}
	}
	return nil
}
