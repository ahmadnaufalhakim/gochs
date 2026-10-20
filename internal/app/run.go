package app

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/engine"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	"github.com/ahmadnaufalhakim/gochs/internal/gui"
	"github.com/ahmadnaufalhakim/gochs/internal/gui3d"
	"github.com/ahmadnaufalhakim/gochs/internal/lichess"
	"github.com/ahmadnaufalhakim/gochs/internal/repository"
	"github.com/ahmadnaufalhakim/gochs/internal/storage"
)

func Run() error {
	loadPuzzle, recordPuzzleAttempt, training, closeDatabases := puzzleService()
	defer closeDatabases()
	stockfishPath, _ := engine.FindStockfish()
	store, err := lichess.NewDefaultTokenStore()
	if err != nil {
		return err
	}
	lichessService := lichess.NewService(nil, store, os.Getenv("GOCHS_LICHESS_TOKEN"))
	syncLichessRating := func(account entity.LichessAccount) (entity.TrainingProfile, error) {
		if training == nil {
			return entity.TrainingProfile{}, fmt.Errorf("offline training is unavailable")
		}
		return training.SyncLichessProfile(account, time.Now())
	}
	loadOfflinePuzzleRating := func() (entity.TrainingProfile, bool, error) {
		if training == nil {
			return entity.TrainingProfile{}, false, nil
		}
		return training.ActiveProfile()
	}
	result, err := gui.Run(loadPuzzle, recordPuzzleAttempt, syncLichessRating, loadOfflinePuzzleRating, stockfishPath, lichessService)
	if err != nil {
		return err
	}
	if result == gui.Exit {
		return nil
	}

	return nil
}

func Run3D() error {
	loadPuzzle, recordPuzzleAttempt, training, closeDatabases := puzzleService()
	defer closeDatabases()
	stockfishPath, _ := engine.FindStockfish()
	store, err := lichess.NewDefaultTokenStore()
	if err != nil {
		return err
	}
	lichessService := lichess.NewService(nil, store, os.Getenv("GOCHS_LICHESS_TOKEN"))
	syncLichessRating := func(account entity.LichessAccount) (entity.TrainingProfile, error) {
		if training == nil {
			return entity.TrainingProfile{}, fmt.Errorf("offline training is unavailable")
		}
		return training.SyncLichessProfile(account, time.Now())
	}
	loadOfflinePuzzleRating := func() (entity.TrainingProfile, bool, error) {
		if training == nil {
			return entity.TrainingProfile{}, false, nil
		}
		return training.ActiveProfile()
	}
	return gui3d.Run(gui3d.Config{
		LoadPuzzle: func() (entity.Puzzle, error) {
			return loadPuzzle()
		},
		RecordPuzzleAttempt: func(puzzle entity.Puzzle, outcome entity.PuzzleOutcome) (entity.TrainingResult, error) {
			return recordPuzzleAttempt(puzzle, outcome)
		},
		SyncLichessRating:       syncLichessRating,
		LoadOfflinePuzzleRating: loadOfflinePuzzleRating,
		Lichess:                 lichessService,
		StockfishPath:           stockfishPath,
	})
}

func puzzleService() (gui.PuzzleLoader, gui.PuzzleAttemptRecorder, *repository.TrainingRepository, func()) {
	_, err := storage.EnsureDataDirectory()
	if err != nil {
		return unavailablePuzzleService(err)
	}
	puzzlePath, err := storage.PuzzleDatabasePath()
	if err != nil {
		return unavailablePuzzleService(err)
	}
	trainingPath, err := storage.TrainingDatabasePath()
	if err != nil {
		return unavailablePuzzleService(err)
	}
	trainingDatabase, err := sql.Open("sqlite", "file:"+trainingPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return unavailablePuzzleService(fmt.Errorf("open training database: %w", err))
	}
	if err := repository.CreateTrainingSchema(trainingDatabase); err != nil {
		trainingDatabase.Close()
		return unavailablePuzzleService(err)
	}
	training := repository.NewTrainingRepository(trainingDatabase)
	if _, err := os.Stat(puzzlePath); err != nil {
		return func() (entity.Puzzle, error) {
				return entity.Puzzle{}, fmt.Errorf("puzzle database not found; run scripts/setup-puzzles.sh")
			}, func(entity.Puzzle, entity.PuzzleOutcome) (entity.TrainingResult, error) {
				return entity.TrainingResult{}, nil
			}, training, func() { trainingDatabase.Close() }
	}
	puzzleDatabase, err := sql.Open("sqlite", "file:"+puzzlePath+"?_pragma=foreign_keys(1)")
	if err != nil {
		trainingDatabase.Close()
		return unavailablePuzzleService(fmt.Errorf("open puzzle database: %w", err))
	}
	puzzles := repository.NewPuzzleRepository(puzzleDatabase)
	return func() (entity.Puzzle, error) {
			target := repository.DefaultPuzzleRating()
			if profile, ok, err := training.ActiveProfile(); err != nil {
				return entity.Puzzle{}, err
			} else if ok {
				target = profile.Rating
			}
			return puzzles.RandomPuzzle(target, 100, nil)
		}, func(puzzle entity.Puzzle, outcome entity.PuzzleOutcome) (entity.TrainingResult, error) {
			return training.RecordAttempt(puzzle, outcome, time.Now())
		}, training, func() {
			puzzleDatabase.Close()
			trainingDatabase.Close()
		}
}

func unavailablePuzzleService(err error) (gui.PuzzleLoader, gui.PuzzleAttemptRecorder, *repository.TrainingRepository, func()) {
	return func() (entity.Puzzle, error) {
			return entity.Puzzle{}, err
		}, func(entity.Puzzle, entity.PuzzleOutcome) (entity.TrainingResult, error) {
			return entity.TrainingResult{}, err
		}, nil, func() {}
}
