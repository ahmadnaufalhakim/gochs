package gui

import "github.com/gdamore/tcell/v2"

type page uint8

const (
	mainMenu page = iota
	comingSoon
	exitConfirmation
)

type menuItem uint8

const (
	playWithStockfish menuItem = iota
	puzzle
	options
	credits
	exit
	menuItemCount
)

var menuLabels = [menuItemCount]string{
	"Play with Stockfish",
	"Puzzle",
	"Options",
	"Credits",
	"Exit",
}

type menuState struct {
	page            page
	selected        menuItem
	unavailableItem menuItem
	confirmExit     bool
}

type menuLayout struct {
	x     int
	y     int
	width int
}

func (s *menuState) handleKey(event *tcell.EventKey) bool {
	switch s.page {
	case mainMenu:
		switch event.Key() {
		case tcell.KeyUp:
			s.moveSelection(-1)
		case tcell.KeyDown:
			s.moveSelection(1)
		case tcell.KeyEnter:
			s.activate(s.selected)
		case tcell.KeyEsc:
			return true
		case tcell.KeyRune:
			switch event.Rune() {
			case 'k':
				s.moveSelection(-1)
			case 'j':
				s.moveSelection(1)
			case 'q':
				return true
			}
		}
	case comingSoon:
		switch event.Key() {
		case tcell.KeyEnter, tcell.KeyEsc:
			s.page = mainMenu
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				s.page = mainMenu
			}
		}
	case exitConfirmation:
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyRight:
			s.confirmExit = !s.confirmExit
		case tcell.KeyEsc:
			s.page = mainMenu
		case tcell.KeyEnter:
			if s.confirmExit {
				return true
			}
			s.page = mainMenu
		case tcell.KeyRune:
			switch event.Rune() {
			case 'y', 'Y':
				return true
			case 'n', 'N', 'q':
				s.page = mainMenu
			}
		}
	}

	return false
}

func (s *menuState) handleMouse(event *tcell.EventMouse, width, height int) bool {
	if event.Buttons() != tcell.Button1 {
		return false
	}

	x, y := event.Position()
	switch s.page {
	case mainMenu:
		item, ok := menuItemAt(x, y, width, height)
		if ok {
			s.selected = item
			s.activate(item)
		}
	case comingSoon:
		s.page = mainMenu
	case exitConfirmation:
		if confirmExitAt(x, y, width, height) {
			return true
		}
		s.page = mainMenu
	}

	return false
}

func (s *menuState) moveSelection(delta int) {
	s.selected = menuItem((int(s.selected) + delta + int(menuItemCount)) % int(menuItemCount))
}

func (s *menuState) activate(item menuItem) {
	if item == exit {
		s.page = exitConfirmation
		s.confirmExit = false
		return
	}

	s.page = comingSoon
	s.unavailableItem = item
}

func draw(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	if width < 28 || height < 14 {
		drawCentered(screen, height/2-1, "Terminal too small", titleStyle)
		drawCentered(screen, height/2+1, "Resize or Esc to exit", mutedStyle)
		return
	}

	switch state.page {
	case mainMenu:
		drawMainMenu(screen, state.selected)
	case comingSoon:
		drawComingSoon(screen, state.unavailableItem)
	case exitConfirmation:
		drawExitConfirmation(screen, state.confirmExit)
	}
}

func drawMainMenu(screen tcell.Screen, selected menuItem) {
	width, height := screen.Size()
	layout := mainMenuLayout(width, height)

	drawCentered(screen, layout.y-5, "gochs", titleStyle)
	drawCentered(screen, layout.y-3, "terminal chess", mutedStyle)
	for item, label := range menuLabels {
		style := backgroundStyle
		if menuItem(item) == selected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+item*2, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(menuItemCount)*2+1, "Up/Down or j/k to select  Enter to choose", mutedStyle)
}

func drawComingSoon(screen tcell.Screen, item menuItem) {
	_, height := screen.Size()
	drawCentered(screen, height/2-2, menuLabels[item], titleStyle)
	drawCentered(screen, height/2, "Coming soon", backgroundStyle)
	drawCentered(screen, height/2+2, "Press Enter, Esc, or click to return", mutedStyle)
}

func drawExitConfirmation(screen tcell.Screen, confirmExit bool) {
	_, height := screen.Size()
	drawCentered(screen, height/2-3, "Are you sure?", titleStyle)
	drawCentered(screen, height/2-1, "No", styleForExitChoice(false, confirmExit))
	drawCentered(screen, height/2+1, "Yes, exit gochs", styleForExitChoice(true, confirmExit))
	drawCentered(screen, height/2+3, "Left/Right to choose  Enter to confirm", mutedStyle)

}

func styleForExitChoice(choice, confirmExit bool) tcell.Style {
	if choice == confirmExit {
		return selectedStyle
	}

	return backgroundStyle
}

func mainMenuLayout(width, height int) menuLayout {
	menuWidth := 0
	for _, label := range menuLabels {
		if len(label) > menuWidth {
			menuWidth = len(label)
		}
	}

	return menuLayout{
		x:     (width - menuWidth - 4) / 2,
		y:     height/2 - int(menuItemCount),
		width: menuWidth + 4,
	}
}

func menuItemAt(x, y, width, height int) (menuItem, bool) {
	layout := mainMenuLayout(width, height)
	if x < layout.x || x >= layout.x+layout.width {
		return 0, false
	}

	for item := range menuItemCount {
		if y == layout.y+int(item)*2 {
			return item, true
		}
	}

	return 0, false
}

func confirmExitAt(x, y, width, height int) bool {
	if x < (width-14)/2 || x >= (width+14)/2 {
		return false
	}

	return y == height/2+1
}

func drawCentered(screen tcell.Screen, y int, text string, style tcell.Style) {
	width, height := screen.Size()
	if y < 0 || y >= height {
		return
	}

	drawString(screen, (width-len(text))/2, y, text, style)
}

func drawPaddedString(screen tcell.Screen, x, y, width int, text string, style tcell.Style) {
	drawString(screen, x, y, "  "+text, style)
	for column := len(text) + 2; column < width; column++ {
		screen.SetContent(x+column, y, ' ', nil, style)
	}
}

func drawString(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	width, height := screen.Size()
	if y < 0 || y >= height {
		return
	}

	for offset, character := range text {
		if x+offset >= 0 && x+offset < width {
			screen.SetContent(x+offset, y, character, nil, style)
		}
	}
}
