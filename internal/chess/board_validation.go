package chess

import "errors"

const (
	fileA Bitboard = 0x0101010101010101
	fileB Bitboard = 0x0202020202020202
	fileC Bitboard = 0x0404040404040404
	fileD Bitboard = 0x0808080808080808
	fileE Bitboard = 0x1010101010101010
	fileF Bitboard = 0x2020202020202020
	fileG Bitboard = 0x4040404040404040
	fileH Bitboard = 0x8080808080808080

	rank1 Bitboard = 0x00000000000000FF
	rank2 Bitboard = 0x000000000000FF00
	rank3 Bitboard = 0x0000000000FF0000
	rank4 Bitboard = 0x00000000FF000000
	rank5 Bitboard = 0x000000FF00000000
	rank6 Bitboard = 0x0000FF0000000000
	rank7 Bitboard = 0x00FF000000000000
	rank8 Bitboard = 0xFF00000000000000
)

func (b *Board) validatePawnPositions() error {
	for color := range PieceColorCount {
		if b.Pieces[color][Pawn]&(rank1|rank8) != 0 {
			return errors.New("pawns cannot be on the first or eighth rank")
		}
	}

	return nil
}
