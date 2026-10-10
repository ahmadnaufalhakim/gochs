package gui

import (
	"strings"
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
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

	if !state.handleKey(keyEvent(tcell.KeyEnter, 0)) || state.result != LocalGame {
		t.Fatalf("Local game activation = %#v, want LocalGame result", state)
	}

	state = menuState{page: playMenu, playSelected: playWithStockfish}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != comingSoon || state.comingSoonLabel != "Play with Stockfish" {
		t.Errorf("Stockfish activation = %#v, want coming-soon page", state)
	}

	state = menuState{page: playMenu, playSelected: back}
	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != mainMenu {
		t.Errorf("Back activation = %#v, want main menu", state)
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
