package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/engine"
)

func main() {
	color := flag.String("color", "random", "white, black, or random")
	difficulty := flag.Int("difficulty", 4, "Stockfish difficulty from 1 to 8")
	minutes := flag.Int("minutes", 10, "minutes per side, or 0 for unlimited")
	increment := flag.Int("increment", 5, "increment seconds")
	path := flag.String("stockfish-path", "", "path to Stockfish executable")
	flag.Parse()

	if *minutes < 0 || *increment < 0 {
		fatal("minutes and increment must not be negative")
	}
	computerColor, err := computerSide(*color)
	if err != nil {
		fatal(err.Error())
	}
	client, err := engine.Start(context.Background(), engine.Config{Path: *path, Difficulty: *difficulty})
	if err != nil {
		fatal("start Stockfish: " + err.Error())
	}
	defer client.Close()

	var board chess.Board
	board.Reset()
	clocks := [chess.PieceColorCount]time.Duration{}
	if *minutes > 0 {
		clocks[chess.White] = time.Duration(*minutes) * time.Minute
		clocks[chess.Black] = time.Duration(*minutes) * time.Minute
	}
	moves := []string{}
	input := bufio.NewScanner(os.Stdin)
	turnStarted := time.Now()

	for {
		printBoard(board, clocks)
		if len(chess.GenerateLegalMoves(board)) == 0 {
			fmt.Println("Game over")
			return
		}
		if board.ColorToMove == computerColor {
			policy, err := engine.DifficultyPolicy(*difficulty)
			if err != nil {
				fatal(err.Error())
			}
			search := engine.Search{WhiteTime: clocks[chess.White], BlackTime: clocks[chess.Black], WhiteIncrement: time.Duration(*increment) * time.Second, BlackIncrement: time.Duration(*increment) * time.Second, MaxMoveTime: policy.MoveTime}
			if *minutes == 0 {
				search.MoveTime = policy.MoveTime
			}
			uci, err := client.BestMove(context.Background(), moves, search)
			if err != nil {
				fatal("Stockfish: " + err.Error())
			}
			legalMoves := chess.GenerateLegalMoves(board)
			legalUCIs := make([]string, len(legalMoves))
			for index, legalMove := range legalMoves {
				legalUCIs[index] = legalMove.UCI()
			}
			move, err := board.ParseUCI(engine.SelectMove(policy, uci, legalUCIs, rand.Intn))
			if err != nil {
				fatal(err.Error())
			}
			if *minutes > 0 {
				clocks[computerColor] = max(0, clocks[computerColor]-time.Since(turnStarted)) + time.Duration(*increment)*time.Second
			}
			fmt.Println("Stockfish:", uci)
			_ = board.MakeMove(move)
			moves = append(moves, uci)
			turnStarted = time.Now()
			continue
		}

		fmt.Print("Your move (UCI or SAN, q to quit): ")
		if !input.Scan() {
			return
		}
		text := strings.TrimSpace(input.Text())
		if text == "q" || text == "quit" {
			return
		}
		move, err := board.ParseUCI(strings.ToLower(text))
		if err != nil {
			move, err = board.ParseSAN(text)
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		if *minutes > 0 {
			elapsed := time.Since(turnStarted)
			if elapsed >= clocks[board.ColorToMove] {
				fmt.Println("You ran out of time")
				return
			}
			clocks[board.ColorToMove] -= elapsed
			clocks[board.ColorToMove] += time.Duration(*increment) * time.Second
		}
		moves = append(moves, move.UCI())
		_ = board.MakeMove(move)
		turnStarted = time.Now()
	}
}

func computerSide(value string) (chess.PieceColor, error) {
	switch value {
	case "white":
		return chess.Black, nil
	case "black":
		return chess.White, nil
	case "random":
		if rand.Intn(2) == 0 {
			return chess.White, nil
		}
		return chess.Black, nil
	default:
		return chess.White, fmt.Errorf("color must be white, black, or random")
	}
}

func printBoard(board chess.Board, clocks [chess.PieceColorCount]time.Duration) {
	fmt.Printf("\nWhite %s  Black %s\n", clocks[chess.White].Round(time.Second), clocks[chess.Black].Round(time.Second))
	for rank := 7; rank >= 0; rank-- {
		fmt.Printf("%d ", rank+1)
		for file := 0; file < 8; file++ {
			piece := board.PieceAt(chess.Square(file + rank*8))
			label := pieceSymbol(piece)
			fmt.Print(label, " ")
		}
		fmt.Println()
	}
	fmt.Println("  a b c d e f g h")
}

func pieceSymbol(piece chess.Piece) string {
	if piece.Type == chess.PieceNone {
		return "."
	}
	label := map[chess.PieceType]string{
		chess.Pawn:   "P",
		chess.Knight: "N",
		chess.Bishop: "B",
		chess.Rook:   "R",
		chess.Queen:  "Q",
		chess.King:   "K",
	}[piece.Type]
	if piece.Color == chess.Black {
		return strings.ToLower(label)
	}
	return label
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
