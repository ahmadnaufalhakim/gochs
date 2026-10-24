package lichess

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type SeekColor string

const (
	SeekWhite  SeekColor = "white"
	SeekBlack  SeekColor = "black"
	SeekRandom SeekColor = "random"
)

type Seek struct {
	TimeSeconds      int
	IncrementSeconds int
	Color            SeekColor
}

type Event interface {
	event()
}

type GameStartEvent struct {
	Game Game
}

func (GameStartEvent) event() {}

type GameEvent interface {
	gameEvent()
}

type GameFullEvent struct {
	Game       Game
	InitialFEN string
	State      GameState
}

func (GameFullEvent) gameEvent() {}

type GameStateEvent struct {
	State GameState
}

func (GameStateEvent) gameEvent() {}

type Game struct {
	ID        string
	White     Player
	Black     Player
	CreatedAt int64
}

type Player struct {
	ID          string
	Name        string
	Rating      int
	Provisional bool
	Color       SeekColor
}

type GameState struct {
	Moves          string
	WhiteTime      int
	BlackTime      int
	WhiteIncrement int
	BlackIncrement int
	Status         string
	Winner         string
	WhiteDraw      bool
	BlackDraw      bool
	WhiteTakeback  bool
	BlackTakeback  bool
}

func (s *Service) CreateSeek(ctx context.Context, seek Seek) error {
	if seek.TimeSeconds <= 0 || seek.TimeSeconds%60 != 0 || seek.IncrementSeconds < 0 {
		return fmt.Errorf("Lichess seek time must be a positive number of whole minutes and increment cannot be negative")
	}
	if seek.Color != SeekWhite && seek.Color != SeekBlack && seek.Color != SeekRandom {
		return fmt.Errorf("invalid Lichess seek color %q", seek.Color)
	}
	values := url.Values{
		"time":      {fmt.Sprintf("%d", seek.TimeSeconds/60)},
		"increment": {fmt.Sprintf("%d", seek.IncrementSeconds)},
		"color":     {string(seek.Color)},
		"rated":     {"false"},
		"variant":   {"standard"},
	}
	response, err := s.streamRequest(ctx, http.MethodPost, s.seekURL, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := boardResponseError(response, "create seek"); err != nil {
		return err
	}
	// Real-time seeks only send empty keep-alive lines. Lichess closes this
	// response when matched; the game ID arrives on the account event stream.
	if _, err := io.Copy(io.Discard, response.Body); err != nil && ctx.Err() == nil {
		return fmt.Errorf("read Lichess seek: %w", err)
	}
	return nil
}

func (s *Service) CancelSeek(ctx context.Context, seekID string) error {
	response, err := s.request(ctx, http.MethodDelete, s.seekURL+"/"+url.PathEscape(seekID), nil, "")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return boardResponseError(response, "cancel seek")
}

func (s *Service) StreamEvents(ctx context.Context) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errors := make(chan error, 1)
	ready := make(chan struct{})
	go s.streamEvents(ctx, events, errors, ready)
	<-ready
	return events, errors
}

func (s *Service) StreamGame(ctx context.Context, gameID string) (<-chan GameEvent, <-chan error) {
	events := make(chan GameEvent)
	errors := make(chan error, 1)
	go s.streamGame(ctx, gameID, events, errors)
	return events, errors
}

func (s *Service) MakeMove(ctx context.Context, gameID, move string) error {
	return s.gameAction(ctx, gameID, "/move/"+url.PathEscape(move))
}

func (s *Service) Resign(ctx context.Context, gameID string) error {
	return s.gameAction(ctx, gameID, "/resign")
}

func (s *Service) SetDraw(ctx context.Context, gameID string, accept bool) error {
	return s.gameAction(ctx, gameID, "/draw/"+fmt.Sprintf("%t", accept))
}

func (s *Service) SetTakeback(ctx context.Context, gameID string, accept bool) error {
	return s.gameAction(ctx, gameID, "/takeback/"+fmt.Sprintf("%t", accept))
}

func (s *Service) streamEvents(ctx context.Context, events chan<- Event, errors chan<- error, ready chan<- struct{}) {
	defer close(events)
	defer close(errors)
	response, err := s.streamRequest(ctx, http.MethodGet, s.eventStreamURL, nil, "")
	if err == nil {
		err = boardResponseError(response, "stream events")
	}
	close(ready)
	if err != nil {
		errors <- err
		return
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(bufio.NewReader(response.Body))
	for {
		var raw struct {
			Type string       `json:"type"`
			Game gameResponse `json:"game"`
		}
		if err := decoder.Decode(&raw); err != nil {
			if ctx.Err() != nil {
				return
			}
			if err != io.EOF {
				errors <- fmt.Errorf("decode Lichess event stream: %w", err)
			}
			return
		}
		if raw.Type == "gameStart" {
			select {
			case events <- GameStartEvent{Game: raw.Game.game()}:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *Service) streamGame(ctx context.Context, gameID string, events chan<- GameEvent, errors chan<- error) {
	defer close(events)
	defer close(errors)
	response, err := s.streamRequest(ctx, http.MethodGet, fmt.Sprintf(s.gameStreamURL, url.PathEscape(gameID)), nil, "")
	if err == nil {
		err = boardResponseError(response, "stream game")
	}
	if err != nil {
		errors <- err
		return
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(bufio.NewReader(response.Body))
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if ctx.Err() != nil {
				return
			}
			if err != io.EOF {
				errors <- fmt.Errorf("decode Lichess game stream: %w", err)
			}
			return
		}
		var kind struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &kind); err != nil {
			errors <- fmt.Errorf("decode Lichess game event: %w", err)
			return
		}
		var event GameEvent
		switch kind.Type {
		case "gameFull":
			var full gameFullResponse
			if err := json.Unmarshal(raw, &full); err != nil {
				errors <- fmt.Errorf("decode Lichess game full: %w", err)
				return
			}
			event = GameFullEvent{Game: full.game(), InitialFEN: full.InitialFEN, State: full.State.state()}
		case "gameState":
			var state gameStateResponse
			if err := json.Unmarshal(raw, &state); err != nil {
				errors <- fmt.Errorf("decode Lichess game state: %w", err)
				return
			}
			event = GameStateEvent{State: state.state()}
		default:
			continue
		}
		select {
		case events <- event:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) gameAction(ctx context.Context, gameID, action string) error {
	response, err := s.request(ctx, http.MethodPost, fmt.Sprintf(s.gameURL, url.PathEscape(gameID))+action, nil, "")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return boardResponseError(response, "game action")
}

func (s *Service) request(ctx context.Context, method, target string, body io.Reader, contentType string) (*http.Response, error) {
	return s.requestWithClient(ctx, s.httpClient, method, target, body, contentType, "application/json")
}

func (s *Service) streamRequest(ctx context.Context, method, target string, body io.Reader, contentType string) (*http.Response, error) {
	client := s.streamHTTPClient
	if client == nil {
		client = s.httpClient
	}
	return s.requestWithClient(ctx, client, method, target, body, contentType, "application/x-ndjson")
}

func (s *Service) requestWithClient(ctx context.Context, client *http.Client, method, target string, body io.Reader, contentType, accept string) (*http.Response, error) {
	token, _, err := s.token()
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("create Lichess board request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", accept)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request Lichess board: %w", err)
	}
	return response, nil
}

func boardResponseError(response *http.Response, action string) error {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Lichess %s failed: %s", action, response.Status)
	}
	return nil
}

type gameResponse struct {
	ID        string         `json:"id"`
	White     playerResponse `json:"white"`
	Black     playerResponse `json:"black"`
	CreatedAt int64          `json:"createdAt"`
}

func (r gameResponse) game() Game {
	return Game{ID: r.ID, White: r.White.player(SeekWhite), Black: r.Black.player(SeekBlack), CreatedAt: r.CreatedAt}
}

type playerResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Rating      int    `json:"rating"`
	Provisional bool   `json:"provisional"`
}

func (r playerResponse) player(color SeekColor) Player {
	return Player{ID: r.ID, Name: r.Name, Rating: r.Rating, Provisional: r.Provisional, Color: color}
}

type gameStateResponse struct {
	Moves     string `json:"moves"`
	WTime     int    `json:"wtime"`
	BTime     int    `json:"btime"`
	WInc      int    `json:"winc"`
	BInc      int    `json:"binc"`
	Status    string `json:"status"`
	Winner    string `json:"winner"`
	WDraw     bool   `json:"wdraw"`
	BDraw     bool   `json:"bdraw"`
	WTakeback bool   `json:"wtakeback"`
	BTakeback bool   `json:"btakeback"`
}

func (r gameStateResponse) state() GameState {
	return GameState{Moves: r.Moves, WhiteTime: r.WTime, BlackTime: r.BTime, WhiteIncrement: r.WInc, BlackIncrement: r.BInc, Status: r.Status, Winner: r.Winner, WhiteDraw: r.WDraw, BlackDraw: r.BDraw, WhiteTakeback: r.WTakeback, BlackTakeback: r.BTakeback}
}

type gameFullResponse struct {
	gameResponse
	InitialFEN string            `json:"initialFen"`
	State      gameStateResponse `json:"state"`
}
