package gui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type page uint8

const (
	mainMenu page = iota
	playMenu
	optionsMenu
	themeMenu
	timeControlMenu
	customTimeControlMenu
	stockfishColorMenu
	stockfishDifficultyMenu
	stockfishUnavailable
	onlineGameTypeMenu
	onlineColorMenu
	lichessMenu
	lichessToken
	lichessSyncConfirmation
	lichessLoading
	comingSoon
	exitConfirmation
)

type menuItem uint8

const (
	play menuItem = iota
	puzzle
	lichess
	options
	credits
	exit
	menuItemCount
)

var menuLabels = [menuItemCount]string{
	"Play",
	"Puzzle",
	"Lichess",
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
	autoFlip
	optionsBack
	optionsMenuItemCount
)

type menuState struct {
	page                 page
	selected             menuItem
	playSelected         playMenuItem
	optionsSelected      optionsMenuItem
	comingSoonLabel      string
	confirmExit          bool
	splashIndex          int
	result               Result
	theme                chess.ColorTheme
	autoFlip             bool
	timeControlSelected  int
	timeControl          timeControl
	customMinutes        string
	customIncrement      string
	customFocus          customTimeField
	customMessage        string
	message              string
	stockfishPath        string
	stockfishColor       stockfishColor
	stockfishDifficulty  int
	stockfishGame        bool
	onlineGame           bool
	onlineRated          bool
	onlineGameTypeSelect int
	onlineColor          lichessapi.SeekColor
	onlineSeeking        bool
	onlineSeekGeneration uint64
	onlineActive         bool
	lichessAccount       *entity.LichessAccount
	lichessSelected      int
	lichessToken         string
	lichessMessage       string
	lichessAction        lichessAction
	offlinePuzzleRating  int
	offlineRatingLoaded  bool
	offlineRatingMessage string
}

type lichessAction uint8

const (
	lichessNoAction lichessAction = iota
	lichessLoad
	lichessConnect
	lichessDisconnect
	lichessSyncRating
)

type stockfishColor uint8

const (
	stockfishWhite stockfishColor = iota
	stockfishBlack
	stockfishRandom
	stockfishColorCount
)

var stockfishColorLabels = [stockfishColorCount]string{"Play as White", "Play as Black", "Play as Random"}

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
			if s.result == Puzzle {
				return true
			}
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
	case timeControlMenu:
		switch event.Key() {
		case tcell.KeyUp:
			s.moveTimeControlSelection(-1)
		case tcell.KeyDown:
			s.moveTimeControlSelection(1)
		case tcell.KeyEnter:
			return s.activateTimeControl()
		case tcell.KeyEsc:
			if s.onlineGame {
				s.page = onlineGameTypeMenu
			} else {
				s.page = playMenu
			}
		}
	case customTimeControlMenu:
		return s.handleCustomTimeControlKey(event)
	case stockfishColorMenu:
		switch event.Key() {
		case tcell.KeyUp:
			s.stockfishColor = (s.stockfishColor + stockfishColorCount - 1) % stockfishColorCount
		case tcell.KeyDown:
			s.stockfishColor = (s.stockfishColor + 1) % stockfishColorCount
		case tcell.KeyEnter:
			s.page = stockfishDifficultyMenu
		case tcell.KeyEsc:
			s.page = playMenu
		}
	case stockfishDifficultyMenu:
		switch event.Key() {
		case tcell.KeyUp:
			s.stockfishDifficulty = max(1, s.stockfishDifficulty-1)
		case tcell.KeyDown:
			s.stockfishDifficulty = min(8, s.stockfishDifficulty+1)
		case tcell.KeyEnter:
			s.page = timeControlMenu
			s.timeControlSelected = 0
			s.stockfishGame = true
		case tcell.KeyEsc:
			s.page = stockfishColorMenu
		}
	case stockfishUnavailable:
		if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyEsc {
			s.page = playMenu
		}
	case onlineGameTypeMenu:
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown:
			s.onlineGameTypeSelect = 1 - s.onlineGameTypeSelect
		case tcell.KeyEnter:
			s.onlineRated = s.onlineGameTypeSelect == 1
			s.page = timeControlMenu
			s.timeControlSelected = onlineTimeControlStart
		case tcell.KeyEsc:
			s.page = lichessMenu
		}
	case onlineColorMenu:
		switch event.Key() {
		case tcell.KeyUp:
			s.moveOnlineColor(-1)
		case tcell.KeyDown:
			s.moveOnlineColor(1)
		case tcell.KeyEnter:
			s.result = OnlineGame
			return true
		case tcell.KeyEsc:
			s.page = timeControlMenu
		}
	case lichessMenu:
		s.handleLichessMenuKey(event)
		if s.result == OnlineGame {
			return true
		}
	case lichessToken:
		s.handleLichessTokenKey(event)
	case lichessSyncConfirmation:
		switch event.Key() {
		case tcell.KeyEnter:
			s.page = lichessLoading
			s.lichessAction = lichessSyncRating
		case tcell.KeyEsc:
			s.page = lichessMenu
		}
	case lichessLoading:
		if event.Key() == tcell.KeyEsc {
			s.page = lichessMenu
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
			} else if s.optionsSelected == autoFlip {
				s.autoFlip = !s.autoFlip
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
				if s.result == Puzzle {
					return true
				}
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
	case timeControlMenu:
		item, ok := timeControlChoiceAt(x, y, width, height, s.onlineGame)
		if ok {
			s.timeControlSelected = item
			if event.Buttons() == tcell.Button1 {
				return s.activateTimeControl()
			}
		}
	case customTimeControlMenu:
		if field, ok := customTimeFieldAt(x, y, width, height); ok {
			s.customFocus = field
			return false
		}
		if event.Buttons() == tcell.Button1 && customBackAt(x, y, width, height) {
			s.page = timeControlMenu
			s.customMessage = ""
		}
	case stockfishColorMenu:
		color, ok := stockfishColorAt(x, y, width, height)
		if ok {
			s.stockfishColor = color
			if event.Buttons() == tcell.Button1 {
				s.page = stockfishDifficultyMenu
			}
		}
	case onlineGameTypeMenu:
		gameType, ok := onlineGameTypeAt(x, y, width, height)
		if ok {
			s.onlineGameTypeSelect = gameType
			if event.Buttons() == tcell.Button1 {
				s.onlineRated = gameType == 1
				s.page = timeControlMenu
				s.timeControlSelected = onlineTimeControlStart
			}
		}
	case onlineColorMenu:
		color, ok := onlineColorAt(x, y, width, height)
		if ok {
			s.onlineColor = color
			if event.Buttons() == tcell.Button1 {
				s.result = OnlineGame
				return true
			}
		}
	case lichessMenu:
		item, ok := lichessMenuItemAt(x, y, width, height, s.lichessAccount != nil)
		if ok {
			s.lichessSelected = item
			if event.Buttons() == tcell.Button1 {
				s.activateLichessMenuItem()
				if s.result == OnlineGame {
					return true
				}
			}
		}
	case optionsMenu:
		item, ok := optionsMenuItemAt(x, y, width, height, s.autoFlip)
		if ok {
			s.optionsSelected = item
			if event.Buttons() == tcell.Button1 {
				if item == theme {
					s.page = themeMenu
				} else if item == autoFlip {
					s.autoFlip = !s.autoFlip
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
	if item == puzzle {
		s.result = Puzzle
		return
	}
	if item == lichess {
		s.page = lichessLoading
		s.lichessAction = lichessLoad
		s.lichessMessage = ""
		return
	}

	s.page = comingSoon
	s.comingSoonLabel = menuLabels[item]
}

func (s *menuState) activatePlayItem(item playMenuItem) {
	switch item {
	case localGame:
		s.onlineGame = false
		s.stockfishGame = false
		s.page = timeControlMenu
		s.timeControlSelected = 0
	case playWithStockfish:
		if s.stockfishPath == "" {
			s.page = stockfishUnavailable
			return
		}
		s.page = stockfishColorMenu
		s.stockfishColor = stockfishWhite
		s.stockfishDifficulty = 1
	case back:
		s.showMainMenu()
	}
}

func (s *menuState) showMainMenu() {
	s.page = mainMenu
	s.onlineGame = false
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
		drawPlayMenu(screen, state)
	case optionsMenu:
		drawOptionsMenu(screen, state.optionsSelected, state.autoFlip)
	case themeMenu:
		drawThemeMenu(screen, state.theme)
	case timeControlMenu:
		drawTimeControlMenu(screen, state)
	case customTimeControlMenu:
		drawCustomTimeControl(screen, state)
	case stockfishColorMenu:
		drawStockfishColorMenu(screen, state)
	case stockfishDifficultyMenu:
		drawStockfishDifficultyMenu(screen, state)
	case stockfishUnavailable:
		drawStockfishUnavailable(screen)
	case onlineGameTypeMenu:
		drawOnlineGameTypeMenu(screen, state)
	case onlineColorMenu:
		drawOnlineColorMenu(screen, state)
	case lichessMenu:
		drawLichessMenu(screen, state)
	case lichessToken:
		drawLichessToken(screen, state)
	case lichessSyncConfirmation:
		drawLichessSyncConfirmation(screen, state)
	case lichessLoading:
		drawLichessLoading(screen)
	case comingSoon:
		drawComingSoon(screen, state.comingSoonLabel)
	case exitConfirmation:
		drawExitConfirmation(screen, state.confirmExit)
	}
}

func (s *menuState) handleLichessMenuKey(event *tcell.EventKey) {
	count := 2
	if s.lichessAccount != nil {
		count = 5
	}
	switch event.Key() {
	case tcell.KeyUp:
		s.lichessSelected = (s.lichessSelected + count - 1) % count
	case tcell.KeyDown:
		s.lichessSelected = (s.lichessSelected + 1) % count
	case tcell.KeyEnter:
		s.activateLichessMenuItem()
	case tcell.KeyEsc:
		s.showMainMenu()
	}
}

func (s *menuState) activateLichessMenuItem() {
	if s.lichessAccount == nil {
		if s.lichessSelected == 0 {
			s.page = lichessToken
			s.lichessMessage = ""
		} else {
			s.showMainMenu()
		}
		return
	}
	switch s.lichessSelected {
	case 0:
		s.page = lichessLoading
		s.lichessAction = lichessLoad
		s.lichessMessage = ""
	case 1:
		s.page = lichessLoading
		s.lichessAction = lichessDisconnect
		s.lichessMessage = ""
	case 2:
		s.offlineRatingLoaded = false
		s.offlineRatingMessage = ""
		s.page = lichessSyncConfirmation
	case 3:
		if s.onlineActive {
			s.result = OnlineGame
			return
		}
		s.onlineGame = true
		s.stockfishGame = false
		s.onlineRated = false
		s.onlineGameTypeSelect = 0
		s.page = onlineGameTypeMenu
	case 4:
		s.showMainMenu()
	}
}

func (s *menuState) handleLichessTokenKey(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyEsc:
		s.lichessToken = ""
		s.page = lichessMenu
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(s.lichessToken) > 0 {
			s.lichessToken = s.lichessToken[:len(s.lichessToken)-1]
		}
	case tcell.KeyEnter:
		if s.lichessToken == "" {
			s.lichessMessage = "Enter a Lichess Personal API Token"
			return
		}
		s.page = lichessLoading
		s.lichessAction = lichessConnect
	case tcell.KeyRune:
		if len(s.lichessToken) < 512 && event.Rune() >= 0x20 && event.Rune() <= 0x7e {
			s.lichessToken += string(event.Rune())
		}
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
	if state.message != "" {
		drawCentered(screen, layout.y+int(menuItemCount)+3, state.message, mutedStyle)
	}
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

func drawPlayMenu(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	layout := simpleMenuLayout(width, height, playMenuLabels[:])

	drawCentered(screen, layout.y-3, "Play", titleStyle)
	for item, label := range playMenuLabels {
		style := backgroundStyle
		if playMenuItem(item) == playWithStockfish && state.stockfishPath == "" {
			style = mutedStyle
		} else if playMenuItem(item) == state.playSelected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+item, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(playMenuItemCount)+1, "Up/Down to select  Enter to choose", mutedStyle)
}

func drawStockfishColorMenu(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	layout := simpleMenuLayout(width, height, stockfishColorLabels[:])
	drawCentered(screen, layout.y-3, "Play > Stockfish", titleStyle)
	drawCentered(screen, layout.y-2, "Please select color to move", mutedStyle)
	for color, label := range stockfishColorLabels {
		style := backgroundStyle
		if stockfishColor(color) == state.stockfishColor {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+color, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(stockfishColorCount)+1, "Up/Down to select  Enter to choose", mutedStyle)
}

func drawStockfishDifficultyMenu(screen tcell.Screen, state menuState) {
	_, height := screen.Size()
	drawCentered(screen, height/2-4, "Play > Stockfish", titleStyle)
	drawCentered(screen, height/2-2, "Please select the difficulty", mutedStyle)
	drawCentered(screen, height/2, fmt.Sprintf("Level %d / 8", state.stockfishDifficulty), selectedStyle)
	drawCentered(screen, height/2+2, "Up/Down to select  Enter to choose", mutedStyle)
}

func drawStockfishUnavailable(screen tcell.Screen) {
	_, height := screen.Size()
	drawCentered(screen, height/2-3, "Stockfish is not installed", titleStyle)
	drawCentered(screen, height/2-1, "Install it with: sudo apt install stockfish", mutedStyle)
	drawCentered(screen, height/2+1, "Then restart gochs", mutedStyle)
	drawCentered(screen, height/2+3, "Press Enter or Esc to return", mutedStyle)
}

func drawLichessMenu(screen tcell.Screen, state menuState) {
	_, height := screen.Size()
	if state.lichessAccount == nil {
		labels := []string{"Connect with token", "Back"}
		drawCentered(screen, height/2-5, "Lichess", titleStyle)
		drawCentered(screen, height/2-3, "Connect with a Lichess Personal API Token", mutedStyle)
		drawLichessActions(screen, labels, state.lichessSelected, height/2)
		if state.lichessMessage != "" {
			drawCentered(screen, height/2+4, state.lichessMessage, mutedStyle)
		}
		return
	}

	account := state.lichessAccount
	name := account.Username
	if account.Title != "" {
		name = account.Title + " " + name
	}
	if account.Online {
		name += "  Online"
	}
	drawCentered(screen, height/2-9, "Lichess", titleStyle)
	drawCentered(screen, height/2-7, name, backgroundStyle)
	if account.Patron {
		drawCentered(screen, height/2-6, "Patron", mutedStyle)
	}
	drawCentered(screen, height/2-4, fmt.Sprintf("Bullet     %s       Blitz      %s", ratingLabel(account.Bullet), ratingLabel(account.Blitz)), backgroundStyle)
	drawCentered(screen, height/2-3, fmt.Sprintf("Rapid      %s       Classical  %s", ratingLabel(account.Rapid), ratingLabel(account.Classical)), backgroundStyle)
	drawCentered(screen, height/2-2, fmt.Sprintf("Puzzle     %s", ratingLabel(account.Puzzle)), backgroundStyle)
	drawCentered(screen, height/2, fmt.Sprintf("Games      %d       %d W / %d D / %d L", account.Games, account.Wins, account.Draws, account.Losses), backgroundStyle)
	drawCentered(screen, height/2+1, fmt.Sprintf("Play time  %s       Member since %s", durationLabel(account.PlayTime), dateLabel(account.MemberSince)), backgroundStyle)
	labels := []string{"Refresh", "Disconnect", "Sync puzzle rating", "Play online", "Back"}
	drawLichessActions(screen, labels, state.lichessSelected, height/2+3)
	if account.FromEnvironment {
		drawCentered(screen, height/2+7, "Using GOCHS_LICHESS_TOKEN", mutedStyle)
	} else if state.lichessMessage != "" {
		messageY := height/2 + 7
		if strings.HasPrefix(state.lichessMessage, "Offline puzzle rating synced to ") {
			messageY += 2
		}
		drawCentered(screen, messageY, state.lichessMessage, mutedStyle)
	}
}

func drawLichessActions(screen tcell.Screen, labels []string, selected, y int) {
	width, _ := screen.Size()
	layout := simpleMenuLayoutAt(width, y, labels)
	for index, label := range labels {
		style := backgroundStyle
		if index == selected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, y+index, layout.width, label, style)
	}
}

func drawLichessToken(screen tcell.Screen, state menuState) {
	_, height := screen.Size()
	drawCentered(screen, height/2-4, "Lichess > Connect", titleStyle)
	drawCentered(screen, height/2-2, "Paste your Lichess Personal API Token:", mutedStyle)
	drawCentered(screen, height/2, "["+maskedToken(state.lichessToken)+"]", selectedStyle)
	drawCentered(screen, height/2+2, "Enter to connect  Esc to cancel", mutedStyle)
	if state.lichessMessage != "" {
		drawCentered(screen, height/2+4, state.lichessMessage, mutedStyle)
	}
}

func drawLichessSyncConfirmation(screen tcell.Screen, state menuState) {
	_, height := screen.Size()
	drawCentered(screen, height/2-4, "Sync Offline Puzzle Rating", titleStyle)
	if state.lichessAccount == nil {
		drawCentered(screen, height/2, "No Lichess account is available", mutedStyle)
		return
	}
	drawCentered(screen, height/2-2, fmt.Sprintf("Lichess Puzzle rating: %d", state.lichessAccount.Puzzle), backgroundStyle)
	currentRating := "No offline rating yet"
	if state.offlineRatingLoaded {
		currentRating = fmt.Sprintf("Current offline rating: %d", state.offlinePuzzleRating)
	}
	if state.offlineRatingMessage != "" {
		currentRating = state.offlineRatingMessage
	}
	drawCentered(screen, height/2, currentRating, backgroundStyle)
	drawCentered(screen, height/2+2, "This sets your offline rating to the Lichess value.", mutedStyle)
	drawCentered(screen, height/2+3, "Puzzle history remains.", mutedStyle)
	drawCentered(screen, height/2+5, "Enter to sync  Esc to cancel", selectedStyle)
}

func maskedToken(token string) string {
	const maximumVisibleCharacters = 32
	if len(token) <= maximumVisibleCharacters {
		return strings.Repeat("*", len(token))
	}
	return strings.Repeat("*", maximumVisibleCharacters) + "..."
}

func drawLichessLoading(screen tcell.Screen) {
	_, height := screen.Size()
	drawCentered(screen, height/2-1, "Lichess", titleStyle)
	drawCentered(screen, height/2+1, "Contacting Lichess...", mutedStyle)
	drawCentered(screen, height/2+3, "Esc to cancel", mutedStyle)
}

func ratingLabel(rating int) string {
	if rating == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", rating)
}

func durationLabel(duration time.Duration) string {
	if duration <= 0 {
		return "-"
	}
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh", hours)
}

func dateLabel(date time.Time) string {
	if date.IsZero() {
		return "-"
	}
	return date.Local().Format("2006-01-02")
}

func drawOptionsMenu(screen tcell.Screen, selected optionsMenuItem, autoFlipEnabled bool) {
	width, height := screen.Size()
	labels := optionsLabels(autoFlipEnabled)
	layout := simpleMenuLayout(width, height, labels)

	drawCentered(screen, layout.y-3, "Options", titleStyle)
	for item, label := range labels {
		style := backgroundStyle
		if optionsMenuItem(item) == selected {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+item, layout.width, label, style)
	}
	drawCentered(screen, layout.y+int(optionsMenuItemCount)+1, "Up/Down to select  Enter to choose", mutedStyle)
}

func optionsLabels(autoFlipEnabled bool) []string {
	autoFlipLabel := "Auto-flip: Off"
	if autoFlipEnabled {
		autoFlipLabel = "Auto-flip: On"
	}
	return []string{"Theme", autoFlipLabel, "Back"}
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

func lichessMenuItemAt(x, y, width, height int, connected bool) (int, bool) {
	labels := []string{"Connect with token", "Back"}
	if connected {
		labels = []string{"Refresh", "Disconnect", "Sync puzzle rating", "Play online", "Back"}
	}
	menuY := height / 2
	if connected {
		menuY = height/2 + 3
	}
	layout := simpleMenuLayoutAt(width, menuY, labels)
	if x < layout.x || x >= layout.x+layout.width {
		return 0, false
	}
	for item := range labels {
		if y == layout.y+item {
			return item, true
		}
	}
	return 0, false
}

var onlineGameTypeLabels = []string{"Casual - no rating change", "Rated - affects your Lichess rating"}

func drawOnlineGameTypeMenu(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	layout := simpleMenuLayout(width, height, onlineGameTypeLabels)
	drawCentered(screen, layout.y-3, "Lichess > Play online", titleStyle)
	drawCentered(screen, layout.y-2, "Select game type", mutedStyle)
	for index, label := range onlineGameTypeLabels {
		style := backgroundStyle
		if state.onlineGameTypeSelect == index {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+index, layout.width, label, style)
	}
	drawCentered(screen, layout.y+len(onlineGameTypeLabels)+1, "Up/Down to select  Enter to choose", mutedStyle)
}

func onlineGameTypeAt(x, y, width, height int) (int, bool) {
	layout := simpleMenuLayout(width, height, onlineGameTypeLabels)
	if x < layout.x || x >= layout.x+layout.width || y < layout.y || y >= layout.y+len(onlineGameTypeLabels) {
		return 0, false
	}
	return y - layout.y, true
}

var onlineColorLabels = []string{"Play as White", "Play as Black", "Play as Random"}

func (s *menuState) moveOnlineColor(delta int) {
	colors := []lichessapi.SeekColor{lichessapi.SeekWhite, lichessapi.SeekBlack, lichessapi.SeekRandom}
	index := 0
	for i, color := range colors {
		if s.onlineColor == color {
			index = i
		}
	}
	s.onlineColor = colors[(index+delta+len(colors))%len(colors)]
}

func drawOnlineColorMenu(screen tcell.Screen, state menuState) {
	width, height := screen.Size()
	layout := simpleMenuLayout(width, height, onlineColorLabels)
	drawCentered(screen, layout.y-3, "Lichess > Play online", titleStyle)
	drawCentered(screen, layout.y-2, "Select your color", mutedStyle)
	colors := []lichessapi.SeekColor{lichessapi.SeekWhite, lichessapi.SeekBlack, lichessapi.SeekRandom}
	for i, label := range onlineColorLabels {
		style := backgroundStyle
		if state.onlineColor == colors[i] {
			style = selectedStyle
		}
		drawPaddedString(screen, layout.x, layout.y+i, layout.width, label, style)
	}
}

func onlineColorAt(x, y, width, height int) (lichessapi.SeekColor, bool) {
	layout := simpleMenuLayout(width, height, onlineColorLabels)
	if x < layout.x || x >= layout.x+layout.width {
		return "", false
	}
	colors := []lichessapi.SeekColor{lichessapi.SeekWhite, lichessapi.SeekBlack, lichessapi.SeekRandom}
	for i := range colors {
		if y == layout.y+i {
			return colors[i], true
		}
	}
	return "", false
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

func stockfishColorAt(x, y, width, height int) (stockfishColor, bool) {
	layout := simpleMenuLayout(width, height, stockfishColorLabels[:])
	if x < layout.x || x >= layout.x+layout.width {
		return 0, false
	}

	for color := range stockfishColorCount {
		if y == layout.y+int(color) {
			return color, true
		}
	}

	return 0, false
}

func optionsMenuItemAt(x, y, width, height int, autoFlipEnabled bool) (optionsMenuItem, bool) {
	layout := simpleMenuLayout(width, height, optionsLabels(autoFlipEnabled))
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
		characterWidth := runewidth.RuneWidth(character)
		if characterWidth == 0 {
			continue
		}
		if x >= 0 && x < width {
			screen.SetContent(x, y, character, nil, style)
		}
		x += characterWidth
	}
}
