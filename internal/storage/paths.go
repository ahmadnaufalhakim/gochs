package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func DataDirectory() (string, error) {
	if directory := os.Getenv("XDG_DATA_HOME"); directory != "" {
		return filepath.Join(directory, "gochs"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "gochs"), nil
}

func EnsureDataDirectory() (string, error) {
	directory, err := DataDirectory()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create data directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", fmt.Errorf("secure data directory: %w", err)
	}
	return directory, nil
}

func PuzzleDatabasePath() (string, error) {
	directory, err := DataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "puzzles.db"), nil
}

func TrainingDatabasePath() (string, error) {
	directory, err := DataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "training.db"), nil
}
