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
	// Add configured Trae CN Remote SSH projects once. Each persisted path
	// uses the requested IP notation while Name stores the SSH config alias
	// required by Trae CN's Remote SSH URI.
	for _, project := range []struct {
		Name string
		Path string
	}{
		{Name: "beishida", Path: "172.24.245.143:/home/beishida/QL"},
		{Name: "cscgbnu", Path: "172.24.245.143:/home/cscgbnu/QL"},
		{Name: "vps", Path: "23.95.48.216:/root/QL"},
	} {
		if _, exists := store.Find(project.Path); exists {
			continue
		}
		if err := store.RecordNamed(project.Path, project.Name); err != nil {
			applog.Log("history: seed %s SSH project: %v", project.Name, err)
		}
	}
	// Seed the Software List on first run: when settings has no software
	// entries yet (fresh settings.json or defaults), persist the detected
	// presets so they become user-manageable. Subsequent launches keep the
	// user's edits, even if a preset is later uninstalled.
	seedSoftware(hist)
	p := panel.New()
	app := NewApp(store, hist, launcher.OSLauncher{}, p)

	// Tray runs on its own OS thread; the Wails main loop owns the main
	// thread. Exiting via the tray menu quits the whole process.
	tray.Run(tray.Callbacks{
		OnLeftClick: p.ToggleFromTray,
		OnOpen:      p.Open,
		OnSettings:  p.OpenSettings,
		OnExit:      app.Quit,
	})

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
		applog.Log("wails.Run failed: %v", err)
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

// seedSoftware persists the detected preset Software List on first run
// only — when the settings store holds no software entries yet. After this
// one-time seed the list is fully user-owned: edits/deletes persist and are
// not re-seeded even if a preset is later uninstalled.
func seedSoftware(s *settings.Store) {
	cur := s.Get()
	if len(cur.Software) > 0 {
		return // already seeded or user-managed
	}
	presets := defaultSoftware()
	if len(presets) == 0 {
		return
	}
	if err := s.Update(settings.Settings{
		HistoryLimit: cur.HistoryLimit,
		AutoStart:    cur.AutoStart,
		Software:     presets,
	}); err != nil {
		applog.Log("settings: seed software list: %v", err)
	}
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
