package gui

import (
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

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
	splashIndex     int
}

type menuLayout struct {
	x     int
	y     int
	width int
}

var gochsWordmark = []string{
	"_         ",
	"  __ _  ___   ___| |__  ___ ",
	" / _` |/ _ \\ / __| '_ \\/ __|",
	"| (_| | (_) | (__| | | \\__ \\",
	" \\__, |\\___/ \\___|_| |_|___/",
	" |___/ ",
}

var pieceSplashes = [][]string{
	{
		"    .::.    ",
		"    _::_   ",
		"  _/____\\_ ",
		"  \\      / ",
		"   \\____/  ",
		"   (____)   ",
		"    |  |    ",
		"    |__|    ",
		"   /    \\   ",
		"  (______)  ",
		" (________) ",
		"(__________) ",
	},
	{
		"     ()     ",
		"   <~~~~>   ",
		"    \\__/    ",
		"   (____)    ",
		"    |  |     ",
		"    |  |     ",
		"    |__|     ",
		"   /____\\    ",
		"  (______)   ",
		" (________)  ",
		"(__________)  ",
	},
	{
		"     <>_    ",
		"   (\\)  )  ",
		"    \\__/   ",
		"   (____)   ",
		"    |  |    ",
		"    |__|    ",
		"   /____\\   ",
		"  (______)  ",
		" (________) ",
		"(__________)  ",
	},
	{
		"   WWWWWW    ",
		"   WWWWWW    ",
		"    |  |     ",
		"    |  |     ",
		"    |__|     ",
		"   /____\\    ",
		"  (______)   ",
	},
	{
		"      __    ",
		"     (  )   ",
		"      ||     ",
		"     /__\\    ",
		"    (____)   ",
		"   (______)  ",
	},
	{
		"    __/\"\"\"\\   ",
		"   ]___ 0  } ",
		"       /   }  ",
		"      /~    } ",
		"      \\____/  ",
		"      /____\\  ",
		"     (______) ",
	},
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
			s.activate(exit)
		case tcell.KeyRune:
			switch event.Rune() {
			case 'k':
				s.moveSelection(-1)
			case 'j':
				s.moveSelection(1)
			case 'q':
				s.activate(exit)
			}
		}
	case comingSoon:
		switch event.Key() {
		case tcell.KeyEnter, tcell.KeyEsc:
			s.showMainMenu()
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				s.showMainMenu()
			}
		}
	case exitConfirmation:
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyRight:
			s.confirmExit = !s.confirmExit
		case tcell.KeyEsc:
			s.showMainMenu()
		case tcell.KeyEnter:
			if s.confirmExit {
				return true
			}
			s.showMainMenu()
		case tcell.KeyRune:
			switch event.Rune() {
			case 'y', 'Y':
				return true
			case 'n', 'N', 'q':
				s.showMainMenu()
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
		s.showMainMenu()
	case exitConfirmation:
		if confirmExitAt(x, y, width, height) {
			return true
		}
		s.showMainMenu()
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

func (s *menuState) showMainMenu() {
	s.page = mainMenu
	s.splashIndex = rand.Intn(len(pieceSplashes))
}

func draw(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	if width < 48 || height < 23 {
		drawCentered(screen, height/2-1, "Terminal too small", titleStyle)
		drawCentered(screen, height/2+1, "Resize or Esc to exit", mutedStyle)
		return
	}

	switch state.page {
	case mainMenu:
		drawMainMenu(screen, state)
	case comingSoon:
		drawComingSoon(screen, state.unavailableItem)
	case exitConfirmation:
		drawExitConfirmation(screen, state.confirmExit)
	}
}

func drawMainMenu(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	layout := mainMenuLayout(width, height)

	drawSplash(screen, layout.y-tallestSplash()-1, state.splashIndex)
	for item, label := range menuLabels {
		style := backgroundStyle
		if menuItem(item) == state.selected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+item, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(menuItemCount)+1, "Up/Down or j/k to select  Enter to choose", mutedStyle)
}

func drawSplash(screen tcell.Screen, y, splashIndex int) {
	width, _ := screen.Size()
	splash := pieceSplashes[splashIndex%len(pieceSplashes)]
	splashWidth := longestLineWidth(gochsWordmark) + 4 + longestLineWidth(splash)
	x := (width - splashWidth) / 2
	for row, line := range splash {
		if row < len(gochsWordmark) {
			drawString(screen, x, y+row, gochsWordmark[row], titleStyle)
		}
		drawString(screen, x+longestLineWidth(gochsWordmark)+4, y+row, line, backgroundStyle)
	}
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
		y:     (height-tallestSplash()-int(menuItemCount)-3)/2 + tallestSplash() + 2,
		width: menuWidth + 4,
	}
}

func menuItemAt(x, y, width, height int) (menuItem, bool) {
	layout := mainMenuLayout(width, height)
	if x < layout.x || x >= layout.x+layout.width {
		return 0, false
	}

	for item := range menuItemCount {
		if y == layout.y+int(item) {
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

func longestLineWidth(lines []string) int {
	width := 0
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}

	return width
}

func tallestSplash() int {
	height := 0
	for _, splash := range pieceSplashes {
		if len(splash) > height {
			height = len(splash)
		}
	}

	return height
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
