package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
)

const (
	puzzleTable      = "puzzles"
	themeTable       = "themes"
	puzzleThemeTable = "puzzle_themes"
)

var ErrPuzzleNotFound = errors.New("puzzle not found")

type PuzzleRepository struct {
	db *sql.DB
}

func NewPuzzleRepository(db *sql.DB) *PuzzleRepository {
	return &PuzzleRepository{db: db}
}

func (r *PuzzleRepository) RandomPuzzle(targetRating, tolerance int, themeIDs []string) (entity.Puzzle, error) {
	columns := []string{
		"P.PUZZLE_ID", "P.FEN", "P.MOVES", "P.RATING", "P.RATING_DEVIATION",
		"P.POPULARITY", "P.PLAY_COUNT", "P.GAME_URL", "P.OPENING_TAGS", "P.DAILY_DATE",
	}
	builder := squirrel.Select(columns...).
		From(puzzleTable + " P").
		Where(squirrel.And{
			squirrel.GtOrEq{"P.RATING": targetRating - tolerance},
			squirrel.LtOrEq{"P.RATING": targetRating + tolerance},
		})
	if len(themeIDs) > 0 {
		builder = builder.
			Join(puzzleThemeTable + " PT ON PT.PUZZLE_ID = P.PUZZLE_ID").
			Where(squirrel.Eq{"PT.THEME_ID": themeIDs}).
			GroupBy("P.PUZZLE_ID")
	}
	query, args, err := builder.OrderBy("RANDOM()").Limit(1).ToSql()
	if err != nil {
		return entity.Puzzle{}, fmt.Errorf("build random puzzle query: %w", err)
	}
	puzzle, err := scanPuzzle(r.db.QueryRow(query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return entity.Puzzle{}, ErrPuzzleNotFound
	}
	if err != nil {
		return entity.Puzzle{}, fmt.Errorf("get random puzzle: %w", err)
	}
	themes, err := r.themesForPuzzle(puzzle.ID)
	if err != nil {
		return entity.Puzzle{}, fmt.Errorf("get puzzle themes: %w", err)
	}
	puzzle.Themes = themes
	return puzzle, nil
}

func (r *PuzzleRepository) Themes() ([]entity.Theme, error) {
	rows, err := r.db.Query("SELECT THEME_ID, DISPLAY_NAME, PUZZLE_COUNT FROM " + themeTable + " ORDER BY DISPLAY_NAME")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanThemes(rows)
}

func (r *PuzzleRepository) themesForPuzzle(puzzleID string) ([]entity.Theme, error) {
	rows, err := r.db.Query(
		"SELECT T.THEME_ID, T.DISPLAY_NAME, T.PUZZLE_COUNT FROM "+themeTable+" T "+
			"JOIN "+puzzleThemeTable+" PT ON PT.THEME_ID = T.THEME_ID "+
			"WHERE PT.PUZZLE_ID = ? ORDER BY T.DISPLAY_NAME",
		puzzleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanThemes(rows)
}

func scanPuzzle(row *sql.Row) (entity.Puzzle, error) {
	var puzzle entity.Puzzle
	var moves string
	var openingTags string
	var dailyDate sql.NullInt64
	if err := row.Scan(
		&puzzle.ID, &puzzle.FEN, &moves, &puzzle.Rating, &puzzle.RatingDeviation,
		&puzzle.Popularity, &puzzle.PlayCount, &puzzle.GameURL, &openingTags, &dailyDate,
	); err != nil {
		return entity.Puzzle{}, err
	}
	board, err := chess.ParseFEN(puzzle.FEN)
	if err != nil {
		return entity.Puzzle{}, fmt.Errorf("parse puzzle FEN: %w", err)
	}
	puzzle.Moves, err = chess.ParseUCIMoves(board, moves)
	if err != nil {
		return entity.Puzzle{}, fmt.Errorf("parse puzzle moves: %w", err)
	}
	puzzle.OpeningTags = strings.Fields(openingTags)
	if dailyDate.Valid {
		date := dailyDate.Int64
		puzzle.DailyDate = &date
	}
	return puzzle, nil
}

func scanThemes(rows *sql.Rows) ([]entity.Theme, error) {
	var themes []entity.Theme
	for rows.Next() {
		var theme entity.Theme
		if err := rows.Scan(&theme.ID, &theme.DisplayName, &theme.PuzzleCount); err != nil {
			return nil, err
		}
		themes = append(themes, theme)
	}
	return themes, rows.Err()
}
