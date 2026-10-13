package app

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/ahmadnaufalhakim/gochs/internal/engine"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	"github.com/ahmadnaufalhakim/gochs/internal/gui"
	"github.com/ahmadnaufalhakim/gochs/internal/repository"
)

func Run() error {
	loadPuzzle, closeDatabase := puzzleLoader()
	defer closeDatabase()
	stockfishPath, _ := engine.FindStockfish()
	result, err := gui.Run(loadPuzzle, stockfishPath)
	if err != nil {
		return err
	}
	if result == gui.Exit {
		return nil
	}

	return nil
}

func puzzleLoader() (gui.PuzzleLoader, func()) {
	const databasePath = "data/puzzle/puzzles.db"
	if _, err := os.Stat(databasePath); err != nil {
		return func() (entity.Puzzle, error) {
			return entity.Puzzle{}, fmt.Errorf("puzzle database not found; run gochs-import-puzzles")
		}, func() {}
	}
	db, err := sql.Open("sqlite", "file:"+databasePath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return func() (entity.Puzzle, error) {
			return entity.Puzzle{}, fmt.Errorf("open puzzle database: %w", err)
		}, func() {}
	}
	repository := repository.NewPuzzleRepository(db)
	return func() (entity.Puzzle, error) {
		return repository.RandomPuzzle(1500, 100, nil)
	}, func() { db.Close() }
}
