package gui3d

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/chess"
	"github.com/ahmadnaufalhakim/gochs/internal/engine"
	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	windowWidth  = 1440
	windowHeight = 900
)

type assistance struct {
	highlights bool
	arrows     bool
	check      bool
}

type PuzzleLoader func() (entity.Puzzle, error)
type PuzzleAttemptRecorder func(entity.Puzzle, entity.PuzzleOutcome) (entity.TrainingResult, error)
type LichessRatingSynchronizer func(entity.LichessAccount) (entity.TrainingProfile, error)
type OfflinePuzzleRatingLoader func() (entity.TrainingProfile, bool, error)

type LichessAccountService interface {
	Load(context.Context) (entity.LichessAccount, error)
	Connect(context.Context, string) (entity.LichessAccount, error)
	Disconnect() error
}

type Config struct {
	LoadPuzzle              PuzzleLoader
	RecordPuzzleAttempt     PuzzleAttemptRecorder
	SyncLichessRating       LichessRatingSynchronizer
	LoadOfflinePuzzleRating OfflinePuzzleRatingLoader
	Lichess                 LichessAccountService
	StockfishPath           string
}

type state struct {
	board          chess.Board
	perspective    chess.PieceColor
	selected       *chess.Square
	lastMove       *chess.Move
	pressSquare    *chess.Square
	pressPosition  rl.Vector2
	arrows         map[string][2]chess.Square
	assistance     assistance
	message        string
	mode           gameMode
	config         Config
	puzzle         *puzzleSession
	engine         *engine.Client
	engineMoves    []string
	engineReady    chan engineStartResult
	engineMove     chan engineMoveResult
	enginePending  bool
	lichessPage    lichessPage
	lichessAccount *entity.LichessAccount
	offlineRating  *entity.TrainingProfile
	lichessMessage string
	lichessPending bool
	lichessResults chan lichessResult
	gameID         uint64
	ctx            context.Context
}

type gameMode uint8

const (
	menuMode gameMode = iota
	localMode
	stockfishLoadingMode
	stockfishMode
	puzzleMode
)

type lichessPage uint8

const (
	lichessHome lichessPage = iota
	lichessAccountPage
	lichessSyncConfirmation
)

type engineStartResult struct {
	gameID uint64
	client *engine.Client
	err    error
}

type engineMoveResult struct {
	gameID uint64
	move   string
	err    error
}

type lichessAction uint8

const (
	loadLichessAccount lichessAction = iota
	syncLichessRating
)

type lichessResult struct {
	action  lichessAction
	account entity.LichessAccount
	profile entity.TrainingProfile
	err     error
}

func Run(config Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(windowWidth, windowHeight, "Gochs 3D")
	rl.SetExitKey(rl.KeyNull)
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)

	var board chess.Board
	board.Reset()
	game := state{
		board:          board,
		arrows:         make(map[string][2]chess.Square),
		assistance:     assistance{highlights: true, arrows: true, check: true},
		perspective:    chess.White,
		message:        "Choose a game mode",
		mode:           menuMode,
		config:         config,
		engineReady:    make(chan engineStartResult),
		engineMove:     make(chan engineMoveResult),
		lichessResults: make(chan lichessResult, 1),
		ctx:            ctx,
	}
	defer game.closeEngine()
	defer cancel()

	for !rl.WindowShouldClose() {
		game.pollEngine()
		game.pollLichess()
		game.handleInput()
		game.draw()
	}
	return nil
}

func (s *state) handleInput() {
	if s.mode == menuMode {
		s.handleMenuInput()
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		s.returnToMenu()
		return
	}
	if rl.IsKeyPressed(rl.KeyV) {
		s.perspective = s.perspective.Opponent()
	}
	if rl.IsKeyPressed(rl.KeyH) {
		s.assistance.highlights = !s.assistance.highlights
	}
	if rl.IsKeyPressed(rl.KeyA) {
		s.assistance.arrows = !s.assistance.arrows
	}
	if rl.IsKeyPressed(rl.KeyC) {
		s.assistance.check = !s.assistance.check
	}

	square, hasSquare := s.squareAtMouse()
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && hasSquare {
		s.pressSquare = &square
		s.pressPosition = rl.GetMousePosition()
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && s.pressSquare != nil {
		mouse := rl.GetMousePosition()
		dragged := rl.Vector2Distance(mouse, s.pressPosition) > 8
		if hasSquare {
			if dragged {
				s.tryMove(*s.pressSquare, square)
			} else {
				s.clickSquare(square)
			}
		}
		s.pressSquare = nil
	}
	if rl.IsMouseButtonPressed(rl.MouseButtonRight) && hasSquare {
		s.pressSquare = &square
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonRight) && s.pressSquare != nil {
		if hasSquare && square != *s.pressSquare {
			s.toggleArrow(*s.pressSquare, square)
		}
		s.pressSquare = nil
	}
}

func (s *state) clickSquare(square chess.Square) {
	if s.selected == nil {
		if s.board.PieceAt(square).Color == s.board.ColorToMove && s.board.PieceAt(square).Type != chess.PieceNone {
			s.selected = &square
		}
		return
	}
	if *s.selected == square {
		s.selected = nil
		return
	}
	if s.tryMove(*s.selected, square) {
		return
	}
	if s.board.PieceAt(square).Color == s.board.ColorToMove && s.board.PieceAt(square).Type != chess.PieceNone {
		s.selected = &square
	} else {
		s.selected = nil
	}
}

func (s *state) tryMove(from, to chess.Square) bool {
	if s.mode == stockfishLoadingMode || s.enginePending {
		return false
	}
	for _, move := range chess.GenerateLegalMoves(s.board) {
		if move.From() != from || move.To() != to {
			continue
		}
		if move.Flag() == chess.PromoteKnight || move.Flag() == chess.PromoteBishop || move.Flag() == chess.PromoteRook {
			continue
		}
		if s.mode == puzzleMode {
			correct, solved, err := s.puzzle.submit(move, s.config.RecordPuzzleAttempt)
			if err != nil {
				s.message = "Puzzle error: " + err.Error()
				return false
			}
			s.board = s.puzzle.board
			s.lastMove = &move
			s.selected = nil
			if !correct {
				s.message = "Wrong move. Try again."
			} else if solved {
				s.message = "Puzzle solved! Press Esc for the menu"
			} else {
				s.message = "Correct. Your move."
			}
			return true
		}
		if err := s.board.MakeMove(move); err != nil {
			s.message = err.Error()
			return false
		}
		s.lastMove = &move
		s.selected = nil
		s.afterMove(move)
		return true
	}
	return false
}

func (s *state) afterMove(move chess.Move) {
	switch s.mode {
	case stockfishMode:
		s.engineMoves = append(s.engineMoves, move.UCI())
		s.requestEngineMove()
	default:
		s.message = fmt.Sprintf("%s to move", s.board.ColorToMove)
	}
}

func (s *state) handleMenuInput() {
	if !rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		return
	}
	mouse := rl.GetMousePosition()
	if s.lichessPage != lichessHome {
		s.handleLichessMenuInput(mouse)
		return
	}
	switch menuButtonAt(mouse) {
	case 0:
		s.startLocal()
	case 1:
		s.startStockfish()
	case 2:
		s.startPuzzle()
	case 3:
		s.startLichessLoad()
	}
}

func (s *state) handleLichessMenuInput(mouse rl.Vector2) {
	if s.lichessPending {
		return
	}
	switch s.lichessPage {
	case lichessAccountPage:
		switch lichessAccountButtonAt(mouse, s.lichessAccount != nil) {
		case 0:
			if s.lichessAccount != nil {
				s.lichessPage = lichessSyncConfirmation
				s.lichessMessage = ""
			}
		case 1:
			s.lichessPage = lichessHome
			s.lichessMessage = "Choose a game mode"
		}
	case lichessSyncConfirmation:
		switch lichessSyncButtonAt(mouse) {
		case 0:
			s.startLichessSync()
		case 1:
			s.lichessPage = lichessAccountPage
		}
	}
}

func (s *state) startLichessLoad() {
	if s.config.Lichess == nil {
		s.lichessPage = lichessAccountPage
		s.lichessMessage = "Lichess is unavailable"
		return
	}
	s.lichessPage = lichessAccountPage
	s.lichessMessage = "Loading Lichess account..."
	s.lichessPending = true
	service := s.config.Lichess
	go func() {
		account, err := service.Load(s.ctx)
		select {
		case s.lichessResults <- lichessResult{action: loadLichessAccount, account: account, err: err}:
		case <-s.ctx.Done():
		}
	}()
}

func (s *state) startLichessSync() {
	if s.lichessAccount == nil || s.config.SyncLichessRating == nil {
		s.lichessMessage = "Lichess rating sync is unavailable"
		s.lichessPage = lichessAccountPage
		return
	}
	s.lichessMessage = "Syncing offline puzzle rating..."
	s.lichessPending = true
	account := *s.lichessAccount
	go func() {
		profile, err := s.config.SyncLichessRating(account)
		select {
		case s.lichessResults <- lichessResult{action: syncLichessRating, profile: profile, err: err}:
		case <-s.ctx.Done():
		}
	}()
}

func (s *state) pollLichess() {
	for {
		select {
		case result := <-s.lichessResults:
			s.applyLichessResult(result)
		default:
			return
		}
	}
}

func (s *state) applyLichessResult(result lichessResult) {
	s.lichessPending = false
	if result.err != nil {
		if errors.Is(result.err, lichessapi.ErrNoToken) {
			s.lichessMessage = "No Lichess token found. Set GOCHS_LICHESS_TOKEN or connect in the terminal app."
		} else {
			s.lichessMessage = "Lichess: " + result.err.Error()
		}
		s.lichessPage = lichessAccountPage
		return
	}
	if result.action == syncLichessRating {
		profile := result.profile
		s.offlineRating = &profile
		s.lichessMessage = fmt.Sprintf("Offline puzzle rating synced to %d", profile.Rating)
		s.lichessPage = lichessAccountPage
		return
	}
	s.lichessAccount = &result.account
	s.offlineRating = nil
	s.lichessMessage = ""
	s.loadOfflinePuzzleRating()
}

func (s *state) loadOfflinePuzzleRating() {
	if s.config.LoadOfflinePuzzleRating == nil {
		return
	}
	profile, ok, err := s.config.LoadOfflinePuzzleRating()
	if err != nil {
		s.lichessMessage = "Load offline rating: " + err.Error()
		return
	}
	if ok {
		s.offlineRating = &profile
	}
}

func (s *state) startLocal() {
	s.resetBoard()
	s.mode = localMode
	s.message = "Local game: select or drag a piece to move"
}

func (s *state) startStockfish() {
	s.resetBoard()
	s.mode = stockfishLoadingMode
	s.message = "Starting Stockfish (difficulty 4)..."
	id := s.gameID
	go func() {
		client, err := engine.Start(s.ctx, engine.Config{Path: s.config.StockfishPath, Difficulty: 4})
		result := engineStartResult{gameID: id, client: client, err: err}
		select {
		case s.engineReady <- result:
		case <-s.ctx.Done():
			if client != nil {
				_ = client.Close()
			}
		}
	}()
}

func (s *state) startPuzzle() {
	if s.config.LoadPuzzle == nil {
		s.message = "Offline puzzles are unavailable"
		return
	}
	puzzle, err := s.config.LoadPuzzle()
	if err != nil {
		s.message = "Load puzzle: " + err.Error()
		return
	}
	session, err := newPuzzleSession(puzzle)
	if err != nil {
		s.message = "Load puzzle: " + err.Error()
		return
	}
	s.resetBoard()
	s.puzzle = &session
	s.board = session.board
	s.mode = puzzleMode
	s.message = "Puzzle: find the best move"
}

func (s *state) resetBoard() {
	s.closeEngine()
	s.gameID++
	var board chess.Board
	board.Reset()
	s.board = board
	s.perspective = chess.White
	s.selected = nil
	s.lastMove = nil
	s.pressSquare = nil
	s.arrows = make(map[string][2]chess.Square)
	s.puzzle = nil
	s.engineMoves = nil
	s.enginePending = false
}

func (s *state) returnToMenu() {
	s.resetBoard()
	s.mode = menuMode
	s.message = "Choose a game mode"
}

func (s *state) closeEngine() {
	if s.engine != nil {
		_ = s.engine.Close()
		s.engine = nil
	}
}

func (s *state) pollEngine() {
	for {
		select {
		case result := <-s.engineReady:
			if result.gameID != s.gameID || s.mode != stockfishLoadingMode {
				if result.client != nil {
					_ = result.client.Close()
				}
				continue
			}
			if result.err != nil {
				s.mode = menuMode
				s.message = "Start Stockfish: " + result.err.Error()
				continue
			}
			s.engine = result.client
			s.mode = stockfishMode
			s.message = "Stockfish game: you play White"
		case result := <-s.engineMove:
			if result.gameID != s.gameID || s.mode != stockfishMode {
				continue
			}
			s.enginePending = false
			if result.err != nil {
				s.message = "Stockfish: " + result.err.Error()
				continue
			}
			move, err := s.board.ParseUCI(result.move)
			if err != nil || s.board.MakeMove(move) != nil {
				s.message = "Stockfish returned an invalid move"
				continue
			}
			s.engineMoves = append(s.engineMoves, move.UCI())
			s.lastMove = &move
			s.message = "Your move"
		default:
			return
		}
	}
}

func (s *state) requestEngineMove() {
	if s.engine == nil || s.enginePending {
		return
	}
	s.enginePending = true
	s.message = "Stockfish is thinking..."
	id := s.gameID
	moves := append([]string(nil), s.engineMoves...)
	client := s.engine
	go func() {
		move, err := client.BestMove(s.ctx, moves, engine.Search{MoveTime: 600 * time.Millisecond})
		select {
		case s.engineMove <- engineMoveResult{gameID: id, move: move, err: err}:
		case <-s.ctx.Done():
		}
	}()
}

func (s *state) toggleArrow(from, to chess.Square) {
	key := from.String() + to.String()
	if _, exists := s.arrows[key]; exists {
		delete(s.arrows, key)
		return
	}
	s.arrows[key] = [2]chess.Square{from, to}
}

func (s *state) squareAtMouse() (chess.Square, bool) {
	width, height := rl.GetScreenWidth(), rl.GetScreenHeight()
	camera := s.camera()
	ray := rl.GetMouseRay(rl.GetMousePosition(), camera)
	if math.Abs(float64(ray.Direction.Y)) < 0.0001 {
		return 0, false
	}
	t := -ray.Position.Y / ray.Direction.Y
	if t < 0 {
		return 0, false
	}
	_ = width
	_ = height
	position := ray.Position.Add(ray.Direction.Scale(t))
	return SquareAtWorld(Vec3{X: position.X, Y: position.Y, Z: position.Z})
}

func (s state) camera() rl.Camera3D {
	position := CameraPosition(s.perspective)
	return rl.Camera3D{
		Position:   rl.NewVector3(position.X, position.Y, position.Z),
		Target:     rl.NewVector3(0, 0.35, 0),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       42,
		Projection: rl.CameraPerspective,
	}
}

func (s state) draw() {
	rl.BeginDrawing()
	defer rl.EndDrawing()
	rl.ClearBackground(rl.NewColor(15, 20, 28, 255))

	rl.BeginMode3D(s.camera())
	s.drawBoard()
	s.drawPieces()
	if s.assistance.arrows {
		s.drawArrows()
	}
	rl.EndMode3D()

	rl.DrawRectangle(18, 18, 410, 112, rl.Fade(rl.Black, 0.72))
	rl.DrawText("GOCHS 3D", 34, 32, 28, rl.NewColor(238, 191, 92, 255))
	rl.DrawText(s.message, 34, 68, 18, rl.RayWhite)
	rl.DrawText("Left click/drag: move  Right drag: arrow", 34, 94, 16, rl.LightGray)
	rl.DrawText("V: side  H: highlights  A: arrows  C: check cue", 34, 114, 16, rl.LightGray)
	if s.mode == menuMode {
		s.drawMenu()
	}
}

func (s state) drawMenu() {
	width, height := rl.GetScreenWidth(), rl.GetScreenHeight()
	panelWidth, panelHeight := int32(470), int32(490)
	x := int32(width)/2 - panelWidth/2
	y := int32(height)/2 - panelHeight/2
	rl.DrawRectangle(0, 0, int32(width), int32(height), rl.Fade(rl.Black, 0.5))
	rl.DrawRectangle(x, y, panelWidth, panelHeight, rl.NewColor(27, 34, 45, 245))
	rl.DrawRectangleLines(x, y, panelWidth, panelHeight, rl.NewColor(238, 191, 92, 255))
	if s.lichessPage != lichessHome {
		s.drawLichessMenu(x, y, panelWidth, panelHeight)
		return
	}
	rl.DrawText("PLAY", x+28, y+28, 34, rl.NewColor(238, 191, 92, 255))
	rl.DrawText("Choose a mode", x+28, y+72, 20, rl.RayWhite)
	for index, label := range []string{"Local game", "Play Stockfish", "Random offline puzzle", "Lichess"} {
		buttonY := y + 118 + int32(index)*78
		color := rl.NewColor(57, 82, 106, 255)
		mouse := rl.GetMousePosition()
		if mouse.X >= float32(x+28) && mouse.X < float32(x+panelWidth-28) && mouse.Y >= float32(buttonY) && mouse.Y < float32(buttonY+58) {
			color = rl.NewColor(78, 111, 142, 255)
		}
		rl.DrawRectangle(x+28, buttonY, panelWidth-56, 58, color)
		rl.DrawText(label, x+48, buttonY+17, 22, rl.RayWhite)
	}
}

func menuButtonAt(position rl.Vector2) int {
	panelWidth, panelHeight := int32(470), int32(490)
	x := int32(rl.GetScreenWidth())/2 - panelWidth/2
	y := int32(rl.GetScreenHeight())/2 - panelHeight/2
	for index := range 4 {
		buttonY := y + 118 + int32(index)*78
		if position.X >= float32(x+28) && position.X < float32(x+panelWidth-28) && position.Y >= float32(buttonY) && position.Y < float32(buttonY+58) {
			return index
		}
	}
	return -1
}

func lichessAccountButtonAt(position rl.Vector2, hasAccount bool) int {
	if !hasAccount {
		return lichessButtonAt(position, 1)
	}
	return lichessButtonAt(position, 2)
}

func lichessSyncButtonAt(position rl.Vector2) int {
	return lichessButtonAt(position, 2)
}

func lichessButtonAt(position rl.Vector2, count int) int {
	panelWidth, panelHeight := int32(470), int32(490)
	x := int32(rl.GetScreenWidth())/2 - panelWidth/2
	y := int32(rl.GetScreenHeight())/2 - panelHeight/2
	for index := range count {
		buttonY := y + 300 + int32(index)*68
		if position.X >= float32(x+28) && position.X < float32(x+panelWidth-28) && position.Y >= float32(buttonY) && position.Y < float32(buttonY+52) {
			return index
		}
	}
	return -1
}

func (s state) drawLichessMenu(x, y, width, _ int32) {
	rl.DrawText("LICHESS", x+28, y+28, 34, rl.NewColor(238, 191, 92, 255))
	switch s.lichessPage {
	case lichessAccountPage:
		s.drawLichessAccount(x, y, width)
	case lichessSyncConfirmation:
		s.drawLichessSyncConfirmation(x, y, width)
	}
}

func (s state) drawLichessAccount(x, y, width int32) {
	if s.lichessAccount == nil {
		label := "Lichess account"
		if s.lichessPending {
			label = "Loading your stored Lichess account..."
		}
		rl.DrawText(label, x+28, y+86, 20, rl.RayWhite)
		s.drawLichessMessage(x, y+124, width)
		s.drawLichessButton(x, y+300, width, "Back")
		return
	}
	account := s.lichessAccount
	rl.DrawText(account.Username, x+28, y+82, 26, rl.RayWhite)
	rl.DrawText(fmt.Sprintf("Lichess Puzzle rating: %d", account.Puzzle), x+28, y+122, 20, rl.RayWhite)
	if s.offlineRating != nil {
		rl.DrawText(fmt.Sprintf("Offline puzzle rating: %d", s.offlineRating.Rating), x+28, y+154, 20, rl.RayWhite)
	} else {
		rl.DrawText("Offline puzzle rating: no profile yet", x+28, y+154, 20, rl.LightGray)
	}
	s.drawLichessMessage(x, y+190, width)
	s.drawLichessButton(x, y+300, width, "Sync offline puzzle rating")
	s.drawLichessButton(x, y+368, width, "Back")
}

func (s state) drawLichessSyncConfirmation(x, y, width int32) {
	rl.DrawText("Sync Offline Puzzle Rating", x+28, y+82, 26, rl.RayWhite)
	if s.lichessAccount != nil {
		rl.DrawText(fmt.Sprintf("Lichess Puzzle rating: %d", s.lichessAccount.Puzzle), x+28, y+122, 20, rl.RayWhite)
	}
	if s.offlineRating != nil {
		rl.DrawText(fmt.Sprintf("Current offline rating: %d", s.offlineRating.Rating), x+28, y+154, 20, rl.RayWhite)
	} else {
		rl.DrawText("Current offline rating: no profile yet", x+28, y+154, 20, rl.LightGray)
	}
	rl.DrawText("This replaces your offline puzzle rating.", x+28, y+202, 18, rl.LightGray)
	s.drawLichessMessage(x, y+234, width)
	s.drawLichessButton(x, y+300, width, "Confirm sync")
	s.drawLichessButton(x, y+368, width, "Cancel")
}

func (s state) drawLichessButton(x, y, width int32, label string) {
	rl.DrawRectangle(x+28, y, width-56, 52, rl.NewColor(57, 82, 106, 255))
	rl.DrawText(label, x+48, y+14, 20, rl.RayWhite)
}

func (s state) drawLichessMessage(x, y, width int32) {
	if s.lichessMessage == "" {
		return
	}
	fontSize := int32(17)
	messageWidth := rl.MeasureText(s.lichessMessage, fontSize)
	if messageWidth > width-56 {
		fontSize = 14
	}
	rl.DrawText(s.lichessMessage, x+28, y, fontSize, rl.LightGray)
}

func (s state) drawBoard() {
	light := rl.NewColor(232, 211, 172, 255)
	dark := rl.NewColor(108, 72, 51, 255)
	border := rl.NewColor(62, 39, 28, 255)
	checkedKing, kingInCheck := s.checkedKingSquare()
	rl.DrawCube(rl.NewVector3(0, -0.18, 0), 8.8, 0.35, 8.8, border)
	for rank := range 8 {
		for file := range 8 {
			square, _ := chess.ParseCoordinate(string(rune('a'+file)) + string(rune('1'+rank)))
			center := SquareCenter(square)
			color := light
			if square.IsDark() {
				color = dark
			}
			if s.assistance.highlights && s.selected != nil && *s.selected == square {
				color = rl.NewColor(226, 183, 72, 255)
			}
			if s.assistance.highlights && s.lastMove != nil && (s.lastMove.From() == square || s.lastMove.To() == square) {
				color = rl.NewColor(137, 177, 86, 255)
			}
			if s.assistance.check && kingInCheck && square == checkedKing {
				color = rl.NewColor(191, 67, 64, 255)
			}
			rl.DrawCube(rl.NewVector3(center.X, -0.02, center.Z), 0.98, 0.08, 0.98, color)
			if s.assistance.highlights && s.isLegalDestination(square) {
				rl.DrawCylinder(rl.NewVector3(center.X, 0.08, center.Z), 0.12, 0.12, 0.04, 16, rl.NewColor(50, 94, 133, 220))
			}
		}
	}
}

func (s state) checkedKingSquare() (chess.Square, bool) {
	color := s.board.ColorToMove
	if !s.board.IsColorInCheck(color) {
		return 0, false
	}
	return s.board.Pieces[color][chess.King].SingleSquare()
}

func (s state) isLegalDestination(square chess.Square) bool {
	if s.selected == nil {
		return false
	}
	for _, move := range chess.GenerateLegalMoves(s.board) {
		if move.From() == *s.selected && move.To() == square {
			return true
		}
	}
	return false
}

func (s state) drawPieces() {
	for square := range chess.Square(64) {
		piece := s.board.PieceAt(square)
		if piece.Type == chess.PieceNone {
			continue
		}
		center := SquareCenter(square)
		color := rl.NewColor(240, 232, 211, 255)
		if piece.Color == chess.Black {
			color = rl.NewColor(43, 48, 56, 255)
		}
		drawPiece(rl.NewVector3(center.X, 0.08, center.Z), piece.Type, color)
	}
}

func drawPiece(position rl.Vector3, pieceType chess.PieceType, color rl.Color) {
	rl.DrawCylinder(position.Add(rl.NewVector3(0, 0.09, 0)), 0.34, 0.42, 0.18, 20, color)
	switch pieceType {
	case chess.Pawn:
		rl.DrawCube(position.Add(rl.NewVector3(0, 0.35, 0)), 0.26, 0.36, 0.26, color)
		rl.DrawSphere(position.Add(rl.NewVector3(0, 0.62, 0)), 0.2, color)
	case chess.Rook:
		rl.DrawCube(position.Add(rl.NewVector3(0, 0.48, 0)), 0.42, 0.62, 0.42, color)
		rl.DrawCube(position.Add(rl.NewVector3(0, 0.82, 0)), 0.5, 0.12, 0.5, color)
	case chess.Knight:
		rl.DrawCube(position.Add(rl.NewVector3(0, 0.47, 0)), 0.34, 0.62, 0.38, color)
		rl.DrawSphere(position.Add(rl.NewVector3(0.12, 0.83, -0.08)), 0.23, color)
	case chess.Bishop:
		rl.DrawCube(position.Add(rl.NewVector3(0, 0.48, 0)), 0.24, 0.62, 0.24, color)
		rl.DrawSphere(position.Add(rl.NewVector3(0, 0.84, 0)), 0.25, color)
	case chess.Queen:
		rl.DrawCube(position.Add(rl.NewVector3(0, 0.5, 0)), 0.3, 0.68, 0.3, color)
		rl.DrawSphere(position.Add(rl.NewVector3(0, 0.9, 0)), 0.27, color)
	case chess.King:
		rl.DrawCube(position.Add(rl.NewVector3(0, 0.53, 0)), 0.32, 0.76, 0.32, color)
		rl.DrawSphere(position.Add(rl.NewVector3(0, 0.98, 0)), 0.22, color)
		rl.DrawCube(position.Add(rl.NewVector3(0, 1.25, 0)), 0.08, 0.3, 0.08, color)
		rl.DrawCube(position.Add(rl.NewVector3(0, 1.25, 0)), 0.3, 0.08, 0.08, color)
	}
}

func (s state) drawArrows() {
	for _, arrow := range s.arrows {
		from := SquareCenter(arrow[0])
		to := SquareCenter(arrow[1])
		start := rl.NewVector3(from.X, 0.17, from.Z)
		end := rl.NewVector3(to.X, 0.17, to.Z)
		rl.DrawLine3D(start, end, rl.NewColor(63, 182, 218, 255))
		rl.DrawSphere(end, 0.12, rl.NewColor(63, 182, 218, 255))
	}
}
