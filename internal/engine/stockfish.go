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

// Score is Stockfish's evaluation of a position. Mate is expressed in moves
// when IsMate is true; otherwise Centipawns is used.
type Score struct {
	Centipawns int
	Mate       int
	IsMate     bool
}

// Analysis is a single principal variation reported by Stockfish.
type Analysis struct {
	Depth    int
	Score    Score
	PV       []string
	BestMove string
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
			if err := c.stopAndDrain(); err != nil {
				return "", err
			}
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

// Analyze searches fen to depth and returns Stockfish's final single-PV
// evaluation. The client remains usable when ctx is cancelled.
func (c *Client) Analyze(ctx context.Context, fen string, depth int) (Analysis, error) {
	return c.AnalyzeWithUpdates(ctx, fen, depth, nil)
}

// AnalyzeWithUpdates searches fen to depth and sends each parsed single-PV
// update to updates when it is ready to receive. Updates are non-blocking and
// may be dropped; the returned Analysis is always the latest result.
func (c *Client) AnalyzeWithUpdates(ctx context.Context, fen string, depth int, updates chan<- Analysis) (Analysis, error) {
	if err := validAnalysisRequest(fen, depth); err != nil {
		return Analysis{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.command("position fen " + fen); err != nil {
		return Analysis{}, err
	}
	if err := c.command("go depth " + strconv.Itoa(depth)); err != nil {
		return Analysis{}, err
	}

	var latest Analysis
	for {
		select {
		case <-ctx.Done():
			if err := c.stopAndDrain(); err != nil {
				return latest, err
			}
			return latest, ctx.Err()
		case line, ok := <-c.lines:
			if !ok {
				return latest, errors.New("Stockfish exited before completing analysis")
			}
			if analysis, ok := parseAnalysis(line); ok {
				latest = analysis
				publishAnalysis(updates, analysis)
				continue
			}
			if move, ok := bestMove(line); ok {
				if move == "(none)" {
					if latest.Depth != 0 {
						return latest, nil
					}
					return latest, errors.New("Stockfish found no move")
				}
				latest.BestMove = move
				publishAnalysis(updates, latest)
				return latest, nil
			}
		}
	}
}

func validAnalysisRequest(fen string, depth int) error {
	if strings.TrimSpace(fen) == "" || strings.ContainsAny(fen, "\r\n") {
		return errors.New("FEN must be a non-empty single line")
	}
	if depth < 1 {
		return errors.New("analysis depth must be positive")
	}
	return nil
}

func parseAnalysis(line string) (Analysis, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 || fields[0] != "info" {
		return Analysis{}, false
	}

	var analysis Analysis
	hasScore := false
	for i := 1; i < len(fields); i++ {
		switch fields[i] {
		case "depth":
			if i+1 < len(fields) {
				analysis.Depth, _ = strconv.Atoi(fields[i+1])
				i++
			}
		case "multipv":
			if i+1 >= len(fields) || fields[i+1] != "1" {
				return Analysis{}, false
			}
			i++
		case "score":
			if i+2 >= len(fields) {
				return Analysis{}, false
			}
			value, err := strconv.Atoi(fields[i+2])
			if err != nil {
				return Analysis{}, false
			}
			switch fields[i+1] {
			case "cp":
				analysis.Score.Centipawns = value
			case "mate":
				analysis.Score.Mate = value
				analysis.Score.IsMate = true
			default:
				return Analysis{}, false
			}
			hasScore = true
			i += 2
		case "pv":
			analysis.PV = append([]string(nil), fields[i+1:]...)
			i = len(fields)
		}
	}
	if analysis.Depth == 0 || !hasScore || len(analysis.PV) == 0 {
		return Analysis{}, false
	}
	return analysis, true
}

func publishAnalysis(updates chan<- Analysis, analysis Analysis) {
	if updates == nil {
		return
	}
	select {
	case updates <- analysis:
	default:
	}
}

func bestMove(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[0] != "bestmove" {
		return "", false
	}
	return fields[1], true
}

// stopAndDrain waits for the command's required bestmove response so a later
// request cannot consume output from the cancelled search.
func (c *Client) stopAndDrain() error {
	if err := c.command("stop"); err != nil {
		return err
	}
	for line := range c.lines {
		if _, ok := bestMove(line); ok {
			return nil
		}
	}
	return errors.New("Stockfish exited while stopping search")
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
