package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/ahmadnaufalhakim/gochs/internal/repository"
	"github.com/ahmadnaufalhakim/gochs/internal/storage"

	_ "modernc.org/sqlite"
)

func main() {
	directory, err := storage.EnsureDataDirectory()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path, err := storage.TrainingDatabasePath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	if err := repository.CreateTrainingSchema(db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Initialized training database in %s\n", directory)
}
