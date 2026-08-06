package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"cwdgo/domain/openactions"
	"cwdgo/domain/recentfolders"
	"cwdgo/domain/search"
	"cwdgo/domain/settings"
	"cwdgo/internal/autostart"
	"cwdgo/internal/launcher"
	"cwdgo/internal/panel"
)

// App is the thin Wails binding layer. All behaviour lives in the domain
// packages; this struct only forwards calls and owns the runtime context.
type App struct {
	ctx      context.Context
	store    *recentfolders.Store
	history  *settings.Store
	launcher openactions.Launcher
	software []openactions.Software
	panel    *panel.Controller
}

// NewApp wires the App binding to the Recent Folders store, the settings
// store, the launcher, the preset Software List and the panel controller.
func NewApp(store *recentfolders.Store, history *settings.Store, launcher openactions.Launcher, software []openactions.Software, p *panel.Controller) *App {
	return &App{store: store, history: history, launcher: launcher, software: software, panel: p}
}

// startup is called once when the Wails runtime is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.panel.SetContext(ctx)
}

// Folder is the view model the panel renders for one Recent Folders entry.
// The domain Entry derives its display name via a method that JSON does not
// serialize, so Name and Path are projected here for the frontend.
type Folder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// toFolders projects domain entries into the frontend view model. It always
// returns a non-nil slice so the frontend receives [] (not null) when empty.
func toFolders(entries []recentfolders.Entry) []Folder {
	out := make([]Folder, 0, len(entries))
	for _, e := range entries {
		out = append(out, Folder{Name: e.Name(), Path: e.Path})
	}
	return out
}

// GetRecentFolders returns the Recent Folders list, newest first, as the
// frontend view model. The panel renders an empty state when this is empty.
func (a *App) GetRecentFolders() []Folder {
	return toFolders(a.store.All())
}

// Search returns the Recent Folders matching query (fuzzy, case-insensitive,
// over name and full path), best match first. An empty query returns every
// folder newest first, so the panel reuses it to reset the list.
func (a *App) Search(query string) []Folder {
	return toFolders(search.Search(a.store.All(), query))
}

// IsDirectory reports whether path is an existing directory. The panel uses
// it to decide whether Enter should open the typed path directly (spec story
// 5: open any folder by pasting its path) instead of the selected entry.
func (a *App) IsDirectory(path string) bool {
	return openactions.IsExistingDir(path)
}

// Open opens folder in Windows Explorer and, on success, records it so it
// moves to the top of Recent Folders. It returns an error — recording
// nothing — if folder is not an existing directory or Explorer could not be
// launched.
func (a *App) Open(folder string) error {
	return openactions.Open(folder, a.launcher, a.store)
}

// GetSoftwareList returns the preset Software List the panel renders as the
// numbered actions (keys 1-9). It is built once at startup from the apps
// actually installed on this machine.
func (a *App) GetSoftwareList() []openactions.Software {
	if a.software == nil {
		return []openactions.Software{}
	}
	return a.software
}

// OpenWith opens folder with the Software List entry at index (0-based, so
// panel key 1 is index 0) and, on success, records it so it moves to the top
// of Recent Folders — the same recency behaviour as the default Explorer
// action. It returns an error if index is out of range, the folder is not an
// existing directory, or the app could not be launched.
func (a *App) OpenWith(folder string, index int) error {
	if index < 0 || index >= len(a.software) {
		return fmt.Errorf("openactions: no software action at index %d", index)
	}
	return openactions.OpenSoftware(folder, a.software[index], a.launcher, a.store)
}

// GetSettings returns the persisted user settings (history cap, auto-start)
// for the settings view to render.
func (a *App) GetSettings() settings.Settings {
	return a.history.Get()
}

// SaveSettings validates and persists historyLimit and autoStart, then
// applies them live: the history cap is enforced on the store immediately
// (trimming any overflow), and the Windows auto-start Run entry is written
// or removed. It returns an error without side effects if the settings are
// invalid; if persistence succeeds but a later apply step fails, the error
// is returned but the persisted value is kept (the next launch reconciles).
func (a *App) SaveSettings(historyLimit int, autoStart bool) error {
	s := settings.Settings{HistoryLimit: historyLimit, AutoStart: autoStart}
	if err := a.history.Update(s); err != nil {
		return err
	}
	// Apply the cap live so the panel reflects the new limit at once.
	if err := a.store.SetLimit(historyLimit); err != nil {
		return err
	}
	// Apply the auto-start registry entry to match the persisted toggle.
	if autoStart {
		if err := autostart.Enable(); err != nil {
			return err
		}
	} else {
		if err := autostart.Disable(); err != nil {
			return err
		}
	}
	return nil
}

// Quit shuts the application down. Safe to call from any goroutine once
// startup has run.
func (a *App) Quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// defaultSoftware builds the preset Software List: PowerShell is always
// available on Windows (opened with -NoExit so the shell stays open in the
// folder); Antigravity and Trae CN are included only when their Start Menu
// shortcut resolves to an installed executable, so any install drive/folder
// is handled. This is platform glue (shortcut resolution / PATH probing)
// and is not unit-tested — the Software command construction and the
// launch/record path it feeds are tested in domain/openactions.
func defaultSoftware() []openactions.Software {
	out := []openactions.Software{}
	if exeAvailable("powershell.exe") {
		out = append(out, openactions.Software{
			Name: "PowerShell",
			Exe:  "powershell.exe",
			Args: []string{"-NoExit", "-Command", "Set-Location '{folder}'"},
		})
	}
	presets := launcher.ResolvePresets()
	for _, p := range launcher.Presets {
		if exe, ok := presets[p.Name]; ok {
			out = append(out, openactions.Software{Name: p.Name, Exe: exe})
		}
	}
	return out
}

// exeAvailable reports whether an executable is reachable: a bare name is
// resolved via PATH (LookPath), an absolute path is checked for existence.
func exeAvailable(exe string) bool {
	if filepath.IsAbs(exe) {
		info, err := os.Stat(exe)
		return err == nil && !info.IsDir()
	}
	_, err := exec.LookPath(exe)
	return err == nil
}
