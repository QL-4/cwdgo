// Package openactions models the Open Actions that act on a Recent Folders
// entry. The default action opens the folder in Windows Explorer; future
// Software List actions (ticket 04) will reuse the same Launcher seam.
//
// Behaviour that matters — command construction and the launch/record
// orchestration — lives here and is unit-tested. The real OS launcher is a
// thin, untested wrapper around os/exec.
package openactions

import (
	"fmt"
	"os"
	"os/exec"
)

// ExplorerExecutable is the program that opens a folder in Windows
// Explorer. Windows resolves it via PATHEXT from %SystemRoot% on PATH.
const ExplorerExecutable = "explorer"

// Launcher runs a program that opens a folder. OSLauncher runs it for real;
// tests inject a fake to exercise success/failure without spawning Explorer.
type Launcher interface {
	Launch(name string, args []string) error
}

// OSLauncher runs the command via os/exec.
type OSLauncher struct{}

// Launch starts the program detached. Explorer is a singleton: the spawned
// process forwards the request to the running instance and exits, so Start
// (not Run) is used — Run can block or report a misleading exit code. The
// handle is released to avoid leaking it, since we never Wait.
func (OSLauncher) Launch(name string, args []string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	return nil
}

// Recorder records a successful folder access. *recentfolders.Store
// satisfies it; openactions deliberately does not import recentfolders so
// the two domain packages stay decoupled.
type Recorder interface {
	Record(path string) error
}

// ExplorerCommand returns the program and arguments that open folder in
// Explorer. It is a pure function: same input always yields the same
// command, with no side effects.
func ExplorerCommand(folder string) (name string, args []string) {
	return ExplorerExecutable, []string{folder}
}

// IsExistingDir reports whether path exists and is a directory. The panel
// uses it to decide whether Enter should open the typed path directly
// (bootstrap any folder — spec story 5) instead of the selected entry.
func IsExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Open opens folder in Explorer via ln and, on a successful launch, records
// it with rec so it moves to the top of Recent Folders.
//
// It returns an error — and records nothing — if folder is not an existing
// directory or the launcher fails. A recorder failure is returned too, but
// only after the folder has already been launched.
func Open(folder string, ln Launcher, rec Recorder) error {
	if !IsExistingDir(folder) {
		return fmt.Errorf("openactions: not an existing directory: %s", folder)
	}
	name, args := ExplorerCommand(folder)
	if err := ln.Launch(name, args); err != nil {
		return fmt.Errorf("openactions: launch %s: %w", name, err)
	}
	if err := rec.Record(folder); err != nil {
		return fmt.Errorf("openactions: record %s: %w", folder, err)
	}
	return nil
}
