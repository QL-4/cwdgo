package recentfolders_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cwdgo/domain/recentfolders"
)

func newStore(t *testing.T) (*recentfolders.Store, string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "history.json")
	return recentfolders.New(p), p
}

func TestNewWithMissingFileStartsEmpty(t *testing.T) {
	s := recentfolders.New(filepath.Join(t.TempDir(), "nope.json"))
	if got := s.All(); len(got) != 0 {
		t.Fatalf("All() = %v, want empty store", got)
	}
}

func TestRecordListsNewestFirst(t *testing.T) {
	s, _ := newStore(t)
	for _, p := range []string{`C:\Users\jerem`, `D:\work\proj`, `E:\media\photos`} {
		if err := s.Record(p); err != nil {
			t.Fatalf("Record(%q): %v", p, err)
		}
	}
	got := s.All()
	want := []string{`E:\media\photos`, `D:\work\proj`, `C:\Users\jerem`}
	if len(got) != len(want) {
		t.Fatalf("All() = %v, want %d entries", got, len(want))
	}
	for i, w := range want {
		if got[i].Path != w {
			t.Fatalf("All()[%d].Path = %q, want %q (all: %v)", i, got[i].Path, w, got)
		}
	}
}

func TestRecordDedupesByCaseInsensitivePathAndBumpsToTop(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Record(`C:\Users\jerem`); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(`D:\work`); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(`c:\users\JEREM`); err != nil { // same folder, different case
		t.Fatal(err)
	}
	got := s.All()
	if len(got) != 2 {
		t.Fatalf("All() = %v, want 2 entries (deduped)", got)
	}
	if got[0].Path != `C:\Users\jerem` {
		t.Fatalf("bumped entry = %q, want first-seen casing %q", got[0].Path, `C:\Users\jerem`)
	}
	if got[1].Path != `D:\work` {
		t.Fatalf("second entry = %q, want %q", got[1].Path, `D:\work`)
	}
}

func TestRecordNormalizesSeparatorsAndTrailingSlash(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Record(`C:\Users\jerem\Documents\`); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(`c:/users/jerem/documents`); err != nil { // forward slashes, different case, no trailing sep
		t.Fatal(err)
	}
	got := s.All()
	if len(got) != 1 {
		t.Fatalf("All() = %v, want 1 entry", got)
	}
	if got[0].Path != `C:\Users\jerem\Documents` {
		t.Fatalf("Path = %q, want cleaned %q", got[0].Path, `C:\Users\jerem\Documents`)
	}
}

func TestRecordRefreshesTimestamp(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Record(`C:\A`); err != nil {
		t.Fatal(err)
	}
	first := s.All()[0].LastUsed
	time.Sleep(20 * time.Millisecond)
	if err := s.Record(`c:\a`); err != nil {
		t.Fatal(err)
	}
	second := s.All()[0].LastUsed
	if !second.After(first) {
		t.Fatalf("timestamp not refreshed: %v then %v", first, second)
	}
}

func TestRecordEvictsOldestBeyondCap(t *testing.T) {
	s, _ := newStore(t)
	for i := 0; i < recentfolders.MaxEntries+1; i++ {
		if err := s.Record(filepath.Join(`C:\evict`, fmt.Sprintf("folder%02d", i))); err != nil {
			t.Fatal(err)
		}
	}
	got := s.All()
	if len(got) != recentfolders.MaxEntries {
		t.Fatalf("All() has %d entries, want %d", len(got), recentfolders.MaxEntries)
	}
	if got[0].Path != filepath.Join(`C:\evict`, "folder50") {
		t.Fatalf("top = %q, want folder50 (newest kept)", got[0].Path)
	}
	if got[len(got)-1].Path != filepath.Join(`C:\evict`, "folder01") {
		t.Fatalf("bottom = %q, want folder01 (oldest remaining)", got[len(got)-1].Path)
	}
}

func TestRecordSameFolderRepeatedlyKeepsOneEntry(t *testing.T) {
	s, _ := newStore(t)
	variants := []string{`C:\Loop`, `c:\loop`, `C:/Loop/`}
	for i := 0; i < 3*recentfolders.MaxEntries; i++ {
		if err := s.Record(variants[i%len(variants)]); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.All(); len(got) != 1 {
		t.Fatalf("All() = %v, want exactly 1 entry", got)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	s := recentfolders.New(path)
	for _, p := range []string{`C:\Users\jerem`, `D:\work\proj`, `E:\media`} {
		if err := s.Record(p); err != nil {
			t.Fatal(err)
		}
	}
	s2 := recentfolders.New(path)
	got, want := s2.All(), s.All()
	if len(got) != len(want) {
		t.Fatalf("reloaded %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Path != want[i].Path {
			t.Fatalf("entry %d: path %q, want %q", i, got[i].Path, want[i].Path)
		}
		if !got[i].LastUsed.Equal(want[i].LastUsed) {
			t.Fatalf("entry %d: LastUsed %v, want %v", i, got[i].LastUsed, want[i].LastUsed)
		}
	}
}

func TestCorruptedFileSilentlyResets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	if err := os.WriteFile(path, []byte("{definitely not json!!!"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := recentfolders.New(path)
	if got := s.All(); len(got) != 0 {
		t.Fatalf("All() = %v, want empty after corrupted file", got)
	}
	// the store still works: recording repairs the file
	if err := s.Record(`C:\recovered`); err != nil {
		t.Fatal(err)
	}
	s2 := recentfolders.New(path)
	if got := s2.All(); len(got) != 1 || got[0].Path != `C:\recovered` {
		t.Fatalf("after recovery All() = %v, want [C:\\recovered]", got)
	}
}

func TestLoadTrimsEntriesBeyondCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	var sb strings.Builder
	sb.WriteString(`{"entries":[`)
	for i := 0; i < recentfolders.MaxEntries+10; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"path":"C:\\bulk\\f%02d","lastUsed":"2026-01-01T00:00:00Z"}`, i)
	}
	sb.WriteString(`]}`)
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	s := recentfolders.New(path)
	got := s.All()
	if len(got) != recentfolders.MaxEntries {
		t.Fatalf("All() has %d entries, want %d", len(got), recentfolders.MaxEntries)
	}
	if got[0].Path != `C:\bulk\f00` {
		t.Fatalf("top = %q, want C:\\bulk\\f00 (newest kept)", got[0].Path)
	}
}

func TestRecordReportsPersistenceFailure(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "blocker"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	s := recentfolders.New(filepath.Join(base, "blocker", "history.json"))
	if err := s.Record(`C:\x`); err == nil {
		t.Fatal("Record: want persist error, got nil")
	}
	if got := s.All(); len(got) != 1 {
		t.Fatalf("All() = %v, want entry kept in memory despite persist error", got)
	}
}

func TestSetLimitTrimsExistingImmediately(t *testing.T) {
	s, _ := newStore(t)
	for _, p := range []string{`C:\a`, `C:\b`, `C:\c`, `C:\d`, `C:\e`} {
		if err := s.Record(p); err != nil {
			t.Fatalf("Record(%q): %v", p, err)
		}
	}
	if err := s.SetLimit(3); err != nil {
		t.Fatalf("SetLimit(3): %v", err)
	}
	got := s.All()
	if len(got) != 3 {
		t.Fatalf("after SetLimit All() = %d entries, want 3 (trimmed)", len(got))
	}
	want := []string{`C:\e`, `C:\d`, `C:\c`} // newest three
	for i, w := range want {
		if got[i].Path != w {
			t.Fatalf("All()[%d].Path = %q, want %q", i, got[i].Path, w)
		}
	}
}

func TestSetLimitPersistsTrimmedList(t *testing.T) {
	s, p := newStore(t)
	for _, x := range []string{`C:\1`, `C:\2`, `C:\3`, `C:\4`} {
		if err := s.Record(x); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetLimit(2); err != nil {
		t.Fatalf("SetLimit(2): %v", err)
	}
	// A fresh store over the same file reads the persisted (trimmed) list.
	got := recentfolders.New(p).All()
	if len(got) != 2 {
		t.Fatalf("reload All() = %d, want 2 (persisted trim)", len(got))
	}
}

func TestSetLimitEnforcedOnSubsequentRecord(t *testing.T) {
	s, _ := newStore(t)
	if err := s.SetLimit(2); err != nil {
		t.Fatalf("SetLimit(2): %v", err)
	}
	for _, p := range []string{`C:\a`, `C:\b`, `C:\c`, `C:\d`} {
		if err := s.Record(p); err != nil {
			t.Fatalf("Record(%q): %v", p, err)
		}
	}
	if got := s.All(); len(got) != 2 {
		t.Fatalf("All() = %d, want 2 (new cap enforced)", len(got))
	}
}

func TestSetLimitRejectsZero(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Record(`C:\x`); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLimit(0); err == nil {
		t.Fatal("SetLimit(0) succeeded, want error")
	}
	// State is unchanged: limit still default, entry still there.
	if got := s.All(); len(got) != 1 {
		t.Fatalf("All() after rejected SetLimit = %d, want 1 (unchanged)", len(got))
	}
}
