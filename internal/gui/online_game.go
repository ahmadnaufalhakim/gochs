package gui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
	"github.com/ahmadnaufalhakim/gochs/internal/pgn"
	"github.com/ahmadnaufalhakim/gochs/internal/sound"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

type onlineResult struct {
	gameStart              *lichessapi.Game
	event                  lichessapi.GameEvent
	err                    error
	action                 string
	fatal                  bool
	ratingChanges          *lichessapi.RatingChanges
	exportedPGN            []byte
	includeClocks          bool
	seek                   lichessapi.Seek
	seekGeneration         uint64
	accountID, accountName string
}

type onlineGameState struct {
	localGameState
	gameID          string
	createdAt       int64
	seek            lichessapi.Seek
	me              chess.PieceColor
	white, black    lichessapi.Player
	state           lichessapi.GameState
	pendingMove     bool
	clockUpdated    time.Time
	requestContext  context.Context
	pendingDraw     bool
	pendingTakeback bool
	ratingChanges   *lichessapi.RatingChanges
	ratingLoading   bool
	ratingChecked   bool
	exportingPGN    bool
	fatal           bool
	offerAnimation  uint8
	hoveredDraw     bool
	hoveredTakeback bool
}

func (g *onlineGameState) loadRatingChanges(service OnlineBoardService, results chan<- onlineResult) {
	if !g.isFinished() || g.ratingLoading || g.ratingChecked {
		return
	}
	g.ratingLoading = true
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	gameID := g.gameID
	go func() {
		defer cancel()
		changes, err := service.RatingChanges(ctx, gameID)
		results <- onlineResult{action: "rating changes", ratingChanges: &changes, err: err}
	}()
}

func startOnlineSeek(menu *menuState, service OnlineBoardService, results chan<- onlineResult, cancel *context.CancelFunc) {
	menu.onlineSeekGeneration++
	generation := menu.onlineSeekGeneration
	if *cancel != nil {
		(*cancel)()
	}
	ctx, newCancel := context.WithCancel(context.Background())
	*cancel = newCancel
	menu.page = lichessLoading
	menu.onlineSeeking = true
	gameType := "Casual"
	if menu.onlineRated {
		gameType = "Rated"
	}
	menu.lichessMessage = gameType + " game: waiting for an opponent... Esc to cancel"
	seek := lichessapi.Seek{TimeSeconds: int(menu.timeControl.initial.Seconds()), IncrementSeconds: int(menu.timeControl.increment.Seconds()), Color: menu.onlineColor, Rated: menu.onlineRated}
	go func() {
		events, errors := service.StreamEvents(ctx)
		go func() {
			if err := service.CreateSeek(ctx, seek); err != nil && ctx.Err() == nil {
				sendOnlineResult(ctx, results, onlineResult{err: err, seekGeneration: generation})
			}
		}()
		for events != nil || errors != nil {
			select {
			case event, ok := <-events:
				if !ok {
					events = nil
					continue
				}
				if started, ok := event.(lichessapi.GameStartEvent); ok {
					sendOnlineResult(ctx, results, onlineResult{gameStart: &started.Game, seekGeneration: generation})
					return
				}
			case err, ok := <-errors:
				if !ok {
					errors = nil
					continue
				}
				if err != nil && ctx.Err() == nil {
					sendOnlineResult(ctx, results, onlineResult{err: err, seekGeneration: generation})
				}
				return
			case <-ctx.Done():
				return
			}
		}
		if ctx.Err() == nil {
			sendOnlineResult(ctx, results, onlineResult{err: fmt.Errorf("Lichess event stream closed while waiting for an opponent"), seekGeneration: generation})
		}
	}()
}

func applyOnlineSeekResult(menu *menuState, result onlineResult, service OnlineBoardService, results chan<- onlineResult, cancel *context.CancelFunc) context.Context {
	if result.seekGeneration != menu.onlineSeekGeneration {
		return nil
	}
	if result.err != nil {
		if *cancel != nil {
			(*cancel)()
		}
		menu.onlineSeekGeneration++
		menu.onlineSeeking = false
		menu.page = lichessMenu
		menu.lichessMessage = result.err.Error()
		return nil
	}
	if result.gameStart == nil || menu.lichessAccount == nil {
		return nil
	}
	menu.onlineSeeking = false
	ctx, newCancel := context.WithCancel(context.Background())
	if *cancel != nil {
		(*cancel)()
	}
	*cancel = newCancel
	game := *result.gameStart
	seek := lichessapi.Seek{TimeSeconds: int(menu.timeControl.initial.Seconds()), IncrementSeconds: int(menu.timeControl.increment.Seconds()), Rated: menu.onlineRated}
	go streamOnlineGame(ctx, service, game.ID, menu.lichessAccount.ID, menu.lichessAccount.Username, seek, results)
	return ctx
}

func sendOnlineResult(ctx context.Context, results chan<- onlineResult, result onlineResult) bool {
	select {
	case results <- result:
		return true
	case <-ctx.Done():
		return false
	}
}

func streamOnlineGame(ctx context.Context, service OnlineBoardService, gameID, accountID, accountName string, seek lichessapi.Seek, results chan<- onlineResult) {
	for {
		events, streamErrors := service.StreamGame(ctx, gameID)
		for events != nil || streamErrors != nil {
			select {
			case event, ok := <-events:
				if !ok {
					events = nil
					continue
				}
				if !sendOnlineResult(ctx, results, onlineResult{event: event, accountID: accountID, accountName: accountName, seek: seek}) {
					return
				}
			case err, ok := <-streamErrors:
				if !ok {
					streamErrors = nil
					continue
				}
				if err != nil {
					var boardError *lichessapi.BoardHTTPError
					if errors.As(err, &boardError) && boardError.IsPermanent() {
						sendOnlineResult(ctx, results, onlineResult{err: fmt.Errorf("Lichess game stream: %w", err), fatal: true})
						return
					}
					select {
					case results <- onlineResult{action: "reconnecting", err: fmt.Errorf("Lichess game stream: %w", err)}:
					case <-ctx.Done():
						return
					}
				}
				events = nil
				streamErrors = nil
			}
		}
		if ctx.Err() != nil {
			return
		}
		if !sendOnlineResult(ctx, results, onlineResult{action: "reconnecting"}) {
			return
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

func newOnlineGame(full lichessapi.GameFullEvent, accountID, accountName string, theme chess.ColorTheme) (*onlineGameState, error) {
	var board chess.Board
	if full.InitialFEN == "startpos" || full.InitialFEN == "" {
		board.Reset()
	} else {
		var err error
		board, err = chess.ParseFEN(full.InitialFEN)
		if err != nil {
			return nil, err
		}
	}
	game := &onlineGameState{gameID: full.Game.ID, createdAt: full.Game.CreatedAt, white: full.Game.White, black: full.Game.Black, state: full.State}
	game.localGameState = newLocalGameState(theme)
	game.board = board
	game.positions = []chess.Board{board}
	if !sameLichessPlayer(full.Game.White, accountID, accountName) && !sameLichessPlayer(full.Game.Black, accountID, accountName) {
		return nil, fmt.Errorf("Lichess game does not include the connected account")
	}
	game.me, game.perspective = chess.White, chess.White
	if sameLichessPlayer(full.Game.Black, accountID, accountName) {
		game.me, game.perspective = chess.Black, chess.Black
	}
	if err := game.rebuild(full.State); err != nil {
		return nil, err
	}
	return game, nil
}

func sameLichessPlayer(player lichessapi.Player, id, name string) bool {
	return (id != "" && strings.EqualFold(player.ID, id)) || (name != "" && strings.EqualFold(player.Name, name))
}

func (g *onlineGameState) rebuild(state lichessapi.GameState) error {
	initial := g.positions[0]
	board := initial
	moves, err := chess.ParseUCIMoves(board, state.Moves)
	if err != nil {
		return err
	}
	g.moves = moves
	g.history = nil
	g.positions = []chess.Board{initial}
	board = initial
	for _, move := range moves {
		san, err := board.SAN(move)
		if err != nil {
			return err
		}
		if err := board.MakeMove(move); err != nil {
			return err
		}
		g.history = append(g.history, san)
		g.positions = append(g.positions, board)
	}
	g.board = board
	if len(moves) > 0 {
		g.lastMove = moves[len(moves)-1]
		g.hasLastMove = true
	} else {
		g.lastMove = chess.Move(0)
		g.hasLastMove = false
	}
	g.viewPly = len(moves)
	g.terminalTitle = ""
	g.drawReason = ""
	g.timedOutBy = nil
	g.resignedBy = nil
	g.state = state
	g.pendingMove = false
	if !g.ownDrawOffer() {
		g.pendingDraw = false
	}
	if !g.ownTakebackRequest() {
		g.pendingTakeback = false
	}
	g.clockRemaining[chess.White] = time.Duration(state.WhiteTime) * time.Millisecond
	g.clockRemaining[chess.Black] = time.Duration(state.BlackTime) * time.Millisecond
	g.clockUpdated = time.Now()
	return nil
}

func (g *onlineGameState) playAuthoritativeUpdate(previousMoves []chess.Move, previousStatus string) {
	matchingPrefix := len(g.moves) >= len(previousMoves)
	if matchingPrefix {
		for index, move := range previousMoves {
			if g.moves[index] != move {
				matchingPrefix = false
				break
			}
		}
	}
	if matchingPrefix {
		for index, move := range g.moves[len(previousMoves):] {
			g.play(sound.Move)
			if isCapture(move) {
				g.play(sound.Capture)
			}
			board := g.positions[len(previousMoves)+index+1]
			if board.IsColorInCheck(board.ColorToMove) && len(chess.GenerateLegalMoves(board)) > 0 {
				g.play(sound.Check)
			}
		}
	}
	if onlineInProgressStatus(previousStatus) && g.isFinished() {
		g.playOnlineTerminal()
	}
}

func (g *onlineGameState) playOnlineTerminal() {
	switch g.state.Status {
	case "mate":
		g.play(sound.Checkmate)
	case "stalemate":
		g.play(sound.Stalemate)
	case "draw":
		g.play(sound.Draw)
	}
	g.endSounds()
}

func onlineInProgressStatus(status string) bool {
	return status == "created" || status == "started"
}

func (g *onlineGameState) advanceClock(now time.Time) {
	if !g.isInProgress() || g.clockUpdated.IsZero() {
		return
	}
	elapsed := now.Sub(g.clockUpdated)
	if elapsed <= 0 {
		return
	}
	color := g.board.ColorToMove
	if elapsed >= g.clockRemaining[color] {
		g.clockRemaining[color] = 0
	} else {
		g.clockRemaining[color] -= elapsed
	}
	g.clockUpdated = now
	g.playOnlineLowTime(color)
	if g.pendingDraw || g.pendingTakeback {
		g.offerAnimation = (g.offerAnimation + 1) % 3
	}
}

func (g *onlineGameState) playOnlineLowTime(color chess.PieceColor) {
	if g.lowTimePlayed[color] || g.seek.TimeSeconds <= 0 {
		return
	}
	threshold := time.Duration(g.seek.TimeSeconds+40*g.seek.IncrementSeconds) * time.Second / 10
	if g.clockRemaining[color] <= threshold {
		g.lowTimePlayed[color] = true
		g.play(sound.LowTime)
	}
}

func (g onlineGameState) isInProgress() bool {
	return g.state.Status == "created" || g.state.Status == "started"
}

func (g onlineGameState) isFinished() bool {
	return !g.isInProgress()
}

func (g *onlineGameState) apply(result onlineResult) {
	if result.fatal {
		g.fatal = true
	}
	if result.err != nil {
		if result.action == "rating changes" {
			g.ratingLoading = false
			g.ratingChecked = true
			return
		}
		if result.action == "save PGN" {
			g.exportingPGN = false
			if _, saveErr := savePGNBytes(g.pgnFileDate(), result.includeClocks, []byte(pgn.Format(g.pgnGame(), result.includeClocks))); saveErr != nil {
				g.exportMessage = "Could not save PGN: " + saveErr.Error()
			} else {
				g.exportMessage = "Saved local PGN (Lichess export unavailable)"
			}
			return
		}
		if result.action == "move" {
			g.pendingMove = false
		}
		if result.action == "draw" {
			g.pendingDraw = false
		}
		if result.action == "takeback" {
			g.pendingTakeback = false
		}
		g.message = result.err.Error()
		return
	}
	switch event := result.event.(type) {
	case nil:
		if result.action == "save PGN" {
			g.exportingPGN = false
			path, err := savePGNBytes(g.pgnFileDate(), result.includeClocks, result.exportedPGN)
			if err != nil {
				g.exportMessage = "Could not save PGN: " + err.Error()
			} else {
				g.exportMessage = "Saved " + filepath.Base(path)
			}
			return
		}
		if result.ratingChanges != nil {
			g.ratingLoading = false
			g.ratingChecked = true
			g.ratingChanges = result.ratingChanges
			return
		}
		if result.action == "reconnecting" {
			g.message = "Reconnecting to Lichess..."
		}
	case lichessapi.GameFullEvent:
		// Full events establish both the authoritative starting position and players.
		if next, err := newOnlineGame(event, result.accountID, result.accountName, g.theme); err != nil {
			g.message = err.Error()
		} else {
			next.perspective = g.perspective
			next.seek = g.seek
			next.sounds = g.sounds
			next.adjustSoundVolume = g.adjustSoundVolume
			next.soundStarted = g.soundStarted
			next.soundEnded = g.soundEnded
			next.lowTimePlayed = g.lowTimePlayed
			previousMoves, previousStatus := g.moves, g.state.Status
			*g = *next
			g.playAuthoritativeUpdate(previousMoves, previousStatus)
		}
	case lichessapi.GameStateEvent:
		previousMoves, previousStatus := g.moves, g.state.Status
		if err := g.rebuild(event.State); err != nil {
			g.message = err.Error()
		} else {
			g.playAuthoritativeUpdate(previousMoves, previousStatus)
		}
	}
}

func (g *onlineGameState) savePGN(service OnlineBoardService, results chan<- onlineResult, includeClocks bool) {
	if g.exportingPGN {
		return
	}
	g.exportingPGN = true
	g.exportMessage = "Saving PGN..."
	gameID := g.gameID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		exported, err := service.ExportPGN(ctx, gameID, includeClocks)
		results <- onlineResult{action: "save PGN", exportedPGN: exported, includeClocks: includeClocks, err: err}
	}()
}

func (g onlineGameState) pgnDate() time.Time {
	if g.createdAt > 0 {
		return time.UnixMilli(g.createdAt).UTC()
	}
	if !g.startedAt.IsZero() {
		return g.startedAt.UTC()
	}
	return time.Now().UTC()
}

func (g onlineGameState) pgnFileDate() time.Time {
	return g.pgnDate().Local()
}

func (g onlineGameState) pgnGame() pgn.Game {
	eventType := "casual"
	if g.seek.Rated {
		eventType = "rated"
	}
	speed := "rapid"
	if g.seek.TimeSeconds >= 30*60 {
		speed = "classical"
	}
	date := g.pgnDate()
	game := pgn.Game{
		Event:       fmt.Sprintf("%s %s game", eventType, speed),
		Site:        "https://lichess.org/game/" + g.gameID,
		Date:        date,
		Round:       "-",
		White:       g.white.Name,
		Black:       g.black.Name,
		Result:      g.pgnResult(),
		Variant:     "Standard",
		TimeControl: fmt.Sprintf("%d+%d", g.seek.TimeSeconds, g.seek.IncrementSeconds),
		Termination: g.pgnTermination(),
		GameID:      g.gameID,
		UTCDate:     date.Format("2006.01.02"),
		UTCTime:     date.Format("15:04:05"),
		WhiteElo:    strconv.Itoa(g.white.Rating),
		BlackElo:    strconv.Itoa(g.black.Rating),
		Moves:       g.history,
		Clocks:      g.clockHistory,
	}
	if g.ratingChanges != nil {
		game.WhiteRatingDiff = fmt.Sprintf("%+d", g.ratingChanges.White)
		game.BlackRatingDiff = fmt.Sprintf("%+d", g.ratingChanges.Black)
	}
	return game
}

func (g onlineGameState) pgnResult() string {
	if winner, ok := lichessWinner(g.state.Winner); ok {
		return winner.EndResult()
	}
	if g.state.Status == "aborted" || g.state.Status == "noStart" {
		return "*"
	}
	return "1/2-1/2"
}

func (g onlineGameState) pgnTermination() string {
	switch g.state.Status {
	case "timeout", "outoftime":
		return "Time forfeit"
	case "aborted", "noStart":
		return "Abandoned"
	case "cheat":
		return "Rules infraction"
	default:
		return "Normal"
	}
}

func (g *onlineGameState) handleKey(event *tcell.EventKey, service OnlineBoardService, results chan<- onlineResult) bool {
	if g.handleSoundVolumeKey(event) {
		return false
	}
	if g.isFinished() || g.fatal {
		if g.review.active {
			if event.Key() == tcell.KeyEsc {
				g.leaveReview()
			}
			return false
		}
		if event.Key() == tcell.KeyRune && (event.Rune() == 'g' || event.Rune() == 'G') {
			g.startReview()
			return false
		}
		return event.Key() == tcell.KeyEsc
	}
	if g.state.WhiteDraw && g.me == chess.Black || g.state.BlackDraw && g.me == chess.White {
		return g.handleOfferKey(event, service, results, "draw")
	}
	if g.state.WhiteTakeback && g.me == chess.Black || g.state.BlackTakeback && g.me == chess.White {
		return g.handleOfferKey(event, service, results, "takeback")
	}
	switch event.Key() {
	case tcell.KeyEsc:
		return false
	case tcell.KeyEnter:
		g.submitMove(service, results, g.uciInput)
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(g.uciInput) > 0 {
			g.uciInput = g.uciInput[:len(g.uciInput)-1]
		}
	case tcell.KeyRune:
		c := unicode.ToLower(event.Rune())
		if (c >= 'a' && c <= 'h') || (c >= '1' && c <= '8') || strings.ContainsRune("qrbn", c) {
			g.uciInput += string(c)
		}
	}
	return false
}

func (g *onlineGameState) handleMouse(event *tcell.EventMouse, width, height int, service OnlineBoardService, results chan<- onlineResult) {
	x, y := event.Position()
	layout, ok := localBoardLayout(width, height)
	if !ok {
		return
	}
	if g.isFinished() {
		g.handleFinishedMouse(event, layout, service, results)
		return
	}
	if g.fatal {
		return
	}
	if event.Buttons() == tcell.ButtonNone {
		g.hoveredResign = g.resignButtonAt(x, y, layout)
		g.hoveredFlip = flipButtonAt(x, y, layout)
		g.hoveredDraw = !g.pendingDraw && !g.incomingDraw() && drawOfferButtonAt(x, y, layout)
		g.hoveredTakeback = !g.pendingTakeback && !g.incomingTakeback() && takebackButtonAt(x, y, layout)
		if g.hoveredResign || g.hoveredFlip || g.hoveredDraw || g.hoveredTakeback {
			return
		}
	}
	if event.Buttons() == tcell.Button1 && g.resignButtonAt(x, y, layout) {
		if g.resignPending {
			g.action(service, results, "resign", true)
		} else {
			g.resignPending = true
		}
		return
	}
	if event.Buttons() == tcell.Button1 && g.resignPending {
		g.resignPending = false
		g.hoveredResign = false
		return
	}
	if event.Buttons() == tcell.Button1 && flipButtonAt(x, y, layout) {
		g.flipBoard()
		return
	}
	if event.Buttons() == tcell.Button1 && !g.pendingDraw && !g.incomingDraw() && drawOfferButtonAt(x, y, layout) {
		g.pendingDraw = true
		g.offerAnimation = 0
		g.action(service, results, "draw", true)
		return
	}
	if event.Buttons() == tcell.Button1 && !g.pendingTakeback && !g.incomingTakeback() && takebackButtonAt(x, y, layout) {
		g.pendingTakeback = true
		g.offerAnimation = 0
		g.action(service, results, "takeback", true)
		return
	}
	if g.pendingMove || g.board.ColorToMove != g.me {
		return
	}
	_, onBoard := squareAtScreenPosition(x, y, layout, g.perspective)
	if !onBoard && len(g.promotionChoices) == 0 {
		return
	}
	moveCount := len(g.moves)
	g.suppressSoundEvents = true
	g.localGameState.handleMouse(event, width, height)
	g.suppressSoundEvents = false
	if len(g.moves) == moveCount {
		return
	}
	move := g.moves[len(g.moves)-1].UCI()
	if err := g.rebuild(g.state); err != nil {
		g.message = err.Error()
		return
	}
	g.uciInput = move
	g.submitMove(service, results, move)
}

func (g *onlineGameState) handleFinishedMouse(event *tcell.EventMouse, layout boardLayout, service OnlineBoardService, results chan<- onlineResult) {
	x, y := event.Position()
	if g.review.active {
		g.handleGameReviewMouse(event, layout, layout.y+13)
		return
	}
	if event.Buttons() == tcell.ButtonNone {
		g.hoveredFlip = flipButtonAt(x, y, layout)
		g.hoveredSavePGN = onlineSavePGNButtonAt(x, y, layout, false)
		g.hoveredSaveClocks = onlineSavePGNButtonAt(x, y, layout, true)
		g.review.hoveredStart = gameReviewButtonAt(x, y, layout)
		return
	}
	buttons := event.Buttons()
	if buttons&tcell.WheelUp != 0 || buttons&tcell.WheelDown != 0 {
		if _, ok := squareAtScreenPosition(x, y, layout, g.perspective); ok {
			direction := 1
			if buttons&tcell.WheelUp != 0 {
				direction = -1
			}
			g.moveReplay(direction)
			return
		}
		if historyAreaAt(x, y, layout) {
			if buttons&tcell.WheelUp != 0 {
				g.historyScroll = max(0, g.historyScroll-1)
			} else {
				g.historyScroll = min(g.historyMaxScroll(), g.historyScroll+1)
			}
		}
		return
	}
	if buttons != tcell.Button1 {
		return
	}
	if gameReviewButtonAt(x, y, layout) {
		g.startReview()
		return
	}
	if flipButtonAt(x, y, layout) {
		g.flipBoard()
		return
	}
	if onlineSavePGNButtonAt(x, y, layout, false) {
		g.savePGN(service, results, false)
		return
	}
	if onlineSavePGNButtonAt(x, y, layout, true) {
		g.savePGN(service, results, true)
		return
	}
	if ply, ok := g.historyPlyAt(x, y, layout); ok {
		g.viewPly = ply
		g.syncHistoryAfterClick(y - (layout.y + 1))
		return
	}
}

func (g *onlineGameState) handleOfferKey(event *tcell.EventKey, service OnlineBoardService, results chan<- onlineResult, action string) bool {
	if event.Key() == tcell.KeyEsc {
		return false
	}
	if event.Key() == tcell.KeyRune && (event.Rune() == 'y' || event.Rune() == 'Y') {
		g.action(service, results, action, true)
	}
	if event.Key() == tcell.KeyRune && (event.Rune() == 'n' || event.Rune() == 'N') {
		g.action(service, results, action, false)
	}
	return false
}

func (g *onlineGameState) submitMove(service OnlineBoardService, results chan<- onlineResult, move string) {
	if g.pendingMove || g.board.ColorToMove != g.me {
		return
	}
	if _, err := g.board.ParseUCI(move); err != nil {
		g.message = err.Error()
		return
	}
	g.pendingMove = true
	g.uciInput = ""
	go func() {
		sendOnlineResult(g.boardContext(), results, onlineResult{action: "move", err: service.MakeMove(g.boardContext(), g.gameID, move)})
	}()
}

func (g *onlineGameState) action(service OnlineBoardService, results chan<- onlineResult, action string, accept bool) {
	g.resignPending = false
	go func() {
		var err error
		switch action {
		case "resign":
			err = service.Resign(g.boardContext(), g.gameID)
		case "draw":
			err = service.SetDraw(g.boardContext(), g.gameID, accept)
		case "takeback":
			err = service.SetTakeback(g.boardContext(), g.gameID, accept)
		}
		sendOnlineResult(g.boardContext(), results, onlineResult{action: action, err: err})
	}()
}

func (g onlineGameState) boardContext() context.Context {
	if g.requestContext != nil {
		return g.requestContext
	}
	return context.Background()
}

func (g onlineGameState) ownDrawOffer() bool {
	return (g.me == chess.White && g.state.WhiteDraw) || (g.me == chess.Black && g.state.BlackDraw)
}

func (g onlineGameState) ownTakebackRequest() bool {
	return (g.me == chess.White && g.state.WhiteTakeback) || (g.me == chess.Black && g.state.BlackTakeback)
}

func drawOnlineGame(screen tcell.Screen, game onlineGameState) {
	width, height := screen.Size()
	layout, ok := localBoardLayout(width, height)
	if !ok {
		drawCentered(screen, height/2, "Terminal too small", titleStyle)
		return
	}
	display := game.localGameState
	if game.isFinished() {
		display.terminalTitle = game.onlineTerminalTitle()
	}
	displayBoard := display.boardForDisplay()
	drawBoard(screen, display, layout)
	drawMoveHistory(screen, display, layout, width)
	top := game.black
	bottom := game.white
	topColor, bottomColor := chess.Black, chess.White
	if game.perspective == chess.Black {
		top, bottom, topColor, bottomColor = bottom, top, chess.White, chess.Black
	}
	drawString(screen, layout.x-2, layout.y-3, game.playerLine(top, topColor), titleStyle)
	drawString(screen, layout.x-2, layout.y-2, formatClock(game.clockRemaining[topColor])+"  "+materialIndicator(displayBoard, topColor), mutedStyle)
	drawString(screen, layout.x-2, layout.y+10, game.playerLine(bottom, bottomColor), titleStyle)
	drawString(screen, layout.x-2, layout.y+11, formatClock(game.clockRemaining[bottomColor])+"  "+materialIndicator(displayBoard, bottomColor), mutedStyle)
	if game.isFinished() {
		if display.review.active {
			drawGameReviewPanel(screen, display, layout, layout.y+13, true)
		} else {
			drawOnlineSavePGNButton(screen, display, layout, false)
			drawOnlineSavePGNButton(screen, display, layout, true)
			drawGameReviewButton(screen, display, layout)
			if game.exportMessage != "" {
				drawString(screen, layout.x-2, layout.y+15, game.exportMessage, mutedStyle)
			}
			drawString(screen, layout.x-2, layout.y+16, reviewSoundHint(game.message), mutedStyle)
		}
		return
	}
	drawString(screen, layout.x-2, layout.y+13, "Move (UCI): "+game.uciInput+"_", backgroundStyle)
	message := game.message
	if game.incomingDraw() {
		message = "Opponent offers a draw: y accept / n decline"
	} else if game.incomingTakeback() {
		message = "Opponent requests takeback: y accept / n decline"
	} else if game.isFinished() {
		message = "Game " + game.state.Status
	}
	drawString(screen, layout.x-2, layout.y+14, message, mutedStyle)
	drawOnlineActionButtons(screen, game, layout)
	drawString(screen, layout.x-2, layout.y+15, "Enter: submit move  ,/.: sound volume", mutedStyle)
	if len(display.promotionChoices) != 0 {
		drawPromotionPicker(screen, display, layout)
	}
}

func (g onlineGameState) onlineTerminalTitle() string {
	winner, hasWinner := lichessWinner(g.state.Winner)
	switch g.state.Status {
	case "mate":
		if hasWinner {
			return fmt.Sprintf("Checkmate · %s wins %s", winner, winner.EndResult())
		}
	case "resign":
		if hasWinner {
			return fmt.Sprintf("%s resigned · %s wins %s", winner.Opponent(), winner, winner.EndResult())
		}
	case "timeout", "outoftime":
		if hasWinner {
			return fmt.Sprintf("%s ran out of time · %s wins %s", winner.Opponent(), winner, winner.EndResult())
		}
	case "stalemate":
		return "Stalemate · ½-½"
	case "draw":
		return "Draw · ½-½"
	case "aborted", "noStart":
		return "Game aborted"
	case "cheat":
		if hasWinner {
			return fmt.Sprintf("Fair-play violation · %s wins %s", winner, winner.EndResult())
		}
	}
	if hasWinner {
		return fmt.Sprintf("%s wins %s", winner, winner.EndResult())
	}
	return "Game " + g.state.Status
}

func lichessWinner(winner string) (chess.PieceColor, bool) {
	switch winner {
	case "white":
		return chess.White, true
	case "black":
		return chess.Black, true
	default:
		return chess.White, false
	}
}

func drawOnlineSavePGNButton(screen tcell.Screen, game localGameState, layout boardLayout, includeClocks bool) {
	x, y := onlineSavePGNButtonPosition(layout, includeClocks)
	style := backgroundStyle
	if (!includeClocks && game.hoveredSavePGN) || (includeClocks && game.hoveredSaveClocks) {
		style = selectedStyle
	}
	drawString(screen, x, y, savePGNButtonLabel(includeClocks), style)
}

func onlineSavePGNButtonPosition(layout boardLayout, includeClocks bool) (int, int) {
	y := layout.y + 13
	if includeClocks {
		y++
	}
	return layout.x - 2, y
}

func onlineSavePGNButtonAt(x, y int, layout boardLayout, includeClocks bool) bool {
	buttonX, buttonY := onlineSavePGNButtonPosition(layout, includeClocks)
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth(savePGNButtonLabel(includeClocks))
}

func playerLine(player lichessapi.Player, color chess.PieceColor) string {
	marker := "●"
	if color == chess.Black {
		marker = "○"
	}
	return fmt.Sprintf("%s %s (%d)", marker, player.Name, player.Rating)
}

func (g onlineGameState) playerLine(player lichessapi.Player, color chess.PieceColor) string {
	line := playerLine(player, color)
	if !g.isFinished() || g.ratingChanges == nil {
		return line
	}
	delta := g.ratingChanges.White
	if color == chess.Black {
		delta = g.ratingChanges.Black
	}
	return fmt.Sprintf("%s %+d -> %d", line, delta, player.Rating+delta)
}

func drawOnlineActionButtons(screen tcell.Screen, game onlineGameState, layout boardLayout) {
	x, y := onlineActionButtonPosition(layout, "draw")
	drawOnlineActionButton(screen, x, y, game.drawOfferLabel(), game.hoveredDraw, game.pendingDraw)
	x, y = onlineActionButtonPosition(layout, "takeback")
	drawOnlineActionButton(screen, x, y, game.takebackLabel(), game.hoveredTakeback, game.pendingTakeback)
}

func drawOnlineActionButton(screen tcell.Screen, x, y int, label string, hovered, pending bool) {
	style := backgroundStyle
	if pending {
		style = mutedStyle
	} else if hovered {
		style = selectedStyle
	}
	drawString(screen, x, y, label, style)
}

func (g onlineGameState) drawOfferLabel() string {
	if !g.pendingDraw {
		return "[🤝] Draw offer"
	}
	return "[🤝] Offering draw" + strings.Repeat(".", int(g.offerAnimation)+1)
}

func (g onlineGameState) takebackLabel() string {
	if !g.pendingTakeback {
		return "[🔙] Takeback request"
	}
	return "[🔙] Requesting takeback" + strings.Repeat(".", int(g.offerAnimation)+1)
}

func onlineActionButtonPosition(layout boardLayout, action string) (int, int) {
	y := layout.y + 8
	if action == "takeback" {
		y++
	}
	return moveHistoryBoxX(layout), y
}

func drawOfferButtonAt(x, y int, layout boardLayout) bool {
	buttonX, buttonY := onlineActionButtonPosition(layout, "draw")
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth("[🤝] Draw offer")
}

func takebackButtonAt(x, y int, layout boardLayout) bool {
	buttonX, buttonY := onlineActionButtonPosition(layout, "takeback")
	return y == buttonY && x >= buttonX && x < buttonX+runewidth.StringWidth("[🔙] Takeback request")
}
func (g onlineGameState) incomingDraw() bool {
	return (g.me == chess.White && g.state.BlackDraw) || (g.me == chess.Black && g.state.WhiteDraw)
}
func (g onlineGameState) incomingTakeback() bool {
	return (g.me == chess.White && g.state.BlackTakeback) || (g.me == chess.Black && g.state.WhiteTakeback)
}
