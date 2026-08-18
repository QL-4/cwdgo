// Package openactions models the Open Actions that act on a Recent Folders
// entry. The default action opens the folder in Windows Explorer; future
// Software List actions (ticket 04) will reuse the same Launcher seam.
//
// Behaviour that matters — command construction and the launch/record
// orchestration — lives here and is unit-tested. The real OS launcher is
// isolated in internal/launcher behind the Launcher seam.
package openactions

import (
	"fmt"
	"os"
	"strings"
)

// ExplorerExecutable is the program that opens a folder in Windows
// Explorer. Windows resolves it via PATHEXT from %SystemRoot% on PATH.
const ExplorerExecutable = "explorer"

// Launcher runs a program that opens a folder. The real Windows implementation
// lives in internal/launcher, deliberately outside the domain package to keep
// platform syscalls out of the domain. Tests inject a fake to exercise
// success/failure without launching anything.
type Launcher interface {
	Launch(name string, args []string) error
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

// IsExistingDir reports whether path exists and is a local directory.
func IsExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsSSHFolder reports whether path uses cwdgo's SSH project notation:
// host-or-ip:/absolute/remote/path.
func IsSSHFolder(path string) bool {
	colon := strings.Index(path, ":/")
	if colon <= 1 {
		return false
	}
	host, remotePath := path[:colon], path[colon+1:]
	return !strings.ContainsAny(host, `\\/`) && strings.HasPrefix(remotePath, "/")
}

// Open opens folder in Explorer via ln and, on a successful launch, records
// it with rec so it moves to the top of Recent Folders.
//
// It returns an error — and records nothing — if folder is not an existing
// directory or the launcher fails. A recorder failure is returned too, but
// only after the folder has already been launched.
func Open(folder string, ln Launcher, rec Recorder) error {
	name, args := ExplorerCommand(folder)
	return launchAndRecord(folder, name, args, ln, rec)
}

// launchAndRecord validates folder, launches name/args via ln, and records
// the access on success. It is the shared body of the Explorer Open and the
// Software OpenSoftware actions, which differ only in the command. The
// validation, launch-then-record ordering, and error wrapping are identical
// for every Open Action.
func launchAndRecord(folder, name string, args []string, ln Launcher, rec Recorder) error {
	if !IsExistingDir(folder) {
		return fmt.Errorf("openactions: not an existing directory: %s", folder)
	}
	if err := ln.Launch(name, args); err != nil {
		return fmt.Errorf("openactions: launch %s: %w", name, err)
	}
	if err := rec.Record(folder); err != nil {
		return fmt.Errorf("openactions: record %s: %w", folder, err)
	}
	return nil
}
