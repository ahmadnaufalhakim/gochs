package lichess

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMapsAccountAndSendsBearerToken(t *testing.T) {
	store := &memoryStore{token: "saved-token"}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer saved-token" {
			t.Errorf("Authorization = %q", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte(`{"id":"example-id","username":"example","title":"FM","patron":true,"online":true,"createdAt":1711929600000,"perfs":{"bullet":{"rating":1820},"blitz":{"rating":1764},"rapid":{"rating":1690},"classical":{"rating":1612},"puzzle":{"rating":1950}},"count":{"all":1245,"win":602,"draw":121,"loss":522},"playTime":{"total":11100}}`))
	}))
	defer server.Close()

	service := NewService(server.Client(), store, "")
	service.accountURL = server.URL
	account, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != "example-id" || account.Username != "example" || account.Title != "FM" || !account.Patron || !account.Online {
		t.Errorf("identity = %#v", account)
	}
	if account.Bullet != 1820 || account.Blitz != 1764 || account.Rapid != 1690 || account.Classical != 1612 || account.Puzzle != 1950 {
		t.Errorf("ratings = %#v", account)
	}
	if account.Games != 1245 || account.Wins != 602 || account.Draws != 121 || account.Losses != 522 {
		t.Errorf("counts = %#v", account)
	}
	if account.PlayTime.String() != "3h5m0s" || account.MemberSince.Format("2006-01-02") != "2024-04-01" {
		t.Errorf("activity = %#v", account)
	}
}

func TestConnectValidatesBeforeSaving(t *testing.T) {
	store := &memoryStore{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer new-token" {
			t.Error("request did not use supplied token")
		}
		writer.Write([]byte(`{"username":"example"}`))
	}))
	defer server.Close()

	service := NewService(server.Client(), store, "")
	service.accountURL = server.URL
	if _, err := service.Connect(context.Background(), " new-token "); err != nil {
		t.Fatal(err)
	}
	if store.token != "new-token" {
		t.Errorf("stored token = %q", store.token)
	}
}

func TestLoadEnvironmentTokenOverridesSavedToken(t *testing.T) {
	store := &memoryStore{token: "saved-token"}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer environment-token" {
			t.Error("environment token did not override saved token")
		}
		writer.Write([]byte(`{"username":"example"}`))
	}))
	defer server.Close()

	service := NewService(server.Client(), store, "environment-token")
	service.accountURL = server.URL
	account, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !account.FromEnvironment {
		t.Error("account should report environment credentials")
	}
}

func TestRejectedTokenDoesNotAppearInError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	service := NewService(server.Client(), &memoryStore{}, "")
	service.accountURL = server.URL
	_, err := service.Connect(context.Background(), "private-token")
	if err == nil || !strings.Contains(err.Error(), "rejected") || strings.Contains(err.Error(), "private-token") {
		t.Errorf("error = %v", err)
	}
}

func TestFileTokenStorePermissionsAndDelete(t *testing.T) {
	store := NewFileTokenStore(t.TempDir())
	if err := store.Save("private-token"); err != nil {
		t.Fatal(err)
	}
	if token, err := store.Load(); err != nil || token != "private-token" {
		t.Errorf("Load() = %q, %v", token, err)
	}
	info, err := os.Stat(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file permissions = %o, want 600", info.Mode().Perm())
	}
	directoryInfo, err := os.Stat(filepath.Dir(store.path))
	if err != nil {
		t.Fatal(err)
	}
	if directoryInfo.Mode().Perm() != 0o700 {
		t.Errorf("directory permissions = %o, want 700", directoryInfo.Mode().Perm())
	}
	if err := store.Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNoToken) {
		t.Errorf("Load() after delete error = %v", err)
	}
}

type memoryStore struct {
	token string
}

func (s *memoryStore) Load() (string, error) {
	if s.token == "" {
		return "", ErrNoToken
	}
	return s.token, nil
}

func (s *memoryStore) Save(token string) error {
	s.token = token
	return nil
}

func (s *memoryStore) Delete() error {
	s.token = ""
	return nil
}
