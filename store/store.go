package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

const writeFileMode = 0o600

type Store struct {
	HeaderLastModified string `json:"lastModified"`
	FeedLastBuildDate  string `json:"feedLastBuildDate"`
}

func (s *Store) LoadStoreFromFile(filename string) error {
	slog.Debug("checking if store file exist")
	_, err := os.Stat(filename)
	if err != nil {
		// Fail if it's any error other than ErrNotExist
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checking if store exists failed: %w", err)
		}

		// First run!
		slog.Debug("store does not exist: first run")

		return nil
	}

	slog.Debug("store exists, reading data")
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read store: %w", err)
	}
	slog.Debug("unmarshalling store items")
	err = json.Unmarshal(jsonData, &s)
	if err != nil {
		return fmt.Errorf("could not unmarshal store: %w", err)
	}

	return nil
}

func (s *Store) SaveStoreToFile(filename string) error {
	slog.Debug("marshalling store")
	jsonString, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("could not marshal store: %w", err)
	}

	slog.Debug("saving store")
	err = os.WriteFile(filename, jsonString, writeFileMode)
	if err != nil {
		return fmt.Errorf("could not save the store: %w", err)
	}

	return nil
}
