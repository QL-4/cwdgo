// Package settings implements the SettingsStore: the persisted user
// configuration for cwdgo (history cap, auto-start). It is a pure-domain
// store with JSON persistence, decoupled from Wails/win32; the auto-start
// registry side-effects are applied by the platform glue layer from the
// value held here.
//
// A missing, unreadable or corrupted file silently yields Defaults, so a
// broken settings file never blocks startup (spec: "历史/设置文件损坏时静默
// 重置为默认值").
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DefaultHistoryLimit caps how many Recent Folders the history keeps, unless
// the user overrides it. Matches the spec default ("上限 50").
const DefaultHistoryLimit = 50

// Settings is the full, persisted configuration. Update replaces it
// wholesale, so every persisted field is represented here.
type Settings struct {
	// HistoryLimit is the Recent Folders cap; must be >= 1.
	HistoryLimit int `json:"historyLimit"`
	// AutoStart is the Windows auto-start toggle (default off). The store
	// only holds the value; the registry write is performed by platform glue.
	AutoStart bool `json:"autoStart"`
}

// Defaults returns the first-run Settings.
func Defaults() Settings {
	return Settings{HistoryLimit: DefaultHistoryLimit, AutoStart: false}
}

// Validate returns an error if s is not a usable configuration.
func (s Settings) Validate() error {
	if s.HistoryLimit < 1 {
		return fmt.Errorf("settings: historyLimit must be >= 1, got %d", s.HistoryLimit)
	}
	return nil
}

// Store is the in-memory Settings with JSON persistence. It is safe for
// concurrent use.
type Store struct {
	mu   sync.RWMutex
	path string
	s    Settings
}

// New loads Settings from the JSON file at filePath. A missing, unreadable
// or corrupted file silently yields Defaults.
func New(filePath string) *Store {
	st := &Store{path: filePath, s: Defaults()}
	st.load()
	return st
}

func (st *Store) load() {
	data, err := os.ReadFile(st.path)
	if err != nil {
		return // missing or unreadable: keep defaults
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return // corrupted: silently reset to defaults
	}
	// A persistable but out-of-range file (e.g. historyLimit:0 written by a
	// hand edit) is treated as corrupt → defaults, never blocks startup.
	if err := s.Validate(); err != nil {
		return
	}
	st.s = s
}

// Get returns the current Settings (value copy).
func (st *Store) Get() Settings {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s
}

// Update validates s, persists it atomically and replaces the in-memory
// value. It returns an error without mutating state if s is invalid or
// persistence fails.
func (st *Store) Update(s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if err := st.persistLocked(s); err != nil {
		return err
	}
	st.s = s
	return nil
}

// persistLocked writes s as pretty JSON atomically (temp file + rename).
func (st *Store) persistLocked(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(st.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
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
	return os.Rename(tmpName, st.path)
}
