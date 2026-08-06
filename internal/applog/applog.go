// Package applog appends timestamped lines to %APPDATA%\cwdgo\cwdgo.log.
// It exists for diagnosing the glue layers (tray, hotkey, panel) that have
// no console in release builds; failures there are otherwise invisible.
package applog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Log appends a line to the log file, creating it if needed. Failures to
// write are ignored: logging must never break the app.
func Log(format string, args ...any) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	path := filepath.Join(dir, "cwdgo", "cwdgo.log")
	line := fmt.Sprintf("%s %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}
