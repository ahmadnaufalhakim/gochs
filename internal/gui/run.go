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
	"github.com/ahmadnaufalhakim/gochs/internal/sound"
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
	OnlineGame
)

type PuzzleLoader func() (entity.Puzzle, error)
type PuzzleAttemptRecorder func(entity.Puzzle, entity.PuzzleOutcome) (entity.TrainingResult, error)
type LichessRatingSynchronizer func(entity.LichessAccount) (entity.TrainingProfile, error)
type OfflinePuzzleRatingLoader func() (entity.TrainingProfile, bool, error)

type LichessAccountService interface {
	Load(context.Context) (entity.LichessAccount, error)
	Connect(context.Context, string) (entity.LichessAccount, error)
	Disconnect() error
}

// OnlineBoardService contains only the authenticated board endpoints used by the UI.
type OnlineBoardService interface {
	CreateSeek(context.Context, lichessapi.Seek) error
	StreamEvents(context.Context) (<-chan lichessapi.Event, <-chan error)
	StreamGame(context.Context, string) (<-chan lichessapi.GameEvent, <-chan error)
	MakeMove(context.Context, string, string) error
	Resign(context.Context, string) error
	SetDraw(context.Context, string, bool) error
	SetTakeback(context.Context, string, bool) error
	RatingChanges(context.Context, string) (lichessapi.RatingChanges, error)
	ExportPGN(context.Context, string, bool) ([]byte, error)
}

// Run starts the terminal user interface.
func Run(loadPuzzle PuzzleLoader, recordPuzzleAttempt PuzzleAttemptRecorder, syncLichessRating LichessRatingSynchronizer, loadOfflinePuzzleRating OfflinePuzzleRatingLoader, stockfishPath string, lichessService LichessAccountService, onlineService OnlineBoardService) (Result, error) {
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

	return run(screen, loadPuzzle, recordPuzzleAttempt, syncLichessRating, loadOfflinePuzzleRating, stockfishPath, lichessService, onlineService), nil
}

func run(screen tcell.Screen, loadPuzzle PuzzleLoader, recordPuzzleAttempt PuzzleAttemptRecorder, syncLichessRating LichessRatingSynchronizer, loadOfflinePuzzleRating OfflinePuzzleRatingLoader, stockfishPath string, lichessService LichessAccountService, onlineService OnlineBoardService) Result {
	sounds := sound.NewPlayer()
	state := newMenuState()
	state.stockfishPath = stockfishPath
	var game *localGameState
	var puzzle *puzzleState
	var online *onlineGameState
	onlineVisible := false
	events := make(chan tcell.Event)
	quitEvents := make(chan struct{})
	defer close(quitEvents)
	go screen.ChannelEvents(events, quitEvents)
	clockTicker := time.NewTicker(100 * time.Millisecond)
	defer clockTicker.Stop()
	lichessResults := make(chan lichessResult, 1)
	onlineResults := make(chan onlineResult, 8)
	var onlineCancel context.CancelFunc
	var onlineContext context.Context
	defer func() {
		if onlineCancel != nil {
			onlineCancel()
		}
	}()

	for {
		screen.Clear()
		if game == nil && puzzle == nil && !onlineVisible {
			draw(screen, state)
		} else if game != nil {
			drawLocalGame(screen, *game)
		} else if online != nil {
			drawOnlineGame(screen, *online)
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
				loadOfflineRating(&state, loadOfflinePuzzleRating)
			}
		case result := <-onlineResults:
			if online == nil {
				if full, ok := result.event.(lichessapi.GameFullEvent); ok {
					newOnline, err := newOnlineGame(full, result.accountID, result.accountName, state.theme)
					if err != nil {
						state.page = lichessMenu
						state.lichessMessage = err.Error()
					} else {
						online = newOnline
						online.sounds = sounds
						online.startSounds()
						if online.isFinished() {
							online.playOnlineTerminal()
						}
						online.seek = result.seek
						online.requestContext = onlineContext
						onlineVisible = true
						state.onlineActive = true
						state.onlineGame = false
					}
				} else {
					onlineContext = applyOnlineSeekResult(&state, result, onlineService, onlineResults, &onlineCancel)
				}
			} else {
				online.apply(result)
				online.loadRatingChanges(onlineService, onlineResults)
				if (online.isFinished() || result.fatal) && onlineCancel != nil {
					onlineCancel()
				}
			}
		case now := <-clockTicker.C:
			if game != nil {
				game.advanceClock(now)
			}
			if puzzle != nil {
				puzzle.advance(now)
			}
			if online != nil {
				online.advanceClock(now)
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
				if online != nil && onlineVisible {
					if online.handleKey(event, onlineService, onlineResults) {
						if online.state.Status == "started" {
							onlineVisible = false
						} else {
							online = nil
							onlineVisible = false
							state.onlineActive = false
						}
						state.showMainMenu()
					}
					continue
				}
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
						newGame.sounds = sounds
						newGame.startSounds()
						game = &newGame
						state.result = Exit
					} else if state.result == OnlineGame {
						if online != nil {
							onlineVisible = true
						} else if onlineService == nil || state.lichessAccount == nil {
							state.message = "Lichess online play is unavailable"
							state.showMainMenu()
						} else {
							startOnlineSeek(&state, onlineService, onlineResults, &onlineCancel)
						}
						state.result = Exit
					} else if state.result == Puzzle {
						puzzle = startPuzzle(loadPuzzle, recordPuzzleAttempt, state.theme, &state)
						state.result = Exit
					} else {
						return state.result
					}
				}
				if state.onlineSeeking && state.page != lichessLoading {
					if onlineCancel != nil {
						onlineCancel()
					}
					state.onlineSeeking = false
					state.onlineSeekGeneration++
				}
				startLichessAction(&state, lichessService, syncLichessRating, lichessResults)
				loadOfflineRating(&state, loadOfflinePuzzleRating)
			case *tcell.EventMouse:
				width, height := screen.Size()
				if game != nil {
					game.handleMouse(event, width, height)
					continue
				}
				if online != nil && onlineVisible {
					online.handleMouse(event, width, height, onlineService, onlineResults)
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
						newGame.sounds = sounds
						newGame.startSounds()
						game = &newGame
						state.result = Exit
					} else if state.result == OnlineGame {
						if online != nil {
							onlineVisible = true
						} else if onlineService == nil || state.lichessAccount == nil {
							state.message = "Lichess online play is unavailable"
							state.showMainMenu()
						} else {
							startOnlineSeek(&state, onlineService, onlineResults, &onlineCancel)
						}
						state.result = Exit
					} else if state.result == Puzzle {
						puzzle = startPuzzle(loadPuzzle, recordPuzzleAttempt, state.theme, &state)
						state.result = Exit
					} else {
						return state.result
					}
				}
				startLichessAction(&state, lichessService, syncLichessRating, lichessResults)
				loadOfflineRating(&state, loadOfflinePuzzleRating)
			}
		}
	}
}

func loadOfflineRating(state *menuState, loadRating OfflinePuzzleRatingLoader) {
	if state.page != lichessSyncConfirmation || state.offlineRatingLoaded || loadRating == nil {
		return
	}
	profile, ok, err := loadRating()
	if err != nil {
		state.offlineRatingMessage = "Could not load offline rating"
		state.offlineRatingLoaded = true
		return
	}
	if ok {
		state.offlinePuzzleRating = profile.Rating
	}
	state.offlineRatingLoaded = true
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
		s.offlineRatingLoaded = false
		s.offlineRatingMessage = ""
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
