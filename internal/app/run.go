package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
)

func Run() {
	reader := bufio.NewReader(os.Stdin)
	renderer := chess.Renderer{
		Theme:       chess.WOOD,
		Perspective: chess.White,
	}

	var board chess.Board
	board.Reset()

	fmt.Println("gochs")
	fmt.Println("Enter lowercase coordinates when prompted, or type quit to exit.")

	for {
		fmt.Println()
		renderer.Print(board)
		fen, err := board.FEN()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fen)
		}

		legalMoves := chess.GenerateLegalMoves(board)
		if len(legalMoves) == 0 {
			if board.IsColorInCheck(board.ColorToMove) {
				fmt.Printf("Checkmate. %s wins.\n", board.ColorToMove.Opponent())
			} else {
				fmt.Println("Stalemate.")
			}
			return
		}

		if board.IsColorInCheck(board.ColorToMove) {
			fmt.Println("Check.")
		}

		from, ok := readSquare(reader, fmt.Sprintf("%s to move. From: ", board.ColorToMove))
		if !ok {
			return
		}

		to, ok := readSquare(reader, "To: ")
		if !ok {
			return
		}

		move, err := selectMove(reader, legalMoves, from, to)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if err := board.MakeMove(move); err != nil {
			fmt.Println(err)
		}
	}
}

func readSquare(reader *bufio.Reader, prompt string) (chess.Square, bool) {
	for {
		fmt.Print(prompt)

		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Printf("Could not read input: %v\n", err)
			return chess.Square(0), false
		}

		input = strings.TrimSpace(input)
		if input == "quit" || (errors.Is(err, io.EOF) && input == "") {
			return chess.Square(0), false
		}

		square, parseErr := chess.ParseCoordinate(input)
		if parseErr == nil {
			return square, true
		}

		fmt.Printf("Invalid square %q: %v\n", input, parseErr)
		if errors.Is(err, io.EOF) {
			return chess.Square(0), false
		}
	}
}

func selectMove(reader *bufio.Reader, legalMoves []chess.Move, from, to chess.Square) (chess.Move, error) {
	candidates := make([]chess.Move, 0, 4)
	for _, move := range legalMoves {
		if move.From() == from && move.To() == to {
			candidates = append(candidates, move)
		}
	}

	switch len(candidates) {
	case 0:
		return chess.Move(0), fmt.Errorf("%s-%s is not a legal move", from, to)
	case 1:
		return candidates[0], nil
	}

	for {
		fmt.Print("Promote to [n/b/r/q]: ")
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return chess.Move(0), fmt.Errorf("could not read promotion: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "quit" || (errors.Is(err, io.EOF) && input == "") {
			return chess.Move(0), errors.New("promotion cancelled")
		}

		for _, move := range candidates {
			if promotionInputMatches(input, move.Flag()) {
				return move, nil
			}
		}

		fmt.Println("Invalid promotion piece. Enter n, b, r, or q.")
	}
}

func promotionInputMatches(input string, flag chess.MoveFlag) bool {
	switch input {
	case "n":
		return flag == chess.PromoteKnight || flag == chess.PromoteCaptureKnight
	case "b":
		return flag == chess.PromoteBishop || flag == chess.PromoteCaptureBishop
	case "r":
		return flag == chess.PromoteRook || flag == chess.PromoteCaptureRook
	case "q":
		return flag == chess.PromoteQueen || flag == chess.PromoteCaptureQueen
	default:
		return false
	}
}
