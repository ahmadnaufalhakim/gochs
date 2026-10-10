package gui

import (
	"math/rand"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type page uint8

const (
	mainMenu page = iota
	playMenu
	optionsMenu
	themeMenu
	comingSoon
	exitConfirmation
)

type menuItem uint8

const (
	play menuItem = iota
	puzzle
	options
	credits
	exit
	menuItemCount
)

var menuLabels = [menuItemCount]string{
	"Play",
	"Puzzle",
	"Options",
	"Credits",
	"Exit",
}

type playMenuItem uint8

const (
	localGame playMenuItem = iota
	playWithStockfish
	back
	playMenuItemCount
)

var playMenuLabels = [playMenuItemCount]string{
	"Local game",
	"Play with Stockfish",
	"Back",
}

type optionsMenuItem uint8

const (
	theme optionsMenuItem = iota
	optionsBack
	optionsMenuItemCount
)

var optionsMenuLabels = [optionsMenuItemCount]string{
	"Theme",
	"Back",
}

type menuState struct {
	page            page
	selected        menuItem
	playSelected    playMenuItem
	optionsSelected optionsMenuItem
	comingSoonLabel string
	confirmExit     bool
	splashIndex     int
	result          Result
	theme           chess.ColorTheme
}

type menuLayout struct {
	x     int
	y     int
	width int
}

var gochsWordmark = []string{
	"                  _         ",
	"  __ _  ___   ___| |__  ___ ",
	" / _` |/ _ \\ / __| '_ \\/ __|",
	"| (_| | (_) | (__| | | \\__ \\",
	" \\__, |\\___/ \\___|_| |_|___/",
	" |___/ ",
}

var pieceSplashes = [][]string{
	{
		"    .::.",
		"    _::_",
		"  _/____\\_",
		"  \\      /",
		"   \\____/",
		"   (____)",
		"    |  |",
		"    |__|",
		"   /    \\",
		"  (______)",
		" (________)",
		"(__________)",
	},
	{
		"     ()",
		"   <~~~~>",
		"    \\__/",
		"   (____)",
		"    |  |",
		"    |  |",
		"    |__|",
		"   /____\\",
		"  (______)",
		" (________)",
		"(__________)",
	},
	{
		"     <>_",
		"   (\\)  )",
		"    \\__/",
		"   (____)",
		"    |  |",
		"    |__|",
		"   /____\\",
		"  (______)",
		" (________)",
		"(__________)",
	},
	{
		"   WWWWWW",
		"   WWWWWW",
		"    |  |",
		"    |  |",
		"    |__|",
		"   /____\\",
		"  (______)",
	},
	{
		"      __",
		"     (  )",
		"      ||",
		"     /__\\",
		"    (____)",
		"   (______)",
	},
	{
		"    __/\"\"\"\\",
		"   ]___ 0  }",
		"       /   }",
		"      /~   }",
		"      \\____/",
		"      /____\\",
		"     (______)",
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
		}
	case playMenu:
		switch event.Key() {
		case tcell.KeyUp:
			s.movePlaySelection(-1)
		case tcell.KeyDown:
			s.movePlaySelection(1)
		case tcell.KeyEnter:
			s.activatePlayItem(s.playSelected)
			if s.result == LocalGame {
				return true
			}
		case tcell.KeyEsc:
			s.showMainMenu()
		}
	case optionsMenu:
		switch event.Key() {
		case tcell.KeyUp:
			s.moveOptionsSelection(-1)
		case tcell.KeyDown:
			s.moveOptionsSelection(1)
		case tcell.KeyEnter:
			if s.optionsSelected == theme {
				s.page = themeMenu
			} else {
				s.showMainMenu()
			}
		case tcell.KeyEsc:
			s.showMainMenu()
		}
	case themeMenu:
		switch event.Key() {
		case tcell.KeyLeft:
			s.moveTheme(-1)
		case tcell.KeyRight:
			s.moveTheme(1)
		case tcell.KeyEnter, tcell.KeyEsc:
			s.page = optionsMenu
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
		case tcell.KeyUp, tcell.KeyDown:
			s.confirmExit = !s.confirmExit
		case tcell.KeyEsc:
			s.showMainMenu()
		case tcell.KeyEnter:
			if s.confirmExit {
				s.result = Exit
				return true
			}
			s.showMainMenu()
		case tcell.KeyRune:
			switch event.Rune() {
			case 'y', 'Y':
				s.result = Exit
				return true
			case 'n', 'N', 'q':
				s.showMainMenu()
			}
		}
	}

	return false
}

func (s *menuState) handleMouse(event *tcell.EventMouse, width, height int) bool {
	x, y := event.Position()
	switch s.page {
	case mainMenu:
		item, ok := menuItemAt(x, y, width, height, len(s.splash()))
		if ok {
			s.selected = item
			if event.Buttons() == tcell.Button1 {
				s.activate(item)
			}
		}
	case playMenu:
		item, ok := playMenuItemAt(x, y, width, height)
		if ok {
			s.playSelected = item
			if event.Buttons() == tcell.Button1 {
				s.activatePlayItem(item)
				if s.result == LocalGame {
					return true
				}
			}
		}
	case optionsMenu:
		item, ok := optionsMenuItemAt(x, y, width, height)
		if ok {
			s.optionsSelected = item
			if event.Buttons() == tcell.Button1 {
				if item == theme {
					s.page = themeMenu
				} else {
					s.showMainMenu()
				}
			}
		}
	case themeMenu:
		if event.Buttons() == tcell.Button1 {
			if x < width/2 {
				s.moveTheme(-1)
			} else {
				s.moveTheme(1)
			}
		}
	case comingSoon:
		if event.Buttons() == tcell.Button1 {
			s.showMainMenu()
		}
	case exitConfirmation:
		choice, ok := exitChoiceAt(x, y, width, height)
		if !ok {
			if event.Buttons() == tcell.Button1 {
				s.showMainMenu()
			}
			return false
		}

		s.confirmExit = choice
		if event.Buttons() == tcell.Button1 {
			if choice {
				s.result = Exit
				return true
			}
			s.showMainMenu()
		}
	}

	return false
}

func (s menuState) splash() []string {
	return pieceSplashes[s.splashIndex%len(pieceSplashes)]
}

func (s *menuState) moveSelection(delta int) {
	s.selected = menuItem((int(s.selected) + delta + int(menuItemCount)) % int(menuItemCount))
}

func (s *menuState) movePlaySelection(delta int) {
	s.playSelected = playMenuItem((int(s.playSelected) + delta + int(playMenuItemCount)) % int(playMenuItemCount))
}

func (s *menuState) moveOptionsSelection(delta int) {
	s.optionsSelected = optionsMenuItem((int(s.optionsSelected) + delta + int(optionsMenuItemCount)) % int(optionsMenuItemCount))
}

func (s *menuState) moveTheme(delta int) {
	themes := chess.ColorThemes()
	index := 0
	for i, candidate := range themes {
		if candidate == s.theme {
			index = i
			break
		}
	}
	s.theme = themes[(index+delta+len(themes))%len(themes)]
}

func (s *menuState) activate(item menuItem) {
	if item == play {
		s.page = playMenu
		s.playSelected = localGame
		return
	}
	if item == exit {
		s.page = exitConfirmation
		s.confirmExit = false
		return
	}
	if item == options {
		s.page = optionsMenu
		s.optionsSelected = theme
		return
	}

	s.page = comingSoon
	s.comingSoonLabel = menuLabels[item]
}

func (s *menuState) activatePlayItem(item playMenuItem) {
	switch item {
	case localGame:
		s.result = LocalGame
	case playWithStockfish:
		s.page = comingSoon
		s.comingSoonLabel = playMenuLabels[item]
	case back:
		s.showMainMenu()
	}
}

func (s *menuState) showMainMenu() {
	s.page = mainMenu
	s.splashIndex = rand.Intn(len(pieceSplashes))
}

func draw(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	if !menuFits(width, height) {
		drawCentered(screen, height/2-1, "Terminal too small", titleStyle)
		drawCentered(screen, height/2+1, "Resize or Esc to exit", mutedStyle)
		return
	}

	switch state.page {
	case mainMenu:
		drawMainMenu(screen, state)
	case playMenu:
		drawPlayMenu(screen, state.playSelected)
	case optionsMenu:
		drawOptionsMenu(screen, state.optionsSelected)
	case themeMenu:
		drawThemeMenu(screen, state.theme)
	case comingSoon:
		drawComingSoon(screen, state.comingSoonLabel)
	case exitConfirmation:
		drawExitConfirmation(screen, state.confirmExit)
	}
}

func drawMainMenu(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	splash := state.splash()
	layout := mainMenuLayout(width, height, len(splash))

	drawSplash(screen, layout.y-len(splash)-1, splash)
	for item, label := range menuLabels {
		style := backgroundStyle
		if menuItem(item) == state.selected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+item, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(menuItemCount)+1, "Up/Down to select  Enter to choose", mutedStyle)
}

func drawSplash(screen tcell.Screen, y int, splash []string) {
	width, _ := screen.Size()
	splashWidth := longestLineWidth(gochsWordmark) + 4 + longestLineWidth(splash)
	x := (width - splashWidth) / 2
	for row, line := range splash {
		if row < len(gochsWordmark) {
			drawString(screen, x, y+row, gochsWordmark[row], titleStyle)
		}
		drawString(screen, x+longestLineWidth(gochsWordmark)+4, y+row, line, backgroundStyle)
	}
}

func drawPlayMenu(screen tcell.Screen, selected playMenuItem) {
	width, height := screen.Size()
	layout := simpleMenuLayout(width, height, playMenuLabels[:])

	drawCentered(screen, layout.y-3, "Play", titleStyle)
	for item, label := range playMenuLabels {
		style := backgroundStyle
		if playMenuItem(item) == selected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+item, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(playMenuItemCount)+1, "Up/Down to select  Enter to choose", mutedStyle)
}

func drawOptionsMenu(screen tcell.Screen, selected optionsMenuItem) {
	width, height := screen.Size()
	layout := simpleMenuLayout(width, height, optionsMenuLabels[:])

	drawCentered(screen, layout.y-3, "Options", titleStyle)
	for item, label := range optionsMenuLabels {
		style := backgroundStyle
		if optionsMenuItem(item) == selected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+item, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(optionsMenuItemCount)+1, "Up/Down to select  Enter to choose", mutedStyle)
}

func drawThemeMenu(screen tcell.Screen, current chess.ColorTheme) {
	width, height := screen.Size()
	light, dark := current.SquareColors()

	drawCentered(screen, height/2-4, "Theme", titleStyle)
	drawCentered(screen, height/2-2, current.String(), backgroundStyle)
	drawCentered(screen, height/2, "Light square", mutedStyle)
	drawColorPreview(screen, width/2-6, height/2+1, light)
	drawCentered(screen, height/2+3, "Dark square", mutedStyle)
	drawColorPreview(screen, width/2-6, height/2+4, dark)
	drawCentered(screen, height/2+6, "Left/Right to change  Enter/Esc to return", mutedStyle)
}

func drawColorPreview(screen tcell.Screen, x, y int, color chess.RGB) {
	style := backgroundStyle.Background(tcell.NewRGBColor(int32(color.R), int32(color.G), int32(color.B)))
	drawString(screen, x, y, "            ", style)
}

func drawComingSoon(screen tcell.Screen, label string) {
	_, height := screen.Size()
	drawCentered(screen, height/2-2, label, titleStyle)
	drawCentered(screen, height/2, "Coming soon", backgroundStyle)
	drawCentered(screen, height/2+2, "Press Enter, Esc, or click to return", mutedStyle)
}

func drawExitConfirmation(screen tcell.Screen, confirmExit bool) {
	_, height := screen.Size()
	drawCentered(screen, height/2-3, "Are you sure?", titleStyle)
	drawCentered(screen, height/2-1, "No", styleForExitChoice(false, confirmExit))
	drawCentered(screen, height/2+1, "Yes, exit gochs", styleForExitChoice(true, confirmExit))
	drawCentered(screen, height/2+3, "Up/Down to choose  Enter to confirm", mutedStyle)

}

func styleForExitChoice(choice, confirmExit bool) tcell.Style {
	if choice == confirmExit {
		return selectedStyle
	}

	return backgroundStyle
}

func mainMenuLayout(width, height, splashHeight int) menuLayout {
	return simpleMenuLayoutAt(
		width,
		(height-splashHeight-int(menuItemCount)-2)/2+splashHeight+1,
		menuLabels[:],
	)
}

func simpleMenuLayout(width, height int, labels []string) menuLayout {
	return simpleMenuLayoutAt(width, (height-len(labels))/2, labels)
}

func simpleMenuLayoutAt(width, y int, labels []string) menuLayout {
	menuWidth := 0
	for _, label := range labels {
		if len(label) > menuWidth {
			menuWidth = len(label)
		}
	}

	return menuLayout{
		x:     (width - menuWidth - 4) / 2,
		y:     y,
		width: menuWidth + 4,
	}
}

func menuItemAt(x, y, width, height, splashHeight int) (menuItem, bool) {
	layout := mainMenuLayout(width, height, splashHeight)
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

func playMenuItemAt(x, y, width, height int) (playMenuItem, bool) {
	layout := simpleMenuLayout(width, height, playMenuLabels[:])
	if x < layout.x || x >= layout.x+layout.width {
		return 0, false
	}

	for item := range playMenuItemCount {
		if y == layout.y+int(item) {
			return item, true
		}
	}

	return 0, false
}

func optionsMenuItemAt(x, y, width, height int) (optionsMenuItem, bool) {
	layout := simpleMenuLayout(width, height, optionsMenuLabels[:])
	if x < layout.x || x >= layout.x+layout.width {
		return 0, false
	}

	for item := range optionsMenuItemCount {
		if y == layout.y+int(item) {
			return item, true
		}
	}

	return 0, false
}

func exitChoiceAt(x, y, width, height int) (bool, bool) {
	if y == height/2-1 && x >= (width-2)/2 && x < (width+2)/2 {
		return false, true
	}
	if y == height/2+1 && x >= (width-15)/2 && x < (width+15)/2 {
		return true, true
	}

	return false, false
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

func drawCentered(screen tcell.Screen, y int, text string, style tcell.Style) {
	width, height := screen.Size()
	if y < 0 || y >= height {
		return
	}

	drawString(screen, (width-runewidth.StringWidth(text))/2, y, text, style)
}

func drawPaddedString(screen tcell.Screen, x, y, width int, text string, style tcell.Style) {
	drawString(screen, x, y, "  "+text, style)
	for column := runewidth.StringWidth(text) + 2; column < width; column++ {
		screen.SetContent(x+column, y, ' ', nil, style)
	}
}

func drawString(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	width, height := screen.Size()
	if y < 0 || y >= height {
		return
	}

	for _, character := range text {
		if x >= 0 && x < width {
			screen.SetContent(x, y, character, nil, style)
		}
		x += runewidth.RuneWidth(character)
	}
}
