package chess

import (
	"fmt"
	"strings"
)

// ParseUCI returns the legal move represented by uci in the current position.
func (b Board) ParseUCI(uci string) (Move, error) {
	if len(uci) != 4 && len(uci) != 5 {
		return Move(0), fmt.Errorf("UCI move %q must contain four or five characters", uci)
	}
	if _, err := ParseCoordinate(uci[:2]); err != nil {
		return Move(0), fmt.Errorf("invalid UCI source: %w", err)
	}
	if _, err := ParseCoordinate(uci[2:4]); err != nil {
		return Move(0), fmt.Errorf("invalid UCI target: %w", err)
	}
	if len(uci) == 5 && !strings.ContainsRune("nbrq", rune(uci[4])) {
		return Move(0), fmt.Errorf("invalid UCI promotion piece %q", uci[4])
	}
	for _, move := range GenerateLegalMoves(b) {
		if move.UCI() == uci {
			return move, nil
		}
	}
	return Move(0), fmt.Errorf("%s is an illegal move", uci)
}

// ParseUCIMoves resolves a space-delimited sequence of legal UCI moves from
// board, applying each move before resolving the next one.
func ParseUCIMoves(board Board, sequence string) ([]Move, error) {
	ucis := strings.Fields(sequence)
	moves := make([]Move, 0, len(ucis))
	for index, uci := range ucis {
		move, err := board.ParseUCI(uci)
		if err != nil {
			return nil, fmt.Errorf("move %d: %w", index+1, err)
		}
		if err := board.MakeMove(move); err != nil {
			return nil, fmt.Errorf("move %d: %w", index+1, err)
		}
		moves = append(moves, move)
	}
	return moves, nil
}
