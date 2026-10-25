package gui

import (
	"strings"
	"testing"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
	"github.com/ahmadnaufalhakim/gochs/internal/sound"
	"github.com/gdamore/tcell/v2"
)

func recordingPlayer(events *[]sound.Event) sound.Player {
	return sound.PlayerFunc(func(event sound.Event) { *events = append(*events, event) })
}

func TestLocalMoveSoundOrder(t *testing.T) {
	var events []sound.Event
	game := newLocalGameState(chess.WOOD)
	game.sounds = recordingPlayer(&events)
	game.startSounds()
	for _, uci := range []string{"e2e4", "d7d5", "e4d5"} {
		game.uciInput = uci
		game.submitUCI()
	}
	want := []sound.Event{sound.Game, sound.Move, sound.Move, sound.Move, sound.Capture}
	assertSoundEvents(t, events, want)
}

func TestCheckmatingCaptureSoundOrder(t *testing.T) {
	var events []sound.Event
	game := newLocalGameState(chess.WOOD)
	game.sounds = recordingPlayer(&events)
	game.startSounds()
	for _, uci := range []string{"e2e4", "e7e5", "f1c4", "b8c6", "d1h5", "g8f6"} {
		game.uciInput = uci
		game.submitUCI()
	}
	events = nil
	game.uciInput = "h5f7"
	game.submitUCI()
	assertSoundEvents(t, events, []sound.Event{sound.Move, sound.Capture, sound.Checkmate, sound.Game})
}

func TestLowTimePlaysOnceForEachColor(t *testing.T) {
	var events []sound.Event
	game := newLocalGameState(chess.WOOD)
	game.sounds = recordingPlayer(&events)
	game.startClock(timeControl{initial: time.Minute, increment: time.Second})
	game.clockRemaining[chess.White] = 10 * time.Second
	game.clockRemaining[chess.Black] = 10 * time.Second
	game.playLowTime(chess.White)
	game.playLowTime(chess.White)
	game.playLowTime(chess.Black)
	assertSoundEvents(t, events, []sound.Event{sound.LowTime, sound.LowTime})
}

func TestReplayPlaysOnlyMoveAndCapture(t *testing.T) {
	var events []sound.Event
	game := newLocalGameState(chess.WOOD)
	game.sounds = recordingPlayer(&events)
	for _, uci := range []string{"e2e4", "d7d5", "e4d5"} {
		game.uciInput = uci
		game.submitUCI()
	}
	game.terminalTitle = "Draw · ½-½"
	events = nil
	game.moveReplay(-1)
	assertSoundEvents(t, events, []sound.Event{sound.Move, sound.Capture})
}

func TestOnlineSoundsUseAuthoritativeUpdatesOnce(t *testing.T) {
	full := lichessapi.GameFullEvent{Game: lichessapi.Game{ID: "game", White: lichessapi.Player{ID: "me"}, Black: lichessapi.Player{ID: "opponent"}}, InitialFEN: "startpos", State: lichessapi.GameState{Status: "started"}}
	game, err := newOnlineGame(full, "me", "Me", chess.WOOD)
	if err != nil {
		t.Fatal(err)
	}
	var events []sound.Event
	game.sounds = recordingPlayer(&events)
	game.startSounds()
	state := lichessapi.GameState{Moves: "e2e4", Status: "started"}
	game.apply(onlineResult{event: lichessapi.GameStateEvent{State: state}})
	game.apply(onlineResult{event: lichessapi.GameStateEvent{State: state}})
	game.apply(onlineResult{event: lichessapi.GameStateEvent{State: lichessapi.GameState{Moves: "e2e4", Status: "draw"}}})
	assertSoundEvents(t, events, []sound.Event{sound.Game, sound.Move, sound.Draw, sound.Game})
}

func TestSpeculativeOnlineMoveDoesNotChangeSoundLifecycle(t *testing.T) {
	var events []sound.Event
	game := newLocalGameState(chess.WOOD)
	game.sounds = recordingPlayer(&events)
	game.suppressSoundEvents = true
	from, _ := chess.ParseCoordinate("e2")
	to, _ := chess.ParseCoordinate("e4")
	game.makeMove(chess.NewMove(from, to, chess.DoublePawnPush))
	game.suppressSoundEvents = false
	assertSoundEvents(t, events, nil)
	if game.soundEnded {
		t.Fatal("speculative online move ended sound lifecycle")
	}
}

func TestGameSoundVolumeKeysAdjustAndReportVolume(t *testing.T) {
	game := newLocalGameState(chess.WOOD)
	var changes []int
	game.adjustSoundVolume = func(delta int) int {
		changes = append(changes, delta)
		return 60
	}
	if game.handleKey(keyEvent(tcell.KeyRune, ','), 80, 24) {
		t.Fatal("volume key returned from game")
	}
	if len(changes) != 1 || changes[0] != -10 || game.message != "Sound volume: 60%" {
		t.Errorf("local volume handling = changes %v, message %q", changes, game.message)
	}
	game.terminalTitle = "Draw · ½-½"
	game.handleKey(keyEvent(tcell.KeyRune, '.'), 80, 24)
	if len(changes) != 2 || changes[1] != 10 {
		t.Errorf("terminal local volume handling = %v", changes)
	}

	online := onlineGameState{localGameState: newLocalGameState(chess.WOOD), state: lichessapi.GameState{Status: "draw"}}
	online.adjustSoundVolume = func(delta int) int {
		changes = append(changes, delta)
		return 50
	}
	if online.handleKey(keyEvent(tcell.KeyRune, ','), nil, nil) {
		t.Fatal("online volume key returned from game")
	}
	if len(changes) != 3 || changes[2] != -10 || online.message != "Sound volume: 50%" {
		t.Errorf("online volume handling = changes %v, message %q", changes, online.message)
	}
}

func TestReviewSoundHintShowsVolumeMessage(t *testing.T) {
	if got := reviewSoundHint("Sound volume: 60%"); got != "Sound volume: 60%  ,/.: Sound volume  Esc: return to menu" {
		t.Errorf("reviewSoundHint() = %q", got)
	}
}

func TestGameSoundVolumeHintsRender(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)
	game := newLocalGameState(chess.WOOD)
	drawLocalGame(screen, game)
	screen.Show()
	contents, width, height := screen.GetContents()
	if !strings.Contains(screenText(contents, width, height), ",/.: sound volume") {
		t.Error("active game does not render sound volume hint")
	}

	game.terminalTitle = "Draw · ½-½"
	game.message = "Sound volume: 60%"
	screen.Clear()
	drawLocalGame(screen, game)
	screen.Show()
	contents, width, height = screen.GetContents()
	if !strings.Contains(screenText(contents, width, height), "Sound volume: 60%  ,/.: Sound volume  Esc: return to menu") {
		t.Error("finished game does not render volume message and return hint")
	}
}

func assertSoundEvents(t *testing.T, got, want []sound.Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("sound events = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("sound events = %v, want %v", got, want)
		}
	}
}
