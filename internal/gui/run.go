package gui

import (
	"fmt"
	"math/rand"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
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
)

// Run starts the terminal user interface.
func Run() (Result, error) {
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

	return run(screen), nil
}

func run(screen tcell.Screen) Result {
	state := newMenuState()
	var game *localGameState

	for {
		screen.Clear()
		if game == nil {
			draw(screen, state)
		} else {
			drawLocalGame(screen, *game)
		}
		screen.Show()

		switch event := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventKey:
			if game != nil {
				width, height := screen.Size()
				if game.handleKey(event, width, height) {
					game = nil
					state.showMainMenu()
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
				if state.result == LocalGame {
					newGame := newLocalGameState(state.theme)
					newGame.autoFlip = state.autoFlip
					game = &newGame
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
			if !menuFits(width, height) {
				continue
			}
			if state.handleMouse(event, width, height) {
				if state.result == LocalGame {
					newGame := newLocalGameState(state.theme)
					newGame.autoFlip = state.autoFlip
					game = &newGame
					state.result = Exit
				} else {
					return state.result
				}
			}
		}
	}
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
