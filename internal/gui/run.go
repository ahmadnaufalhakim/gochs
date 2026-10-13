package gui

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	"github.com/gdamore/tcell/v2"
)

var (
	backgroundStyle = tcell.StyleDefault.Background(tcell.NewRGBColor(10, 18, 33)).Foreground(tcell.NewRGBColor(239, 230, 211))
	titleStyle      = backgroundStyle.Foreground(tcell.NewRGBColor(224, 181, 85)).Bold(true)
	selectedStyle   = backgroundStyle.Background(tcell.NewRGBColor(224, 181, 85)).Foreground(tcell.NewRGBColor(10, 18, 33)).Bold(true)
	mutedStyle      = backgroundStyle.Foreground(tcell.NewRGBColor(157, 172, 191))
)

type Result uint8

const (
	Exit Result = iota
	LocalGame
	StockfishGame
	Puzzle
)

type PuzzleLoader func() (entity.Puzzle, error)

// Run starts the terminal user interface.
func Run(loadPuzzle PuzzleLoader, stockfishPath string) (Result, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return Exit, fmt.Errorf("create terminal screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return Exit, fmt.Errorf("initialize terminal screen: %w", err)
	}
	defer screen.Fini()

	screen.SetStyle(backgroundStyle)
	screen.EnableMouse(tcell.MouseButtonEvents, tcell.MouseMotionEvents)

	return run(screen, loadPuzzle, stockfishPath), nil
}

func run(screen tcell.Screen, loadPuzzle PuzzleLoader, stockfishPath string) Result {
	state := newMenuState()
	state.stockfishPath = stockfishPath
	var game *localGameState
	var puzzle *puzzleState
	events := make(chan tcell.Event)
	quitEvents := make(chan struct{})
	defer close(quitEvents)
	go screen.ChannelEvents(events, quitEvents)
	clockTicker := time.NewTicker(100 * time.Millisecond)
	defer clockTicker.Stop()

	for {
		screen.Clear()
		if game == nil && puzzle == nil {
			draw(screen, state)
		} else if game != nil {
			drawLocalGame(screen, *game)
		} else {
			drawPuzzle(screen, *puzzle)
		}
		screen.Show()

		var engineMoves <-chan engineResult
		if game != nil {
			engineMoves = game.engineResults
		}
		select {
		case result := <-engineMoves:
			if game != nil {
				game.applyEngineResult(result)
			}
		case now := <-clockTicker.C:
			if game != nil {
				game.advanceClock(now)
			}
			if puzzle != nil {
				puzzle.advance(now)
			}
			continue
		case rawEvent, ok := <-events:
			if !ok {
				return Exit
			}
			switch event := rawEvent.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				if game != nil {
					width, height := screen.Size()
					if game.handleKey(event, width, height) {
						game.closeStockfish()
						game = nil
						state.showMainMenu()
					}
					continue
				}
				if puzzle != nil {
					width, height := screen.Size()
					switch puzzle.handleKey(event, width, height) {
					case puzzleReturnToMenu:
						puzzle = nil
						state.showMainMenu()
					case puzzleContinueTraining:
						puzzle = startPuzzle(loadPuzzle, state.theme, &state)
					}
					continue
				}
				width, height := screen.Size()
				if !menuFits(width, height) {
					if event.Key() == tcell.KeyEsc {
						return Exit
					}
					continue
				}
				if state.handleKey(event) {
					if state.result == LocalGame || state.result == StockfishGame {
						newGame := newLocalGameState(state.theme)
						newGame.autoFlip = state.autoFlip
						newGame.startClock(state.timeControl)
						if state.result == StockfishGame {
							if err := newGame.startStockfish(state.stockfishPath, state.stockfishColor, state.stockfishDifficulty); err != nil {
								state.message = "Could not start Stockfish: " + err.Error()
								state.showMainMenu()
								state.result = Exit
								continue
							}
						}
						game = &newGame
						state.result = Exit
					} else if state.result == Puzzle {
						puzzle = startPuzzle(loadPuzzle, state.theme, &state)
						state.result = Exit
					} else {
						return state.result
					}
				}
			case *tcell.EventMouse:
				width, height := screen.Size()
				if game != nil {
					game.handleMouse(event, width, height)
					continue
				}
				if puzzle != nil {
					switch puzzle.handleMouse(event, width, height) {
					case puzzleReturnToMenu:
						puzzle = nil
						state.showMainMenu()
					case puzzleContinueTraining:
						puzzle = startPuzzle(loadPuzzle, state.theme, &state)
					}
					continue
				}
				if !menuFits(width, height) {
					continue
				}
				if state.handleMouse(event, width, height) {
					if state.result == LocalGame || state.result == StockfishGame {
						newGame := newLocalGameState(state.theme)
						newGame.autoFlip = state.autoFlip
						newGame.startClock(state.timeControl)
						if state.result == StockfishGame {
							if err := newGame.startStockfish(state.stockfishPath, state.stockfishColor, state.stockfishDifficulty); err != nil {
								state.message = "Could not start Stockfish: " + err.Error()
								state.showMainMenu()
								state.result = Exit
								continue
							}
						}
						game = &newGame
						state.result = Exit
					} else if state.result == Puzzle {
						puzzle = startPuzzle(loadPuzzle, state.theme, &state)
						state.result = Exit
					} else {
						return state.result
					}
				}
			}
		}
	}
}

func startPuzzle(loadPuzzle PuzzleLoader, theme chess.ColorTheme, state *menuState) *puzzleState {
	if loadPuzzle == nil {
		state.message = "Puzzle database is unavailable"
		return nil
	}
	puzzle, err := loadPuzzle()
	if err != nil {
		state.message = "Could not load a puzzle: " + err.Error()
		return nil
	}
	newPuzzle, err := newPuzzleState(puzzle, theme, time.Now())
	if err != nil {
		state.message = "Could not prepare puzzle: " + err.Error()
		return nil
	}
	state.message = ""
	return &newPuzzle
}

func menuFits(width, height int) bool {
	return width >= 48 && height >= 23
}

func newMenuState() menuState {
	return menuState{
		selected:    play,
		splashIndex: rand.Intn(len(pieceSplashes)),
		theme:       chess.WOOD,
	}
}
