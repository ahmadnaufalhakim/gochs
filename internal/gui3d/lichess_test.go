package gui3d

import (
	"errors"
	"testing"

	"github.com/ahmadnaufalhakim/gochs/internal/entity"
	lichessapi "github.com/ahmadnaufalhakim/gochs/internal/lichess"
)

func TestApplyLichessLoadUsesOfflineProfile(t *testing.T) {
	game := state{config: Config{
		LoadOfflinePuzzleRating: func() (entity.TrainingProfile, bool, error) {
			return entity.TrainingProfile{Rating: 1450}, true, nil
		},
	}}

	game.applyLichessResult(lichessResult{
		action:  loadLichessAccount,
		account: entity.LichessAccount{Username: "example", Puzzle: 1800},
	})

	if game.lichessAccount == nil || game.lichessAccount.Username != "example" {
		t.Fatalf("account = %#v, want loaded account", game.lichessAccount)
	}
	if game.offlineRating == nil || game.offlineRating.Rating != 1450 {
		t.Fatalf("offlineRating = %#v, want rating 1450", game.offlineRating)
	}
}

func TestApplyLichessLoadWithoutTokenExplainsHowToConfigureOne(t *testing.T) {
	game := state{}

	game.applyLichessResult(lichessResult{action: loadLichessAccount, err: lichessapi.ErrNoToken})

	if game.lichessMessage != "No Lichess token found. Set GOCHS_LICHESS_TOKEN or connect in the terminal app." {
		t.Fatalf("message = %q", game.lichessMessage)
	}
}

func TestApplyLichessSyncUpdatesOfflineRating(t *testing.T) {
	game := state{lichessPage: lichessSyncConfirmation}

	game.applyLichessResult(lichessResult{
		action:  syncLichessRating,
		profile: entity.TrainingProfile{Rating: 1800},
	})

	if game.lichessPage != lichessAccountPage {
		t.Fatalf("page = %d, want account page", game.lichessPage)
	}
	if game.offlineRating == nil || game.offlineRating.Rating != 1800 {
		t.Fatalf("offlineRating = %#v, want rating 1800", game.offlineRating)
	}
}

func TestApplyLichessLoadKeepsUnexpectedErrors(t *testing.T) {
	game := state{}

	game.applyLichessResult(lichessResult{action: loadLichessAccount, err: errors.New("unavailable")})

	if game.lichessMessage != "Lichess: unavailable" {
		t.Fatalf("message = %q", game.lichessMessage)
	}
}
