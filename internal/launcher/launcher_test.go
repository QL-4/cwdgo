package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// TestOSLauncherUsesFreshUserEnvironment is the regression test for cwdgo's
// long-lived-process bug: environment variables can change in the registry
// after cwdgo starts, and newly launched programs must receive those current
// values rather than cwdgo's stale process snapshot.
func TestOSLauncherUsesFreshUserEnvironment(t *testing.T) {
	name := fmt.Sprintf("CWDGO_LAUNCHER_ENV_%d", os.Getpid())
	staleValue := "stale-parent-value"
	freshValue := "fresh-registry-value"
	t.Setenv(name, staleValue)

	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		t.Fatalf("open HKCU\\Environment: %v", err)
	}
	defer key.Close()
	previous, previousType, previousErr := key.GetStringValue(name)
	defer func() {
		if previousErr == registry.ErrNotExist {
			_ = key.DeleteValue(name)
			return
		}
		if previousErr == nil {
			if previousType == registry.EXPAND_SZ {
				_ = key.SetExpandStringValue(name, previous)
			} else {
				_ = key.SetStringValue(name, previous)
			}
		}
	}()
	if err := key.SetStringValue(name, freshValue); err != nil {
		t.Fatalf("write test environment variable: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("test executable: %v", err)
	}
	output := filepath.Join(t.TempDir(), "child-env.txt")
	args := []string{"-test.run=^TestOSLauncherEnvironmentHelper$", "--", name, output}
	if err := (OSLauncher{}).Launch(exe, args); err != nil {
		t.Fatalf("Launch helper: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := os.ReadFile(output)
		if err == nil {
			if value := string(got); value != freshValue {
				t.Fatalf("child environment %s = %q, want fresh registry value %q", name, value, freshValue)
			}
			return
		}
		if !os.IsNotExist(err) {
			t.Fatalf("read helper output: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for launched helper")
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestResolveExecutableUsesProvidedFreshPath(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "newly-installed.exe")
	if err := os.WriteFile(exe, []byte("not actually executed"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveExecutable("newly-installed", []string{"PATH=" + dir, "PATHEXT=.EXE"})
	if err != nil {
		t.Fatalf("resolveExecutable: %v", err)
	}
	want, err := filepath.Abs(exe)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(got, want) {
		t.Fatalf("resolved %q, want %q", got, want)
	}
}

func TestNativeCommandWrapsBatchFilesWithSystemCmd(t *testing.T) {
	exe, args, err := nativeCommand(`C:\tools\open-folder.cmd`, []string{"arg with spaces"})
	if err != nil {
		t.Fatalf("nativeCommand: %v", err)
	}
	systemDir, err := windows.GetSystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	wantExe := filepath.Join(systemDir, "cmd.exe")
	if !strings.EqualFold(exe, wantExe) {
		t.Fatalf("exe = %q, want %q", exe, wantExe)
	}
	wantArgs := []string{"/d", "/c", `C:\tools\open-folder.cmd`, "arg with spaces"}
	if strings.Join(args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("args = %q, want %q", args, wantArgs)
	}
}

// TestOSLauncherEnvironmentHelper is launched as a separate process by the
// regression test above. It records one inherited environment value so the
// parent test can assert which environment block OSLauncher supplied.
func TestOSLauncherEnvironmentHelper(t *testing.T) {
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 {
		return // helper-only test; a normal package test run has no payload
	}
	if len(os.Args) != separator+3 {
		t.Fatalf("helper args = %q, want -- <environment-name> <output-path>", strings.Join(os.Args, " "))
	}
	name, output := os.Args[separator+1], os.Args[separator+2]
	if err := os.WriteFile(output, []byte(os.Getenv(name)), 0o600); err != nil {
		t.Fatalf("write helper output: %v", err)
	}
}
