package lichess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/entity"
)

const (
	accountURL          = "https://lichess.org/api/account"
	seekURL             = "https://lichess.org/api/board/seek"
	eventStreamURL      = "https://lichess.org/api/stream/event"
	gameStreamURLFormat = "https://lichess.org/api/board/game/stream/%s"
	gameURLFormat       = "https://lichess.org/api/board/game/%s"
	gameExportURLFormat = "https://lichess.org/game/export/%s?pgnInJson=true"
)

type Service struct {
	httpClient       *http.Client
	streamHTTPClient *http.Client
	store            TokenStore
	environmentToken string
	accountURL       string
	seekURL          string
	eventStreamURL   string
	gameStreamURL    string
	gameURL          string
	gameExportURL    string
}

func NewService(httpClient *http.Client, store TokenStore, environmentToken string) *Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	streamHTTPClient := *httpClient
	streamHTTPClient.Timeout = 0
	return &Service{
		httpClient:       httpClient,
		streamHTTPClient: &streamHTTPClient,
		store:            store,
		environmentToken: strings.TrimSpace(environmentToken),
		accountURL:       accountURL,
		seekURL:          seekURL,
		eventStreamURL:   eventStreamURL,
		gameStreamURL:    gameStreamURLFormat,
		gameURL:          gameURLFormat,
		gameExportURL:    gameExportURLFormat,
	}
}

func (s *Service) Load(ctx context.Context) (entity.LichessAccount, error) {
	token, fromEnvironment, err := s.token()
	if err != nil {
		return entity.LichessAccount{}, err
	}
	account, err := s.account(ctx, token)
	if err != nil {
		return entity.LichessAccount{}, err
	}
	account.FromEnvironment = fromEnvironment
	return account, nil
}

func (s *Service) Connect(ctx context.Context, token string) (entity.LichessAccount, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return entity.LichessAccount{}, fmt.Errorf("enter a Lichess Personal API Token")
	}
	account, err := s.account(ctx, token)
	if err != nil {
		return entity.LichessAccount{}, err
	}
	if err := s.store.Save(token); err != nil {
		return entity.LichessAccount{}, err
	}
	return account, nil
}

func (s *Service) Disconnect() error {
	return s.store.Delete()
}

func (s *Service) token() (string, bool, error) {
	if s.environmentToken != "" {
		return s.environmentToken, true, nil
	}
	token, err := s.store.Load()
	return token, false, err
}

func (s *Service) account(ctx context.Context, token string) (entity.LichessAccount, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.accountURL, nil)
	if err != nil {
		return entity.LichessAccount{}, fmt.Errorf("create Lichess account request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return entity.LichessAccount{}, fmt.Errorf("request Lichess account: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return entity.LichessAccount{}, fmt.Errorf("Lichess rejected the token")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return entity.LichessAccount{}, fmt.Errorf("Lichess account request failed: %s", response.Status)
	}

	var responseAccount accountResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&responseAccount); err != nil {
		return entity.LichessAccount{}, fmt.Errorf("decode Lichess account: %w", err)
	}
	return responseAccount.account(), nil
}

type accountResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Title     string `json:"title"`
	Patron    bool   `json:"patron"`
	Online    bool   `json:"online"`
	CreatedAt int64  `json:"createdAt"`
	Perfs     struct {
		Bullet    performance `json:"bullet"`
		Blitz     performance `json:"blitz"`
		Rapid     performance `json:"rapid"`
		Classical performance `json:"classical"`
		Puzzle    performance `json:"puzzle"`
	} `json:"perfs"`
	Count struct {
		All  int `json:"all"`
		Win  int `json:"win"`
		Draw int `json:"draw"`
		Loss int `json:"loss"`
	} `json:"count"`
	PlayTime struct {
		Total int64 `json:"total"`
	} `json:"playTime"`
}

type performance struct {
	Rating int `json:"rating"`
}

func (r accountResponse) account() entity.LichessAccount {
	account := entity.LichessAccount{
		ID:        r.ID,
		Username:  r.Username,
		Title:     r.Title,
		Patron:    r.Patron,
		Online:    r.Online,
		Bullet:    r.Perfs.Bullet.Rating,
		Blitz:     r.Perfs.Blitz.Rating,
		Rapid:     r.Perfs.Rapid.Rating,
		Classical: r.Perfs.Classical.Rating,
		Puzzle:    r.Perfs.Puzzle.Rating,
		Games:     r.Count.All,
		Wins:      r.Count.Win,
		Draws:     r.Count.Draw,
		Losses:    r.Count.Loss,
		PlayTime:  time.Duration(r.PlayTime.Total) * time.Second,
	}
	if r.CreatedAt > 0 {
		account.MemberSince = time.UnixMilli(r.CreatedAt)
	}
	return account
}

func IsNoToken(err error) bool {
	return errors.Is(err, ErrNoToken)
}
