package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var (
	errEmptyClient   = errors.New("client id must not be empty")
	errNotSubscribed = errors.New("client is not subscribed to this filter")
)

// LoadBroker reads state from path; a missing file yields a fresh broker.
func LoadBroker(path string) (*Broker, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewBroker(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	b := NewBroker()
	if err := json.Unmarshal(data, b); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if b.Subs == nil {
		b.Subs = map[string]map[string]bool{}
	}
	if b.Retained == nil {
		b.Retained = map[string]string{}
	}
	return b, nil
}

// SaveBroker persists state atomically (write temp file, then rename).
func SaveBroker(path string, b *Broker) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("commit state: %w", err)
	}
	return nil
}
