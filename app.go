package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"cwdgo/domain/recentfolders"
	"cwdgo/internal/panel"
)

// App is the thin Wails binding layer. All behaviour lives in the domain
// packages; this struct only forwards calls and owns the runtime context.
type App struct {
	ctx   context.Context
	store *recentfolders.Store
	panel *panel.Controller
}

// NewApp wires the App binding to the Recent Folders store and the panel
// controller.
func NewApp(store *recentfolders.Store, p *panel.Controller) *App {
	return &App{store: store, panel: p}
}

// startup is called once when the Wails runtime is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.panel.SetContext(ctx)
}

// GetRecentFolders returns the Recent Folders list, newest first. The panel
// renders an empty state when this is empty.
func (a *App) GetRecentFolders() []recentfolders.Entry {
	return a.store.All()
}

// Quit shuts the application down. Safe to call from any goroutine once
// startup has run.
func (a *App) Quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
