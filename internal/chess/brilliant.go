package chess

type brilliantPiece struct {
	square Square
	piece  Piece
}

type brilliantAttack struct {
	move  Move
	piece Piece
}

var brilliantPieceValues = [...]int{
	PieceNone: 0,
	Pawn:      1,
	Knight:    3,
	Bishop:    3,
	Rook:      5,
	Queen:     9,
	King:      100,
}

// IsBrilliantSacrifice reports whether move creates the tactical, apparently
// unsafe material situation used by WintrChess's brilliant-move heuristic.
// Engine evaluation gates belong to the caller because this check is purely
// position based.
func IsBrilliantSacrifice(before Board, move Move) bool {
	movingPiece := before.PieceAt(move.From())
	if movingPiece.Type == PieceNone || isQueenPromotion(move) {
		return false
	}

	after := before
	if err := after.MakeMove(move); err != nil {
		return false
	}
	captured := before.PieceAt(move.To())
	if move.Flag() == EnPassant {
		captured = Piece{Color: movingPiece.Color.Opponent(), Type: Pawn}
	}
	if !brilliantCouldHaveUnsafePiece(after, movingPiece.Color, captured.Type) {
		return false
	}

	previousUnsafe := brilliantUnsafePieces(before, movingPiece.Color, PieceNone)
	unsafe := brilliantUnsafePieces(after, movingPiece.Color, captured.Type)
	if !after.IsColorInCheck(after.ColorToMove) && len(unsafe) < len(previousUnsafe) {
		return false
	}

	for _, threatened := range unsafe {
		if !brilliantHasDangerLevels(after, threatened, brilliantAttacks(after, threatened, false)) {
			goto dangerFound
		}
	}
	return false

dangerFound:
	previousTrapped := brilliantTrappedPieces(before, previousUnsafe)
	trapped := brilliantTrappedPieces(after, unsafe)
	if len(trapped) == len(unsafe) || brilliantContainsSquare(previousTrapped, move.From()) || len(trapped) < len(previousTrapped) {
		return false
	}

	return len(unsafe) > 0
}

func brilliantCouldHaveUnsafePiece(board Board, color PieceColor, capturedType PieceType) bool {
	for square := a1; square <= h8; square++ {
		piece := board.PieceAt(square)
		if piece.Color == color && piece.Type != PieceNone && piece.Type != Pawn && piece.Type != King && brilliantPieceValues[piece.Type] > brilliantPieceValues[capturedType] && board.IsSquareAttackedBy(square, color.Opponent()) {
			return true
		}
	}
	return false
}

func isQueenPromotion(move Move) bool {
	return move.Flag() == PromoteQueen || move.Flag() == PromoteCaptureQueen
}

func brilliantUnsafePieces(board Board, color PieceColor, capturedType PieceType) []brilliantPiece {
	unsafe := make([]brilliantPiece, 0)
	for square := a1; square <= h8; square++ {
		piece := board.PieceAt(square)
		if piece.Color != color || piece.Type == PieceNone || piece.Type == Pawn || piece.Type == King || brilliantPieceValues[piece.Type] <= brilliantPieceValues[capturedType] {
			continue
		}
		candidate := brilliantPiece{square: square, piece: piece}
		if !brilliantPieceSafe(board, candidate, capturedType) {
			unsafe = append(unsafe, candidate)
		}
	}
	return unsafe
}

func brilliantPieceSafe(board Board, target brilliantPiece, capturedType PieceType) bool {
	direct := brilliantAttacks(board, target, false)
	attackers := brilliantAttacks(board, target, true)
	defenders := brilliantDefenders(board, target)
	if target.piece.Type == Rook && capturedType == Knight && len(attackers) == 1 && len(defenders) > 0 && attackers[0].piece.Type == Knight {
		return true
	}
	for _, attacker := range direct {
		if brilliantPieceValues[attacker.piece.Type] < brilliantPieceValues[target.piece.Type] {
			return false
		}
	}
	if len(attackers) <= len(defenders) {
		return true
	}
	if len(direct) == 0 {
		return true
	}
	lowest := direct[0]
	for _, attacker := range direct[1:] {
		if brilliantPieceValues[attacker.piece.Type] < brilliantPieceValues[lowest.piece.Type] {
			lowest = attacker
		}
	}
	if brilliantPieceValues[target.piece.Type] < brilliantPieceValues[lowest.piece.Type] {
		for _, defender := range defenders {
			if brilliantPieceValues[defender.piece.Type] < brilliantPieceValues[lowest.piece.Type] {
				return true
			}
		}
	}
	for _, defender := range defenders {
		if defender.piece.Type == Pawn {
			return true
		}
	}
	return false
}

func brilliantAttacks(board Board, target brilliantPiece, transitive bool) []brilliantAttack {
	attacks := brilliantDirectAttacks(board, target)
	if !transitive {
		return attacks
	}
	type frontierEntry struct {
		board  Board
		square Square
		piece  Piece
	}
	frontier := make([]frontierEntry, 0, len(attacks))
	for _, attack := range attacks {
		frontier = append(frontier, frontierEntry{board: board, square: attack.move.From(), piece: attack.piece})
	}
	for len(frontier) > 0 {
		entry := frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		if entry.piece.Type == King {
			continue
		}
		old := brilliantDirectAttacks(entry.board, target)
		revealedBoard := entry.board
		revealedBoard.ClearSquare(entry.square)
		for _, revealed := range brilliantDirectAttacks(revealedBoard, target) {
			if revealed.move.From() == entry.square || brilliantHasAttack(old, revealed) {
				continue
			}
			attacks = append(attacks, revealed)
			frontier = append(frontier, frontierEntry{board: revealedBoard, square: revealed.move.From(), piece: revealed.piece})
		}
	}
	return attacks
}

func brilliantDirectAttacks(board Board, target brilliantPiece) []brilliantAttack {
	attackerColor := target.piece.Color.Opponent()
	attackBoard := board
	attackBoard.ColorToMove = attackerColor
	attacks := make([]brilliantAttack, 0)
	for _, move := range GenerateLegalMoves(attackBoard) {
		if move.Flag() != EnPassant && move.To() == target.square && attackBoard.PieceAt(move.To()).Color == target.piece.Color {
			attacks = append(attacks, brilliantAttack{move: move, piece: attackBoard.PieceAt(move.From())})
		}
	}
	for square := a1; square <= h8; square++ {
		piece := board.PieceAt(square)
		if piece == (Piece{Color: attackerColor, Type: King}) && GenerateKingAttacks(board, attackerColor)&target.square.Mask() != 0 {
			attack := brilliantAttack{move: NewMove(square, target.square, Capture), piece: piece}
			if !brilliantHasAttack(attacks, attack) {
				attacks = append(attacks, attack)
			}
		}
	}
	return attacks
}

func brilliantDefenders(board Board, target brilliantPiece) []brilliantAttack {
	attackers := brilliantAttacks(board, target, false)
	var smallest []brilliantAttack
	for _, attacker := range attackers {
		captureBoard := board
		captureBoard.ColorToMove = attacker.piece.Color
		captureBoard.applyMove(attacker.move)
		recaptured := captureBoard.PieceAt(attacker.move.To())
		recaptures := brilliantAttacks(captureBoard, brilliantPiece{square: attacker.move.To(), piece: recaptured}, true)
		if smallest == nil || len(recaptures) < len(smallest) {
			smallest = recaptures
		}
	}
	if smallest != nil {
		return smallest
	}
	flipped := board
	flipped.SetPieceAt(target.square, Piece{Color: target.piece.Color.Opponent(), Type: target.piece.Type})
	return brilliantAttacks(flipped, brilliantPiece{square: target.square, piece: flipped.PieceAt(target.square)}, true)
}

func brilliantHasDangerLevels(board Board, threatened brilliantPiece, attacks []brilliantAttack) bool {
	for _, attack := range attacks {
		if !brilliantMoveLeavesGreaterThreat(board, threatened, attack.move) {
			return false
		}
	}
	return true
}

func brilliantMoveLeavesGreaterThreat(board Board, threatened brilliantPiece, move Move) bool {
	actionBoard := board
	moving := actionBoard.PieceAt(move.From())
	actionBoard.ColorToMove = moving.Color
	captured := actionBoard.PieceAt(move.To())
	actionBoard.applyMove(move)
	if len(brilliantRelativeUnsafeAttacks(actionBoard, threatened, moving.Color, captured.Type)) > 0 {
		return true
	}
	return brilliantHasCheckmatingMove(actionBoard, threatened.piece.Type)
}

func brilliantMoveCreatesGreaterThreat(board Board, threatened brilliantPiece, move Move) bool {
	moving := board.PieceAt(move.From())
	previous := brilliantRelativeUnsafeAttacks(board, threatened, moving.Color, PieceNone)
	actionBoard := board
	actionBoard.ColorToMove = moving.Color
	captured := actionBoard.PieceAt(move.To())
	actionBoard.applyMove(move)
	for _, attack := range brilliantRelativeUnsafeAttacks(actionBoard, threatened, moving.Color, captured.Type) {
		if !brilliantHasAttack(previous, attack) {
			return true
		}
	}
	return brilliantHasCheckmatingMove(actionBoard, threatened.piece.Type)
}

func brilliantRelativeUnsafeAttacks(board Board, threatened brilliantPiece, color PieceColor, capturedType PieceType) []brilliantAttack {
	attacks := make([]brilliantAttack, 0)
	for _, unsafe := range brilliantUnsafePieces(board, color, capturedType) {
		if unsafe.square != threatened.square && brilliantPieceValues[unsafe.piece.Type] >= brilliantPieceValues[threatened.piece.Type] {
			attacks = append(attacks, brilliantAttacks(board, unsafe, false)...)
		}
	}
	return attacks
}

func brilliantHasCheckmatingMove(board Board, threatenedType PieceType) bool {
	if brilliantPieceValues[threatenedType] >= brilliantPieceValues[Queen] {
		return false
	}
	for _, move := range GenerateLegalMoves(board) {
		next := board
		next.applyMove(move)
		if len(GenerateLegalMoves(next)) == 0 && next.IsColorInCheck(next.ColorToMove) {
			return true
		}
	}
	return false
}

func brilliantTrappedPieces(board Board, pieces []brilliantPiece) []brilliantPiece {
	trapped := make([]brilliantPiece, 0, len(pieces))
	for _, piece := range pieces {
		if brilliantPieceTrapped(board, piece) {
			trapped = append(trapped, piece)
		}
	}
	return trapped
}

func brilliantPieceTrapped(board Board, target brilliantPiece) bool {
	if brilliantPieceSafe(board, target, PieceNone) {
		return false
	}
	escapeBoard := board
	escapeBoard.ColorToMove = target.piece.Color
	for _, move := range GenerateLegalMoves(escapeBoard) {
		if move.From() != target.square {
			continue
		}
		if !brilliantMoveCreatesGreaterThreat(escapeBoard, target, move) {
			next := escapeBoard
			captured := next.PieceAt(move.To())
			next.applyMove(move)
			if brilliantPieceSafe(next, brilliantPiece{square: move.To(), piece: next.PieceAt(move.To())}, captured.Type) {
				return false
			}
		}
	}
	return true
}

func brilliantHasAttack(attacks []brilliantAttack, target brilliantAttack) bool {
	for _, attack := range attacks {
		if attack.move == target.move && attack.piece == target.piece {
			return true
		}
	}
	return false
}

func brilliantContainsSquare(pieces []brilliantPiece, square Square) bool {
	for _, piece := range pieces {
		if piece.square == square {
			return true
		}
	}
	return false
}
