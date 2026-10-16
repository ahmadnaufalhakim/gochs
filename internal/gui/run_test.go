package gui

import (
	"context"
	"testing"
	"time"

	"github.com/ahmadnaufalhakim/gochs/internal/entity"
)

func TestStartLichessActionDispatchesLoad(t *testing.T) {
	state := menuState{page: lichessLoading, lichessAction: lichessLoad}
	results := make(chan lichessResult, 1)
	startLichessAction(&state, fakeLichessService{}, nil, results)
	if state.lichessAction != lichessNoAction {
		t.Errorf("action = %d, want no pending action", state.lichessAction)
	}

	select {
	case result := <-results:
		if result.action != lichessLoad || result.account.Username != "example" || result.err != nil {
			t.Errorf("result = %#v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("Lichess action did not complete")
	}
}

type fakeLichessService struct{}

func (fakeLichessService) Load(context.Context) (entity.LichessAccount, error) {
	return entity.LichessAccount{Username: "example"}, nil
}

func (fakeLichessService) Connect(context.Context, string) (entity.LichessAccount, error) {
	return entity.LichessAccount{}, nil
}

func (fakeLichessService) Disconnect() error {
	return nil
}
