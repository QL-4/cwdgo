package openactions_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cwdgo/domain/openactions"
)

// fakeLauncher records the program it was asked to run and returns a
// configurable error, so Open's success/failure paths can be exercised
// without spawning Explorer.
type fakeLauncher struct {
	name  string
	args  []string
	calls int
	err   error
}

func (f *fakeLauncher) Launch(name string, args []string) error {
	f.calls++
	f.name, f.args = name, args
	return f.err
}

// fakeRecorder records the paths Open told it to remember.
type fakeRecorder struct {
	paths []string
	err   error
}

func (r *fakeRecorder) Record(path string) error {
	r.paths = append(r.paths, path)
	return r.err
}

func TestExplorerCommandOpensFolder(t *testing.T) {
	name, args := openactions.ExplorerCommand(`D:\projects\foo`)
	if name != openactions.ExplorerExecutable {
		t.Fatalf("name = %q, want %q", name, openactions.ExplorerExecutable)
	}
	if len(args) != 1 || args[0] != `D:\projects\foo` {
		t.Fatalf("args = %v, want [D:\\projects\\foo]", args)
	}
}

func TestOpenValidDirectoryLaunchesExplorerAndRecords(t *testing.T) {
	dir := t.TempDir()
	var ln fakeLauncher
	var rec fakeRecorder

	if err := openactions.Open(dir, &ln, &rec); err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}
	if ln.calls != 1 {
		t.Fatalf("launcher called %d times, want 1", ln.calls)
	}
	if ln.name != openactions.ExplorerExecutable {
		t.Fatalf("launched %q, want %q", ln.name, openactions.ExplorerExecutable)
	}
	if len(ln.args) != 1 || ln.args[0] != dir {
		t.Fatalf("launcher args = %v, want [%s]", ln.args, dir)
	}
	if len(rec.paths) != 1 || rec.paths[0] != dir {
		t.Fatalf("recorded %v, want [%s]", rec.paths, dir)
	}
}

func TestOpenNonexistentPathLaunchesNothing(t *testing.T) {
	var ln fakeLauncher
	var rec fakeRecorder
	err := openactions.Open(filepath.Join(t.TempDir(), "missing"), &ln, &rec)
	if err == nil {
		t.Fatal("Open: want error for nonexistent path, got nil")
	}
	if ln.calls != 0 || len(rec.paths) != 0 {
		t.Fatal("nonexistent path must not launch Explorer or record")
	}
}

func TestOpenPlainFileIsNotLaunched(t *testing.T) {
	file := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var ln fakeLauncher
	var rec fakeRecorder
	if err := openactions.Open(file, &ln, &rec); err == nil {
		t.Fatal("Open: want error for a plain file, got nil")
	}
	if ln.calls != 0 || len(rec.paths) != 0 {
		t.Fatal("a plain file must not launch Explorer or record")
	}
}

func TestOpenLaunchFailureDoesNotRecord(t *testing.T) {
	dir := t.TempDir()
	ln := fakeLauncher{err: errors.New("boom")}
	var rec fakeRecorder
	err := openactions.Open(dir, &ln, &rec)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Open: want error wrapping 'boom', got %v", err)
	}
	if len(rec.paths) != 0 {
		t.Fatal("must not record when the launch failed")
	}
}

func TestOpenRecorderFailureIsReportedAfterLaunch(t *testing.T) {
	dir := t.TempDir()
	var ln fakeLauncher
	rec := fakeRecorder{err: errors.New("disk full")}
	err := openactions.Open(dir, &ln, &rec)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("Open: want error wrapping 'disk full', got %v", err)
	}
	if ln.calls != 1 {
		t.Fatal("Explorer must launch before the recorder is attempted")
	}
}

func TestIsExistingDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !openactions.IsExistingDir(dir) {
		t.Errorf("IsExistingDir(tempDir) = false, want true")
	}
	if openactions.IsExistingDir(file) {
		t.Errorf("IsExistingDir(file) = true, want false")
	}
	if openactions.IsExistingDir(filepath.Join(dir, "nope")) {
		t.Errorf("IsExistingDir(missing) = true, want false")
	}
}
