package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"cwdgo/domain/recentfolders"
	"cwdgo/domain/settings"
	"cwdgo/internal/applog"
	"cwdgo/internal/hotkey"
	"cwdgo/internal/launcher"
	"cwdgo/internal/panel"
	"cwdgo/internal/tray"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	applog.Log("cwdgo starting")
	hist := settings.New(settingsPath())
	store := recentfolders.New(historyPath())
	// Apply the persisted history cap at startup (defaults to 50 until the
	// user changes it in settings).
	if err := store.SetLimit(hist.Get().HistoryLimit); err != nil {
		applog.Log("settings: apply history limit at startup: %v", err)
	}
	software := defaultSoftware()
	p := panel.New()
	app := NewApp(store, hist, launcher.OSLauncher{}, software, p)

	// Tray runs on its own OS thread; the Wails main loop owns the main
	// thread. Exiting via the tray menu quits the whole process.
	tray.Run(p.Open, p.OpenSettings, app.Quit)

	// Global Launcher Hotkey: default Alt+X. If the OS rejects it (e.g. it
	// is taken by another program) show a readable message and keep running
	// — the tray still opens the panel.
	stopHotkey, err := hotkey.Listen(hotkey.AltX, p.Open)
	if err != nil {
		hotkey.ReportFailure(err)
	} else {
		defer stopHotkey()
	}

	err = wails.Run(&options.App{
		Title:            "cwdgo",
		Width:            720,
		Height:           440,
		MinWidth:         720,
		MinHeight:        440,
		MaxWidth:         720,
		MaxHeight:        440,
		DisableResize:    true,
		Frameless:        true,
		AlwaysOnTop:      true,
		StartHidden:      true,
		BackgroundColour: &options.RGBA{R: 0x1E, G: 0x22, B: 0x2E, A: 0xFF},
		Logger:           fileLogger{},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
	})
	if err != nil {
		os.Exit(1)
	}
}

// historyPath returns the JSON file that persists Recent Folders:
// %APPDATA%\cwdgo\history.json.
func historyPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "cwdgo", "history.json")
}

// settingsPath returns the JSON file that persists Settings:
// %APPDATA%\cwdgo\settings.json.
func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "cwdgo", "settings.json")
}

// fileLogger routes Wails log output (including frontend LogDebug calls)
// into the app log file so release builds stay diagnosable.
type fileLogger struct{}

func (fileLogger) Print(m string)   { applog.Log("%s", m) }
func (fileLogger) Trace(m string)   { applog.Log("TRACE %s", m) }
func (fileLogger) Debug(m string)   { applog.Log("DEBUG %s", m) }
func (fileLogger) Info(m string)    { applog.Log("INFO %s", m) }
func (fileLogger) Warning(m string) { applog.Log("WARN %s", m) }
func (fileLogger) Error(m string)   { applog.Log("ERROR %s", m) }
func (fileLogger) Fatal(m string)   { applog.Log("FATAL %s", m) }

var _ logger.Logger = fileLogger{}
