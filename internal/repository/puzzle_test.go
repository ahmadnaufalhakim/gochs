package repository

import (
	"database/sql"
	"errors"
	"testing"
)

func TestPuzzleRepositoryRandomPuzzleFiltersAnyTheme(t *testing.T) {
	db := openTestDatabase(t)
	insertTestTheme(t, db, "fork", "Fork", 1)
	insertTestTheme(t, db, "pin", "Pin", 2)
	insertTestPuzzle(t, db, "fork-puzzle", 1500)
	insertTestPuzzle(t, db, "pin-puzzle", 1500)
	if _, err := db.Exec("INSERT INTO puzzle_themes (PUZZLE_ID, THEME_ID) VALUES (?, ?), (?, ?)", "fork-puzzle", "fork", "pin-puzzle", "pin"); err != nil {
		t.Fatal(err)
	}

	repository := NewPuzzleRepository(db)
	puzzle, err := repository.RandomPuzzle(1500, 0, []string{"fork", "pin"})
	if err != nil {
		t.Fatal(err)
	}
	if puzzle.ID != "fork-puzzle" && puzzle.ID != "pin-puzzle" {
		t.Errorf("puzzle ID = %q, want a theme-matching puzzle", puzzle.ID)
	}
	if len(puzzle.Moves) != 1 || puzzle.Moves[0].UCI() != "e2e4" {
		t.Errorf("moves = %v, want [e2e4]", puzzle.Moves)
	}
	if puzzle.DailyDate == nil || *puzzle.DailyDate != 1_607_774_862_751 {
		t.Errorf("daily date = %v", puzzle.DailyDate)
	}
}

func TestPuzzleRepositoryReportsNoPuzzle(t *testing.T) {
	db := openTestDatabase(t)
	repository := NewPuzzleRepository(db)
	_, err := repository.RandomPuzzle(1500, 0, nil)
	if !errors.Is(err, ErrPuzzleNotFound) {
		t.Errorf("error = %v, want ErrPuzzleNotFound", err)
	}
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:puzzle-repository-test?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := CreatePuzzleSchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func insertTestTheme(t *testing.T, db *sql.DB, id, label string, count int) {
	t.Helper()
	if _, err := db.Exec("INSERT INTO themes (THEME_ID, DISPLAY_NAME, PUZZLE_COUNT) VALUES (?, ?, ?)", id, label, count); err != nil {
		t.Fatal(err)
	}
}

func insertTestPuzzle(t *testing.T, db *sql.DB, id string, rating int) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO puzzles (
		PUZZLE_ID, FEN, MOVES, RATING, RATING_DEVIATION, POPULARITY, PLAY_COUNT,
		THEME_IDS, GAME_URL, OPENING_TAGS, DAILY_DATE
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"e2e4",
		rating,
		80,
		90,
		100,
		"fork pin",
		"https://lichess.org/example",
		"Kings_Pawn_Game",
		int64(1_607_774_862_751),
	)
	if err != nil {
		t.Fatal(err)
	}
}
