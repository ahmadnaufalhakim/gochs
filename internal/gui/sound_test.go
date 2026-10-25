package gui

import (
	"testing"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
	"github.com/ahmadnaufalhakim/gochs/internal/sound"
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
