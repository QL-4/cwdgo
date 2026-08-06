package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"cwdgo/domain/openactions"
	"cwdgo/domain/recentfolders"
	"cwdgo/domain/search"
	"cwdgo/domain/settings"
	"cwdgo/internal/autostart"
	"cwdgo/internal/folderscan"
	"cwdgo/internal/launcher"
	"cwdgo/internal/panel"
)

// App is the thin Wails binding layer. All behaviour lives in the domain
// packages; this struct only forwards calls and owns the runtime context.
type App struct {
	ctx      context.Context
	store    *recentfolders.Store
	settings *settings.Store
	launcher openactions.Launcher
	panel    *panel.Controller
}

// NewApp wires the App binding to the Recent Folders store, the settings
// store, the launcher and the panel controller.
func NewApp(store *recentfolders.Store, settings *settings.Store, launcher openactions.Launcher, p *panel.Controller) *App {
	return &App{store: store, settings: settings, launcher: launcher, panel: p}
}

// startup is called once when the Wails runtime is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.panel.SetContext(ctx)
}

// Folder is the view model the panel renders for one result.
// Recorded distinguishes entries already in Recent Folders (true) from
// folders discovered live from the filesystem by path completion (false),
// so the panel can render them differently. The domain Entry derives its
// display name via a method that JSON does not serialize, so Name and Path
// are projected here for the frontend.
type Folder struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Recorded bool   `json:"recorded"`
}

// toFolders projects domain entries into the frontend view model, all marked
// Recorded (they come from history). It always returns a non-nil slice so
// the frontend receives [] (not null) when empty.
func toFolders(entries []recentfolders.Entry) []Folder {
	out := make([]Folder, 0, len(entries))
	for _, e := range entries {
		out = append(out, Folder{Name: e.Name(), Path: e.Path, Recorded: true})
	}
	return out
}

// GetRecentFolders returns the Recent Folders list, newest first, as the
// frontend view model. The panel renders an empty state when this is empty.
func (a *App) GetRecentFolders() []Folder {
	return toFolders(a.store.All())
}

// Search returns results matching query from two sources, merged and
// de-duplicated case-insensitively:
//
//  1. Recent Folders fuzzy match (name + full path), in history order
//     (newest first, NOT reordered by relevance) — these are marked
//     Recorded.
//  2. Filesystem path completion: the query is split into a parent directory
//     and a name prefix (see folderscan.Split), and matching child folders
//     are listed. Matches already in history are marked Recorded; the rest
//     are live discoveries (Recorded=false) the panel renders distinctly so
//     the user can bookmark them with Enter.
//
// An empty query returns every history folder newest first (no filesystem
// scan), so the panel reuses it to reset the list.
func (a *App) Search(query string) []Folder {
	query = strings.TrimSpace(query)
	out := toFolders(search.Filter(a.store.All(), query))
	seen := make(map[string]bool, len(out))
	for _, f := range out {
		seen[normPath(f.Path)] = true
	}
	hist := a.historyNormSet()
	dir, prefix := folderscan.Split(query)
	if dir != "" {
		for _, p := range folderscan.Scan(dir, prefix) {
			k := normPath(p)
			if seen[k] {
				continue
			}
			seen[k] = true
			entry := recentfolders.Entry{Path: p}
			out = append(out, Folder{Name: entry.Name(), Path: p, Recorded: hist[k]})
		}
	}
	return out
}

// normPath is the case-insensitive de-dup key for a path (clean + lower).
func normPath(p string) string {
	return strings.ToLower(filepath.Clean(p))
}

// historyNormSet returns the set of normalized paths currently in history,
// used to flag filesystem completions that happen to already be recorded.
func (a *App) historyNormSet() map[string]bool {
	set := make(map[string]bool)
	for _, e := range a.store.All() {
		set[normPath(e.Path)] = true
	}
	return set
}

// Record adds folder to the top of Recent Folders without opening it, so the
// user can bookmark a path quickly (Enter in the panel) and then choose how
// to open it (click / digit key). It returns an error without recording if
// folder is not an existing directory.
func (a *App) Record(folder string) error {
	if !openactions.IsExistingDir(folder) {
		return fmt.Errorf("openactions: not an existing directory: %s", folder)
	}
	return a.store.Record(folder)
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

// HideAfterOpen closes the panel after a successful Open/OpenWith without
// restoring the pre-panel foreground window. The freshly launched process
// should inherit the foreground instead of being pushed behind the window
// the user had focused before opening the panel.
func (a *App) HideAfterOpen() {
	a.panel.HideAfterOpen()
}

// GetSoftwareList returns the preset Software List the panel renders as the
// numbered actions (keys 1-9). It is built once at startup from the apps
// actually installed on this machine.
func (a *App) GetSoftwareList() []openactions.Software {
	return toActionSoftware(a.settings.Get().Software)
}

// OpenWith opens folder with the Software List entry at index (0-based, so
// panel key 1 is index 0) and, on success, records it so it moves to the top
// of Recent Folders — the same recency behaviour as the default Explorer
// action. It returns an error if index is out of range, the folder is not an
// existing directory, or the app could not be launched.
func (a *App) OpenWith(folder string, index int) error {
	sw := toActionSoftware(a.settings.Get().Software)
	if index < 0 || index >= len(sw) {
		return fmt.Errorf("openactions: no software action at index %d", index)
	}
	return openactions.OpenSoftware(folder, sw[index], a.launcher, a.store)
}

// GetSettings returns the persisted user settings (history cap, auto-start)
// for the settings view to render.
func (a *App) GetSettings() settings.Settings {
	return a.settings.Get()
}

// SaveSettings validates and persists historyLimit and autoStart, then
// applies them live: the history cap is enforced on the store immediately
// (trimming any overflow), and the Windows auto-start Run entry is written
// or removed. It returns an error without side effects if the settings are
// invalid; if persistence succeeds but a later apply step fails, the error
// is returned but the persisted value is kept (the next launch reconciles).
func (a *App) SaveSettings(historyLimit int, autoStart bool) error {
	// Preserve the current Software List: SaveSettings is the form-driven
	// path for cap + auto-start only; software edits go through the
	// Add/Update/Delete bindings.
	cur := a.settings.Get()
	s := settings.Settings{HistoryLimit: historyLimit, AutoStart: autoStart, Software: cur.Software}
	if err := a.settings.Update(s); err != nil {
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

// AddSoftware appends a new Software List entry and persists it. Validation
// (non-empty name/exe) happens inside the settings store. The panel picks
// up the new entry on its next GetSoftwareList call / panel-opened refresh.
func (a *App) AddSoftware(name, exe string, args []string) error {
	return a.mutateSoftware(func(list []settings.Software) []settings.Software {
		return append(list, settings.Software{Name: name, Exe: exe, Args: args})
	})
}

// UpdateSoftware replaces the Software List entry at index with the given
// fields. It returns an error if index is out of range or the new fields are
// invalid.
func (a *App) UpdateSoftware(index int, name, exe string, args []string) error {
	return a.mutateSoftware(func(list []settings.Software) []settings.Software {
		list[index] = settings.Software{Name: name, Exe: exe, Args: args}
		return list
	})
}

// DeleteSoftware removes the Software List entry at index. Indices beyond the
// removed one are renumbered automatically (no holes).
func (a *App) DeleteSoftware(index int) error {
	return a.mutateSoftware(func(list []settings.Software) []settings.Software {
		return append(list[:index], list[index+1:]...)
	})
}

// mutateSoftware reads the current settings, applies fn to a copy of the
// Software List (fn must bounds-check itself), validates the whole resulting
// settings, and persists. Other settings fields (cap, auto-start) are
// preserved.
func (a *App) mutateSoftware(fn func([]settings.Software) []settings.Software) error {
	cur := a.settings.Get()
	next := fn(append([]settings.Software(nil), cur.Software...))
	return a.settings.Update(settings.Settings{
		HistoryLimit: cur.HistoryLimit,
		AutoStart:    cur.AutoStart,
		Software:     next,
	})
}

// toActionSoftware converts the persisted config model to the open-action
// executor model. The two are structurally identical but live in separate
// packages so the settings store stays independent of the action executor.
func toActionSoftware(in []settings.Software) []openactions.Software {
	if len(in) == 0 {
		return []openactions.Software{}
	}
	out := make([]openactions.Software, 0, len(in))
	for _, s := range in {
		args := s.Args
		if args == nil {
			args = []string{}
		}
		out = append(out, openactions.Software{Name: s.Name, Exe: s.Exe, Args: args})
	}
	return out
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
func defaultSoftware() []settings.Software {
	out := []settings.Software{}
	if exeAvailable("powershell.exe") {
		out = append(out, settings.Software{
			Name: "PowerShell",
			Exe:  "powershell.exe",
			Args: []string{"-NoExit", "-Command", "Set-Location '{folder}'"},
		})
	}
	presets := launcher.ResolvePresets()
	for _, p := range launcher.Presets {
		if exe, ok := presets[p.Name]; ok {
			out = append(out, settings.Software{Name: p.Name, Exe: exe})
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
