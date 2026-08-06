package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"cwdgo/domain/settings"
)

func newStore(t *testing.T) (*settings.Store, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "settings.json")
	return settings.New(p), p
}

func TestDefaultsValues(t *testing.T) {
	d := settings.Defaults()
	if d.HistoryLimit != 50 {
		t.Fatalf("Defaults().HistoryLimit = %d, want 50", d.HistoryLimit)
	}
	if d.AutoStart {
		t.Fatalf("Defaults().AutoStart = true, want false")
	}
}

func TestNewWithMissingFileReturnsDefaults(t *testing.T) {
	s := settings.New(filepath.Join(t.TempDir(), "nope.json"))
	got := s.Get()
	if got != settings.Defaults() {
		t.Fatalf("Get() = %+v, want defaults %+v", got, settings.Defaults())
	}
}

func TestUpdatePersistsAndRoundTrips(t *testing.T) {
	s, p := newStore(t)
	want := settings.Settings{HistoryLimit: 25, AutoStart: true}
	if err := s.Update(want); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// A fresh Store over the same file reads the persisted values.
	got := settings.New(p).Get()
	if got != want {
		t.Fatalf("round-trip Get = %+v, want %+v", got, want)
	}
}

func TestCorruptedFileResetsToDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(p, []byte("{ this is not valid json !!!"), 0o644); err != nil {
		t.Fatalf("seed corrupt file: %v", err)
	}
	// Loading a corrupted file must not panic and must yield defaults.
	s := settings.New(p)
	if got := s.Get(); got != settings.Defaults() {
		t.Fatalf("Get() after corrupt load = %+v, want defaults", got)
	}
}

func TestUpdateRejectsInvalidHistoryLimit(t *testing.T) {
	s, _ := newStore(t)
	cases := []int{0, -1, -100}
	for _, n := range cases {
		if err := s.Update(settings.Settings{HistoryLimit: n}); err == nil {
			t.Fatalf("Update(HistoryLimit=%d) succeeded, want error", n)
		}
	}
}

func TestUpdateIsFullReplace(t *testing.T) {
	s, _ := newStore(t)
	// Set AutoStart on, HistoryLimit high.
	if err := s.Update(settings.Settings{HistoryLimit: 99, AutoStart: true}); err != nil {
		t.Fatalf("Update first: %v", err)
	}
	// A second Update replaces wholesale: AutoStart must go back to false.
	if err := s.Update(settings.Settings{HistoryLimit: 10, AutoStart: false}); err != nil {
		t.Fatalf("Update second: %v", err)
	}
	got := s.Get()
	want := settings.Settings{HistoryLimit: 10, AutoStart: false}
	if got != want {
		t.Fatalf("Get = %+v, want %+v", got, want)
	}
}
