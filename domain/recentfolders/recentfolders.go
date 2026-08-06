// Package recentfolders implements the Recent Folders store: the
// self-tracked, newest-first list of folders shown in the launcher panel.
// Paths are deduplicated case-insensitively (Windows semantics), the list is
// capped at MaxEntries, and the state is persisted as a JSON file.
package recentfolders

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MaxEntries caps how many folders the history keeps.
const MaxEntries = 50

// Entry is a single Recent Folders entry.
type Entry struct {
	Path     string    `json:"path"`
	LastUsed time.Time `json:"lastUsed"`
}

// Name returns the folder's base name (its last path segment).
func (e Entry) Name() string {
	base := filepath.Base(e.Path)
	sep := string(filepath.Separator)
	if base == "" || base == "." || base == sep || base == sep+sep {
		return e.Path
	}
	return base
}

// Store is the in-memory Recent Folders list with JSON persistence. It is
// safe for concurrent use.
type Store struct {
	mu    sync.RWMutex
	path  string
	items []Entry // newest first
}

// New loads the store from the JSON file at filePath. A missing, unreadable
// or corrupted file silently yields an empty store.
func New(filePath string) *Store {
	s := &Store{path: filePath}
	s.load()
	return s
}

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return // missing or unreadable: start empty
	}
	var f struct {
		Entries []Entry `json:"entries"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return // corrupted content: silently reset to defaults
	}
	for _, e := range f.Entries {
		if e.Path != "" {
			s.items = append(s.items, e)
		}
	}
	if len(s.items) > MaxEntries {
		s.items = s.items[:MaxEntries]
	}
}

// Record records an access to the folder at path. The path is cleaned and
// deduplicated case-insensitively: an existing entry is moved to the top with
// a refreshed timestamp, a new entry is added at the top, and the oldest
// entry is evicted when the list would exceed MaxEntries. The updated list is
// persisted to disk before returning; on a persistence failure the memory
// state is kept and the error is returned.
func (s *Store) Record(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := normalize(path)
	now := time.Now()
	kept := make([]Entry, 0, len(s.items))
	var bumped Entry
	found := false
	for _, e := range s.items {
		if normalize(e.Path) == key {
			bumped, found = e, true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		bumped = Entry{Path: filepath.Clean(path), LastUsed: now}
	} else {
		bumped.LastUsed = now // keep first-seen casing, refresh timestamp
	}
	next := append([]Entry{bumped}, kept...)
	if len(next) > MaxEntries {
		next = next[:MaxEntries]
	}
	s.items = next
	return s.persistLocked()
}

// All returns the folders newest first.
func (s *Store) All() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Entry(nil), s.items...)
}

// normalize returns the case-insensitive dedup key for a folder path.
func normalize(path string) string {
	return strings.ToLower(filepath.Clean(path))
}

// persistLocked writes the current list atomically (temp file + rename) as
// JSON.
func (s *Store) persistLocked() error {
	data, err := json.MarshalIndent(struct {
		Entries []Entry `json:"entries"`
	}{Entries: s.items}, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".history-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}
