package gui

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
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
type PuzzleAttemptRecorder func(entity.Puzzle, entity.PuzzleOutcome) (entity.TrainingResult, error)
type LichessRatingSynchronizer func(entity.LichessAccount) (entity.TrainingProfile, error)

type LichessAccountService interface {
	Load(context.Context) (entity.LichessAccount, error)
	Connect(context.Context, string) (entity.LichessAccount, error)
	Disconnect() error
}

// Run starts the terminal user interface.
func Run(loadPuzzle PuzzleLoader, recordPuzzleAttempt PuzzleAttemptRecorder, syncLichessRating LichessRatingSynchronizer, stockfishPath string, lichessService LichessAccountService) (Result, error) {
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

	return run(screen, loadPuzzle, recordPuzzleAttempt, syncLichessRating, stockfishPath, lichessService), nil
}

func run(screen tcell.Screen, loadPuzzle PuzzleLoader, recordPuzzleAttempt PuzzleAttemptRecorder, syncLichessRating LichessRatingSynchronizer, stockfishPath string, lichessService LichessAccountService) Result {
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
	lichessResults := make(chan lichessResult, 1)

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
		case result := <-lichessResults:
			if state.page == lichessLoading {
				state.applyLichessResult(result)
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
						puzzle = startPuzzle(loadPuzzle, recordPuzzleAttempt, state.theme, &state)
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
						puzzle = startPuzzle(loadPuzzle, recordPuzzleAttempt, state.theme, &state)
						state.result = Exit
					} else {
						return state.result
					}
				}
				startLichessAction(&state, lichessService, syncLichessRating, lichessResults)
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
						puzzle = startPuzzle(loadPuzzle, recordPuzzleAttempt, state.theme, &state)
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
						puzzle = startPuzzle(loadPuzzle, recordPuzzleAttempt, state.theme, &state)
						state.result = Exit
					} else {
						return state.result
					}
				}
				startLichessAction(&state, lichessService, syncLichessRating, lichessResults)
			}
		}
	}
}

type lichessResult struct {
	action  lichessAction
	account entity.LichessAccount
	profile entity.TrainingProfile
	err     error
}

func startLichessAction(state *menuState, service LichessAccountService, syncRating LichessRatingSynchronizer, results chan<- lichessResult) {
	action := state.lichessAction
	if action == lichessNoAction {
		return
	}
	state.lichessAction = lichessNoAction
	if service == nil || (action == lichessSyncRating && syncRating == nil) {
		state.page = lichessMenu
		state.lichessMessage = "Lichess is unavailable"
		return
	}
	token := state.lichessToken
	account := state.lichessAccount
	go func() {
		result := lichessResult{action: action}
		switch action {
		case lichessLoad:
			result.account, result.err = service.Load(context.Background())
		case lichessConnect:
			result.account, result.err = service.Connect(context.Background(), token)
		case lichessDisconnect:
			result.err = service.Disconnect()
		case lichessSyncRating:
			if account == nil {
				result.err = fmt.Errorf("Lichess account is unavailable")
			} else {
				result.profile, result.err = syncRating(*account)
			}
		}
		results <- result
	}()
}

func (s *menuState) applyLichessResult(result lichessResult) {
	if result.err != nil {
		if errors.Is(result.err, lichessapi.ErrNoToken) {
			s.lichessAccount = nil
			s.lichessMessage = ""
			s.page = lichessMenu
			return
		}
		s.lichessMessage = result.err.Error()
		if result.action == lichessConnect {
			s.page = lichessToken
		} else {
			s.page = lichessMenu
		}
		return
	}
	if result.action == lichessDisconnect {
		s.lichessAccount = nil
		s.lichessToken = ""
		s.lichessSelected = 0
		s.lichessMessage = ""
		s.page = lichessMenu
		return
	}
	if result.action == lichessSyncRating {
		s.lichessMessage = fmt.Sprintf("Offline puzzle rating synced to %d", result.profile.Rating)
		s.page = lichessMenu
		return
	}
	s.lichessAccount = &result.account
	s.lichessToken = ""
	s.lichessSelected = 0
	s.lichessMessage = ""
	if result.action == lichessConnect && result.account.Puzzle != 0 {
		s.page = lichessSyncConfirmation
		return
	}
	s.page = lichessMenu
}

func startPuzzle(loadPuzzle PuzzleLoader, recordPuzzleAttempt PuzzleAttemptRecorder, theme chess.ColorTheme, state *menuState) *puzzleState {
	if loadPuzzle == nil {
		state.message = "Puzzle database is unavailable"
		return nil
	}
	puzzle, err := loadPuzzle()
	if err != nil {
		state.message = "Could not load a puzzle: " + err.Error()
		return nil
	}
	newPuzzle, err := newPuzzleStateWithRecorder(puzzle, recordPuzzleAttempt, theme, time.Now())
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
