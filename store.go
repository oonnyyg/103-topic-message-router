package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// LoadBroker restores broker state from path; missing file means fresh state.
func LoadBroker(path string) (*Broker, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return NewBroker(), nil
	}
	if err != nil {
		return nil, err
	}
	b := NewBroker()
	if err := json.Unmarshal(data, b); err != nil {
		return nil, err
	}
	if b.Subs == nil {
		b.Subs = map[string]map[string]bool{}
	}
	if b.Retained == nil {
		b.Retained = map[string]string{}
	}
	return b, nil
}

// SaveBroker persists broker state atomically (temp file + rename).
func SaveBroker(path string, b *Broker) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// defaultStatePath resolves the state file location.
func defaultStatePath() string {
	if p := os.Getenv("ROUTER_STATE"); p != "" {
		return p
	}
	return filepath.Join(".", "router-state.json")
}
