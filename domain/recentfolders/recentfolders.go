// Package recentfolders implements the Recent Folders store: the
// self-tracked, newest-first list of folders shown in the launcher panel.
// Paths are deduplicated case-insensitively (Windows semantics), the list is
// capped at MaxEntries, and the state is persisted as a JSON file.
package recentfolders

import (
	"encoding/json"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MaxEntries caps how many folders the history keeps.
const MaxEntries = 50

// Entry is a single Recent Folders entry.
type Entry struct {
	DisplayName string    `json:"name,omitempty"`
	Path        string    `json:"path"`
	LastUsed    time.Time `json:"lastUsed"`
}

// Name returns the configured display name, or the folder's base name when
// no explicit name is stored.
func (e Entry) Name() string {
	if strings.TrimSpace(e.DisplayName) != "" {
		return e.DisplayName
	}
	base := filepath.Base(e.Path)
	sep := string(filepath.Separator)
	if base == "" || base == "." || base == sep || base == sep+sep {
		return e.Path
	}
	return base
}

// Store is the in-memory Recent Folders list with JSON persistence. It is
// safe for concurrent use. The cap (limit) defaults to MaxEntries and can
// be changed at runtime via SetLimit (e.g. when the user edits the history
// cap in settings).
type Store struct {
	mu    sync.RWMutex
	path  string
	limit int     // current cap; MaxEntries by default
	items []Entry // newest first
}

// New loads the store from the JSON file at filePath. A missing, unreadable
// or corrupted file silently yields an empty store. The cap is MaxEntries
// until SetLimit changes it (settings applied by the host at startup).
func New(filePath string) *Store {
	s := &Store{path: filePath, limit: MaxEntries}
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
	if len(s.items) > s.limit {
		s.items = s.items[:s.limit]
	}
}

// Record records an access to the folder at path, preserving any configured
// display name already stored for it.
func (s *Store) Record(path string) error {
	return s.record(path, "")
}

// RecordNamed records a folder with an explicit display name. It is used for
// targets such as SSH projects whose user-facing name is not derivable from
// their path.
func (s *Store) RecordNamed(path, displayName string) error {
	return s.record(path, strings.TrimSpace(displayName))
}

func (s *Store) record(path, displayName string) error {
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
		bumped = Entry{DisplayName: displayName, Path: clean(path), LastUsed: now}
	} else {
		bumped.LastUsed = now
		if displayName != "" {
			bumped.DisplayName = displayName
		}
	}
	next := append([]Entry{bumped}, kept...)
	if len(next) > s.limit {
		next = next[:s.limit]
	}
	s.items = next
	return s.persistLocked()
}

// SetLimit changes the cap to limit and immediately trims any entries beyond
// it, persisting the trimmed list. It is called when the user edits the
// history cap in settings; the cap itself is owned by the settings store, so
// it is not persisted here (only the resulting trimmed entries are). A
// limit < 1 is rejected without mutating state.
func (s *Store) SetLimit(limit int) error {
	if limit < 1 {
		return fmt.Errorf("recentfolders: limit must be >= 1, got %d", limit)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limit = limit
	if len(s.items) > limit {
		s.items = s.items[:limit]
	}
	return s.persistLocked()
}

// All returns the folders newest first.
func (s *Store) All() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Entry(nil), s.items...)
}

// Find returns the recorded entry matching path.
func (s *Store) Find(path string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := normalize(path)
	for _, e := range s.items {
		if normalize(e.Path) == key {
			return e, true
		}
	}
	return Entry{}, false
}

// clean preserves SSH targets in host:/remote/path form while cleaning their
// POSIX path. Local paths retain Windows filepath semantics.
func clean(value string) string {
	value = strings.TrimSpace(value)
	if host, remotePath, ok := splitSSHPath(value); ok {
		return host + ":" + pathpkg.Clean(remotePath)
	}
	return filepath.Clean(value)
}

// normalize returns the dedup key for a local or SSH folder path. Windows
// paths are case-insensitive; SSH host names are case-insensitive while their
// remote POSIX paths remain case-sensitive.
func normalize(value string) string {
	value = clean(value)
	if host, remotePath, ok := splitSSHPath(value); ok {
		return strings.ToLower(host) + ":" + remotePath
	}
	return strings.ToLower(value)
}

func splitSSHPath(value string) (host, remotePath string, ok bool) {
	colon := strings.Index(value, ":/")
	if colon <= 1 {
		return "", "", false
	}
	host, remotePath = value[:colon], value[colon+1:]
	if strings.ContainsAny(host, `\\/`) || remotePath == "" {
		return "", "", false
	}
	return host, remotePath, true
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
