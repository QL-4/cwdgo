package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"cwdgo/domain/openactions"
	"cwdgo/domain/recentfolders"
	"cwdgo/domain/search"
	"cwdgo/internal/panel"
)

// App is the thin Wails binding layer. All behaviour lives in the domain
// packages; this struct only forwards calls and owns the runtime context.
type App struct {
	ctx      context.Context
	store    *recentfolders.Store
	launcher openactions.Launcher
	panel    *panel.Controller
}

// NewApp wires the App binding to the Recent Folders store, the Explorer
// launcher and the panel controller.
func NewApp(store *recentfolders.Store, launcher openactions.Launcher, p *panel.Controller) *App {
	return &App{store: store, launcher: launcher, panel: p}
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

// Quit shuts the application down. Safe to call from any goroutine once
// startup has run.
func (a *App) Quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
