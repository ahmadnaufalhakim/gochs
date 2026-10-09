package gui

import (
	"strings"
	"testing"

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

	draw(screen, menuState{selected: puzzle})
	screen.Show()
	contents, width, height := screen.GetContents()
	text := screenText(contents, width, height)
	for _, want := range []string{"gochs", "Play with Stockfish", "Puzzle", "Options", "Credits", "Exit"} {
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
	state := menuState{selected: playWithStockfish}

	state.handleKey(keyEvent(tcell.KeyUp, 0))
	if state.selected != exit {
		t.Errorf("selection after Up = %d, want Exit", state.selected)
	}

	state.handleKey(keyEvent(tcell.KeyRune, 'j'))
	if state.selected != playWithStockfish {
		t.Errorf("selection after j = %d, want Play with Stockfish", state.selected)
	}
}

func TestMenuActivation(t *testing.T) {
	state := menuState{selected: options}

	state.handleKey(keyEvent(tcell.KeyEnter, 0))
	if state.page != comingSoon || state.unavailableItem != options {
		t.Errorf("options activation = %#v, want coming-soon Options page", state)
	}

	state.handleKey(keyEvent(tcell.KeyEsc, 0))
	if state.page != mainMenu {
		t.Errorf("Esc from coming-soon page = %d, want main menu", state.page)
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
	state.handleKey(keyEvent(tcell.KeyRight, 0))
	if !state.confirmExit {
		t.Fatal("Right did not select Yes")
	}
	if !state.handleKey(keyEvent(tcell.KeyEnter, 0)) {
		t.Fatal("Enter with Yes selected did not exit")
	}
}

func TestMouseActivatesMenuItem(t *testing.T) {
	const width, height = 80, 24
	layout := mainMenuLayout(width, height)
	state := menuState{}

	state.handleMouse(tcell.NewEventMouse(layout.x, layout.y+int(credits)*2, tcell.Button1, tcell.ModNone), width, height)
	if state.page != comingSoon || state.unavailableItem != credits {
		t.Errorf("credits click = %#v, want coming-soon Credits page", state)
	}

	state = menuState{}
	state.handleMouse(tcell.NewEventMouse(layout.x, layout.y+int(exit)*2, tcell.Button1, tcell.ModNone), width, height)
	if state.page != exitConfirmation || state.confirmExit {
		t.Errorf("exit click = %#v, want confirmation with No selected", state)
	}
}
