// Package sound provides game sound events and Linux playback.
package sound

import (
	"fmt"
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

const (
	DefaultVolume = 70
	MaxVolume     = 100
)

// Controller applies a shared session volume while retaining the event-based
// Player interface used by game code and tests.
type Controller struct {
	player         Player
	volume         int
	previousVolume int
}

func NewController(player Player) *Controller {
	return &Controller{player: player, volume: DefaultVolume, previousVolume: DefaultVolume}
}

func (c *Controller) Volume() int { return c.volume }

func (c *Controller) Adjust(delta int) int {
	c.volume = min(MaxVolume, max(0, c.volume+delta))
	if c.volume > 0 {
		c.previousVolume = c.volume
	}
	return c.volume
}

func (c *Controller) ToggleMute() int {
	if c.volume > 0 {
		c.previousVolume = c.volume
		c.volume = 0
		return c.volume
	}
	c.volume = c.previousVolume
	if c.volume == 0 {
		c.volume = DefaultVolume
	}
	return c.volume
}

func (c *Controller) Play(event Event) {
	if c.volume == 0 || c.player == nil {
		return
	}
	if player, ok := c.player.(interface{ PlayVolume(Event, int) }); ok {
		player.PlayVolume(event, c.volume)
		return
	}
	c.player.Play(event)
}

type paplayPlayer struct {
	once sync.Once
	dir  string
	cmd  string
	args []string
}

// NewPlayer returns a non-blocking player. Missing audio support is ignored.
func NewPlayer() Player { return &paplayPlayer{} }

func (p *paplayPlayer) Play(event Event) {
	p.PlayVolume(event, MaxVolume)
}

func (p *paplayPlayer) PlayVolume(event Event, volume int) {
	p.once.Do(p.materialize)
	if p.dir == "" {
		return
	}
	path := filepath.Join(p.dir, string(event)+".wav")
	if _, err := os.Stat(path); err != nil {
		return
	}
	args := append([]string(nil), p.args...)
	if p.cmd == "paplay" {
		args = append(args, "--volume", fmt.Sprintf("%d", volume*65536/MaxVolume))
	}
	command := exec.Command(p.cmd, append(args, path)...)
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
