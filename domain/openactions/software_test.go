package openactions_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"cwdgo/domain/openactions"
)

// --- Software.Command (pure function) ---

func TestSoftwareCommandNoArgsAppendsFolder(t *testing.T) {
	sw := openactions.Software{Exe: "code"}
	name, args := sw.Command(`D:\foo`)
	if name != "code" {
		t.Fatalf("name = %q, want %q", name, "code")
	}
	want := []string{`D:\foo`}
	if !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestSoftwareCommandPlaceholderIsSubstituted(t *testing.T) {
	sw := openactions.Software{
		Exe:  "powershell.exe",
		Args: []string{"-NoExit", "-Command", "Set-Location '{folder}'"},
	}
	name, args := sw.Command(`D:\projects\foo`)
	if name != "powershell.exe" {
		t.Fatalf("name = %q, want powershell.exe", name)
	}
	want := []string{"-NoExit", "-Command", "Set-Location 'D:\\projects\\foo'"}
	if !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	// folder must not be appended again when a placeholder was used
	if len(args) != 3 {
		t.Fatalf("got %d args, want 3 (no duplicate folder append)", len(args))
	}
}

func TestSoftwareCommandArgsWithoutPlaceholderAppendFolder(t *testing.T) {
	sw := openactions.Software{Exe: "code", Args: []string{"-n"}}
	_, args := sw.Command(`D:\foo`)
	want := []string{"-n", `D:\foo`}
	if !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func TestSoftwareCommandMultiplePlaceholdersAllReplaced(t *testing.T) {
	sw := openactions.Software{Exe: "x", Args: []string{"{folder}", "echo {folder}"}}
	_, args := sw.Command(`D:\foo`)
	want := []string{`D:\foo`, `echo D:\foo`}
	if !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestSSHFolderURI(t *testing.T) {
	got, err := openactions.SSHFolderURI(`172.24.245.143:/home/beishida/QL`, "beishida")
	if err != nil {
		t.Fatal(err)
	}
	want := "vscode-remote://ssh-remote%2B7b22686f73744e616d65223a226265697368696461222c2266726f6d436f6e666967223a747275657d/home/beishida/QL"
	if got != want {
		t.Fatalf("SSHFolderURI() = %q, want %q", got, want)
	}
}

func TestRemoteCommandUsesFolderURI(t *testing.T) {
	sw := openactions.Software{Exe: `D:\Programs\Trae CN\Trae CN.exe`}
	name, args, err := sw.RemoteCommand(`172.24.245.143:/home/beishida/QL`, "beishida")
	if err != nil {
		t.Fatal(err)
	}
	if name != sw.Exe || len(args) != 2 || args[0] != "--folder-uri" || !strings.Contains(args[1], "/home/beishida/QL") {
		t.Fatalf("RemoteCommand() = %q %v", name, args)
	}
}

// --- OpenSoftware (success/failure via fake Launcher) ---

func TestOpenSoftwareLaunchesCommandAndRecords(t *testing.T) {
	dir := t.TempDir()
	sw := openactions.Software{Exe: "code", Args: []string{"-n"}}
	var ln fakeLauncher
	var rec fakeRecorder

	if err := openactions.OpenSoftware(dir, sw, &ln, &rec); err != nil {
		t.Fatalf("OpenSoftware: unexpected error: %v", err)
	}
	if ln.calls != 1 {
		t.Fatalf("launcher called %d times, want 1", ln.calls)
	}
	if ln.name != "code" {
		t.Fatalf("launched %q, want code", ln.name)
	}
	wantArgs := []string{"-n", dir}
	if !equalArgs(ln.args, wantArgs) {
		t.Fatalf("launcher args = %v, want %v", ln.args, wantArgs)
	}
	if len(rec.paths) != 1 || rec.paths[0] != dir {
		t.Fatalf("recorded %v, want [%s]", rec.paths, dir)
	}
}

func TestOpenSoftwarePlaceholderCommandIsLaunched(t *testing.T) {
	dir := t.TempDir()
	sw := openactions.Software{
		Exe:  "powershell.exe",
		Args: []string{"-NoExit", "-Command", "Set-Location '{folder}'"},
	}
	var ln fakeLauncher
	var rec fakeRecorder
	if err := openactions.OpenSoftware(dir, sw, &ln, &rec); err != nil {
		t.Fatalf("OpenSoftware: %v", err)
	}
	if ln.name != "powershell.exe" {
		t.Fatalf("launched %q, want powershell.exe", ln.name)
	}
	wantArgs := []string{"-NoExit", "-Command", "Set-Location '" + dir + "'"}
	if !equalArgs(ln.args, wantArgs) {
		t.Fatalf("launcher args = %v, want %v", ln.args, wantArgs)
	}
}

func TestOpenSSHSoftwareLaunchesRemoteURIAndRecords(t *testing.T) {
	sw := openactions.Software{Exe: `D:\Programs\Trae CN\Trae CN.exe`}
	var ln fakeLauncher
	var rec fakeRecorder
	const target = `172.24.245.143:/home/beishida/QL`
	if err := openactions.OpenSSHSoftware(target, "beishida", sw, &ln, &rec); err != nil {
		t.Fatal(err)
	}
	if ln.name != sw.Exe || len(ln.args) != 2 || ln.args[0] != "--folder-uri" {
		t.Fatalf("launched %q %v", ln.name, ln.args)
	}
	if len(rec.paths) != 1 || rec.paths[0] != target {
		t.Fatalf("recorded %v, want [%s]", rec.paths, target)
	}
}

func TestOpenSoftwareNonexistentPathLaunchesNothing(t *testing.T) {
	sw := openactions.Software{Exe: "code"}
	var ln fakeLauncher
	var rec fakeRecorder
	err := openactions.OpenSoftware(filepath.Join(t.TempDir(), "missing"), sw, &ln, &rec)
	if err == nil {
		t.Fatal("want error for nonexistent path, got nil")
	}
	if ln.calls != 0 || len(rec.paths) != 0 {
		t.Fatal("nonexistent path must not launch or record")
	}
}

func TestOpenSoftwareLaunchFailureDoesNotRecord(t *testing.T) {
	dir := t.TempDir()
	sw := openactions.Software{Exe: "nope"}
	ln := fakeLauncher{err: errors.New("exec: not found")}
	var rec fakeRecorder
	err := openactions.OpenSoftware(dir, sw, &ln, &rec)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want error wrapping 'not found', got %v", err)
	}
	if len(rec.paths) != 0 {
		t.Fatal("must not record when launch failed")
	}
}

func TestOpenSoftwareRecorderFailureReportedAfterLaunch(t *testing.T) {
	dir := t.TempDir()
	sw := openactions.Software{Exe: "code"}
	var ln fakeLauncher
	rec := fakeRecorder{err: errors.New("disk full")}
	err := openactions.OpenSoftware(dir, sw, &ln, &rec)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("want error wrapping 'disk full', got %v", err)
	}
	if ln.calls != 1 {
		t.Fatal("app must launch before the recorder is attempted")
	}
}
