package gui

import (
	"fmt"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

var (
	backgroundStyle = tcell.StyleDefault.Background(tcell.NewRGBColor(10, 18, 33)).Foreground(tcell.NewRGBColor(239, 230, 211))
	titleStyle      = backgroundStyle.Foreground(tcell.NewRGBColor(224, 181, 85)).Bold(true)
	selectedStyle   = backgroundStyle.Background(tcell.NewRGBColor(224, 181, 85)).Foreground(tcell.NewRGBColor(10, 18, 33)).Bold(true)
	mutedStyle      = backgroundStyle.Foreground(tcell.NewRGBColor(157, 172, 191))
)

// Run starts the terminal user interface.
func Run() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("create terminal screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("initialize terminal screen: %w", err)
	}
	defer screen.Fini()

	screen.SetStyle(backgroundStyle)
	screen.EnableMouse(tcell.MouseButtonEvents)

	run(screen)
	return nil
}

func run(screen tcell.Screen) {
	state := newMenuState()

	for {
		screen.Clear()
		draw(screen, state)
		screen.Show()

		switch event := screen.PollEvent().(type) {
		case *tcell.EventResize:
			screen.Sync()
		case *tcell.EventKey:
			if state.handleKey(event) {
				return
			}
		case *tcell.EventMouse:
			width, height := screen.Size()
			if state.handleMouse(event, width, height) {
				return
			}
		}
	}
}

func newMenuState() menuState {
	return menuState{
		selected:    playWithStockfish,
		splashIndex: rand.Intn(len(pieceSplashes)),
	}
}
