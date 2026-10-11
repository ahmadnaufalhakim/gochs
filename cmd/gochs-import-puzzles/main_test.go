package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/repository"
)

func TestRunImportsAllPuzzleColumns(t *testing.T) {
	directory := t.TempDir()
	csvPath := filepath.Join(directory, "puzzles.csv")
	themesPath := filepath.Join(directory, "themes.json")
	databasePath := filepath.Join(directory, "puzzles.db")
	csv := strings.Join([]string{
		"PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags,DailyDate",
		"test01,rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1,e2e4,1500,80,90,100,fork pin,https://lichess.org/test,Italian_Game,1607774862751",
	}, "\n")
	if err := os.WriteFile(csvPath, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}
	themes := `{"themes":[{"key":"fork","puzzleCount":1,"xmlLabel":"Fork"},{"key":"pin","puzzleCount":1,"xmlLabel":"Pin"}]}`
	if err := os.WriteFile(themesPath, []byte(themes), 0o600); err != nil {
		t.Fatal(err)
	}

	var progress strings.Builder
	if err := run(csvPath, themesPath, databasePath, &progress); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+databasePath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	puzzle, err := repository.NewPuzzleRepository(db).RandomPuzzle(1500, 0, []string{"fork"})
	if err != nil {
		t.Fatal(err)
	}
	if puzzle.GameURL != "https://lichess.org/test" || len(puzzle.OpeningTags) != 1 || puzzle.OpeningTags[0] != "Italian_Game" {
		t.Errorf("puzzle metadata = %#v", puzzle)
	}
	if len(puzzle.Themes) != 2 {
		t.Errorf("themes = %#v, want both normalized themes", puzzle.Themes)
	}
}
