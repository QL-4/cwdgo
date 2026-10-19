package openactions_test

import (
	"errors"
	"net/url"
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

func TestSoftwareCommandURIPlaceholderIsQueryEscaped(t *testing.T) {
	sw := openactions.Software{
		Exe:  `C:\Programs\ZCode\ZCode.exe`,
		Args: []string{"zcode://workspace/open?path={folderUri}"},
	}
	_, args := sw.Command(`D:\My Projects\foo&bar`)
	want := []string{"zcode://workspace/open?path=" + url.QueryEscape(`D:\My Projects\foo&bar`)}
	if !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	// the folder must not also be appended positionally
	if len(args) != 1 {
		t.Fatalf("got %d args, want 1 (no duplicate folder append)", len(args))
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
	got, err := openactions.SSHFolderURI(`203.0.113.10:/home/server-a/project`, "server-a")
	if err != nil {
		t.Fatal(err)
	}
	want := "vscode-remote://ssh-remote%2B7b22686f73744e616d65223a227365727665722d61222c2266726f6d436f6e666967223a747275657d/home/server-a/project"
	if got != want {
		t.Fatalf("SSHFolderURI() = %q, want %q", got, want)
	}
}

func TestRemoteCommandUsesFolderURI(t *testing.T) {
	sw := openactions.Software{
		Exe:        `D:\Programs\Trae CN\Trae CN.exe`,
		RemoteArgs: []string{"--folder-uri", openactions.SSHURIPlaceholder},
	}
	name, args, err := sw.RemoteCommand(`203.0.113.10:/home/server-a/project`, "server-a")
	if err != nil {
		t.Fatal(err)
	}
	if name != sw.Exe || len(args) != 2 || args[0] != "--folder-uri" || !strings.Contains(args[1], "/home/server-a/project") {
		t.Fatalf("RemoteCommand() = %q %v", name, args)
	}
}

func TestRemoteCommandAppendsURIWhenNoPlaceholder(t *testing.T) {
	sw := openactions.Software{Exe: "code", RemoteArgs: []string{"--folder-uri"}}
	_, args, err := sw.RemoteCommand(`203.0.113.10:/home/server-a/project`, "server-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || args[0] != "--folder-uri" || !strings.Contains(args[1], "vscode-remote://") {
		t.Fatalf("RemoteCommand() args = %v, want URI appended after --folder-uri", args)
	}
}

func TestRemoteCommandWithoutRemoteArgsErrors(t *testing.T) {
	sw := openactions.Software{Exe: "code"}
	if _, _, err := sw.RemoteCommand(`203.0.113.10:/home/server-a/project`, "server-a"); err == nil {
		t.Fatal("want error for SSH target without RemoteArgs, got nil")
	}
}

func TestRemoteCommandLocalFolderUsesCommand(t *testing.T) {
	sw := openactions.Software{Exe: "code"}
	_, args, err := sw.RemoteCommand(`D:\foo`, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{`D:\foo`}; !equalArgs(args, want) {
		t.Fatalf("args = %v, want %v (local folders need no RemoteArgs)", args, want)
	}
}

func TestCanOpenSSHRequiresRemoteArgs(t *testing.T) {
	if (openactions.Software{Exe: "code", Name: "Trae"}).CanOpenSSH() {
		t.Fatal("entry without RemoteArgs must not be SSH-capable, whatever its name")
	}
	if !(openactions.Software{Exe: "code", RemoteArgs: []string{"--folder-uri"}}).CanOpenSSH() {
		t.Fatal("entry with RemoteArgs must be SSH-capable")
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
	sw := openactions.Software{
		Exe:        `D:\Programs\Trae CN\Trae CN.exe`,
		RemoteArgs: []string{"--folder-uri", openactions.SSHURIPlaceholder},
	}
	var ln fakeLauncher
	var rec fakeRecorder
	const target = `203.0.113.10:/home/server-a/project`
	if err := openactions.OpenSSHSoftware(target, "server-a", sw, &ln, &rec); err != nil {
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
