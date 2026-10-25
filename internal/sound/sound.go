// Package sound provides game sound events and Linux playback.
package sound

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/ahmadnaufalhakim/gochs/data/sounds"
)

type Event string

const (
	Game      Event = "game"
	Move      Event = "move"
	Capture   Event = "capture"
	Check     Event = "check"
	Checkmate Event = "checkmate"
	Draw      Event = "draw"
	Stalemate Event = "stalemate"
	LowTime   Event = "low-time"
)

type Player interface {
	Play(Event)
}

type PlayerFunc func(Event)

func (f PlayerFunc) Play(event Event) { f(event) }

type paplayPlayer struct {
	once sync.Once
	dir  string
	cmd  string
	args []string
}

// NewPlayer returns a non-blocking player. Missing audio support is ignored.
func NewPlayer() Player { return &paplayPlayer{} }

func (p *paplayPlayer) Play(event Event) {
	p.once.Do(p.materialize)
	if p.dir == "" {
		return
	}
	path := filepath.Join(p.dir, string(event)+".wav")
	if _, err := os.Stat(path); err != nil {
		return
	}
	command := exec.Command(p.cmd, append(p.args, path)...)
	if command.Start() == nil {
		go func() { _ = command.Wait() }()
	}
}

func (p *paplayPlayer) materialize() {
	if _, err := exec.LookPath("paplay"); err == nil {
		p.cmd = "paplay"
		// PulseAudio defaults to a relatively large buffer. Sound effects need
		// short latency so rapid replay navigation stays responsive.
		p.args = []string{"--latency-msec=20", "--process-time-msec=20"}
	} else if _, err := exec.LookPath("aplay"); err == nil {
		p.cmd = "aplay"
	} else {
		return
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	dir = filepath.Join(dir, "gochs", "sounds")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	for _, event := range []Event{Game, Move, Capture, Check, Checkmate, Draw, Stalemate, LowTime} {
		name := string(event) + ".wav"
		contents, err := sounds.Files.ReadFile(name)
		if err != nil {
			return
		}
		if err := os.WriteFile(filepath.Join(dir, name), contents, 0o600); err != nil {
			return
		}
	}
	p.dir = dir
}
