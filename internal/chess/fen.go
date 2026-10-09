package chess

var pieceToFENSymbol = map[Piece]rune{
	{Color: White, Type: Pawn}:   'P',
	{Color: White, Type: Knight}: 'N',
	{Color: White, Type: Bishop}: 'B',
	{Color: White, Type: Rook}:   'R',
	{Color: White, Type: Queen}:  'Q',
	{Color: White, Type: King}:   'K',

	{Color: Black, Type: Pawn}:   'p',
	{Color: Black, Type: Knight}: 'n',
	{Color: Black, Type: Bishop}: 'b',
	{Color: Black, Type: Rook}:   'r',
	{Color: Black, Type: Queen}:  'q',
	{Color: Black, Type: King}:   'k',
}

var castlingRightFENSymbols = [PieceColorCount][CastlingSideCount]rune{
	White: {KingSide: 'K', QueenSide: 'Q'},
	Black: {KingSide: 'k', QueenSide: 'q'},
}
