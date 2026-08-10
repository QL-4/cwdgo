// Package launcher provides the real Launcher implementation that starts
// Open Action programs on Windows. cwdgo is long-lived, so each launch builds
// a fresh registry-derived environment block for the current user instead of
// passing on the environment snapshot cwdgo inherited when it started.
//
// Programs are started with CreateProcessW and CREATE_NEW_CONSOLE. The latter
// gives interactive console apps such as PowerShell independent standard I/O;
// os/exec would otherwise attach os.DevNull handles because cwdgo is a GUI
// subsystem process, making an interactive host exit immediately.
package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"cwdgo/domain/openactions"

	"golang.org/x/sys/windows"
)

// OSLauncher implements openactions.Launcher with CreateProcessW. Unlike
// ShellExecuteW (which inherits cwdgo's stale process environment), every
// launch receives a new environment block built from the current system and
// user registry settings.
type OSLauncher struct{}

// Launch starts name with args in a new console and gives it the current
// registry-derived user environment. A bare executable name is resolved
// against that fresh PATH/PATHEXT, so software installed after cwdgo started
// can be launched without restarting cwdgo.
func (OSLauncher) Launch(name string, args []string) error {
	token := windows.GetCurrentProcessToken()
	var envBlock *uint16
	if err := windows.CreateEnvironmentBlock(&envBlock, token, false); err != nil {
		return fmt.Errorf("launcher: build environment block: %w", err)
	}
	defer windows.DestroyEnvironmentBlock(envBlock)

	exe, err := resolveExecutable(name, environmentStrings(envBlock))
	if err != nil {
		return fmt.Errorf("launcher: resolve %s: %w", name, err)
	}
	exe, args, err = nativeCommand(exe, args)
	if err != nil {
		return fmt.Errorf("launcher: prepare %s: %w", name, err)
	}
	app, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return fmt.Errorf("launcher: executable path: %w", err)
	}
	commandLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{exe}, args...)))
	if err != nil {
		return fmt.Errorf("launcher: command line: %w", err)
	}

	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var process windows.ProcessInformation
	if err := windows.CreateProcess(
		app,
		commandLine,
		nil,
		nil,
		false,
		windows.CREATE_NEW_CONSOLE|windows.CREATE_UNICODE_ENVIRONMENT,
		envBlock,
		nil,
		&startup,
		&process,
	); err != nil {
		return fmt.Errorf("launcher: start %s: %w", exe, err)
	}
	windows.CloseHandle(process.Thread)
	windows.CloseHandle(process.Process)
	return nil
}

// environmentStrings projects a CreateEnvironmentBlock result into the form
// used by executable resolution. The block remains owned by the caller.
func environmentStrings(block *uint16) []string {
	var env []string
	const elementSize = unsafe.Sizeof(*block)
	for *block != 0 {
		end := unsafe.Pointer(block)
		for *(*uint16)(end) != 0 {
			end = unsafe.Add(end, elementSize)
		}
		entry := unsafe.Slice(block, (uintptr(end)-uintptr(unsafe.Pointer(block)))/elementSize)
		env = append(env, windows.UTF16ToString(entry))
		block = (*uint16)(unsafe.Add(end, elementSize))
	}
	return env
}

// resolveExecutable resolves a bare command name against PATH and PATHEXT from
// env. Names that already contain a directory are checked directly (plus
// PATHEXT candidates when no extension is present).
func resolveExecutable(name string, env []string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty executable name")
	}
	paths := []string{""}
	if !strings.ContainsAny(name, `:\/`) {
		paths = executableSearchDirs(env)
	}
	exts := executableExtensions(name, envValue(env, "PATHEXT"))
	for _, dir := range paths {
		candidate := name
		if dir != "" {
			candidate = filepath.Join(dir, name)
		}
		for _, ext := range exts {
			path := candidate + ext
			info, err := os.Stat(path)
			if err == nil && !info.IsDir() {
				abs, err := filepath.Abs(path)
				if err != nil {
					return "", err
				}
				return abs, nil
			}
		}
	}
	return "", os.ErrNotExist
}

// executableSearchDirs approximates CreateProcess's executable search order,
// but uses PATH from the fresh environment instead of cwdgo's process.
func executableSearchDirs(env []string) []string {
	dirs := []string{}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}
	if systemDir, err := windows.GetSystemDirectory(); err == nil {
		dirs = append(dirs, systemDir)
	}
	if windowsDir, err := windows.GetWindowsDirectory(); err == nil {
		dirs = append(dirs, filepath.Join(windowsDir, "System"), windowsDir)
	}
	dirs = append(dirs, filepath.SplitList(envValue(env, "PATH"))...)
	return dirs
}

// nativeCommand turns a batch-file action into an explicit invocation of the
// trusted system command interpreter. CreateProcess cannot execute .cmd/.bat
// files directly, unlike ShellExecute.
func nativeCommand(exe string, args []string) (string, []string, error) {
	ext := strings.ToLower(filepath.Ext(exe))
	if ext != ".cmd" && ext != ".bat" {
		return exe, args, nil
	}
	systemDir, err := windows.GetSystemDirectory()
	if err != nil {
		return "", nil, err
	}
	cmd := filepath.Join(systemDir, "cmd.exe")
	return cmd, append([]string{"/d", "/c", exe}, args...), nil
}

// executableExtensions returns the suffixes to probe. A name that already has
// an extension is tried as-is first; extensionless names use fresh PATHEXT.
func executableExtensions(name, pathExt string) []string {
	if filepath.Ext(name) != "" {
		return []string{""}
	}
	if pathExt == "" {
		pathExt = ".COM;.EXE;.BAT;.CMD"
	}
	exts := make([]string, 0, 4)
	for _, ext := range filepath.SplitList(pathExt) {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		if ext[0] != '.' {
			ext = "." + ext
		}
		exts = append(exts, ext)
	}
	return exts
}

// envValue reads one value from a Windows environment slice
// case-insensitively. Hidden drive-current-directory entries (=C:=...) are
// ignored because they are not normal variables.
func envValue(env []string, name string) string {
	for _, entry := range env {
		if strings.HasPrefix(entry, "=") {
			continue
		}
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

// Compile-time check that OSLauncher satisfies the domain Launcher seam.
var _ openactions.Launcher = OSLauncher{}
