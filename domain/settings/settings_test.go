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
	if len(d.Software) != 0 {
		t.Fatalf("Defaults().Software = %v, want empty (preset detection happens in glue)", d.Software)
	}
}

func assertSettingsEqual(t *testing.T, got, want settings.Settings) {
	t.Helper()
	if got.HistoryLimit != want.HistoryLimit || got.AutoStart != want.AutoStart {
		t.Fatalf("Settings = {Limit:%d Auto:%v}, want {Limit:%d Auto:%v}",
			got.HistoryLimit, got.AutoStart, want.HistoryLimit, want.AutoStart)
	}
	if len(got.Software) != len(want.Software) {
		t.Fatalf("Software len = %d, want %d (got %+v want %+v)", len(got.Software), len(want.Software), got.Software, want.Software)
	}
	for i := range want.Software {
		if got.Software[i].Name != want.Software[i].Name || got.Software[i].Exe != want.Software[i].Exe {
			t.Fatalf("Software[%d] = %+v, want %+v", i, got.Software[i], want.Software[i])
		}
		if len(got.Software[i].Args) != len(want.Software[i].Args) {
			t.Fatalf("Software[%d].Args len = %d, want %d", i, len(got.Software[i].Args), len(want.Software[i].Args))
		}
		for j := range want.Software[i].Args {
			if got.Software[i].Args[j] != want.Software[i].Args[j] {
				t.Fatalf("Software[%d].Args[%d] = %q, want %q", i, j, got.Software[i].Args[j], want.Software[i].Args[j])
			}
		}
	}
}

func TestNewWithMissingFileReturnsDefaults(t *testing.T) {
	s := settings.New(filepath.Join(t.TempDir(), "nope.json"))
	assertSettingsEqual(t, s.Get(), settings.Defaults())
}

func TestUpdatePersistsAndRoundTrips(t *testing.T) {
	s, p := newStore(t)
	want := settings.Settings{HistoryLimit: 25, AutoStart: true, Software: []settings.Software{
		{Name: "PowerShell", Exe: "powershell.exe", Args: []string{"-NoExit"}},
	}}
	if err := s.Update(want); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// A fresh Store over the same file reads the persisted values.
	assertSettingsEqual(t, settings.New(p).Get(), want)
}

func TestCorruptedFileResetsToDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(p, []byte("{ this is not valid json !!!"), 0o644); err != nil {
		t.Fatalf("seed corrupt file: %v", err)
	}
	// Loading a corrupted file must not panic and must yield defaults.
	assertSettingsEqual(t, settings.New(p).Get(), settings.Defaults())
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

func TestUpdateRejectsSoftwareWithEmptyName(t *testing.T) {
	s, _ := newStore(t)
	bad := settings.Settings{HistoryLimit: 50, Software: []settings.Software{
		{Name: "", Exe: "a.exe"},
	}}
	if err := s.Update(bad); err == nil {
		t.Fatal("Update with empty Name succeeded, want error")
	}
}

func TestUpdateRejectsSoftwareWithEmptyExe(t *testing.T) {
	s, _ := newStore(t)
	bad := settings.Settings{HistoryLimit: 50, Software: []settings.Software{
		{Name: "App", Exe: ""},
	}}
	if err := s.Update(bad); err == nil {
		t.Fatal("Update with empty Exe succeeded, want error")
	}
}

// A software entry with a valid name and exe but nil args is fine (the app
// takes the folder as a positional argument); args default to nil/empty.
func TestUpdateAcceptsSoftwareWithoutArgs(t *testing.T) {
	s, p := newStore(t)
	want := settings.Settings{HistoryLimit: 50, Software: []settings.Software{
		{Name: "VS Code", Exe: "code"},
	}}
	if err := s.Update(want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertSettingsEqual(t, settings.New(p).Get(), want)
}

func TestUpdateIsFullReplace(t *testing.T) {
	s, _ := newStore(t)
	// Set AutoStart on, HistoryLimit high, with one software entry.
	if err := s.Update(settings.Settings{HistoryLimit: 99, AutoStart: true, Software: []settings.Software{
		{Name: "A", Exe: "a.exe"},
	}}); err != nil {
		t.Fatalf("Update first: %v", err)
	}
	// A second Update replaces wholesale: AutoStart back to false, Software
	// cleared (not merged).
	if err := s.Update(settings.Settings{HistoryLimit: 10, AutoStart: false}); err != nil {
		t.Fatalf("Update second: %v", err)
	}
	assertSettingsEqual(t, s.Get(), settings.Settings{HistoryLimit: 10, AutoStart: false})
}
