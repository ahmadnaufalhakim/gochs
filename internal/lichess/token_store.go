package lichess

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNoToken = errors.New("no Lichess token configured")

type TokenStore interface {
	Load() (string, error)
	Save(token string) error
	Delete() error
}

type FileTokenStore struct {
	path string
}

type storedToken struct {
	Token string `json:"token"`
}

func NewFileTokenStore(configDirectory string) *FileTokenStore {
	return &FileTokenStore{path: filepath.Join(configDirectory, "gochs", "lichess.json")}
}

func NewDefaultTokenStore() (*FileTokenStore, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("find configuration directory: %w", err)
	}
	return NewFileTokenStore(directory), nil
}

func (s *FileTokenStore) Load() (string, error) {
	contents, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNoToken
	}
	if err != nil {
		return "", fmt.Errorf("read Lichess token: %w", err)
	}

	var stored storedToken
	if err := json.Unmarshal(contents, &stored); err != nil {
		return "", fmt.Errorf("parse Lichess token: %w", err)
	}
	if stored.Token == "" {
		return "", ErrNoToken
	}
	return stored.Token, nil
}

func (s *FileTokenStore) Save(token string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create Lichess configuration directory: %w", err)
	}
	if err := os.Chmod(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("secure Lichess configuration directory: %w", err)
	}
	contents, err := json.Marshal(storedToken{Token: token})
	if err != nil {
		return fmt.Errorf("encode Lichess token: %w", err)
	}
	if err := os.WriteFile(s.path, contents, 0o600); err != nil {
		return fmt.Errorf("save Lichess token: %w", err)
	}
	if err := os.Chmod(s.path, 0o600); err != nil {
		return fmt.Errorf("secure Lichess token: %w", err)
	}
	return nil
}

func (s *FileTokenStore) Delete() error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete Lichess token: %w", err)
	}
	return nil
}
