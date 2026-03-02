package store

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HistoryEntry records a single open event.
type HistoryEntry struct {
	Time   time.Time `json:"time"`
	Type   string    `json:"type"`   // "link", "group", or "url"
	Target string    `json:"target"` // link key, group name, or direct URL
	URLs   []string  `json:"urls"`   // resolved URLs opened
}

// LoadHistory reads all entries from the JSONL history file.
// Returns an empty slice if the file does not exist.
func LoadHistory(path string) ([]HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []HistoryEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e HistoryEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	return entries, sc.Err()
}

// SaveHistory writes entries to the file in JSONL format, overwriting existing content.
func SaveHistory(path string, entries []HistoryEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			continue
		}
		w.Write(line)
		w.WriteByte('\n')
	}
	return w.Flush()
}

// AppendHistory appends entry to the history file according to the dedup strategy.
//
// dedup values:
//   - "" or "none":        always append (pure O(1) write)
//   - "consecutive":       skip if same target as the last recorded entry
//   - "all":               remove all previous occurrences of same target, then append
//
// size > 0 is enforced only when dedup="all" (since we rewrite the file anyway).
// For other modes, use `compact` to enforce size.
func AppendHistory(path string, entry HistoryEntry, size int, dedup string) error {
	switch dedup {
	case "consecutive":
		last, err := readLastEntry(path)
		if err == nil && last != nil && last.Target == entry.Target {
			return nil // skip consecutive duplicate
		}
		return appendLine(path, entry)

	case "all":
		entries, err := LoadHistory(path)
		if err != nil {
			return err
		}
		// Remove all previous occurrences of the same target
		filtered := entries[:0]
		for _, e := range entries {
			if e.Target != entry.Target {
				filtered = append(filtered, e)
			}
		}
		filtered = append(filtered, entry)
		// Apply size limit (only enforced in this path)
		if size > 0 && len(filtered) > size {
			filtered = filtered[len(filtered)-size:]
		}
		return SaveHistory(path, filtered)

	default: // "none" or empty
		return appendLine(path, entry)
	}
}

// appendLine appends a single entry as a JSON line.
func appendLine(path string, entry HistoryEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	_, err = f.Write(line)
	return err
}

// readLastEntry reads only the last non-empty line of the file, avoiding a full read.
// Returns nil, nil if the file is empty or does not exist.
func readLastEntry(path string) (*HistoryEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil || fi.Size() == 0 {
		return nil, nil
	}

	// Read at most 4 KB from the end — enough for any single JSON entry
	bufSize := int64(4096)
	if fi.Size() < bufSize {
		bufSize = fi.Size()
	}
	buf := make([]byte, bufSize)
	if _, err := f.ReadAt(buf, fi.Size()-bufSize); err != nil {
		return nil, err
	}

	s := strings.TrimRight(string(buf), "\n")
	lastNL := strings.LastIndex(s, "\n")
	var lastLine string
	if lastNL >= 0 {
		lastLine = s[lastNL+1:]
	} else {
		lastLine = s
	}
	if lastLine == "" {
		return nil, nil
	}

	var e HistoryEntry
	if err := json.Unmarshal([]byte(lastLine), &e); err != nil {
		return nil, err
	}
	return &e, nil
}
