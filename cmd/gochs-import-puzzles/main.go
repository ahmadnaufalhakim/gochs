package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ahmadnaufalhakim/gochs/internal/repository"
)

const importBatchSize = 10_000

type themeFile struct {
	Themes []themeMetadata `json:"themes"`
}

type themeMetadata struct {
	Key         string `json:"key"`
	PuzzleCount int    `json:"puzzleCount"`
	XMLLabel    string `json:"xmlLabel"`
}

func main() {
	inputPath := flag.String("input", "data/puzzle/lichess_db_puzzle.csv", "path to the Lichess puzzle CSV")
	themesPath := flag.String("themes", "data/puzzle/themes.json", "path to generated theme metadata")
	databasePath := flag.String("database", "data/puzzle/puzzles.db", "path to write the SQLite puzzle database")
	flag.Parse()

	if err := run(*inputPath, *themesPath, *databasePath, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath, themesPath, databasePath string, progress io.Writer) error {
	themes, err := readThemes(themesPath)
	if err != nil {
		return fmt.Errorf("read theme metadata: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+databasePath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("open puzzle database: %w", err)
	}
	defer db.Close()
	if err := repository.CreatePuzzleSchema(db); err != nil {
		return err
	}
	if err := repository.DropPuzzleIndexes(db); err != nil {
		return err
	}
	if err := clearDatabase(db); err != nil {
		return err
	}
	if err := insertThemes(db, themes); err != nil {
		return err
	}
	count, err := importPuzzles(db, inputPath, progress)
	if err != nil {
		return err
	}
	if err := repository.CreatePuzzleIndexes(db); err != nil {
		return err
	}
	if err := verifyImport(db, count, len(themes)); err != nil {
		return err
	}
	fmt.Fprintf(progress, "Imported %d puzzles.\n", count)
	return nil
}

func readThemes(path string) ([]themeMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var metadata themeFile
	if err := json.NewDecoder(file).Decode(&metadata); err != nil {
		return nil, err
	}
	if len(metadata.Themes) == 0 {
		return nil, errors.New("theme metadata contains no themes")
	}
	return metadata.Themes, nil
}

func clearDatabase(db *sql.DB) error {
	for _, table := range []string{"puzzle_themes", "puzzles", "themes"} {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	return nil
}

func insertThemes(db *sql.DB, themes []themeMetadata) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statement, err := tx.Prepare("INSERT INTO themes (THEME_ID, DISPLAY_NAME, PUZZLE_COUNT) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, theme := range themes {
		if _, err := statement.Exec(theme.Key, theme.XMLLabel, theme.PuzzleCount); err != nil {
			return fmt.Errorf("insert theme %q: %w", theme.Key, err)
		}
	}
	return tx.Commit()
}

func importPuzzles(db *sql.DB, path string, progress io.Writer) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	reader := csv.NewReader(bufio.NewReaderSize(file, 1024*1024))
	header, err := reader.Read()
	if err != nil {
		return 0, err
	}
	columns, err := puzzleColumns(header)
	if err != nil {
		return 0, err
	}

	count := 0
	var tx *sql.Tx
	var insertPuzzle *sql.Stmt
	var insertTheme *sql.Stmt
	closeBatch := func() error {
		if insertPuzzle == nil {
			return nil
		}
		if err := insertPuzzle.Close(); err != nil {
			return err
		}
		if err := insertTheme.Close(); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		insertPuzzle = nil
		insertTheme = nil
		tx = nil
		return nil
	}
	startBatch := func() error {
		tx, err = db.Begin()
		if err != nil {
			return err
		}
		insertPuzzle, err = tx.Prepare(`INSERT INTO puzzles (
			PUZZLE_ID, FEN, MOVES, RATING, RATING_DEVIATION, POPULARITY, PLAY_COUNT,
			THEME_IDS, GAME_URL, OPENING_TAGS, DAILY_DATE
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			tx.Rollback()
			return err
		}
		insertTheme, err = tx.Prepare("INSERT INTO puzzle_themes (PUZZLE_ID, THEME_ID) VALUES (?, ?)")
		if err != nil {
			insertPuzzle.Close()
			tx.Rollback()
			return err
		}
		return nil
	}

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return count, fmt.Errorf("read record %d: %w", count+2, err)
		}
		if len(record) != len(header) {
			return count, fmt.Errorf("record %d has %d fields, want %d", count+2, len(record), len(header))
		}
		if insertPuzzle == nil {
			if err := startBatch(); err != nil {
				return count, err
			}
		}
		if err := insertPuzzleRecord(insertPuzzle, insertTheme, record, columns); err != nil {
			tx.Rollback()
			return count, fmt.Errorf("import record %d: %w", count+2, err)
		}
		count++
		if count%importBatchSize == 0 {
			if err := closeBatch(); err != nil {
				return count, err
			}
			fmt.Fprintf(progress, "Imported %d puzzles...\n", count)
		}
	}
	if err := closeBatch(); err != nil {
		return count, err
	}
	return count, nil
}

func puzzleColumns(header []string) (map[string]int, error) {
	required := []string{"PuzzleId", "FEN", "Moves", "Rating", "RatingDeviation", "Popularity", "NbPlays", "Themes", "GameUrl", "OpeningTags", "DailyDate"}
	columns := make(map[string]int, len(header))
	for index, name := range header {
		columns[name] = index
	}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("CSV header has no %s column", name)
		}
	}
	return columns, nil
}

func insertPuzzleRecord(puzzles, puzzleThemes *sql.Stmt, record []string, columns map[string]int) error {
	value := func(name string) string { return record[columns[name]] }
	rating, err := strconv.Atoi(value("Rating"))
	if err != nil {
		return fmt.Errorf("parse rating: %w", err)
	}
	ratingDeviation, err := strconv.Atoi(value("RatingDeviation"))
	if err != nil {
		return fmt.Errorf("parse rating deviation: %w", err)
	}
	popularity, err := strconv.Atoi(value("Popularity"))
	if err != nil {
		return fmt.Errorf("parse popularity: %w", err)
	}
	playCount, err := strconv.Atoi(value("NbPlays"))
	if err != nil {
		return fmt.Errorf("parse play count: %w", err)
	}
	var dailyDate any
	if rawDate := value("DailyDate"); rawDate != "" {
		dailyDate, err = strconv.ParseInt(rawDate, 10, 64)
		if err != nil {
			return fmt.Errorf("parse daily date: %w", err)
		}
	}
	puzzleID := value("PuzzleId")
	themes := value("Themes")
	if _, err := puzzles.Exec(
		puzzleID, value("FEN"), value("Moves"), rating, ratingDeviation, popularity, playCount,
		themes, value("GameUrl"), value("OpeningTags"), dailyDate,
	); err != nil {
		return err
	}
	for _, themeID := range strings.Fields(themes) {
		if _, err := puzzleThemes.Exec(puzzleID, themeID); err != nil {
			return err
		}
	}
	return nil
}

func verifyImport(db *sql.DB, puzzleCount, themeCount int) error {
	var storedPuzzleCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM puzzles").Scan(&storedPuzzleCount); err != nil {
		return err
	}
	if storedPuzzleCount != puzzleCount {
		return fmt.Errorf("stored %d puzzles, imported %d", storedPuzzleCount, puzzleCount)
	}
	var storedThemeCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM themes").Scan(&storedThemeCount); err != nil {
		return err
	}
	if storedThemeCount != themeCount {
		return fmt.Errorf("stored %d themes, expected %d", storedThemeCount, themeCount)
	}
	var orphanedThemeCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM puzzle_themes AS PT
		LEFT JOIN themes AS T ON T.THEME_ID = PT.THEME_ID
		WHERE T.THEME_ID IS NULL`).Scan(&orphanedThemeCount); err != nil {
		return err
	}
	if orphanedThemeCount != 0 {
		return fmt.Errorf("found %d puzzle themes without metadata", orphanedThemeCount)
	}
	return nil
}
