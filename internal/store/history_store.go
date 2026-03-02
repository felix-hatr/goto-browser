package store

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// HistoryEntry records a single open event.
type HistoryEntry struct {
	Time   time.Time `yaml:"time"`
	Type   string    `yaml:"type"`   // "link", "group", or "url"
	Target string    `yaml:"target"` // link key, group name, or direct URL
	URLs   []string  `yaml:"urls"`   // resolved URLs opened
}

// HistoryFile is the on-disk format for history.yaml.
type HistoryFile struct {
	Version string         `yaml:"version"`
	History []HistoryEntry `yaml:"history"`
}

// LoadHistory reads the history file. Returns an empty file if not found.
func LoadHistory(path string) (*HistoryFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &HistoryFile{Version: "1", History: []HistoryEntry{}}, nil
		}
		return nil, err
	}
	var hf HistoryFile
	if err := yaml.Unmarshal(data, &hf); err != nil {
		return nil, err
	}
	if hf.History == nil {
		hf.History = []HistoryEntry{}
	}
	return &hf, nil
}

// SaveHistory writes the history file, creating parent directories as needed.
func SaveHistory(path string, hf *HistoryFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(hf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// AppendHistory appends an entry to the history file, applying TTL and limit.
//
// Sentinel values:
//   - limit == -1: history disabled; returns immediately without writing
//   - limit == 0: use default (caller passes the resolved default)
//   - ttlDays == -1: keep forever
//   - ttlDays == 0: use default (caller passes the resolved default)
func AppendHistory(path string, entry HistoryEntry, limit int, ttlDays int) error {
	if limit == -1 {
		return nil // history disabled
	}

	hf, err := LoadHistory(path)
	if err != nil {
		return err
	}

	// Apply TTL: remove entries older than ttlDays
	if ttlDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -ttlDays)
		filtered := hf.History[:0]
		for _, e := range hf.History {
			if e.Time.After(cutoff) {
				filtered = append(filtered, e)
			}
		}
		hf.History = filtered
	}

	hf.History = append(hf.History, entry)

	// Apply limit: keep only the last N entries
	if limit > 0 && len(hf.History) > limit {
		hf.History = hf.History[len(hf.History)-limit:]
	}

	return SaveHistory(path, hf)
}
