package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Search struct {
	WhiteTime      time.Duration
	BlackTime      time.Duration
	WhiteIncrement time.Duration
	BlackIncrement time.Duration
	MoveTime       time.Duration
	MaxMoveTime    time.Duration
}

type Config struct {
	Path       string
	Difficulty int
}

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan string
	done   chan error
	mu     sync.Mutex
	closed bool
}

type Difficulty struct {
	SkillLevel     int
	MoveTime       time.Duration
	BlunderPercent int
}

func FindStockfish() (string, error) {
	if path := os.Getenv("GOCHS_STOCKFISH_PATH"); path != "" {
		return executable(path)
	}
	if path, err := exec.LookPath("stockfish"); err == nil {
		return path, nil
	}
	return executable("/usr/games/stockfish")
}

func executable(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("%s is not executable", filepath.Clean(path))
	}
	return path, nil
}

func DifficultySkill(level int) (int, error) {
	policy, err := DifficultyPolicy(level)
	return policy.SkillLevel, err
}

func UnlimitedMoveTime(level int) (time.Duration, error) {
	policy, err := DifficultyPolicy(level)
	return policy.MoveTime, err
}

func DifficultyPolicy(level int) (Difficulty, error) {
	if level < 1 || level > 8 {
		return Difficulty{}, fmt.Errorf("difficulty must be between 1 and 8")
	}
	return []Difficulty{
		{SkillLevel: 0, MoveTime: 150 * time.Millisecond, BlunderPercent: 65},
		{SkillLevel: 3, MoveTime: 300 * time.Millisecond, BlunderPercent: 52},
		{SkillLevel: 6, MoveTime: 450 * time.Millisecond, BlunderPercent: 40},
		{SkillLevel: 9, MoveTime: 600 * time.Millisecond, BlunderPercent: 27},
		{SkillLevel: 11, MoveTime: 750 * time.Millisecond, BlunderPercent: 16},
		{SkillLevel: 14, MoveTime: 900 * time.Millisecond, BlunderPercent: 9},
		{SkillLevel: 17, MoveTime: 1050 * time.Millisecond, BlunderPercent: 3},
		{SkillLevel: 20, MoveTime: 1200 * time.Millisecond},
	}[level-1], nil
}

// SelectMove returns the engine move unless this level deliberately chooses a
// different legal move. randomInt must return a value in [0, n).
func SelectMove(policy Difficulty, best string, legal []string, randomInt func(int) int) string {
	if policy.BlunderPercent == 0 || len(legal) < 2 || randomInt(100) >= policy.BlunderPercent {
		return best
	}

	alternatives := make([]string, 0, len(legal)-1)
	for _, move := range legal {
		if move != best {
			alternatives = append(alternatives, move)
		}
	}
	if len(alternatives) == 0 {
		return best
	}
	return alternatives[randomInt(len(alternatives))]
}

func Start(ctx context.Context, config Config) (*Client, error) {
	if config.Path == "" {
		path, err := FindStockfish()
		if err != nil {
			return nil, fmt.Errorf("find Stockfish: %w", err)
		}
		config.Path = path
	}
	policy, err := DifficultyPolicy(config.Difficulty)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, config.Path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open Stockfish stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open Stockfish stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start Stockfish: %w", err)
	}

	client := &Client{
		cmd:   cmd,
		stdin: stdin,
		lines: make(chan string, 128),
		done:  make(chan error, 1),
	}
	go client.readOutput(stdout)
	go func() { client.done <- cmd.Wait() }()

	if err := client.command("uci"); err != nil {
		client.Close()
		return nil, err
	}
	if err := client.waitFor(ctx, "uciok"); err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize Stockfish: %w", err)
	}
	if err := client.command("setoption name Skill Level value " + strconv.Itoa(policy.SkillLevel)); err != nil {
		client.Close()
		return nil, err
	}
	if err := client.ready(ctx); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) readOutput(output io.Reader) {
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		c.lines <- strings.TrimSpace(scanner.Text())
	}
	close(c.lines)
}

func (c *Client) BestMove(ctx context.Context, moves []string, search Search) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	position := "position startpos"
	if len(moves) != 0 {
		position += " moves " + strings.Join(moves, " ")
	}
	if err := c.command(position); err != nil {
		return "", err
	}
	if search.MoveTime > 0 {
		if err := c.command("go movetime " + strconv.Itoa(int(search.MoveTime/time.Millisecond))); err != nil {
			return "", err
		}
	} else {
		command := fmt.Sprintf("go wtime %d btime %d winc %d binc %d",
			search.WhiteTime/time.Millisecond,
			search.BlackTime/time.Millisecond,
			search.WhiteIncrement/time.Millisecond,
			search.BlackIncrement/time.Millisecond,
		)
		if search.MaxMoveTime > 0 {
			command += " movetime " + strconv.Itoa(int(search.MaxMoveTime/time.Millisecond))
		}
		if err := c.command(command); err != nil {
			return "", err
		}
	}

	for {
		select {
		case <-ctx.Done():
			_ = c.command("stop")
			return "", ctx.Err()
		case line, ok := <-c.lines:
			if !ok {
				return "", errors.New("Stockfish exited before returning a move")
			}
			if fields := strings.Fields(line); len(fields) >= 2 && fields[0] == "bestmove" {
				if fields[1] == "(none)" {
					return "", errors.New("Stockfish found no move")
				}
				return fields[1], nil
			}
		}
	}
}

func (c *Client) ready(ctx context.Context) error {
	if err := c.command("isready"); err != nil {
		return err
	}
	return c.waitFor(ctx, "readyok")
}

func (c *Client) waitFor(ctx context.Context, expected string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, ok := <-c.lines:
			if !ok {
				return errors.New("Stockfish exited during initialization")
			}
			if line == expected {
				return nil
			}
		}
	}
}

func (c *Client) command(command string) error {
	if c.closed {
		return errors.New("Stockfish is closed")
	}
	_, err := io.WriteString(c.stdin, command+"\n")
	return err
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	_, _ = io.WriteString(c.stdin, "quit\n")
	_ = c.stdin.Close()
	return <-c.done
}
