package chess

type Move struct {
	From      Square
	To        Square
	Promotion PieceType
}

func GeneratePawnMoves(b Board) Bitboard {
	unoccupied := ^b.Occupied()
	pawns := b.Pieces[b.ColorToMove][Pawn]

	var pawnSinglePushFn func(pawns Bitboard) Bitboard
	var doublePushDestinationRank Bitboard
	switch b.ColorToMove {
	case White:
		pawnSinglePushFn = func(pawns Bitboard) Bitboard {
			return (pawns << 8)
		}
		doublePushDestinationRank = rank4
	case Black:
		pawnSinglePushFn = func(pawns Bitboard) Bitboard {
			return (pawns >> 8)
		}
		doublePushDestinationRank = rank5
	}

	pawnSinglePushes := pawnSinglePushFn(pawns) & unoccupied
	pawnDoublePushes := (pawnSinglePushes << 8) & unoccupied & doublePushDestinationRank

	return pawnSinglePushes | pawnDoublePushes
}
