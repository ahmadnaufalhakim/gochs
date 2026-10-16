package storage

import (
	"path/filepath"
	"testing"
)

func TestDataPathsUseXDGDataHome(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("XDG_DATA_HOME", directory)
	dataDirectory, err := EnsureDataDirectory()
	if err != nil {
		t.Fatal(err)
	}
	if dataDirectory != filepath.Join(directory, "gochs") {
		t.Errorf("DataDirectory() = %q", dataDirectory)
	}
	puzzlePath, err := PuzzleDatabasePath()
	if err != nil {
		t.Fatal(err)
	}
	if puzzlePath != filepath.Join(dataDirectory, "puzzles.db") {
		t.Errorf("PuzzleDatabasePath() = %q", puzzlePath)
	}
}
