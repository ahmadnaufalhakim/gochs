package lichess

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBoardResponseErrorIncludesBoundedDetails(t *testing.T) {
	detail := strings.Repeat("x", maxBoardErrorDetailBytes+1)
	err := boardResponseError(&http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(strings.NewReader(detail)),
	}, "make move")
	var boardErr *BoardHTTPError
	if !errors.As(err, &boardErr) {
		t.Fatalf("error = %T, want *BoardHTTPError", err)
	}
	if boardErr.StatusCode != http.StatusTooManyRequests || boardErr.Operation != "make move" {
		t.Errorf("error = %#v", boardErr)
	}
	if len(boardErr.Detail) != maxBoardErrorDetailBytes {
		t.Errorf("detail length = %d, want %d", len(boardErr.Detail), maxBoardErrorDetailBytes)
	}
	if boardErr.IsPermanent() {
		t.Error("429 error should be retryable")
	}
}

func TestBoardHTTPErrorPermanent(t *testing.T) {
	for _, statusCode := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		if !(&BoardHTTPError{StatusCode: statusCode}).IsPermanent() {
			t.Errorf("status %d should be permanent", statusCode)
		}
	}
}

func TestStreamSetupClosesBodyOnStatusError(t *testing.T) {
	for _, stream := range []string{"events", "game"} {
		t.Run(stream, func(t *testing.T) {
			body := &closeTrackingBody{Reader: strings.NewReader("not allowed")}
			client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusForbidden, Body: body, Header: make(http.Header)}, nil
			})}
			service := NewService(client, &memoryStore{token: "token"}, "")

			var err error
			if stream == "events" {
				_, errs := service.StreamEvents(context.Background())
				err = <-errs
			} else {
				_, errs := service.StreamGame(context.Background(), "game-id")
				err = <-errs
			}
			if !body.closed() {
				t.Error("response body was not closed")
			}
			var boardErr *BoardHTTPError
			if !errors.As(err, &boardErr) || boardErr.StatusCode != http.StatusForbidden {
				t.Errorf("error = %v", err)
			}
		})
	}
}

func TestCreateSeekSendsCasualStandardForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s", request.Method)
		}
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("content type = %q", request.Header.Get("Content-Type"))
		}
		if request.Header.Get("Accept") != "application/x-ndjson" {
			t.Errorf("accept = %q", request.Header.Get("Accept"))
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		want := url.Values{"time": {"5"}, "increment": {"5"}, "color": {"random"}, "rated": {"false"}, "variant": {"standard"}}
		if got := request.PostForm; !valuesEqual(got, want) {
			t.Errorf("form = %v, want %v", got, want)
		}
		writer.Write([]byte("\n"))
	}))
	defer server.Close()

	service := NewService(server.Client(), &memoryStore{token: "token"}, "")
	service.seekURL = server.URL
	err := service.CreateSeek(context.Background(), Seek{TimeSeconds: 300, IncrementSeconds: 5, Color: SeekRandom})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateSeekWaitsPastRegularClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(40 * time.Millisecond)
		_, _ = writer.Write([]byte("\n"))
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 5 * time.Millisecond
	service := NewService(client, &memoryStore{token: "token"}, "")
	service.seekURL = server.URL
	err := service.CreateSeek(context.Background(), Seek{TimeSeconds: 60, Color: SeekRandom})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStreamsDoNotInheritRequestTimeout(t *testing.T) {
	client := &http.Client{Timeout: time.Second}
	service := NewService(client, &memoryStore{token: "token"}, "")
	if service.streamHTTPClient.Timeout != 0 {
		t.Errorf("stream timeout = %s, want no client timeout", service.streamHTTPClient.Timeout)
	}
}

func TestBoardActionsUseAuthenticatedEndpoints(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		paths = append(paths, request.Method+" "+request.URL.Path)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service := NewService(server.Client(), &memoryStore{token: "token"}, "")
	service.seekURL = server.URL + "/seek"
	service.gameURL = server.URL + "/game/%s"
	ctx := context.Background()
	if err := service.CancelSeek(ctx, "seek id"); err != nil {
		t.Fatal(err)
	}
	if err := service.MakeMove(ctx, "game id", "e2e4"); err != nil {
		t.Fatal(err)
	}
	if err := service.Resign(ctx, "game id"); err != nil {
		t.Fatal(err)
	}
	if err := service.SetDraw(ctx, "game id", true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetTakeback(ctx, "game id", false); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"DELETE /seek/seek id",
		"POST /game/game id/move/e2e4",
		"POST /game/game id/resign",
		"POST /game/game id/draw/true",
		"POST /game/game id/takeback/false",
	}
	if strings.Join(paths, "\n") != strings.Join(want, "\n") {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestRatingChangesExportsCompletedGame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s", request.Method)
		}
		if request.URL.Path != "/game/export/game-id" || request.URL.Query().Get("pgnInJson") != "true" {
			t.Errorf("URL = %s", request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		_, _ = writer.Write([]byte(`{"players":{"white":{"ratingDiff":7},"black":{"ratingDiff":-7}}}`))
	}))
	defer server.Close()

	service := NewService(server.Client(), &memoryStore{token: "token"}, "")
	service.gameExportURL = server.URL + "/game/export/%s?pgnInJson=true"
	changes, err := service.RatingChanges(context.Background(), "game-id")
	if err != nil {
		t.Fatal(err)
	}
	if changes != (RatingChanges{White: 7, Black: -7}) {
		t.Errorf("changes = %#v", changes)
	}
}

func TestStreamEventsParsesGameStartAndCancels(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/x-ndjson")
		writer.Write([]byte(`{"type":"gameStart","game":{"id":"game-id","white":{"id":"white","name":"White","rating":1800},"black":{"id":"black","name":"Black","rating":1700}}}` + "\n"))
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	service := NewService(server.Client(), &memoryStore{token: "token"}, "")
	service.eventStreamURL = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	events, errs := service.StreamEvents(ctx)
	event := <-events
	start, ok := event.(GameStartEvent)
	if !ok || start.Game.ID != "game-id" || start.Game.White.Color != SeekWhite || start.Game.Black.Rating != 1700 {
		t.Errorf("event = %#v", event)
	}
	<-started
	cancel()
	select {
	case _, ok := <-events:
		if ok {
			t.Error("events channel remains open")
		}
	case <-time.After(time.Second):
		t.Fatal("event stream did not stop after cancellation")
	}
	if _, ok := <-errs; ok {
		t.Error("unexpected stream error")
	}
}

func TestStreamGameParsesFullAndState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing bearer token")
		}
		if request.Header.Get("Accept") != "application/x-ndjson" {
			t.Errorf("accept = %q", request.Header.Get("Accept"))
		}
		writer.Header().Set("Content-Type", "application/x-ndjson")
		writer.Write([]byte(`{"type":"gameFull","id":"game-id","initialFen":"startpos","white":{"id":"white","name":"White","rating":1800,"provisional":true},"black":{"id":"black","name":"Black","rating":1700},"state":{"moves":"e2e4","wtime":299000,"btime":300000,"winc":5000,"binc":5000,"status":"started","wdraw":true,"btakeback":true}}` + "\n"))
		writer.Write([]byte(`{"type":"gameState","moves":"e2e4 e7e5","wtime":298000,"btime":299000,"winc":5000,"binc":5000,"status":"mate","winner":"white","bdraw":true}` + "\n"))
	}))
	defer server.Close()

	service := NewService(server.Client(), &memoryStore{token: "token"}, "")
	service.gameStreamURL = server.URL + "/%s"
	events, errs := service.StreamGame(context.Background(), "game-id")
	full := <-events
	fullEvent, ok := full.(GameFullEvent)
	if !ok || fullEvent.Game.ID != "game-id" || fullEvent.InitialFEN != "startpos" || !fullEvent.Game.White.Provisional || !fullEvent.State.WhiteDraw || !fullEvent.State.BlackTakeback {
		t.Errorf("full event = %#v", full)
	}
	state := <-events
	stateEvent, ok := state.(GameStateEvent)
	if !ok || stateEvent.State.Moves != "e2e4 e7e5" || stateEvent.State.Winner != "white" || !stateEvent.State.BlackDraw {
		t.Errorf("state event = %#v", state)
	}
	if _, ok := <-events; ok {
		t.Error("events channel remains open")
	}
	if _, ok := <-errs; ok {
		t.Error("unexpected stream error")
	}
}

func valuesEqual(got, want url.Values) bool {
	return got.Encode() == want.Encode()
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type closeTrackingBody struct {
	io.Reader
	mu        sync.Mutex
	wasClosed bool
}

func (b *closeTrackingBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.wasClosed = true
	return nil
}

func (b *closeTrackingBody) closed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.wasClosed
}
