package gui

import (
	"strings"
	"testing"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
	"github.com/gdamore/tcell/v2"
)

func keyEvent(key tcell.Key, character rune) *tcell.EventKey {
	return tcell.NewEventKey(key, character, tcell.ModNone)
}

func TestDrawMainMenu(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	draw(screen, menuState{selected: puzzle, splashIndex: 1})
	screen.Show()
	contents, width, height := screen.GetContents()
	text := screenText(contents, width, height)
	for _, want := range []string{"___| |__  ___", "<~~~~>", "Play", "Puzzle", "Options", "Credits", "Exit"} {
		if !strings.Contains(text, want) {
			t.Errorf("menu rendering does not contain %q", want)
		}
	}
}

func TestDrawSmallTerminalMessage(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(20, 10)

	draw(screen, menuState{})
	screen.Show()
	contents, width, height := screen.GetContents()
	if !strings.Contains(screenText(contents, width, height), "Terminal too small") {
		t.Error("small terminal message was not rendered")
	}
}

func TestDrawStringUsesDisplayWidth(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(20, 1)

	drawString(screen, 0, 0, "[♞ ] [♛ ]", backgroundStyle)
	screen.Show()
	contents, width, height := screen.GetContents()
	if got := screenText(contents, width, height); !strings.Contains(got, "[♞ ] [♛ ]") {
		t.Errorf("drawString() = %q, want intact chess glyph labels", got)
	}
}

func screenText(cells []tcell.SimCell, width, height int) string {
	var result strings.Builder
	for y := range height {
		for x := range width {
			result.Write(cells[y*width+x].Bytes)
		}
		result.WriteByte('\n')
	}

	return result.String()
}

func TestMenuSelectionWraps(t *testing.T) {
	state := menuState{selected: play}

	state.handleKey(keyEvent(tcell.KeyUp, 0))
	if state.selected != exit {
		t.Errorf("selection after Up = %d, want Exit", state.selected)
	}

	state.handleKey(keyEvent(tcell.KeyDown, 0))
	if state.selected != play {
		t.Errorf("selection after Down = %d, want Play", state.selected)
	}
}

func TestMenuActivation(t *testing.T) {
	state := menuState{selected: options}

	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != optionsMenu || state.optionsSelected != theme {
		t.Errorf("options activation = %#v, want Options menu with Theme selected", state)
	}

	state.handleKey(keyEvent(tcell.KeyEsc, 0))
	if state.page != mainMenu {
		t.Errorf("Esc from Options menu = %d, want main menu", state.page)
	}

	state.selected = exit
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != exitConfirmation || state.confirmExit {
		t.Errorf("exit activation = %#v, want confirmation with No selected", state)
	}
}

func TestLichessMenuFlow(t *testing.T) {
	state := menuState{selected: lichess}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != lichessLoading || state.lichessAction != lichessLoad {
		t.Fatalf("Lichess activation = %#v", state)
	}

	state.applyLichessResult(lichessResult{action: lichessLoad, err: lichessapi.ErrNoToken})
	if state.page != lichessMenu || state.lichessAccount != nil {
		t.Fatalf("no-token result = %#v", state)
	}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != lichessToken {
		t.Fatalf("connect selection page = %d, want token page", state.page)
	}
	for _, character := range "abc" {
		state.handleKey(keyEvent(tcell.KeyRune, character))
	}
	if state.lichessToken != "abc" {
		t.Errorf("token input = %q", state.lichessToken)
	}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != lichessLoading || state.lichessAction != lichessConnect {
		t.Fatalf("token submit = %#v", state)
	}

	account := entity.LichessAccount{Username: "example", Bullet: 1800}
	state.applyLichessResult(lichessResult{action: lichessConnect, account: account})
	if state.page != lichessMenu || state.lichessAccount == nil || state.lichessAccount.Username != "example" {
		t.Fatalf("connected account = %#v", state)
	}
}

func TestDrawLichessTokenMasksToken(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	draw(screen, menuState{page: lichessToken, lichessToken: "private-token"})
	screen.Show()
	contents, width, height := screen.GetContents()
	text := screenText(contents, width, height)
	if strings.Contains(text, "private-token") {
		t.Error("token was rendered in plain text")
	}
	if !strings.Contains(text, "*************") {
		t.Error("masked token was not rendered")
	}
}

func TestLichessDisconnect(t *testing.T) {
	state := menuState{page: lichessMenu, lichessAccount: &entity.LichessAccount{Username: "example"}, lichessSelected: 1, lichessToken: "private-token"}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != lichessLoading || state.lichessAction != lichessDisconnect {
		t.Fatalf("disconnect selection = %#v", state)
	}
	state.applyLichessResult(lichessResult{action: lichessDisconnect})
	if state.lichessAccount != nil || state.lichessToken != "" || state.page != lichessMenu {
		t.Errorf("disconnect result = %#v", state)
	}
}

func TestLichessPuzzleRatingSyncConfirmation(t *testing.T) {
	state := menuState{page: lichessMenu, lichessAccount: &entity.LichessAccount{ID: "example", Username: "example", Puzzle: 1800}, lichessSelected: 2}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != lichessSyncConfirmation {
		t.Fatalf("sync selection page = %d", state.page)
	}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != lichessLoading || state.lichessAction != lichessSyncRating {
		t.Fatalf("sync confirmation = %#v", state)
	}
	state.applyLichessResult(lichessResult{action: lichessSyncRating, profile: entity.TrainingProfile{Rating: 1800}})
	if state.page != lichessMenu || !strings.Contains(state.lichessMessage, "1800") {
		t.Errorf("sync result = %#v", state)
	}
}

func TestDrawLichessSyncShowsOfflineRating(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	draw(screen, menuState{
		page:                lichessSyncConfirmation,
		lichessAccount:      &entity.LichessAccount{Puzzle: 1800},
		offlineRatingLoaded: true,
		offlinePuzzleRating: 1725,
	})
	screen.Show()
	contents, width, height := screen.GetContents()
	if !strings.Contains(screenText(contents, width, height), "Current offline rating: 1725") {
		t.Error("sync confirmation does not show the current offline rating")
	}
}

func TestExitConfirmation(t *testing.T) {
	state := menuState{page: exitConfirmation}
	if state.handleKey(keyEvent(tcell.KeyEnter, 0)) {
		t.Fatal("Enter with No selected exited")
	}
	if state.page != mainMenu {
		t.Errorf("Enter with No selected changed page to %d, want main menu", state.page)
	}

	state = menuState{page: exitConfirmation}
	state.handleKey(keyEvent(tcell.KeyDown, 0))
	if !state.confirmExit {
		t.Fatal("Down did not select Yes")
	}
	if !state.handleKey(keyEvent(tcell.KeyEnter, 0)) {
		t.Fatal("Enter with Yes selected did not exit")
	}
}

func TestMouseActivatesMenuItem(t *testing.T) {
	const width, height = 80, 24
	state := menuState{}
	layout := mainMenuLayout(width, height, len(state.splash()))

	state.handleMouse(tcell.NewEventMouse(layout.x, layout.y+int(credits), tcell.Button1, tcell.ModNone), width, height)
	if state.page != comingSoon || state.comingSoonLabel != "Credits" {
		t.Errorf("credits click = %#v, want coming-soon Credits page", state)
	}

	state = menuState{}
	state.handleMouse(tcell.NewEventMouse(layout.x, layout.y+int(exit), tcell.Button1, tcell.ModNone), width, height)
	if state.page != exitConfirmation || state.confirmExit {
		t.Errorf("exit click = %#v, want confirmation with No selected", state)
	}
}

func TestMouseHoverSelectsMenuItem(t *testing.T) {
	const width, height = 80, 24
	state := menuState{}
	layout := mainMenuLayout(width, height, len(state.splash()))

	state.handleMouse(tcell.NewEventMouse(layout.x, layout.y+int(puzzle), tcell.ButtonNone, tcell.ModNone), width, height)
	if state.selected != puzzle || state.page != mainMenu {
		t.Errorf("puzzle hover = %#v, want Puzzle selected on main menu", state)
	}
}

func TestMouseHoverSelectsExitChoice(t *testing.T) {
	const width, height = 80, 24
	state := menuState{page: exitConfirmation}

	state.handleMouse(tcell.NewEventMouse(width/2, height/2+1, tcell.ButtonNone, tcell.ModNone), width, height)
	if !state.confirmExit || state.page != exitConfirmation {
		t.Errorf("Yes hover = %#v, want Yes selected on confirmation page", state)
	}

	state.handleMouse(tcell.NewEventMouse(width/2, height/2-1, tcell.ButtonNone, tcell.ModNone), width, height)
	if state.confirmExit || state.page != exitConfirmation {
		t.Errorf("No hover = %#v, want No selected on confirmation page", state)
	}
}

func TestPlayMenuActivation(t *testing.T) {
	state := menuState{selected: play}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != playMenu || state.playSelected != localGame {
		t.Fatalf("Play activation = %#v, want play menu with Local game selected", state)
	}

	if state.handleKey(keyEvent(tcell.KeyEnter, 0)) || state.page != timeControlMenu {
		t.Fatalf("Local game activation = %#v, want time-control menu", state)
	}

	if !state.handleKey(keyEvent(tcell.KeyEnter, 0)) || state.result != LocalGame || state.timeControl != timeControlChoices[0].control {
		t.Fatalf("Time-control activation = %#v, want LocalGame result with 1+0", state)
	}

	state = menuState{page: playMenu, playSelected: playWithStockfish}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != stockfishUnavailable {
		t.Errorf("unavailable Stockfish activation = %#v, want unavailable page", state)
	}

	state = menuState{page: playMenu, playSelected: playWithStockfish, stockfishPath: "/usr/games/stockfish"}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != stockfishColorMenu || state.stockfishDifficulty != 1 {
		t.Errorf("Stockfish activation = %#v, want color menu", state)
	}

	state = menuState{page: playMenu, playSelected: back}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != mainMenu {
		t.Errorf("Back activation = %#v, want main menu", state)
	}
}

func TestStockfishColorMenuMouseSelection(t *testing.T) {
	const width, height = 80, 24
	state := menuState{page: stockfishColorMenu}
	layout := simpleMenuLayout(width, height, stockfishColorLabels[:])

	state.handleMouse(tcell.NewEventMouse(layout.x, layout.y+int(stockfishBlack), tcell.ButtonNone, tcell.ModNone), width, height)
	if state.stockfishColor != stockfishBlack {
		t.Errorf("hovered color = %d, want black", state.stockfishColor)
	}

	state.handleMouse(tcell.NewEventMouse(layout.x, layout.y+int(stockfishRandom), tcell.Button1, tcell.ModNone), width, height)
	if state.stockfishColor != stockfishRandom || state.page != stockfishDifficultyMenu {
		t.Errorf("clicked color state = %#v", state)
	}
}

func TestPuzzleMenuActivation(t *testing.T) {
	state := menuState{selected: puzzle}
	if !state.handleKey(keyEvent(tcell.KeyEnter, 0)) {
		t.Fatal("Puzzle activation did not return a transition")
	}
	if state.result != Puzzle {
		t.Errorf("Puzzle activation result = %d, want Puzzle", state.result)
	}
}

func TestCustomTimeControlValidation(t *testing.T) {
	state := menuState{page: customTimeControlMenu, customMinutes: "0", customIncrement: "1"}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != customTimeControlMenu || state.customMessage == "" {
		t.Fatalf("invalid custom control = %#v, want validation message", state)
	}

	state.customMinutes = "10"
	state.customIncrement = "5"
	if !state.handleKey(keyEvent(tcell.KeyEnter, 0)) || state.result != LocalGame {
		t.Fatalf("valid custom control = %#v, want LocalGame result", state)
	}
	if state.timeControl != (timeControl{initial: 10 * time.Minute, increment: 5 * time.Second, event: "custom game"}) {
		t.Errorf("custom time control = %#v, want 10+5", state.timeControl)
	}
}

func TestTimeControlMenuIsCentered(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 30)

	draw(screen, menuState{page: timeControlMenu})
	screen.Show()
	contents, width, height := screen.GetContents()
	lines := strings.Split(screenText(contents, width, height), "\n")
	if !strings.Contains(lines[4], "Play > Local game") {
		t.Errorf("time-control title line = %q, want vertically centered title", lines[4])
	}
	if !strings.Contains(lines[25], "Up/Down to select  Enter to choose") {
		t.Errorf("time-control hint line = %q", lines[25])
	}
	if choice, ok := timeControlChoiceAt(width/2, 8, width, height); !ok || choice != 0 {
		t.Errorf("centered first choice = %d, %t; want 0, true", choice, ok)
	}
}

func TestThemeMenuChangesTheme(t *testing.T) {
	state := menuState{page: optionsMenu, optionsSelected: theme, theme: chess.WOOD}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != themeMenu {
		t.Fatalf("Theme activation = %#v, want theme menu", state)
	}

	state.handleKey(keyEvent(tcell.KeyRight, 0))
	if state.theme == chess.WOOD {
		t.Fatal("Right did not change theme")
	}

	state.handleKey(keyEvent(tcell.KeyEsc, 0))
	if state.page != optionsMenu {
		t.Errorf("Esc from theme menu = %d, want Options menu", state.page)
	}
}

func TestOptionsMenuTogglesAutoFlip(t *testing.T) {
	state := menuState{page: optionsMenu, optionsSelected: autoFlip}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if !state.autoFlip || state.page != optionsMenu {
		t.Errorf("auto-flip activation = %#v, want enabled in Options", state)
	}

	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.autoFlip {
		t.Error("second auto-flip activation did not disable it")
	}
}
