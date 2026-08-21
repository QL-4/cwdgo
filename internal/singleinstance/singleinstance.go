// Package singleinstance gives cwdgo replace-style single-instance
// behaviour: a fresh start terminates every older process with the same
// executable file name, so the global hotkey and tray icon are always owned
// by exactly one process — the most recently started one.
//
// A process counts as "previous" when its image file name matches ours and
// it was created before this process. Matching by name (not full path) also
// replaces an older build running from a different folder; ordering by
// creation time keeps rapid double-launch races safe: the newest start
// survives, every earlier one is terminated.
//
// This is OS glue (process snapshot + terminate) and is not unit-tested;
// failures are logged and ignored because a failed cleanup must never block
// the new instance from starting.
package singleinstance

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"cwdgo/internal/applog"

	"golang.org/x/sys/windows"
)

// exitWaitTimeout is how long a terminated older process is given to
// disappear, so resources it holds (global hotkey, tray icon) are released
// before the new instance registers its own.
const exitWaitTimeout = 2 * time.Second

// ReplacePrevious terminates every process whose executable file name
// matches this one's and that was started before this process, then waits
// for each to exit. It never kills the calling process.
func ReplacePrevious() {
	self, err := os.Executable()
	if err != nil {
		applog.Log("singleinstance: resolve own executable: %v", err)
		return
	}
	selfName := strings.ToLower(filepath.Base(self))
	selfCreate, err := ownCreationTime()
	if err != nil {
		// Without our own creation time we cannot order processes
		// reliably; do nothing rather than risk killing the wrong one.
		applog.Log("singleinstance: read own creation time: %v", err)
		return
	}
	me := windows.GetCurrentProcessId()

	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		applog.Log("singleinstance: process snapshot: %v", err)
		return
	}
	defer windows.CloseHandle(snap)

	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Process32First(snap, &e); err != nil {
		applog.Log("singleinstance: enumerate processes: %v", err)
		return
	}
	for {
		if e.ProcessID != me && e.ProcessID != 0 &&
			strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), selfName) {
			terminateIfOlder(e.ProcessID, selfCreate)
		}
		if err := windows.Process32Next(snap, &e); err != nil {
			break
		}
	}
}

// ownCreationTime returns this process's creation time.
func ownCreationTime() (windows.Filetime, error) {
	var create, exitTime, kernel, user windows.Filetime
	err := windows.GetProcessTimes(windows.CurrentProcess(), &create, &exitTime, &kernel, &user)
	return create, err
}

// terminateIfOlder terminates pid when it was created before selfCreate,
// then waits for the exit. A pid that cannot be opened or queried, or that
// is not older than this process, is skipped with a log line.
func terminateIfOlder(pid uint32, selfCreate windows.Filetime) {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, pid)
	if err != nil {
		// Another user's or a protected process — not ours to kill.
		applog.Log("singleinstance: open pid %d: %v", pid, err)
		return
	}
	defer windows.CloseHandle(h)

	var create, exitTime, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &create, &exitTime, &kernel, &user); err != nil {
		applog.Log("singleinstance: query pid %d times: %v", pid, err)
		return
	}
	if ftKey(create) >= ftKey(selfCreate) {
		return // started after (or with) us — not a previous instance
	}
	image := imagePath(h) // query before terminating; a dead process has none
	if err := windows.TerminateProcess(h, 1); err != nil {
		applog.Log("singleinstance: terminate pid %d: %v", pid, err)
		return
	}
	if event, err := windows.WaitForSingleObject(h, uint32(exitWaitTimeout.Milliseconds())); err != nil || event == uint32(windows.WAIT_TIMEOUT) {
		applog.Log("singleinstance: pid %d did not exit in time", pid)
		return
	}
	applog.Log("singleinstance: replaced previous instance (pid %d, %s)", pid, image)
}

// ftKey projects a Filetime into a comparable uint64 for ordering.
func ftKey(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

// imagePath returns pid's full executable path for logging; "" when it
// cannot be queried.
func imagePath(h windows.Handle) string {
	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}
