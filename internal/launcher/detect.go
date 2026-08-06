package launcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"

	"cwdgo/internal/applog"
)

// PresetApp is a preset IDE the panel offers as a numbered action when found
// installed. Link is the Start Menu shortcut name (without .lnk) to look for;
// Name is the display name shown on its action badge.
type PresetApp struct {
	Name string
	Link string
}

// Presets is the ordered list of IDE presets to detect. PowerShell is handled
// separately (always present on Windows, resolved via PATH). The order here
// fixes the key binding after PowerShell: index 0 → key 2, etc.
var Presets = []PresetApp{
	{Name: "Antigravity", Link: "Antigravity"},
	{Name: "Trae CN", Link: "Trae CN"},
}

// ResolvePresets scans the Start Menu Programs folders (user and all-users)
// for the preset IDE shortcuts and resolves each to its target executable via
// the shell shortcut API. It returns display name -> exe path for those found;
// presets whose shortcut is absent or whose target does not exist are omitted.
//
// This is best-effort platform glue (not unit-tested): any failure yields an
// empty or partial result so startup never blocks on detection. PowerShell is
// always available on Windows, so it is used to resolve the .lnk targets; if
// it is somehow unavailable, detection simply returns nothing.
func ResolvePresets() map[string]string {
	result := make(map[string]string)
	lnkByName := make(map[string]string) // preset Name -> .lnk path
	for _, dir := range programsDirs() {
		for _, lnk := range walkLnk(dir) {
			base := strings.ToLower(strings.TrimSuffix(filepath.Base(lnk), ".lnk"))
			for _, p := range Presets {
				if base == strings.ToLower(p.Link) {
					if _, ok := lnkByName[p.Name]; !ok {
						lnkByName[p.Name] = lnk
					}
				}
			}
		}
	}
	if len(lnkByName) == 0 {
		return result
	}
	for name, exe := range resolveLnks(lnkByName) {
		if exe != "" && fileExists(exe) {
			result[name] = exe
		}
	}
	return result
}

// programsDirs returns the Start Menu Programs folders that exist: the
// per-user folder (%APPDATA%\...\Start Menu\Programs) and the all-users
// folder (%ProgramData%\...\Start Menu\Programs).
func programsDirs() []string {
	var dirs []string
	roots := []string{os.Getenv("APPDATA"), os.Getenv("ProgramData")}
	for _, root := range roots {
		if root == "" {
			continue
		}
		dir := filepath.Join(root, "Microsoft", "Windows", "Start Menu", "Programs")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// walkLnk returns every .lnk file under dir (recursively).
func walkLnk(dir string) []string {
	var lnks []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".lnk") {
			lnks = append(lnks, path)
		}
		return nil
	})
	return lnks
}

// resolveLnks resolves each .lnk to its TargetPath via PowerShell's
// WScript.Shell in a single invocation. It returns preset Name -> target path.
// Each preset is emitted as one stdout line "name\ttarget".
func resolveLnks(lnkByName map[string]string) map[string]string {
	result := make(map[string]string, len(lnkByName))
	var script strings.Builder
	script.WriteString("$ErrorActionPreference='SilentlyContinue';")
	script.WriteString("$sh=New-Object -ComObject WScript.Shell;")
	// Stable order for readable output.
	names := make([]string, 0, len(lnkByName))
	for name := range lnkByName {
		names = append(names, name)
	}
	for _, name := range names {
		lnk := lnkByName[name]
		script.WriteString("Write-Output ('")
		script.WriteString(psSingleQuote(name))
		script.WriteString("' + \"`t\" + $sh.CreateShortcut('")
		script.WriteString(psSingleQuote(lnk))
		script.WriteString("').TargetPath);")
	}
	out, err := runPowerShell(script.String())
	if err != nil {
		applog.Log("launcher: preset resolve failed: %v", err)
		return result
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		i := strings.IndexByte(line, '\t')
		if i < 0 {
			continue
		}
		name, target := line[:i], line[i+1:]
		if name != "" && target != "" {
			result[name] = target
		}
	}
	return result
}

// runPowerShell executes a PowerShell script via -EncodedCommand (UTF-16LE
// base64) so no shell quoting of the script body is needed, and returns its
// stdout. Non-interactive: the process runs, prints, and exits.
func runPowerShell(script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-EncodedCommand", encodeUTF16LEBase64(script))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

// psSingleQuote makes a Go string safe to embed inside a PowerShell
// single-quoted string literal, by doubling embedded single quotes.
func psSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// encodeUTF16LEBase64 encodes s as UTF-16 little-endian then base64, the form
// PowerShell's -EncodedCommand expects.
func encodeUTF16LEBase64(s string) string {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// fileExists reports whether path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
